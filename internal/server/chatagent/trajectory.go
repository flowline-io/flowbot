package chatagent

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/flowline-io/flowbot/pkg/agent/harness"
	"github.com/flowline-io/flowbot/pkg/agent/msg"
	"github.com/flowline-io/flowbot/pkg/agent/session"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/google/uuid"
)

// TrajectoryView is the assembled trajectory ledger for one session branch.
type TrajectoryView struct {
	Rows []TrajectoryRow `json:"rows"`
}

// TrajectoryRow is one labeled step in the trajectory log.
type TrajectoryRow struct {
	ID         string                 `json:"id"`
	Turn       int                    `json:"turn"`
	Role       string                 `json:"role"`
	Kind       string                 `json:"kind"`
	Text       string                 `json:"text"`
	DurationMs int64                  `json:"duration_ms,omitempty"`
	AssembleMs int64                  `json:"assemble_ms,omitempty"`
	ToolName   string                 `json:"tool_name,omitempty"`
	ToolStatus string                 `json:"tool_status,omitempty"`
	Subagent   string                 `json:"subagent,omitempty"`
	CreatedAt  time.Time              `json:"created_at"`
	Raw        any                    `json:"raw,omitempty"`
	Sections   []session.TraceSection `json:"sections,omitempty"`
}

func appendTurnTrace(ctx context.Context, h *harness.Harness, req RunRequest, trace promptTrace) error {
	if req.Kind == RunKindPipeline || h == nil || h.Session() == nil {
		return nil
	}
	if len(trace.Sections) == 0 && trace.AssembleMs == 0 {
		return nil
	}
	parentID, err := h.Session().LeafID(ctx)
	if err != nil {
		return fmt.Errorf("chatagent turn_trace leaf: %w", err)
	}
	entryID := uuid.NewString()
	if err := h.Session().Append(ctx, session.TreeEntry{
		ID:         entryID,
		ParentID:   parentID,
		Type:       session.EntryTurnTrace,
		Sections:   trace.Sections,
		AssembleMs: trace.AssembleMs,
	}); err != nil {
		return fmt.Errorf("chatagent turn_trace append: %w", err)
	}
	if req.API != nil && req.API.Publisher != nil {
		if err := req.API.Publisher.Publish(StreamEvent{
			Type:       EventTypeTurnTrace,
			ID:         entryID,
			Sections:   trace.Sections,
			AssembleMs: trace.AssembleMs,
		}); err != nil {
			flog.Warn("[chat-agent] turn_trace sse session=%s: %v", req.SessionID, err)
		}
	}
	return nil
}

