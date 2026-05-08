package analysis

import (
	"go/token"

	"golang.org/x/tools/go/ssa"
)

type BlockResult struct {
	In  EscapeMap
	Out EscapeMap
}

type FunctionResult struct {
	BlockResults map[int]*BlockResult
	Final        EscapeMap
}

func RunWorklist(
	fn *ssa.Function,
	summaries Summaries,
	callSiteCallees CallSiteCallees,
	fset *token.FileSet,
) *FunctionResult {
	blockResults := make(map[int]*BlockResult, len(fn.Blocks))
	for _, block := range fn.Blocks {
		blockResults[block.Index] = &BlockResult{
			In:  make(EscapeMap),
			Out: make(EscapeMap),
		}
	}

	// initialize all globals referenced in this function as HEAP
	entryOut := blockResults[fn.Blocks[0].Index].Out
	for _, block := range fn.Blocks {
		for _, instr := range block.Instrs {
			for _, operand := range instr.Operands(nil) {
				if operand == nil {
					continue
				}
				if _, isGlobal := (*operand).(*ssa.Global); isGlobal {
					entryOut.MarkHeap((*operand).Name())
				}
			}
		}
	}

	queue := newBlockQueue()
	for _, block := range fn.Blocks {
		queue.enqueue(block)
	}

	for !queue.isEmpty() {
		block := queue.dequeue()
		result := blockResults[block.Index]

		newIn := make(EscapeMap)
		for _, pred := range block.Preds {
			MergeInto(newIn, blockResults[pred.Index].Out)
		}

		newOut := newIn.Clone()
		if block.Index == 0 {
			for k, v := range entryOut {
				newOut.Set(k, v)
			}
		}

		for {
			blockChanged := false
			for _, instr := range block.Instrs {
				if ApplyInstruction(instr, newOut, summaries, callSiteCallees, fset) {
					blockChanged = true
				}
			}
			if !blockChanged {
				break
			}
		}

		outChanged := !newOut.Equal(result.Out)
		result.In = newIn
		result.Out = newOut

		if outChanged {
			for _, succ := range block.Succs {
				queue.enqueue(succ)
			}
		}
	}

	final := make(EscapeMap)
	for _, result := range blockResults {
		MergeInto(final, result.Out)
	}

	return &FunctionResult{
		BlockResults: blockResults,
		Final:        final,
	}
}

type blockQueue struct {
	items   []*ssa.BasicBlock
	inQueue map[int]bool
}

func newBlockQueue() *blockQueue {
	return &blockQueue{
		items:   make([]*ssa.BasicBlock, 0, 8),
		inQueue: make(map[int]bool),
	}
}

func (q *blockQueue) enqueue(b *ssa.BasicBlock) {
	if !q.inQueue[b.Index] {
		q.items = append(q.items, b)
		q.inQueue[b.Index] = true
	}
}

func (q *blockQueue) dequeue() *ssa.BasicBlock {
	b := q.items[0]
	q.items = q.items[1:]
	delete(q.inQueue, b.Index)
	return b
}

func (q *blockQueue) isEmpty() bool {
	return len(q.items) == 0
}
