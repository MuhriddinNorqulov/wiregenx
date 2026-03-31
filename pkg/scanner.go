package pkg

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var (
	reInject      = regexp.MustCompile(`^\s*//\s*@Inject(?:\((\w+)\))?\s*$`)
	reApplication = regexp.MustCompile(`^\s*//\s*@Application\s*$`)
)

// scanProviders walks the directory tree and finds all annotated provider functions.
func scanProviders(root string, ignoreVendor, ignoreHidden bool) ([]Provider, error) {
	var out []Provider
	fset := token.NewFileSet()

	err := filepath.WalkDir(root, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}

		if d.IsDir() {
			name := d.Name()
			if ignoreHidden && strings.HasPrefix(name, ".") {
				return filepath.SkipDir
			}
			if ignoreVendor && name == "vendor" {
				return filepath.SkipDir
			}
			return nil
		}

		if filepath.Ext(path) != ".go" || strings.HasSuffix(path, "_test.go") {
			return nil
		}

		file, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
		if err != nil {
			return fmt.Errorf("parse %s: %w", path, err)
		}

		pkgName := file.Name.Name
		imports := buildImportMap(file)

		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Doc == nil || fn.Recv != nil {
				continue
			}

			ann := parseAnnotations(fn.Doc)
			if !ann.isProvider {
				continue
			}

			params := extractParams(fn.Type.Params, imports)

			retType, returnsError, err := extractReturnType(fn.Type.Results, imports)
			if err != nil {
				return fmt.Errorf("%s: %s: %w", path, fn.Name.Name, err)
			}

			scope := ScopeSingleton
			if ann.scope != "" {
				scope = ann.scope
			}

			out = append(out, Provider{
				FuncName:     fn.Name.Name,
				PkgName:      pkgName,
				File:         path,
				Params:       params,
				ReturnType:   retType,
				Scope:        scope,
				ReturnsError: returnsError,
				IsApp:        ann.isApp,
			})
		}

		return nil
	})

	if err != nil {
		return nil, err
	}
	return out, nil
}

type annotations struct {
	isProvider bool
	isApp      bool
	scope      Scope
}

func parseAnnotations(cg *ast.CommentGroup) annotations {
	var a annotations
	for _, c := range cg.List {
		switch {
		case reInject.MatchString(c.Text):
			a.isProvider = true
			matches := reInject.FindStringSubmatch(c.Text)
			if len(matches) >= 2 && matches[1] != "" {
				switch strings.ToLower(matches[1]) {
				case "prototype":
					a.scope = ScopePrototype
				case "singleton":
					a.scope = ScopeSingleton
				}
			}
		case reApplication.MatchString(c.Text):
			a.isProvider = true
			a.isApp = true
		}
	}
	return a
}

// buildImportMap creates a mapping from local alias/name to full import path.
func buildImportMap(file *ast.File) map[string]string {
	m := make(map[string]string)
	for _, imp := range file.Imports {
		path := strings.Trim(imp.Path.Value, `"`)
		if imp.Name != nil {
			m[imp.Name.Name] = path
		} else {
			seg := path
			if idx := strings.LastIndex(seg, "/"); idx >= 0 {
				seg = seg[idx+1:]
			}
			m[seg] = path
		}
	}
	return m
}

// extractParams extracts Param (name + type) for each function parameter.
func extractParams(fields *ast.FieldList, imports map[string]string) []Param {
	if fields == nil {
		return nil
	}
	var params []Param
	for _, field := range fields.List {
		tr := resolveTypeExpr(field.Type, imports)
		if len(field.Names) == 0 {
			params = append(params, Param{Type: tr})
		} else {
			for _, name := range field.Names {
				params = append(params, Param{Name: name.Name, Type: tr})
			}
		}
	}
	return params
}

// extractReturnType extracts the primary return type.
// Auto-detects (T, error) signatures → returnsError=true.
func extractReturnType(results *ast.FieldList, imports map[string]string) (TypeRef, bool, error) {
	if results == nil || len(results.List) == 0 {
		return TypeRef{}, false, fmt.Errorf("provider function must have a return type")
	}

	if len(results.List) == 1 {
		return resolveTypeExpr(results.List[0].Type, imports), false, nil
	}

	if len(results.List) == 2 {
		second := resolveTypeExpr(results.List[1].Type, imports)
		if second.TypeName == "error" && second.ImportPath == "" && !second.IsPointer {
			return resolveTypeExpr(results.List[0].Type, imports), true, nil
		}
		return TypeRef{}, false, fmt.Errorf("second return value must be error, got %s", second.Raw)
	}

	return TypeRef{}, false, fmt.Errorf("provider function must return 1 or 2 values, got %d", len(results.List))
}

var builtinTypes = map[string]bool{
	"bool": true, "byte": true, "complex64": true, "complex128": true,
	"error": true, "float32": true, "float64": true,
	"int": true, "int8": true, "int16": true, "int32": true, "int64": true,
	"rune": true, "string": true,
	"uint": true, "uint8": true, "uint16": true, "uint32": true, "uint64": true, "uintptr": true,
	"any": true,
}

// resolveTypeExpr converts an AST type expression to a TypeRef.
func resolveTypeExpr(expr ast.Expr, imports map[string]string) TypeRef {
	switch t := expr.(type) {
	case *ast.StarExpr:
		inner := resolveTypeExpr(t.X, imports)
		inner.IsPointer = true
		inner.Raw = "*" + inner.Raw
		return inner

	case *ast.SelectorExpr:
		ident, ok := t.X.(*ast.Ident)
		if !ok {
			return TypeRef{Raw: formatExpr(expr)}
		}
		pkgAlias := ident.Name
		typeName := t.Sel.Name
		importPath := imports[pkgAlias]
		return TypeRef{
			ImportPath: importPath,
			TypeName:   typeName,
			Raw:        pkgAlias + "." + typeName,
		}

	case *ast.Ident:
		name := t.Name
		if builtinTypes[name] {
			return TypeRef{TypeName: name, Raw: name}
		}
		return TypeRef{TypeName: name, Raw: name}

	case *ast.ArrayType:
		elem := resolveTypeExpr(t.Elt, imports)
		return TypeRef{
			ImportPath: elem.ImportPath,
			TypeName:   elem.TypeName,
			IsPointer:  elem.IsPointer,
			Raw:        "[]" + elem.Raw,
		}

	case *ast.MapType:
		return TypeRef{Raw: formatExpr(expr)}

	case *ast.InterfaceType:
		return TypeRef{TypeName: "interface{}", Raw: "interface{}"}

	default:
		return TypeRef{Raw: formatExpr(expr)}
	}
}

func formatExpr(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.StarExpr:
		return "*" + formatExpr(t.X)
	case *ast.SelectorExpr:
		return formatExpr(t.X) + "." + t.Sel.Name
	case *ast.ArrayType:
		return "[]" + formatExpr(t.Elt)
	default:
		return "unknown"
	}
}
