package ir

import (
	"fmt"
	"go/token"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/tools/go/packages"
	"golang.org/x/tools/go/ssa"
	"golang.org/x/tools/go/ssa/ssautil"
)

type LoadedProgram struct {
	Program  *ssa.Program
	Packages []*ssa.Package
	Fset     *token.FileSet
}

const packageLoadMode = packages.NeedName | packages.NeedFiles | packages.NeedCompiledGoFiles |
	packages.NeedSyntax | packages.NeedTypes | packages.NeedTypesInfo |
	packages.NeedDeps | packages.NeedImports

// Load parses Go code and builds the SSA intermediate representation.
func Load(patterns []string) (*LoadedProgram, error) {
	fset := token.NewFileSet()
	cfg := &packages.Config{
		Mode:  packageLoadMode,
		Fset:  fset,
		Tests: false,
	}

	pkgs, err := packages.Load(cfg, patterns...)
	if err != nil {
		return nil, fmt.Errorf("ir/loader: failed to load packages %v: %w", patterns, err)
	}

	ssaMode := ssa.SanityCheckFunctions | ssa.InstantiateGenerics
	prog, ssaPkgs := ssautil.AllPackages(pkgs, ssaMode)
	prog.Build()

	var validPkgs []*ssa.Package
	for _, p := range ssaPkgs {
		if p != nil {
			validPkgs = append(validPkgs, p)
		}
	}

	if len(validPkgs) == 0 {
		return nil, fmt.Errorf("ir/loader: no valid SSA packages built from %v", patterns)
	}

	return &LoadedProgram{
		Program:  prog,
		Packages: validPkgs,
		Fset:     fset,
	}, nil
}

// SaveSSA writes the human-readable SSA text for user functions only.
func (lp *LoadedProgram) SaveSSA(outDir string) error {
	ssaDir := filepath.Join(outDir, "ssa")
	os.MkdirAll(ssaDir, 0755)

	for fn := range ssautil.AllFunctions(lp.Program) {
		if fn.Blocks == nil || !isUserFunction(fn) {
			continue
		}

		pkgName := "unknown"
		if fn.Package() != nil {
			pkgName = sanitizeName(fn.Package().Pkg.Path())
		}
		funcName := sanitizeName(fn.Name())
		filename := filepath.Join(ssaDir, pkgName+"."+funcName+".ssa")

		f, _ := os.Create(filename)
		fn.WriteTo(f)
		f.Close()
	}

	return nil
}

// isUserFunction returns true if fn belongs to a user-defined package.
func isUserFunction(fn *ssa.Function) bool {
	if fn.Package() == nil {
		return false
	}
	path := fn.Package().Pkg.Path()
	firstComponent := strings.SplitN(path, "/", 2)[0]
	return strings.Contains(firstComponent, ".")
}

func sanitizeName(s string) string {
	replacer := strings.NewReplacer("/", "_", "\\", "_", ":", "_", "*", "_", "(", "_", ")", "_", "[", "_", "]", "_")
	return replacer.Replace(s)
}
