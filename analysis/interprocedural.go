package analysis

import (
	"fmt"
	"go/token"
	"sort"
	"strings"

	"golang.org/x/tools/go/callgraph"
	"golang.org/x/tools/go/ssa"
	"golang.org/x/tools/go/ssa/ssautil"
)

type InterResult struct {
	FuncResults map[string]*IntraResult
	Summaries   Summaries
}

func Analyze(prog *ssa.Program, cg *callgraph.Graph, fset *token.FileSet) *InterResult {
	result := &InterResult{
		FuncResults: make(map[string]*IntraResult),
		Summaries:   make(Summaries),
	}

	allFuncs := collectAnalyzableFunctions(prog)

	for fn := range ssautil.AllFunctions(prog) {
		if fn.Blocks == nil {
			name := qualifiedFuncName(fn)
			result.Summaries[name] = ConservativeSummary(len(fn.Params))
		}
	}

	callSiteCallees := buildCallSiteCallees(cg)

	sccs := computeSCCs(cg, allFuncs)
	for _, scc := range sccs {
		if len(scc) == 1 && !isSelfRecursive(scc[0], cg) {
			analyzeSCC(scc, result, callSiteCallees, fset, false)
		} else {
			analyzeSCC(scc, result, callSiteCallees, fset, true)
		}
	}

	return result
}

func buildCallSiteCallees(cg *callgraph.Graph) CallSiteCallees {
	result := make(CallSiteCallees)
	_ = callgraph.GraphVisitEdges(cg, func(edge *callgraph.Edge) error {
		if edge.Site != nil && edge.Callee.Func != nil {
			result[edge.Site] = append(result[edge.Site], edge.Callee.Func)
		}
		return nil
	})
	return result
}

func analyzeSCC(
	scc []*ssa.Function,
	result *InterResult,
	callSiteCallees CallSiteCallees,
	fset *token.FileSet,
	recursive bool,
) {
	if recursive {
		for _, fn := range scc {
			name := qualifiedFuncName(fn)
			result.Summaries[name] = EmptySummary()
		}
	}

	maxIter := 1
	if recursive {
		maxIter = 10
	}

	for iter := 0; iter < maxIter; iter++ {
		changed := false
		for _, fn := range scc {
			name := qualifiedFuncName(fn)
			oldSummary := result.Summaries[name]
			if oldSummary == nil {
				oldSummary = EmptySummary()
			}

			intraResult := AnalyzeFunc(fn, result.Summaries, callSiteCallees, fset)
			newSummary := BuildSummary(intraResult)

			if SummaryChanged(oldSummary, newSummary) {
				changed = true
			}
			result.FuncResults[name] = intraResult
			result.Summaries[name] = newSummary
		}
		if !changed {
			break
		}
	}
}

func collectAnalyzableFunctions(prog *ssa.Program) map[*ssa.Function]bool {
	result := make(map[*ssa.Function]bool)
	for fn := range ssautil.AllFunctions(prog) {
		if fn.Blocks == nil || !isUserFunction(fn) {
			continue
		}
		result[fn] = true
	}
	return result
}

func isSelfRecursive(fn *ssa.Function, cg *callgraph.Graph) bool {
	node, exists := cg.Nodes[fn]
	if !exists {
		return false
	}
	for _, edge := range node.Out {
		if edge.Callee.Func == fn {
			return true
		}
	}
	return false
}

func (r *InterResult) Print() {
	r.printEscapeTable()
	r.printSummaryTable()
}

func (r *InterResult) printEscapeTable() {

	fmt.Printf("\n  ─────────────────────── Escape Analysis ────────────────────── \n")
	fmt.Printf("  %-22s %-14s %-12s %s\n", "Function", "Variable", "Decision", "Location")
	fmt.Printf("  %s\n", strings.Repeat("─", 62))

	// Sort function names for deterministic output.
	names := sortedKeys(r.FuncResults)

	for _, name := range names {
		intra := r.FuncResults[name]
		short := shortFuncName(name)

		// Skip closure bodies — captured vars shown in enclosing function.
		if strings.Contains(short, "$") {
			continue
		}
		// Skip init — never interesting.
		if short == "init" {
			continue
		}

		if len(intra.AllocSites) == 0 {
			fmt.Printf("  %-22s %s\n", short, "(no escapes)")
			continue
		}

		for _, site := range intra.AllocSites {
			fmt.Printf("  %-22s %-14s %-12s %s\n",
				short,
				site.VarName,
				site.Decision,
				shortPos(site.Pos),
			)
		}
	}
}

