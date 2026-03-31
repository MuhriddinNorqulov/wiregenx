package pkg

// Scope defines the lifecycle of a provider.
type Scope string

const (
	ScopeSingleton Scope = "singleton"
	ScopePrototype Scope = "prototype"
)

// Provider represents a function annotated with @Inject or @app.
type Provider struct {
	FuncName   string
	PkgName    string // package name in source file
	ImportPath string // full import path (from go list)
	File       string // absolute file path

	Params     []Param // function parameters (dependencies)
	ReturnType TypeRef // primary return type (what this provides)

	Scope        Scope
	ReturnsError bool // auto-detected: function returns (T, error)
	IsApp        bool // @Application annotation
}

// AppGroup represents one @app and all its resolved dependencies.
type AppGroup struct {
	App       Provider   // the @app provider
	Providers []Provider // topologically sorted (dependencies + app itself)
	Name      string     // container name prefix, e.g. "Http"
}

// Param represents a single function parameter with its name and type.
type Param struct {
	Name string  // parameter name from source, empty if unnamed
	Type TypeRef // resolved type
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