// ListSessionTrajectory assembles the trajectory view for a session branch.
func ListSessionTrajectory(ctx context.Context, sessionID string) (*TrajectoryView, error) {
	storage := NewDBStorage(sessionID, types.Uid(""), "")
	branch, err := storage.GetBranch(ctx, "")
	if err != nil {
		return nil, err
	}
	createdAtByID, err := entryCreatedAtMap(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	return assembleTrajectory(branch, createdAtByID), nil
}

func assembleTrajectory(branch []session.TreeEntry, createdAtByID map[string]time.Time) *TrajectoryView {
	view := &TrajectoryView{Rows: make([]TrajectoryRow, 0, len(branch)*2)}
	turn := 0
	subagentByCall := map[string]string{}
	for _, entry := range branch {
		createdAt := createdAtByID[entry.ID]
		if createdAt.IsZero() {
			createdAt = time.Now().UTC()
		}
		switch entry.Type {
		case session.EntryTurnTrace:
			nextTurn := turn + 1
			view.Rows = append(view.Rows, turnTraceRows(entry, nextTurn, createdAt)...)
		case session.EntryCompaction:
			text := strings.TrimSpace(entry.Summary)
			if text == "" {
				continue
			}
			view.Rows = append(view.Rows, TrajectoryRow{
				ID:        entry.ID,
				Turn:      turn,
				Role:      "compaction",
				Kind:      "compaction",
				Text:      text,
				CreatedAt: createdAt,
				Raw:       map[string]any{"summary": text, "first_kept_entry_id": entry.FirstKeptEntryID},
			})
		case session.EntryMessage:
			if entry.Message == nil {
				continue
			}
			view.Rows = append(view.Rows, trajectoryRowsFromMessage(entry, &turn, createdAt, subagentByCall)...)
		}
	}
	return view
}

func turnTraceRows(entry session.TreeEntry, turn int, createdAt time.Time) []TrajectoryRow {
	rows := make([]TrajectoryRow, 0, len(entry.Sections))
	seen := map[string]int{}
	for _, section := range entry.Sections {
		name := strings.TrimSpace(section.Name)
		if name == "" {
			continue
		}
		seen[name]++
		id := entry.ID + "/" + name
		if n := seen[name]; n > 1 {
			id += "/" + strconv.Itoa(n)
		}
		row := TrajectoryRow{
			ID:        id,
			Turn:      turn,
			Role:      "context",
			Kind:      "context",
			Text:      section.Content,
			CreatedAt: createdAt,
			Raw:       section,
		}
		if name == TraceSectionSystemBody {
			row.Role = "system"
			row.Kind = "system"
			row.AssembleMs = entry.AssembleMs
		}
		rows = append(rows, row)
	}
	return rows
}

func trajectoryRowsFromMessage(entry session.TreeEntry, turn *int, createdAt time.Time, subagentByCall map[string]string) []TrajectoryRow {
	switch m := entry.Message.(type) {
	case msg.UserMessage:
		*turn++
		text := strings.TrimSpace(textFromParts(m.Parts))
		ts := messageTimestamp(m.Timestamp, createdAt)
		if text == "" && len(historyAttachments(m.Parts)) == 0 {
			return nil
		}
		return []TrajectoryRow{{
			ID:        entry.ID,
			Turn:      *turn,
			Role:      "user",
			Kind:      "user",
			Text:      text,
			CreatedAt: ts,
			Raw:       map[string]any{"role": "user", "text": text},
		}}
	case msg.AssistantMessage:
		return trajectoryRowsFromAssistant(entry.ID, *turn, createdAt, m, subagentByCall)
	case msg.ToolResultMessage:
		return trajectoryRowsFromToolResult(entry.ID, *turn, createdAt, m, subagentByCall)
	default:
		return nil
	}
}

func trajectoryRowsFromAssistant(entryID string, turn int, createdAt time.Time, m msg.AssistantMessage, subagentByCall map[string]string) []TrajectoryRow {
	ts := messageTimestamp(m.Timestamp, createdAt)
	out := make([]TrajectoryRow, 0, 2+len(m.ToolCalls()))
	if thinking := strings.TrimSpace(m.ThinkingText); thinking != "" {
		out = append(out, TrajectoryRow{
			ID:         entryID + "/thinking",
			Turn:       turn,
			Role:       "assistant",
			Kind:       "thinking",
			Text:       thinking,
			DurationMs: m.ThinkingDurationMs,
			CreatedAt:  ts,
			Raw:        map[string]any{"kind": "thinking", "text": thinking},
		})
	}
	if len(m.ToolCalls()) > 0 {
		text := strings.TrimSpace(msg.TrimToolCallStreamContent(m.TextContent()))
		if text != "" {
			out = append(out, TrajectoryRow{
				ID:         entryID + "/assistant",
				Turn:       turn,
				Role:       "assistant",
				Kind:       "assistant",
				Text:       text,
				DurationMs: m.TurnDurationMs,
				CreatedAt:  ts,
				Raw:        map[string]any{"role": "assistant", "text": text},
			})
		}
		for i, call := range m.ToolCalls() {
			out = append(out, trajectoryRowFromToolCall(entryID, turn, ts, i, call, subagentByCall))
		}
		return out
	}
	text := strings.TrimSpace(msg.AssistantDisplayText(m))
	if text == "" {
		return out
	}
	return append(out, TrajectoryRow{
		ID:         entryID,
		Turn:       turn,
		Role:       "assistant",
		Kind:       "assistant",
		Text:       text,
		DurationMs: m.TurnDurationMs,
		CreatedAt:  ts,
		Raw:        map[string]any{"role": "assistant", "text": text},
	})
}

func trajectoryRowFromToolCall(entryID string, turn int, ts time.Time, i int, call msg.ToolCallPart, subagentByCall map[string]string) TrajectoryRow {
	callID := strings.TrimSpace(call.ID)
	if callID == "" {
		callID = strconv.Itoa(i)
	}
	subagent := ""
	if call.Name == delegateSubagentToolName {
		subagent = subagentTypeFromArgs(call.Arguments)
		if callID != "" && subagent != "" && subagentByCall != nil {
			subagentByCall[callID] = subagent
		}
	}
	return TrajectoryRow{
		ID:        entryID + "/tool_call/" + callID,
		Turn:      turn,
		Role:      "assistant",
		Kind:      "tool_call",
		Text:      call.Arguments,
		ToolName:  call.Name,
		Subagent:  subagent,
		CreatedAt: ts,
		Raw: map[string]any{
			"id":        call.ID,
			"name":      call.Name,
			"arguments": call.Arguments,
			"subagent":  subagent,
		},
	}
}

func trajectoryRowsFromToolResult(entryID string, turn int, createdAt time.Time, m msg.ToolResultMessage, subagentByCall map[string]string) []TrajectoryRow {
	text := strings.TrimSpace(textFromParts(m.Parts))
	status := "completed"
	if m.IsError {
		status = "error"
	}
	subagent := ""
	if subagentByCall != nil {
		subagent = subagentByCall[m.ToolCallID]
	}
	return []TrajectoryRow{{
		ID:         entryID,
		Turn:       turn,
		Role:       "tool",
		Kind:       "tool",
		Text:       text,
		ToolName:   m.Name,
		ToolStatus: status,
		Subagent:   subagent,
		DurationMs: m.DurationMs,
		CreatedAt:  messageTimestamp(m.Timestamp, createdAt),
		Raw: map[string]any{
			"name":         m.Name,
			"tool_call_id": m.ToolCallID,
			"text":         text,
			"is_error":     m.IsError,
			"subagent":     subagent,
		},
	}}
}
