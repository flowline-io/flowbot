package ctxmgr

import (
	"github.com/flowline-io/flowbot/pkg/agent/msg"
	"github.com/flowline-io/flowbot/pkg/agent/session"
)

// CutPointResult describes where to split session history during compaction.
type CutPointResult struct {
	FirstKeptEntryIndex int
	TurnStartIndex      int
	IsSplitTurn         bool
}

// FindCutPoint selects the oldest entry index to keep within a token budget.
func FindCutPoint(entries []session.TreeEntry, startIndex, endIndex, keepRecentTokens int) CutPointResult {
	cutPoints := findValidCutPoints(entries, startIndex, endIndex)
	if len(cutPoints) == 0 {
		return CutPointResult{FirstKeptEntryIndex: startIndex, TurnStartIndex: -1}
	}

	cutIndex := selectCutIndexByTokens(entries, startIndex, endIndex, keepRecentTokens, cutPoints)
	cutIndex = adjustCutIndexFromCompaction(entries, cutIndex, startIndex)
	turnStart, isSplitTurn := cutTurnInfo(entries, cutIndex, startIndex)
	return CutPointResult{
		FirstKeptEntryIndex: cutIndex,
		TurnStartIndex:      turnStart,
		IsSplitTurn:         isSplitTurn,
	}
}

func findValidCutPoints(entries []session.TreeEntry, startIndex, endIndex int) []int {
	cutPoints := make([]int, 0, endIndex-startIndex)
	for i := startIndex; i < endIndex; i++ {
		entry := entries[i]
		switch entry.Type {
		case session.EntryMessage:
			if entry.Message == nil {
				continue
			}
			switch entry.Message.Role() {
			case msg.RoleUser, msg.RoleAssistant, msg.RoleCustom, msg.RoleBranchSummary, msg.RoleCompactionSummary:
				cutPoints = append(cutPoints, i)
			case msg.RoleToolResult:
			}
		case session.EntryBranchSummary:
			cutPoints = append(cutPoints, i)
		}
	}
	return cutPoints
}

func findTurnStartIndex(entries []session.TreeEntry, entryIndex, startIndex int) int {
	for i := entryIndex; i >= startIndex; i-- {
		entry := entries[i]
		if entry.Type == session.EntryBranchSummary {
			return i
		}
		if entry.Type != session.EntryMessage || entry.Message == nil {
			continue
		}
		if entry.Message.Role() == msg.RoleUser {
			return i
		}
	}
	return -1
}
