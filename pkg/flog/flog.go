package flog

import (
	"fmt"
	"io"
	"os"
	"time"

	"github.com/flowline-io/flowbot/pkg/alarm"
	jsoniter "github.com/json-iterator/go"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/pkgerrors"
)

var l zerolog.Logger

func Init(fileLogEnabled bool) {
	// error stack
	zerolog.ErrorStackMarshaler = pkgerrors.MarshalStack
	// json marshaling
	json := jsoniter.ConfigCompatibleWithStandardLibrary
	zerolog.InterfaceMarshalFunc = json.Marshal

	var writer []io.Writer
	// console
	console := zerolog.ConsoleWriter{
		Out:        os.Stdout,
		TimeFormat: time.DateTime,
		NoColor:    true,
		FormatLevel: func(i interface{}) string {
			return fmt.Sprintf("%s", i)
		},
	}
	writer = append(writer, console)

	// file
	if fileLogEnabled {
		runLogFile, err := os.OpenFile(
			"flowbot.log", // todo file path
			os.O_APPEND|os.O_CREATE|os.O_WRONLY,
			0664,
		)
		if err != nil {
			Error(err)
		} else {
			writer = append(writer, runLogFile)
		}
	}

	multi := zerolog.MultiLevelWriter(writer...)
	l = zerolog.New(multi).With().Timestamp().Logger()
}

func GetLogger() zerolog.Logger {
	return l
}

const (
	DebugLevel = "debug"
	InfoLevel  = "info"
	WarnLevel  = "warn"
	ErrorLevel = "error"
	FatalLevel = "fatal"
	PanicLevel = "panic"
)

// SetLevel sets the global logging level based on the provided level.
//
// level: The logging level to set. Valid values are "debug", "info", "warn",
//
//	"error", "fatal", "panic". If an invalid level is provided, the
//	default level is set to "info".
func SetLevel(level string) {
	switch level {
	case DebugLevel:
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	case InfoLevel:
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	case WarnLevel:
		zerolog.SetGlobalLevel(zerolog.WarnLevel)
	case ErrorLevel:
		zerolog.SetGlobalLevel(zerolog.ErrorLevel)
	case FatalLevel:
		zerolog.SetGlobalLevel(zerolog.FatalLevel)
	case PanicLevel:
		zerolog.SetGlobalLevel(zerolog.PanicLevel)
	default:
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	}
}

func Debug(format string, a ...any) {
	l.Debug().Caller(1).Msgf(format, a...)
}

func Info(format string, a ...any) {
	l.Info().Caller(1).Msgf(format, a...)
}

func Warn(format string, a ...any) {
	l.Warn().Caller(1).Msgf(format, a...)
}

func Error(err error) {
	// alarm error
	alarm.Alarm(err, 0)
	// print error
	l.Error().Caller(1).Stack().Err(err).Msg(err.Error())
}

func Fatal(format string, a ...any) {
	l.Fatal().Caller(1).Msgf(format, a...)
}

func Panic(format string, a ...any) {
	l.Panic().Caller(1).Msgf(format, a...)
}
