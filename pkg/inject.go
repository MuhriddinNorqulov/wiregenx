package pkg

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func Inject() {
	root := flag.String("root", ".", "repo root to scan")
	out := flag.String("out", "container_gen.go", "output file (relative to root)")
	outPkg := flag.String("pkg", "container", "output package name")
	ignoreVendor := flag.Bool("no-vendor", true, "ignore vendor/")
	ignoreHidden := flag.Bool("no-hidden", true, "ignore hidden dirs like .git/")
	flag.Parse()

	absRoot, err := filepath.Abs(*root)
	must(err)

	// 1. Scan for annotated providers
	providers, err := scanProviders(absRoot, *ignoreVendor, *ignoreHidden)
	must(err)

	if len(providers) == 0 {
		fmt.Println("no annotated provider functions found (@Inject, @Factory, @Application)")
		return
	}

	fmt.Printf("found %d provider(s)\n", len(providers))

	// 2. Resolve import paths via go list
	dirToImport, err := resolveImportPaths(absRoot, providers)
	must(err)

	for i := range providers {
		dir := filepath.Dir(providers[i].File)
		providers[i].ImportPath = dirToImport[dir]

		// Resolve local types (no ImportPath) to their package's import path
		if providers[i].ReturnType.ImportPath == "" && !builtinTypes[providers[i].ReturnType.TypeName] {
			providers[i].ReturnType.ImportPath = providers[i].ImportPath
		}
		for j := range providers[i].Params {
			if providers[i].Params[j].ImportPath == "" && !builtinTypes[providers[i].Params[j].TypeName] {
				providers[i].Params[j].ImportPath = providers[i].ImportPath
			}
		}
	}

	// 3. Resolve dependency graph (topological sort)
	sorted, err := resolveGraph(providers)
	must(err)

	// 4. Render container
	code, err := renderContainer(*outPkg, sorted)
	must(err)

	// 5. Write output file
	outPath := filepath.Join(absRoot, filepath.FromSlash(*out))
	must(os.MkdirAll(filepath.Dir(outPath), 0o755))
	must(os.WriteFile(outPath, code, 0o644))

	fmt.Println("generated:", outPath)
}

func resolveImportPaths(root string, providers []Provider) (map[string]string, error) {
	uniq := map[string]struct{}{}
	for _, p := range providers {
		uniq[filepath.Dir(p.File)] = struct{}{}
	}

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
