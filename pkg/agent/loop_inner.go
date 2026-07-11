package agent

import (
	"context"
	"time"

	agentevent "github.com/flowline-io/flowbot/pkg/agent/event"
	"github.com/flowline-io/flowbot/pkg/agent/tool"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/metrics"
	"github.com/flowline-io/flowbot/pkg/trace"
)

type innerLoopState struct {
	ctx         context.Context
	current     *Context
	cfg         Config
	deps        LoopDeps
	emit        func(agentevent.Event) error
	newMessages *[]AgentMessage
	pending     *[]AgentMessage
	steps       *int
}

func (s *innerLoopState) runTurn() (stopInner bool, err error) {
	turnStart := time.Now()
	ctx, span := trace.StartSpan(s.ctx, "agent.turn")
	defer span.End()
	s.ctx = ctx

	if len(*s.pending) > 0 {
		for _, message := range *s.pending {
			s.current.Messages = append(s.current.Messages, message)
			*s.newMessages = append(*s.newMessages, message)
			if err := emitMessage(s.emit, message); err != nil {
				metrics.Agent().ObserveTurnDuration("error", time.Since(turnStart).Seconds())
				return false, err
			}
		}
		*s.pending = nil
	}

	*s.steps++
	if *s.steps > s.cfg.MaxSteps {
		metrics.Agent().ObserveTurnDuration("error", time.Since(turnStart).Seconds())
		return false, ErrMaxSteps
	}

	if err := s.emit(agentevent.Event{Type: agentevent.TypeTurnStart}); err != nil {
		metrics.Agent().ObserveTurnDuration("error", time.Since(turnStart).Seconds())
		return false, err
	}

	flog.Debug("agent loop: turn start step=%d model=%s", *s.steps, turnModelName(s.cfg, s.current))

	assistant, err := streamAssistant(s.ctx, s.current, s.cfg, s.deps, s.emit)
	if err != nil {
		metrics.Agent().ObserveTurnDuration("error", time.Since(turnStart).Seconds())
		trace.RecordError(s.ctx, err)
		return false, err
	}
	assistant.Timestamp = time.Now().UTC()
	assistantIdx := len(s.current.Messages)
	s.current.Messages = append(s.current.Messages, assistant)
	*s.newMessages = append(*s.newMessages, assistant)

	toolResults, terminate, hasToolCalls, err := s.executeTools(assistant)
	if err != nil {
		metrics.Agent().ObserveTurnDuration("error", time.Since(turnStart).Seconds())
		return false, err
	}

	turnDurationMs := time.Since(turnStart).Milliseconds()
	assistant.TurnDurationMs = turnDurationMs
	s.current.Messages[assistantIdx] = assistant
	newAssistantIdx := len(*s.newMessages) - len(toolResults) - 1
	if newAssistantIdx >= 0 && newAssistantIdx < len(*s.newMessages) {
		(*s.newMessages)[newAssistantIdx] = assistant
	}

	if err := s.emit(agentevent.Event{
		Type:        agentevent.TypeTurnEnd,
		Message:     assistant,
		ToolResults: toolResults,
		DurationMs:  turnDurationMs,
		Step:        *s.steps,
	}); err != nil {
		metrics.Agent().ObserveTurnDuration("error", time.Since(turnStart).Seconds())
		return false, err
	}

	if err := s.applyTurnHooks(assistant, toolResults); err != nil {
		metrics.Agent().ObserveTurnDuration("error", time.Since(turnStart).Seconds())
		return false, err
	}
	if terminate {
		metrics.Agent().ObserveTurnDuration("ok", time.Since(turnStart).Seconds())
		return false, errStopAfterTurn
	}

	stopInner, contErr := s.continueAfterTurn(hasToolCalls)
	if contErr != nil {
		metrics.Agent().ObserveTurnDuration("error", time.Since(turnStart).Seconds())
		return stopInner, contErr
	}
	metrics.Agent().ObserveTurnDuration("ok", time.Since(turnStart).Seconds())
	return stopInner, nil
}

func (s *innerLoopState) executeTools(assistant AssistantMessage) ([]ToolResultMessage, bool, bool, error) {
	hasToolCalls := len(assistant.ToolCalls()) > 0
	if !hasToolCalls {
		return nil, false, false, nil
	}

	batch, err := tool.ExecuteBatch(s.ctx, tool.BatchRequest{
		Assistant: assistant,
		Context:   s.current,
		Registry:  s.deps.Registry,
		Mode:      s.cfg.ToolExecution,
		Before:    s.cfg.BeforeToolCall,
		After:     s.cfg.AfterToolCall,
		Emit: func(_ context.Context, ev agentevent.Event) error {
			return s.emit(ev)
		},
	})
	if err != nil {
		return nil, false, hasToolCalls, err
	}

	for i := range batch.Messages {
		batch.Messages[i].Timestamp = time.Now().UTC()
		s.current.Messages = append(s.current.Messages, batch.Messages[i])
		*s.newMessages = append(*s.newMessages, batch.Messages[i])
		if err := emitMessage(s.emit, batch.Messages[i]); err != nil {
			return nil, batch.Terminate, hasToolCalls, err
		}
	}
	return batch.Messages, batch.Terminate, hasToolCalls, nil
}

func (s *innerLoopState) applyTurnHooks(assistant AssistantMessage, toolResults []ToolResultMessage) error {
	turnCtx := TurnContext{
		Message:     assistant,
		ToolResults: toolResults,
		Context:     s.current,
		NewMessages: append([]AgentMessage(nil), *s.newMessages...),
	}
	if s.cfg.PrepareNextTurn != nil {
		update, err := s.cfg.PrepareNextTurn(turnCtx)
		if err != nil {
			return err
		}
		if update != nil {
			if update.Context != nil {
				s.current = update.Context
			}
			if update.ModelName != "" {
				prevModel := turnModelName(s.cfg, s.current)
				s.current.ModelName = update.ModelName
				s.cfg.ModelName = update.ModelName
				if update.ModelName != prevModel {
					flog.Debug("agent loop: model switch step=%d from=%s to=%s after_tools=%t",
						*s.steps, prevModel, update.ModelName, len(toolResults) > 0)
				}
			}
		}
	}
	if s.cfg.ShouldStopAfterTurn != nil {
		stop, err := s.cfg.ShouldStopAfterTurn(turnCtx)
		if err != nil {
			return err
		}
		if stop {
			return errStopAfterTurn
		}
	}
	return nil
}

type stopAfterTurnError struct{}

func (stopAfterTurnError) Error() string { return "agent: stop after turn" }

var errStopAfterTurn = stopAfterTurnError{}

func (s *innerLoopState) continueAfterTurn(hasToolCalls bool) (bool, error) {
	if !hasToolCalls {
		if err := s.drainSteering(); err != nil {
			return false, err
		}
		return len(*s.pending) == 0, nil
	}
	if err := s.drainSteering(); err != nil {
		return false, err
	}
	return false, nil
}

func (s *innerLoopState) drainSteering() error {
	if s.cfg.GetSteeringMessages == nil {
		return nil
	}
	steering, err := s.cfg.GetSteeringMessages()
	if err != nil {
		return err
	}
	*s.pending = drainQueue(*s.pending, steering, s.cfg.SteeringMode)
	return nil
}
