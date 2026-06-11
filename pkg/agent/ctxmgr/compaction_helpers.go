package ctxmgr

import (
	"github.com/flowline-io/flowbot/pkg/agent/msg"
	"github.com/flowline-io/flowbot/pkg/agent/result"
	"github.com/flowline-io/flowbot/pkg/agent/session"
)

type compactionBounds struct {
	prevCompactionIndex int
	previousSummary     string
	boundaryStart       int
	boundaryEnd         int
	keepRecent          int
}

func lastCompactionIndex(pathEntries []session.TreeEntry) int {
	for i := len(pathEntries) - 1; i >= 0; i-- {
		if pathEntries[i].Type == session.EntryCompaction {
			return i
		}
	}
	return -1
}

func firstKeptStartIndex(pathEntries []session.TreeEntry, prev session.TreeEntry, fallback int) int {
	firstKeptIndex := indexOfEntry(pathEntries, prev.FirstKeptEntryID)
	if firstKeptIndex >= 0 {
		return firstKeptIndex
	}
	return fallback
}

func computeCompactionBounds(
	pathEntries []session.TreeEntry,
	settings Settings,
	leafIsCompaction, force bool,
) compactionBounds {
	bounds := compactionBounds{
		prevCompactionIndex: lastCompactionIndex(pathEntries),
		boundaryStart:       0,
		boundaryEnd:         len(pathEntries),
		keepRecent:          settings.KeepRecentTokens,
	}

	if bounds.prevCompactionIndex < 0 {
		return bounds
	}

	prev := pathEntries[bounds.prevCompactionIndex]
	bounds.previousSummary = prev.Summary

	if leafIsCompaction && force {
		bounds.boundaryStart = firstKeptStartIndex(pathEntries, prev, bounds.prevCompactionIndex+1)
		bounds.boundaryEnd = bounds.prevCompactionIndex
		if bounds.keepRecent > 1 {
			bounds.keepRecent = bounds.keepRecent / 2
		}
		return bounds
	}

	bounds.boundaryStart = firstKeptStartIndex(pathEntries, prev, bounds.prevCompactionIndex+1)
	return bounds
}

func prepareExtraOnlyCompaction(
	pathEntries []session.TreeEntry,
	opts PrepareOptions,
	prevCompactionIndex int,
	previousSummary string,
	tokensBefore int,
	settings Settings,
) *CompactionPreparation {
	prev := pathEntries[prevCompactionIndex]
	return &CompactionPreparation{
		FirstKeptEntryID:    prev.FirstKeptEntryID,
		MessagesToSummarize: append([]msg.AgentMessage(nil), opts.ExtraMessages...),
		TokensBefore:        tokensBefore,
		PreviousSummary:     previousSummary,
		FileOps:             ExtractFileOperations(opts.ExtraMessages, pathEntries, prevCompactionIndex),
		Settings:            settings,
	}
}

func finalizeMessagesToSummarize(
	pathEntries []session.TreeEntry,
	bounds compactionBounds,
	cutPoint CutPointResult,
	firstKept session.TreeEntry,
	opts PrepareOptions,
) (messagesToSummarize []msg.AgentMessage, turnPrefix []msg.AgentMessage, firstKeptEntry session.TreeEntry) {
	historyEnd := cutPoint.FirstKeptEntryIndex
	if cutPoint.IsSplitTurn && cutPoint.TurnStartIndex >= 0 {
		historyEnd = cutPoint.TurnStartIndex
	}

	messagesToSummarize = collectMessages(pathEntries, bounds.boundaryStart, historyEnd)
	messagesToSummarize = append(messagesToSummarize, opts.ExtraMessages...)
	turnPrefix = collectMessages(pathEntries, cutPoint.TurnStartIndex, cutPoint.FirstKeptEntryIndex)
	if !cutPoint.IsSplitTurn {
		turnPrefix = nil
	}

	if len(messagesToSummarize) == 0 && len(turnPrefix) == 0 {
		if opts.Force && bounds.boundaryEnd > bounds.boundaryStart {
			messagesToSummarize = collectMessages(pathEntries, bounds.boundaryStart, bounds.boundaryEnd)
			if len(messagesToSummarize) > 0 {
				firstKept = pathEntries[bounds.boundaryEnd-1]
			}
		}
	}
	return messagesToSummarize, turnPrefix, firstKept
}

func buildCompactionPreparation(
	pathEntries []session.TreeEntry,
	bounds compactionBounds,
	opts PrepareOptions,
	tokensBefore int,
	settings Settings,
) result.Result[*CompactionPreparation, result.CompactionError] {
	cutPoint := FindCutPoint(pathEntries, bounds.boundaryStart, bounds.boundaryEnd, bounds.keepRecent)
	firstKept := pathEntries[cutPoint.FirstKeptEntryIndex]
	if firstKept.ID == "" {
		return result.Err[*CompactionPreparation, result.CompactionError](
			result.NewCompactionError("invalid_session", "missing first kept entry id", nil),
		)
	}

	messagesToSummarize, turnPrefix, firstKept := finalizeMessagesToSummarize(
		pathEntries, bounds, cutPoint, firstKept, opts,
	)
	if len(messagesToSummarize) == 0 && len(turnPrefix) == 0 {
		return result.Ok[*CompactionPreparation, result.CompactionError](nil)
	}

	fileOps := ExtractFileOperations(messagesToSummarize, pathEntries, bounds.prevCompactionIndex)
	if cutPoint.IsSplitTurn {
		for _, message := range turnPrefix {
			ExtractFileOpsFromMessage(message, fileOps)
		}
	}

	return result.Ok[*CompactionPreparation, result.CompactionError](&CompactionPreparation{
		FirstKeptEntryID:    firstKept.ID,
		MessagesToSummarize: messagesToSummarize,
		TurnPrefixMessages:  turnPrefix,
		IsSplitTurn:         cutPoint.IsSplitTurn,
		TokensBefore:        tokensBefore,
		PreviousSummary:     bounds.previousSummary,
		FileOps:             fileOps,
		Settings:            settings,
	})
}
