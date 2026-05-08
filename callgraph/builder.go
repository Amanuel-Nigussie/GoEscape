package callgraph

import (
	"encoding/json"
	"fmt"
	"go/token"
	"os"
	"path/filepath"

	"golang.org/x/tools/go/callgraph"
	"golang.org/x/tools/go/callgraph/cha"
	"golang.org/x/tools/go/callgraph/rta"
	"golang.org/x/tools/go/callgraph/vta"
	"golang.org/x/tools/go/ssa"
	"golang.org/x/tools/go/ssa/ssautil"
)

type Algorithm string

const (
	CHA Algorithm = "cha"
	RTA Algorithm = "rta"
	VTA Algorithm = "vta"
)

type Result struct {
	Graph     *callgraph.Graph
	Algorithm Algorithm
	Fset      *token.FileSet
}

func Build(prog *ssa.Program, pkgs []*ssa.Package, algo Algorithm, fset *token.FileSet) (*Result, error) {
	var cg *callgraph.Graph

	switch algo {
	case CHA:
		cg = cha.CallGraph(prog)
	case RTA:
		mainFns, err := findEntryPoints(pkgs)
		if err != nil {
			return nil, fmt.Errorf("callgraph: RTA requires entry points: %w", err)
		}
		rtaResult := rta.Analyze(mainFns, true)
		cg = rtaResult.CallGraph
	case VTA:
		initialCHA := cha.CallGraph(prog)
		funcs := collectFuncsForVTA(prog)
		cg = vta.CallGraph(funcs, initialCHA)
	default:
		return nil, fmt.Errorf("callgraph: unknown algorithm %q", algo)
	}

	return &Result{Graph: cg, Algorithm: algo, Fset: fset}, nil
}

func (r *Result) Stats() (nodes int, edges int) {
	nodeSet := make(map[*ssa.Function]bool)
	_ = callgraph.GraphVisitEdges(r.Graph, func(edge *callgraph.Edge) error {
		if edge.Caller.Func != nil {
			nodeSet[edge.Caller.Func] = true
		}
		if edge.Callee.Func != nil {
			nodeSet[edge.Callee.Func] = true
		}
		edges++
		return nil
	})
	return len(nodeSet), edges
}

func (r *Result) SaveJSON(outDir string) error {
	cgDir := filepath.Join(outDir, "callgraph")
	os.MkdirAll(cgDir, 0755)

	var edges []jsonEdge
	nodeSet := make(map[string]bool)

	_ = callgraph.GraphVisitEdges(r.Graph, func(edge *callgraph.Edge) error {
		if edge.Caller.Func == nil || edge.Callee.Func == nil {
			return nil
		}
		site := "unknown"
		if edge.Site != nil && edge.Site.Pos().IsValid() {
			site = r.Fset.Position(edge.Site.Pos()).String()
		}
		callerName := funcQualifiedName(edge.Caller.Func)
		calleeName := funcQualifiedName(edge.Callee.Func)
		edges = append(edges, jsonEdge{Caller: callerName, Callee: calleeName, Site: site})
		nodeSet[callerName] = true
		nodeSet[calleeName] = true
		return nil
	})

	output := jsonCallGraph{
		Algorithm: string(r.Algorithm),
		NodeCount: len(nodeSet),
		EdgeCount: len(edges),
		Edges:     edges,
	}
	data, _ := json.MarshalIndent(output, "", "  ")
	return os.WriteFile(filepath.Join(cgDir, string(r.Algorithm)+".json"), data, 0644)
}

// For RTA
func findEntryPoints(pkgs []*ssa.Package) ([]*ssa.Function, error) {
	mains := ssautil.MainPackages(pkgs)
	var entries []*ssa.Function
	for _, pkg := range mains {
		if initFn := pkg.Func("init"); initFn != nil {
			entries = append(entries, initFn)
		}
		if mainFn := pkg.Func("main"); mainFn != nil {
			entries = append(entries, mainFn)
		}
	}
	if len(entries) == 0 {
		return nil, fmt.Errorf("no main() functions found; RTA requires a main package")
	}
	return entries, nil
}

func collectFuncsForVTA(prog *ssa.Program) map[*ssa.Function]bool {
	funcs := make(map[*ssa.Function]bool)
	for fn := range ssautil.AllFunctions(prog) {
		if fn.Blocks != nil {
			funcs[fn] = true
		}
	}
	return funcs
}

type jsonEdge struct {
	Caller string `json:"caller"`
	Callee string `json:"callee"`
	Site   string `json:"site"`
}

type jsonCallGraph struct {
	Algorithm string     `json:"algorithm"`
	NodeCount int        `json:"node_count"`
	EdgeCount int        `json:"edge_count"`
	Edges     []jsonEdge `json:"edges"`
}

func funcQualifiedName(fn *ssa.Function) string {
	if fn.Package() != nil {
		return fn.Package().Pkg.Path() + "." + fn.Name()
	}
	return fn.Name()
}
