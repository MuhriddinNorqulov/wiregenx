package pkg

// Scope defines the lifecycle of a provider.
type Scope string

const (
	ScopeSingleton Scope = "singleton"
	ScopePrototype Scope = "prototype"
)

// Provider represents a function annotated with @Inject, @Factory, or @Application.
type Provider struct {
	FuncName   string
	PkgName    string // package name in source file
	ImportPath string // full import path (from go list)
	File       string // absolute file path

	Params     []TypeRef // function parameters (dependencies)
	ReturnType TypeRef   // primary return type (what this provides)

	Scope     Scope
	IsFactory bool // returns (T, error)
	IsApp     bool // @Application
}

// TypeRef represents a resolved Go type reference.
type TypeRef struct {
	ImportPath string // e.g. "database/sql", empty for builtins/local before resolution
	TypeName   string // e.g. "DB", "Config", "string"
	IsPointer  bool
	Raw        string // original source representation, e.g. "*sql.DB"
}

// FullName returns the canonical type string used for dependency matching.
// Examples: "*database/sql.DB", "github.com/app/config.Config", "string"
func (t TypeRef) FullName() string {
	prefix := ""
	if t.IsPointer {
		prefix = "*"
	}
	if t.ImportPath != "" {
		return prefix + t.ImportPath + "." + t.TypeName
	}
	return prefix + t.TypeName
}

// GoListPkg holds the parsed output of `go list -json`.
type GoListPkg struct {
	ImportPath string
	Name       string
	Dir        string
}
