package chatagent

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/types"
)

// EphemeralRunParams describes one isolated autonomous agent turn.
type EphemeralRunParams struct {
	UID         types.Uid
	Prompt      string
	Kind        RunKind
	Tools       []string
	Skills      []string
	MemoryScope string
}

// EphemeralRunResult holds the outcome of one ephemeral run.
type EphemeralRunResult struct {
	SessionID string
	Reply     string
}

// IsAutonomousRunKind reports whether the run uses scheduled-style permissions.
func IsAutonomousRunKind(kind RunKind) bool {
	return kind == RunKindScheduled || kind == RunKindPipeline
}

// BeginEphemeralSession creates a temporary chat session for one autonomous run.
func BeginEphemeralSession(ctx context.Context, uid types.Uid) (sessionID string, err error) {
	sessionID = types.Id()
	if err := CreateSession(ctx, uid, sessionID); err != nil {
		return "", err
	}
	return sessionID, nil
}

// CloseEphemeralSession closes a temporary session. Failures are logged and not returned.
func CloseEphemeralSession(ctx context.Context, sessionID string) {
	if sessionID == "" {
		return
	}
	if err := CloseSession(ctx, sessionID); err != nil {
		flog.Warn("[chat-agent] ephemeral session close session=%s: %v", sessionID, err)
	}
}

// RunAutonomousPrompt executes one prompt in an existing session with RunTimeout.
func RunAutonomousPrompt(ctx context.Context, svc *Service, sessionID, prompt string, kind RunKind, tools, skills []string, memoryScope string) (string, error) {
	runCtx, cancel := context.WithTimeout(ctx, RunTimeout())
	defer cancel()
	return svc.Run(runCtx, RunRequest{
		SessionID:   sessionID,
		Text:        prompt,
		Kind:        kind,
		Tools:       tools,
		Skills:      skills,
		MemoryScope: memoryScope,
	}, nil)
}

// RunEphemeral creates a temporary session, runs one prompt, and closes the session.
func RunEphemeral(ctx context.Context, svc *Service, params EphemeralRunParams) (EphemeralRunResult, error) {
	sessionID, err := BeginEphemeralSession(ctx, params.UID)
	if err != nil {
		return EphemeralRunResult{}, fmt.Errorf("begin ephemeral session: %w", err)
	}
	if params.Kind == RunKindPipeline {
		flog.Info("[pipeline-agent] ephemeral session created uid=%s session=%s", params.UID, sessionID)
	}
	defer CloseEphemeralSession(ctx, sessionID)

	if params.Kind == RunKindPipeline {
		flog.Info("[pipeline-agent] autonomous prompt start session=%s prompt_len=%d timeout=%s",
			sessionID, len(strings.TrimSpace(params.Prompt)), RunTimeout())
	}
	promptStart := time.Now()
	reply, err := RunAutonomousPrompt(ctx, svc, sessionID, params.Prompt, params.Kind, params.Tools, params.Skills, params.MemoryScope)
	if params.Kind == RunKindPipeline {
		flog.Info("[pipeline-agent] autonomous prompt end session=%s duration=%s err=%v",
			sessionID, time.Since(promptStart).Round(time.Millisecond), err)
	}
	if err != nil {
		return EphemeralRunResult{SessionID: sessionID}, err
	}
	return EphemeralRunResult{SessionID: sessionID, Reply: reply}, nil
}
