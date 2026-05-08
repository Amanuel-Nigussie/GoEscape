package analysis

import (
	"encoding/json"
	"fmt"
	"go/token"
	"os"
	"path/filepath"

	"golang.org/x/tools/go/ssa"
)

type AllocSite struct {
	Name     string
	VarName  string
	Pos      string
	Decision EscapeVal
}

type IntraResult struct {
	FuncName       string
	AllocSites     []AllocSite
	EscapingParams map[int]bool
	ReturnEscapes  bool
	WorklistResult *FunctionResult
}

func AnalyzeFunc(
	fn *ssa.Function,
	summaries Summaries,
	callSiteCallees CallSiteCallees,
	fset *token.FileSet,
) *IntraResult {
	worklistResult := RunWorklist(fn, summaries, callSiteCallees, fset)
	funcName := qualifiedFuncName(fn)

	result := &IntraResult{
		FuncName:       funcName,
		EscapingParams: make(map[int]bool),
		WorklistResult: worklistResult,
	}

	for _, block := range fn.Blocks {
		for _, instr := range block.Instrs {
			alloc, ok := instr.(*ssa.Alloc)
			if !ok {
				continue
			}
			decision := worklistResult.Final.Get(alloc.Name())
			pos := "unknown"
			if fset != nil && alloc.Pos().IsValid() {
				pos = fset.Position(alloc.Pos()).String()
			}
			varName := alloc.Comment
			if varName == "" {
				varName = alloc.Name()
			}
			result.AllocSites = append(result.AllocSites, AllocSite{
				Name:     alloc.Name(),
				VarName:  varName,
				Pos:      pos,
				Decision: decision,
			})
		}
	}

	for i, param := range fn.Params {
		if worklistResult.Final.Get(param.Name()) == HEAP {
			result.EscapingParams[i] = true
		}
	}

	result.ReturnEscapes = checkReturnEscapes(fn, worklistResult.Final)
	return result
}

func checkReturnEscapes(fn *ssa.Function, final EscapeMap) bool {
	for _, block := range fn.Blocks {
		for _, instr := range block.Instrs {
			ret, ok := instr.(*ssa.Return)
			if !ok {
				continue
			}
			for _, result := range ret.Results {
				realRes := Unwrap(result)
				if final.Get(realRes.Name()) == HEAP {
					return true
				}
			}
		}
	}
	return false
}

func qualifiedFuncName(fn *ssa.Function) string {
	return fn.String()
}

func shortPos(pos string) string {
	s := pos
	for i := len(s) - 1; i >= 0; i-- {
		if s[i] == '/' || s[i] == '\\' {
			s = s[i+1:]
			break
		}
	}

	for i := len(s) - 1; i >= 0; i-- {
		if s[i] == ':' {
			s = s[:i]
			break
		}
	}
	return s
}

func shortFuncName(name string) string {
	for i := len(name) - 1; i >= 0; i-- {
		if name[i] == '.' {
			return name[i+1:]
		}
	}
	return name
}

// --- JSON output (for reproducibility) ---

type jsonAllocSite struct {
	Name     string `json:"name"`
	VarName  string `json:"var_name"`
	Pos      string `json:"pos"`
	Decision string `json:"decision"`
}

type jsonIntraResult struct {
	FuncName       string          `json:"func_name"`
	AllocSites     []jsonAllocSite `json:"alloc_sites"`
	EscapingParams []int           `json:"escaping_params"`
	ReturnEscapes  bool            `json:"return_escapes"`
}

func (r *IntraResult) SaveJSON(outDir string, fset *token.FileSet) error {
	analysisDir := filepath.Join(outDir, "analysis")
	if err := os.MkdirAll(analysisDir, 0755); err != nil {
		return fmt.Errorf("intraprocedural: cannot create output dir: %w", err)
	}

	var sites []jsonAllocSite
	for _, site := range r.AllocSites {
		sites = append(sites, jsonAllocSite{
			Name:     site.Name,
			VarName:  site.VarName,
			Pos:      site.Pos,
			Decision: site.Decision.String(),
		})
	}

	var escapingParams []int
	for i := range r.EscapingParams {
		escapingParams = append(escapingParams, i)
	}

	output := jsonIntraResult{
		FuncName:       r.FuncName,
		AllocSites:     sites,
		EscapingParams: escapingParams,
		ReturnEscapes:  r.ReturnEscapes,
	}

	data, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		return fmt.Errorf("intraprocedural: JSON marshal error: %w", err)
	}

	filename := filepath.Join(analysisDir, sanitizeFuncName(r.FuncName)+".json")
	return os.WriteFile(filename, data, 0644)
}

func sanitizeFuncName(name string) string {
	result := make([]byte, len(name))
	for i, c := range []byte(name) {
		switch c {
		case '/', '\\', ':', '*', '?', '"', '<', '>', '|', '.':
			result[i] = '_'
		default:
			result[i] = c
		}
	}
	return string(result)
}
