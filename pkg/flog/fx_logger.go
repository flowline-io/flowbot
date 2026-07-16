package flog

import (
	"strings"

	"go.uber.org/fx/fxevent"
)

// FxLogger adapts Fx lifecycle events to flog with reduced verbosity.
// Successful provide/invoke/hook chatter is discarded so debug log level stays
// usable; failures and process signals remain Info/Error.
type FxLogger struct{}

// NewFxLogger returns an fxevent.Logger that routes Fx events through flog.
func NewFxLogger() fxevent.Logger {
	return &FxLogger{}
}

// LogEvent implements fxevent.Logger.
func (*FxLogger) LogEvent(event fxevent.Event) {
	if logFxHookEvent(event) {
		return
	}
	if logFxGraphEvent(event) {
		return
	}
	logFxProcessEvent(event)
}

func logFxHookEvent(event fxevent.Event) bool {
	switch e := event.(type) {
	case *fxevent.OnStartExecuting, *fxevent.OnStopExecuting:
		return true
	case *fxevent.OnStartExecuted:
		logFxErrOnly(e.Err)
		return true
	case *fxevent.OnStopExecuted:
		logFxErrOnly(e.Err)
		return true
	default:
		return false
	}
}

func logFxGraphEvent(event fxevent.Event) bool {
	switch e := event.(type) {
	case *fxevent.Supplied:
		logFxErrOnly(e.Err)
		return true
	case *fxevent.Provided:
		logFxErrOnly(e.Err)
		return true
	case *fxevent.Replaced:
		logFxErrOnly(e.Err)
		return true
	case *fxevent.Decorated:
		logFxErrOnly(e.Err)
		return true
	case *fxevent.BeforeRun, *fxevent.Invoking:
		return true
	case *fxevent.Run:
		logFxErrOnly(e.Err)
		return true
	case *fxevent.Invoked:
		logFxErrOnly(e.Err)
		return true
	default:
		return false
	}
}

func logFxProcessEvent(event fxevent.Event) {
	switch e := event.(type) {
	case *fxevent.Stopping:
		Info("[fx] received signal %s", strings.ToUpper(e.Signal.String()))
	case *fxevent.Stopped:
		logFxErrOnly(e.Err)
	case *fxevent.RollingBack:
		Error(e.StartErr)
	case *fxevent.RolledBack:
		logFxErrOnly(e.Err)
	case *fxevent.Started:
		logFxErrOrInfo(e.Err, "[fx] started")
	case *fxevent.LoggerInitialized:
		logFxErrOnly(e.Err)
	}
}

func logFxErrOnly(err error) {
	if err != nil {
		Error(err)
	}
}

func logFxErrOrInfo(err error, format string, a ...any) {
	if err != nil {
		Error(err)
		return
	}
	Info(format, a...)
}
