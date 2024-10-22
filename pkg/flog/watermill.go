package flog

import (
	"github.com/ThreeDotsLabs/watermill"
	"github.com/rs/zerolog"
)

var WatermillLogger = &watermillLogger{}

type watermillLogger struct{}

func (w *watermillLogger) Error(msg string, err error, fields watermill.LogFields) {
	t := l.Error().Caller(1).Err(err)
	if fields != nil {
		addWatermillFieldsData(t, fields)
	}
	t.Msg(msg)
}

func (w *watermillLogger) Info(msg string, fields watermill.LogFields) {
	t := l.Info().Caller(1)
	if fields != nil {
		addWatermillFieldsData(t, fields)
	}
	t.Msg(msg)
}

func (w *watermillLogger) Debug(msg string, fields watermill.LogFields) {
	t := l.Debug().Caller(1)
	if fields != nil {
		addWatermillFieldsData(t, fields)
	}
	t.Msg(msg)
}

func (w *watermillLogger) Trace(msg string, fields watermill.LogFields) {
	t := l.Trace().Caller(1)
	if fields != nil {
		addWatermillFieldsData(t, fields)
	}
	t.Msg(msg)
}

func (w *watermillLogger) With(fields watermill.LogFields) watermill.LoggerAdapter {
	if fields == nil {
		return w
	}
	subLog := l.With()
	for i, v := range fields {
		subLog = subLog.Any(i, v)
	}

	return w
}

func addWatermillFieldsData(event *zerolog.Event, fields watermill.LogFields) {
	for i, v := range fields {
		event.Any(i, v)
	}
}