func (r *InterResult) printSummaryTable() {
	fmt.Printf("\n  ────────────────── Interprocedural Summaries ─────────────────\n")
	fmt.Printf("  %-22s %-22s %s\n", "Function", "Escaping Params", "Return Escapes")
	fmt.Printf("  %s\n", strings.Repeat("─", 62))

	names := sortedKeys(r.FuncResults)
	printed := false

	for _, name := range names {
		intra := r.FuncResults[name]
		short := shortFuncName(name)

		// Skip trivial summaries.
		if len(intra.EscapingParams) == 0 && !intra.ReturnEscapes {
			continue
		}
		// Skip closure bodies.
		if strings.Contains(short, "$") {
			continue
		}

		params := "none"
		if len(intra.EscapingParams) > 0 {
			var ps []string
			for i := range intra.EscapingParams {
				ps = append(ps, fmt.Sprintf("param[%d]", i))
			}
			sort.Strings(ps)
			params = strings.Join(ps, ", ")
		}

		ret := "no"
		if intra.ReturnEscapes {
			ret = "yes"
		}

		fmt.Printf("  %-22s %-22s %s\n", short, params, ret)
		printed = true
	}

	if !printed {
		fmt.Printf("  (no functions with escaping parameters)\n")
	}
}

func sortedKeys(m map[string]*IntraResult) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

type sccState struct {
	index      map[*ssa.Function]int
	lowlink    map[*ssa.Function]int
	onStack    map[*ssa.Function]bool
	stack      []*ssa.Function
	counter    int
	sccs       [][]*ssa.Function
	cg         *callgraph.Graph
	analyzable map[*ssa.Function]bool
}

func computeSCCs(cg *callgraph.Graph, analyzable map[*ssa.Function]bool) [][]*ssa.Function {
	state := &sccState{
		index:      make(map[*ssa.Function]int),
		lowlink:    make(map[*ssa.Function]int),
		onStack:    make(map[*ssa.Function]bool),
		stack:      make([]*ssa.Function, 0),
		cg:         cg,
		analyzable: analyzable,
	}
	for fn := range analyzable {
		if _, visited := state.index[fn]; !visited {
			state.tarjan(fn)
		}
	}
	return state.sccs
}

func (s *sccState) tarjan(fn *ssa.Function) {
	s.index[fn] = s.counter
	s.lowlink[fn] = s.counter
	s.counter++
	s.stack = append(s.stack, fn)
	s.onStack[fn] = true

	if node, exists := s.cg.Nodes[fn]; exists {
		for _, edge := range node.Out {
			callee := edge.Callee.Func
			if !s.analyzable[callee] {
				continue
			}
			if _, visited := s.index[callee]; !visited {
				s.tarjan(callee)
				if s.lowlink[callee] < s.lowlink[fn] {
					s.lowlink[fn] = s.lowlink[callee]
				}
			} else if s.onStack[callee] {
				if s.index[callee] < s.lowlink[fn] {
					s.lowlink[fn] = s.index[callee]
				}
			}
		}
	}

	if s.lowlink[fn] == s.index[fn] {
		var scc []*ssa.Function
		for {
			top := s.stack[len(s.stack)-1]
			s.stack = s.stack[:len(s.stack)-1]
			s.onStack[top] = false
			scc = append(scc, top)
			if top == fn {
				break
			}
		}
		s.sccs = append(s.sccs, scc)
	}
}

func isUserFunction(fn *ssa.Function) bool {
	if fn.Package() == nil {
		return false
	}
	path := fn.Package().Pkg.Path()
	firstComponent := strings.SplitN(path, "/", 2)[0]
	return strings.Contains(firstComponent, ".")
}
