package internal

type InjectFunc struct {
	PkgName    string // package name in file
	ImportPath string // full import path (from go list)
	FuncName   string
	File       string // abs
}

type GoListPkg struct {
	ImportPath string
	Name       string
	Dir        string
}
