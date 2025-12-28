package pkg

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

var injectRe = regexp.MustCompile(`^\s*//\s*@inject(\b|:)`)

func Inject() {
	root := flag.String("root", ".", "repo root to scan")
	out := flag.String("out", "wire_provider.go", "output file (relative to root)")
	outPkg := flag.String("pkg", "wire", "output package name")
	ignoreVendor := flag.Bool("no-vendor", true, "ignore vendor/")
	ignoreHidden := flag.Bool("no-hidden", true, "ignore hidden dirs like .git/")
	flag.Parse()

	absRoot, err := filepath.Abs(*root)
	must(err)

	funcs, err := scanInjectFuncs(absRoot, *ignoreVendor, *ignoreHidden)
	must(err)

	if len(funcs) == 0 {
		fmt.Println("no @inject functions found")
		// Still generate empty set (useful for stable builds)
	}

	// Map dir -> import path (via go list)
	dirToImport, err := resolveImportPaths(absRoot, funcs)
	must(err)
	for i := range funcs {
		funcs[i].ImportPath = dirToImport[filepath.Dir(funcs[i].File)]
	}

	// Generate file
	outPath := filepath.Join(absRoot, filepath.FromSlash(*out))
	must(os.MkdirAll(filepath.Dir(outPath), 0o755))

	code, err := renderWireProviders(*outPkg, funcs)
	must(err)

	must(os.WriteFile(outPath, code, 0o644))
	fmt.Println("generated:", outPath)
}

func resolveImportPaths(root string, funcs []InjectFunc) (map[string]string, error) {
	// Collect unique dirs
	uniq := map[string]struct{}{}
	for _, f := range funcs {
		uniq[filepath.Dir(f.File)] = struct{}{}
	}

	// For each dir, run: go list -json .
	// (we set cmd.Dir=dir so it resolves that package)
	dirToImport := map[string]string{}

	for dir := range uniq {
		pkg, err := goListPkg(dir)
		if err != nil {
			return nil, fmt.Errorf("go list failed in %s: %w", dir, err)
		}
		dirToImport[dir] = pkg.ImportPath
	}

	return dirToImport, nil
}

func goListPkg(dir string) (*GoListPkg, error) {
	cmd := exec.Command("go", "list", "-json", ".")
	cmd.Dir = dir
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("%v: %s", err, strings.TrimSpace(stderr.String()))
	}

	var pkg GoListPkg
	if err := json.Unmarshal(stdout.Bytes(), &pkg); err != nil {
		return nil, err
	}
	return &pkg, nil
}
