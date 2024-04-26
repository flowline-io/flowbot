package flog

import (
	"fmt"

	"github.com/hibiken/asynq"
)

var AsynqLogger = &asynqLogger{}

type asynqLogger struct {
	Level asynq.LogLevel
}

func (a *asynqLogger) Debug(args ...interface{}) {
	l.Debug().Caller(3).Msg(fmt.Sprint(args...))
}

func (a *asynqLogger) Info(args ...interface{}) {
	l.Info().Caller(3).Msg(fmt.Sprint(args...))
}

func (a *asynqLogger) Warn(args ...interface{}) {
	l.Warn().Caller(3).Msg(fmt.Sprint(args...))
}

func (a *asynqLogger) Error(args ...interface{}) {
	l.Error().Caller(3).Msg(fmt.Sprint(args...))
}

func (a *asynqLogger) Fatal(args ...interface{}) {
	l.Fatal().Caller(3).Msg(fmt.Sprint(args...))
}

func AsynqLogLevel(level string) asynq.LogLevel {
	switch level {
	case DebugLevel:
		return asynq.DebugLevel
	case InfoLevel:
		return asynq.InfoLevel
	case WarnLevel:
		return asynq.WarnLevel
	case ErrorLevel:
		return asynq.ErrorLevel
	case FatalLevel:
		return asynq.FatalLevel
	default:
		return asynq.InfoLevel
	}
}
