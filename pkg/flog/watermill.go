package flog

import (
	"fmt"

	"github.com/ThreeDotsLabs/watermill"
)

var WatermillLogger = &watermillLogger{}

type watermillLogger struct{}

func (w *watermillLogger) Error(msg string, err error, fields watermill.LogFields) {
	t := l.Error().Caller(1)
	for k, v := range fields {
		t = t.Any(k, v)
	}
	t.Msg(fmt.Sprintf("%s error: %v", msg, err))
}

func (w *watermillLogger) Info(msg string, fields watermill.LogFields) {
	t := l.Info().Caller(1)
	for k, v := range fields {
		t = t.Any(k, v)
	}
	t.Msg(msg)
}

func (w *watermillLogger) Debug(msg string, fields watermill.LogFields) {
	t := l.Debug().Caller(1)
	for k, v := range fields {
		t = t.Any(k, v)
	}
	t.Msg(msg)
}

func (w *watermillLogger) Trace(msg string, fields watermill.LogFields) {
	t := l.Trace().Caller(1)
	for k, v := range fields {
		t = t.Any(k, v)
	}
	t.Msg(msg)
}

func (w *watermillLogger) With(fields watermill.LogFields) watermill.LoggerAdapter {
	return w
}
