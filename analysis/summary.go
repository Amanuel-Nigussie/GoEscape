package analysis

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

func BuildSummary(result *IntraResult) *FuncSummary {
	return &FuncSummary{
		EscapingParams: result.EscapingParams,
		ReturnEscapes:  result.ReturnEscapes,
	}
}

func ConservativeSummary(paramCount int) *FuncSummary {
	params := make(map[int]bool, paramCount)
	for i := 0; i < paramCount; i++ {
		params[i] = true
	}
	return &FuncSummary{
		EscapingParams: params,
		ReturnEscapes:  true,
	}
}

func EmptySummary() *FuncSummary {
	return &FuncSummary{
		EscapingParams: make(map[int]bool),
		ReturnEscapes:  false,
	}
}

func MergeSummaries(a, b *FuncSummary) *FuncSummary {
	merged := &FuncSummary{
		EscapingParams: make(map[int]bool),
		ReturnEscapes:  a.ReturnEscapes || b.ReturnEscapes,
	}

	for i := range a.EscapingParams {
		merged.EscapingParams[i] = true
	}
	for i := range b.EscapingParams {
		merged.EscapingParams[i] = true
	}

	return merged
}

func SummaryChanged(oldSummary, newSummary *FuncSummary) bool {
	if newSummary.ReturnEscapes && !oldSummary.ReturnEscapes {
		return true
	}

	for i := range newSummary.EscapingParams {
		if !oldSummary.EscapingParams[i] {
			return true
		}
	}

	return false
}

type jsonSummary struct {
	FuncName       string `json:"func_name"`
	EscapingParams []int  `json:"escaping_params"`
	ReturnEscapes  bool   `json:"return_escapes"`
}

func SaveSummaries(summaries Summaries, outDir string) error {
	summaryDir := filepath.Join(outDir, "summaries")
	if err := os.MkdirAll(summaryDir, 0755); err != nil {
		return fmt.Errorf("summary: cannot create output dir: %w", err)
	}

	var entries []jsonSummary
	for name, summary := range summaries {
		var params []int
		for i := range summary.EscapingParams {
			params = append(params, i)
		}
		sort.Ints(params)

		entries = append(entries, jsonSummary{
			FuncName:       name,
			EscapingParams: params,
			ReturnEscapes:  summary.ReturnEscapes,
		})
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].FuncName < entries[j].FuncName
	})

	data, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		return fmt.Errorf("summary: JSON marshal error: %w", err)
	}

	filename := filepath.Join(summaryDir, "summaries.json")
	if err := os.WriteFile(filename, data, 0644); err != nil {
		return fmt.Errorf("summary: cannot write %s: %w", filename, err)
	}

	return nil
}
