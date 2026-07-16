package flog

import (
	"bytes"
	"errors"
	"strings"
	"syscall"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/fx/fxevent"
)

func TestFxLogger_LogEvent(t *testing.T) {
	tests := []struct {
		name       string
		event      fxevent.Event
		wantEmpty  bool
		wantSubstr string
		wantLevel  string
	}{
		{
			name:      "provided success is discarded",
			event:     &fxevent.Provided{ConstructorName: "pkg.NewThing()"},
			wantEmpty: true,
		},
		{
			name:      "invoking is discarded",
			event:     &fxevent.Invoking{FunctionName: "pkg.Init()"},
			wantEmpty: true,
		},
		{
			name:      "on start executing is discarded",
			event:     &fxevent.OnStartExecuting{FunctionName: "start", CallerName: "caller"},
			wantEmpty: true,
		},
		{
			name:      "on start executed success is discarded",
			event:     &fxevent.OnStartExecuted{FunctionName: "start", CallerName: "caller"},
			wantEmpty: true,
		},
		{
			name:       "started is info",
			event:      &fxevent.Started{},
			wantSubstr: "[fx] started",
			wantLevel:  "info",
		},
		{
			name:       "stopping is info",
			event:      &fxevent.Stopping{Signal: syscall.SIGINT},
			wantSubstr: "[fx] received signal",
			wantLevel:  "info",
		},
		{
			name:       "started error is error",
			event:      &fxevent.Started{Err: errors.New("boot failed")},
			wantSubstr: "boot failed",
			wantLevel:  "error",
		},
		{
			name:       "invoked error is error",
			event:      &fxevent.Invoked{FunctionName: "bad", Err: errors.New("invoke boom")},
			wantSubstr: "invoke boom",
			wantLevel:  "error",
		},
		{
			name:       "provided error is error",
			event:      &fxevent.Provided{ConstructorName: "bad", Err: errors.New("provide boom")},
			wantSubstr: "provide boom",
			wantLevel:  "error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			prev := l
			t.Cleanup(func() { l = prev })
			l = zerolog.New(&buf).Level(zerolog.DebugLevel)

			logger := NewFxLogger()
			require.NotNil(t, logger)
			logger.LogEvent(tt.event)

			out := buf.String()
			if tt.wantEmpty {
				assert.Empty(t, out)
				return
			}
			assert.Contains(t, out, tt.wantSubstr)
			assert.Contains(t, strings.ToLower(out), `"level":"`+tt.wantLevel+`"`)
		})
	}
}

func TestFxLogger_NewFxLogger(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
	}{
		{name: "returns non-nil logger"},
		{name: "implements fxevent.Logger"},
		{name: "safe to construct repeatedly"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var _ fxevent.Logger = NewFxLogger()
			assert.NotNil(t, NewFxLogger())
		})
	}
}

func TestWatermillLogger_InfoIsDiscarded(t *testing.T) {
	tests := []struct {
		name string
		msg  string
	}{
		{name: "adding handler message", msg: "Adding handler"},
		{name: "subscribing message", msg: "Subscribing to topic"},
		{name: "router closed message", msg: "Router closed"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			prev := l
			t.Cleanup(func() { l = prev })
			l = zerolog.New(&buf).Level(zerolog.DebugLevel)

			WatermillLogger.Info(tt.msg, nil)
			assert.Empty(t, buf.String())
		})
	}
}
