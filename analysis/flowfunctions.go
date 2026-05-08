package analysis

import (
	"go/token"
	"go/types"

	"golang.org/x/tools/go/ssa"
)

// CallSiteCallees maps each call instruction to its possible callees
type CallSiteCallees map[ssa.Instruction][]*ssa.Function

// Summaries maps qualified function name → FuncSummary.
type Summaries map[string]*FuncSummary

// FuncSummary describes a function's escape behaviour for callers.
type FuncSummary struct {
	EscapingParams map[int]bool
	ReturnEscapes  bool
}

func ApplyInstruction(
	instr ssa.Instruction,
	state EscapeMap,
	summaries Summaries,
	callSiteCallees CallSiteCallees,
	fset *token.FileSet,
) bool {
	changed := false

	switch v := instr.(type) {

	case *ssa.Alloc:
		if _, seen := state[v.Name()]; !seen {
			state[v.Name()] = STACK
		}

	case *ssa.Store:
		if _, isGlobal := v.Addr.(*ssa.Global); isGlobal {
			realVal := Unwrap(v.Val)
			if isPointerLike(realVal.Type()) {
				if state.MarkHeap(realVal.Name()) {
					changed = true
				}
			}
		}

		realAddr := Unwrap(v.Addr)
		if state.Get(realAddr.Name()) == HEAP {
			realVal := Unwrap(v.Val)
			if isPointerLike(realVal.Type()) {
				if state.MarkHeap(realVal.Name()) {
					changed = true
				}
			}
		}

	case *ssa.UnOp:
		if v.Op == token.MUL {
			if state.Get(v.X.Name()) == HEAP {
				if state.MarkHeap(v.Name()) {
					changed = true
				}
			}
		}

	case *ssa.Return:
		for _, result := range v.Results {
			realRes := Unwrap(result)
			if isPointerLike(realRes.Type()) {
				if state.MarkHeap(realRes.Name()) {
					changed = true
				}
			}
		}

	case *ssa.MakeInterface:
		if state.Get(v.Name()) == HEAP {
			if state.MarkHeap(v.X.Name()) {
				changed = true
			}
		}

	case *ssa.Slice:
		if state.Get(v.Name()) == HEAP {
			if state.MarkHeap(v.X.Name()) {
				changed = true
			}
		}
		if state.Get(v.X.Name()) == HEAP {
			if state.MarkHeap(v.Name()) {
				changed = true
			}
		}

	case *ssa.MakeClosure:
		if state.Get(v.Name()) == HEAP {
			for _, binding := range v.Bindings {
				if isPointerLike(binding.Type()) {
					if state.MarkHeap(binding.Name()) {
						changed = true
					}
				}
			}
		}

	case *ssa.Go:
		if isPointerLike(v.Call.Value.Type()) {
			if state.MarkHeap(v.Call.Value.Name()) {
				changed = true
			}
		}
		for _, arg := range v.Call.Args {
			if isPointerLike(arg.Type()) {
				if state.MarkHeap(arg.Name()) {
					changed = true
				}
			}
		}

	case *ssa.Send:
		if isPointerLike(v.X.Type()) {
			if state.MarkHeap(v.X.Name()) {
				changed = true
			}
		}

	case *ssa.Defer:
		if isPointerLike(v.Call.Value.Type()) {
			if state.MarkHeap(v.Call.Value.Name()) {
				changed = true
			}
		}
		for _, arg := range v.Call.Args {
			if isPointerLike(arg.Type()) {
				if state.MarkHeap(arg.Name()) {
					changed = true
				}
			}
		}

	case *ssa.Phi:
		var edgeVals []EscapeVal
		for _, edge := range v.Edges {
			edgeVals = append(edgeVals, state.Get(edge.Name()))
		}
		if state.Set(v.Name(), Join(state.Get(v.Name()), JoinAll(edgeVals))) {
			changed = true
		}

	case *ssa.Call:
		if v.Call.IsInvoke() {
			callees := callSiteCallees[instr]
			if applyInvokeSummary(v, callees, state, summaries) {
				changed = true
			}
		} else {
			calleeName := ""
			if fn := v.Call.StaticCallee(); fn != nil {
				calleeName = qualifiedFuncName(fn)
			}
			summary, hasSummary := summaries[calleeName]
			if hasSummary && calleeName != "" {
				for i, arg := range v.Call.Args {
					if summary.EscapingParams[i] && isPointerLike(arg.Type()) {
						if state.MarkHeap(arg.Name()) {
							changed = true
						}
					}
				}
				if summary.ReturnEscapes && isPointerLike(v.Type()) {
					if state.MarkHeap(v.Name()) {
						changed = true
					}
				}
			} else {
				// Unknown callee: conservatively escape all pointer args.
				for _, arg := range v.Call.Args {
					if isPointerLike(arg.Type()) {
						if state.MarkHeap(arg.Name()) {
							changed = true
						}
					}
				}
			}
		}

	case *ssa.FieldAddr:
		// 1. If the field pointer escapes, its parent struct escapes.
		if state.Get(v.Name()) == HEAP {
			if state.MarkHeap(v.X.Name()) {
				changed = true
			}
		}
		// 2. If the parent struct escapes, the field pointer points to the heap.
		if state.Get(v.X.Name()) == HEAP {
			if state.MarkHeap(v.Name()) {
				changed = true
			}
		}

	case *ssa.IndexAddr:
		if state.Get(v.Name()) == HEAP {
			if state.MarkHeap(v.X.Name()) {
				changed = true
			}
		}

		if state.Get(v.X.Name()) == HEAP {
			if state.MarkHeap(v.Name()) {
				changed = true
			}
		}

	case *ssa.If, *ssa.Jump, *ssa.Panic:
		// No escape effect.
	}

	return changed
}

// applyInvokeSummary applies the joined summary of all possible callees
// at an interface invoke site.

func applyInvokeSummary(
	v *ssa.Call,
	callees []*ssa.Function,
	state EscapeMap,
	summaries Summaries,
) bool {
	changed := false

	if len(callees) == 0 {
		// No callees resolved: conservatively mark receiver as HEAP
		if state.MarkHeap(v.Call.Value.Name()) {
			changed = true
		}
		return changed
	}

	// Join summaries of all possible callees
	effective := EmptySummary()
	for _, callee := range callees {
		name := qualifiedFuncName(callee)
		if s, ok := summaries[name]; ok {
			effective = MergeSummaries(effective, s)
		}
	}

	if effective.EscapingParams[0] {
		realVal := Unwrap(v.Call.Value)
		if state.MarkHeap(realVal.Name()) {
			changed = true
		}
	}

	for i, arg := range v.Call.Args {
		if effective.EscapingParams[i+1] && isPointerLike(arg.Type()) {
			if state.MarkHeap(arg.Name()) {
				changed = true
			}
		}
	}

	if effective.ReturnEscapes && isPointerLike(v.Type()) {
		if state.MarkHeap(v.Name()) {
			changed = true
		}
	}

	return changed
}

func isPointerLike(t types.Type) bool {
	t = t.Underlying()
	switch t.(type) {
	case *types.Pointer, *types.Map, *types.Chan, *types.Signature, *types.Interface, *types.Slice:
		return true
	default:
		return false
	}
}

func Unwrap(v ssa.Value) ssa.Value {
	for {
		switch val := v.(type) {
		case *ssa.MakeInterface:
			v = val.X
		case *ssa.ChangeType:
			v = val.X
		case *ssa.Slice:
			v = val.X
		default:
			return v
		}
	}
}
