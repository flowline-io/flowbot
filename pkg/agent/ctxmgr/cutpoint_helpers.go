package ctxmgr

import (
	"github.com/flowline-io/flowbot/pkg/agent/msg"
	"github.com/flowline-io/flowbot/pkg/agent/session"
)

func selectCutIndexByTokens(
	entries []session.TreeEntry,
	startIndex, endIndex, keepRecentTokens int,
	cutPoints []int,
) int {
	cutIndex := cutPoints[0]
	accumulated := 0
	for i := endIndex - 1; i >= startIndex; i-- {
		entry := entries[i]
		if entry.Type != session.EntryMessage || entry.Message == nil {
			continue
		}
		accumulated += EstimateTokens(entry.Message)
		if accumulated >= keepRecentTokens {
			for _, candidate := range cutPoints {
				if candidate >= i {
					cutIndex = candidate
					break
				}
			}
			break
		}
	}
	return cutIndex
}

func adjustCutIndexFromCompaction(entries []session.TreeEntry, cutIndex, startIndex int) int {
	for cutIndex > startIndex {
		prev := entries[cutIndex-1]
		if prev.Type == session.EntryCompaction {
			break
		}
		if prev.Type == session.EntryMessage {
			break
		}
		cutIndex--
	}
	return cutIndex
}

func cutTurnInfo(entries []session.TreeEntry, cutIndex, startIndex int) (turnStart int, isSplitTurn bool) {
	cutEntry := entries[cutIndex]
	isUser := cutEntry.Type == session.EntryMessage && cutEntry.Message != nil && cutEntry.Message.Role() == msg.RoleUser
	isBranch := cutEntry.Type == session.EntryBranchSummary
	if isUser || isBranch {
		return -1, false
	}
	turnStart = findTurnStartIndex(entries, cutIndex, startIndex)
	return turnStart, turnStart >= 0
}
