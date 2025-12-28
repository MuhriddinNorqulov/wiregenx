package internal

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
)

func scanInjectFuncs(root string, ignoreVendor, ignoreHidden bool) ([]InjectFunc, error) {
	var out []InjectFunc
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

		pkg := file.Name.Name

		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Doc == nil {
				continue
			}
			if !hasInject(fn.Doc) {
				continue
			}

			// Only accept top-level funcs (methods are allowed too, but wire providers are usually funcs).
			// If you don't want methods, uncomment:
			// if fn.Recv != nil { continue }

			out = append(out, InjectFunc{
				PkgName:  pkg,
				FuncName: fn.Name.Name,
				File:     path,
			})
		}

		return nil
	})

	if err != nil {
		return nil, err
	}
	return out, nil
}

func hasInject(cg *ast.CommentGroup) bool {
	for _, c := range cg.List {
		if injectRe.MatchString(c.Text) {
			return true
		}
	}
	return false
}
