package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Amanuel-Nigussie/GoEscape/analysis"
	gocallgraph "github.com/Amanuel-Nigussie/GoEscape/callgraph"
	"github.com/Amanuel-Nigussie/GoEscape/ir"
)

func main() {
	cgAlgo := flag.String("cg", "cha", "call graph algorithm: cha | rta | vta")
	saveIntermediates := flag.Bool("save", false, "save intermediate outputs")
	outDir := flag.String("out", "./output", "output directory")
	flag.Parse()

	patterns := flag.Args()
	if len(patterns) == 0 {
		fmt.Fprintln(os.Stderr, "usage: goesc [--cg=cha|rta|vta] [--save] [--out=dir] <package patterns>")
		os.Exit(1)
	}

	algo := gocallgraph.Algorithm(*cgAlgo)
	if algo != gocallgraph.CHA && algo != gocallgraph.RTA && algo != gocallgraph.VTA {
		fmt.Fprintf(os.Stderr, "error: unknown algorithm %q (want cha, rta, or vta)\n", *cgAlgo)
		os.Exit(1)
	}

	algoDesc := map[gocallgraph.Algorithm]string{
		gocallgraph.CHA: "CHA (Class Hierarchy Analysis)",
		gocallgraph.RTA: "RTA (Rapid Type Analysis)",
		gocallgraph.VTA: "VTA (Variable Type Analysis)",
	}

	// ── Header ───────────────────────────────────────────────────────
	fmt.Printf("GoEscape — Static Escape Analysis\n")
	fmt.Printf("  packages : %s\n\n", strings.Join(patterns, ", "))

	// ── Phase 1: IR ───────────────────────────────────────────────────
	lp, err := ir.Load(patterns)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	ssaMsg := ""
	if *saveIntermediates {
		lp.SaveSSA(*outDir)
		ssaMsg = fmt.Sprintf(", SSA → %s", filepath.Join(*outDir, "ssa"))
	}
	fmt.Printf("  IR         %d package(s) loaded%s\n", len(lp.Packages), ssaMsg)

	// ── Phase 2: Call Graph ───────────────────────────────────────────
	cgResult, err := gocallgraph.Build(lp.Program, lp.Packages, algo, lp.Fset)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	nodes, edges := cgResult.Stats()
	cgMsg := ""
	if *saveIntermediates {
		cgResult.SaveJSON(*outDir)
		cgMsg = fmt.Sprintf(" → %s", filepath.Join(*outDir, "callgraph", string(algo)+".json"))
	}
	fmt.Printf("  CallGraph  %d nodes, %d edges  [%s]%s\n",
		nodes, edges, algoDesc[algo], cgMsg)

	// ── Phase 3+4: Escape Analysis ────────────────────────────────────
	interResult := analysis.Analyze(lp.Program, cgResult.Graph, lp.Fset)
	interResult.Print()

	if *saveIntermediates {
		for _, intra := range interResult.FuncResults {
			intra.SaveJSON(*outDir, lp.Fset)
		}
		analysis.SaveSummaries(interResult.Summaries, *outDir)
	}

	// ── Summary ───────────────────────────────────────────────────────
	printFinalSummary(interResult, *outDir, *saveIntermediates)
}

func printFinalSummary(r *analysis.InterResult, outDir string, saved bool) {
	totalAllocs := 0
	heapCount := 0
	stackCount := 0

	for _, intra := range r.FuncResults {
		for _, site := range intra.AllocSites {
			totalAllocs++
			if site.Decision == analysis.HEAP {
				heapCount++
			} else {
				stackCount++
			}
		}
	}

	fmt.Printf("\n────────────────── Summary ──────────────────\n")
	fmt.Printf("  functions analyzed    %d\n", len(r.FuncResults))
	fmt.Printf("  allocations found     %d\n", totalAllocs)
	fmt.Printf("  HEAP  (escaped)       %d\n", heapCount)
	fmt.Printf("  STACK     (safe)      %d\n", stackCount)

	if saved {
		fmt.Printf("  output saved to       %s\n", outDir)
	}
}
