package flog

import (
	"fmt"
	"github.com/hibiken/asynq"
)

var AsynqLogger = &asynqLogger{
	Level: asynq.DebugLevel,
}

type asynqLogger struct {
	Level asynq.LogLevel
}

func (a *asynqLogger) Debug(args ...interface{}) {
	l.Debug().Caller(2).Msg(fmt.Sprint(args...))
}

func (a *asynqLogger) Info(args ...interface{}) {
	l.Info().Caller(2).Msg(fmt.Sprint(args...))
}

func (a *asynqLogger) Warn(args ...interface{}) {
	l.Warn().Caller(2).Msg(fmt.Sprint(args...))
}

func (a *asynqLogger) Error(args ...interface{}) {
	l.Error().Caller(2).Msg(fmt.Sprint(args...))
}

func (a *asynqLogger) Fatal(args ...interface{}) {
	l.Fatal().Caller(2).Msg(fmt.Sprint(args...))
}
