// Package flog provides structured logging for Flowbot.
package flog

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	"github.com/adrg/xdg"
	"github.com/bytedance/sonic"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/pkgerrors"
	"go.opentelemetry.io/otel/trace"
	"gopkg.in/natefinch/lumberjack.v2"
)

var (
	stateMu            sync.RWMutex
	l                  zerolog.Logger
	sampled            zerolog.Logger
	callerOn           atomic.Bool
	stackOn            atomic.Bool
	moduleLogs         sync.Map // map[string]*zerolog.Logger
	moduleLvls         sync.Map // map[string]zerolog.Level
	defaultLvl         zerolog.Level
	zerologGlobalsInit sync.Once
)

// Config holds all logging configuration.
type Config struct {
	Level       string
	Caller      bool
	StackTrace  bool
	JSONOutput  bool
	FileLog     bool
	FileLogPath string
	ModuleLevel map[string]string
	Sampling    *SamplingConfig
	Rotation    *RotationConfig
}

// SamplingConfig controls log sampling to reduce noise from high-frequency log points.
type SamplingConfig struct {
	Burst  int
	Period time.Duration
}

// RotationConfig controls log file rotation via lumberjack.
type RotationConfig struct {
	MaxSize    int
	MaxAge     int
	MaxBackups int
	Compress   bool
}

// Init initializes the logging subsystem. Must be called once at startup.
func Init(cfg Config) {
	zerologGlobalsInit.Do(func() {
		zerolog.ErrorStackMarshaler = pkgerrors.MarshalStack
		zerolog.InterfaceMarshalFunc = sonic.Marshal
	})

	callerOn.Store(cfg.Caller)
	stackOn.Store(cfg.StackTrace)

	var writers []io.Writer

	// stdout
	if cfg.JSONOutput {
		writers = append(writers, os.Stdout)
	} else {
		console := zerolog.ConsoleWriter{
			Out:        os.Stdout,
			TimeFormat: time.DateTime,
			NoColor:    true,
			FormatLevel: func(i any) string {
				return fmt.Sprintf("%s", i)
			},
		}
		writers = append(writers, console)
	}

	// file
	if cfg.FileLog {
		logPath := cfg.FileLogPath
		if logPath == "" {
			dir := filepath.Join(xdg.ConfigHome, "flowbot")
			if err := os.MkdirAll(dir, 0700); err != nil {
				panic(err)
			}
			logPath = filepath.Join(dir, "flowbot.log")
		}

		if cfg.Rotation != nil && cfg.Rotation.MaxSize > 0 {
			writers = append(writers, &lumberjack.Logger{
				Filename:   logPath,
				MaxSize:    cfg.Rotation.MaxSize,
				MaxAge:     cfg.Rotation.MaxAge,
				MaxBackups: cfg.Rotation.MaxBackups,
				Compress:   cfg.Rotation.Compress,
			})
		} else {
			f, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
			if err != nil {
				Err(fmt.Errorf("flog: failed to open log file: %w", err))
			} else {
				writers = append(writers, f)
			}
		}
	}

	multi := zerolog.MultiLevelWriter(writers...)

	stateMu.Lock()

	l = zerolog.New(multi).With().Timestamp().Logger()

	// level
	defaultLvl = zerologLevel(cfg.Level)

	// per-module levels
	for name, lvlStr := range cfg.ModuleLevel {
		setModuleLevelLocked(name, lvlStr)
	}

	// sampling
	if cfg.Sampling != nil && cfg.Sampling.Burst > 0 {
		period := cfg.Sampling.Period
		if period == 0 {
			period = time.Second
		}
		sampled = l.Sample(&zerolog.BurstSampler{
			Burst:  uint32(cfg.Sampling.Burst),
			Period: period,
		})
	} else {
		sampled = l
	}

	stateMu.Unlock()

	syncGlobalLevelLocked()
}

// GetLogger returns the underlying zerolog.Logger.
func GetLogger() zerolog.Logger {
	stateMu.RLock()
	defer stateMu.RUnlock()
	return l
}

// Ctx returns a zerolog.Logger annotated with trace_id and span_id from the
// OpenTelemetry span in the given context. If no span is present, the global
// logger is returned unchanged.
func Ctx(ctx context.Context) *zerolog.Logger {
	span := trace.SpanFromContext(ctx)
	if !span.SpanContext().IsValid() {
		stateMu.RLock()
		lg := l
		stateMu.RUnlock()
		return &lg
	}
	stateMu.RLock()
	lg := l
	stateMu.RUnlock()
	lctx := lg.With().
		Str("trace_id", span.SpanContext().TraceID().String()).
		Str("span_id", span.SpanContext().SpanID().String()).
		Logger()
	return &lctx
}

// Module returns a logger with the configured per-module log level.
// Falls back to the global logger if the module is not configured.
func Module(name string) *zerolog.Logger {
	if lgr, ok := moduleLogs.Load(name); ok {
		if log, ok := lgr.(*zerolog.Logger); ok {
			return log
		}
	}
	stateMu.RLock()
	lg := l
	stateMu.RUnlock()
	return &lg
}

// Sampled returns a logger with burst sampling applied.
func Sampled() *zerolog.Logger {
	stateMu.RLock()
	s := sampled
	stateMu.RUnlock()
	return &s
}

const (
	DebugLevel = "debug"
	InfoLevel  = "info"
	WarnLevel  = "warn"
	ErrorLevel = "error"
	FatalLevel = "fatal"
	PanicLevel = "panic"
)

// SetLevel sets the global default log level.
func SetLevel(level string) {
	stateMu.Lock()
	defaultLvl = zerologLevel(level)
	stateMu.Unlock()
	syncGlobalLevelLocked()
}

// SetModuleLevel sets the log level for a specific module.
func SetModuleLevel(name, lvlStr string) {
	stateMu.Lock()
	setModuleLevelLocked(name, lvlStr)
	stateMu.Unlock()
	syncGlobalLevelLocked()
}

func setModuleLevelLocked(name, lvlStr string) {
	lvl := zerologLevel(lvlStr)
	moduleLvls.Store(name, lvl)
	ml := l.Level(lvl)
	moduleLogs.Store(name, &ml)
}

func syncGlobalLevelLocked() {
	stateMu.RLock()
	minLvl := defaultLvl
	stateMu.RUnlock()
	moduleLvls.Range(func(_ any, v any) bool {
		lvl, ok := v.(zerolog.Level)
		if !ok {
			return true
		}
		if lvl < minLvl {
			minLvl = lvl
		}
		return true
	})
	zerolog.SetGlobalLevel(minLvl)
}

func mustCaller() bool {
	if callerOn.Load() {
		return true
	}
	return zerolog.GlobalLevel() <= zerolog.DebugLevel
}

func mustStack() bool {
	return stackOn.Load()
}

// Event helpers for structured logging. Use these when you need to attach
// typed fields (.Str, .Int, .Dur, etc.) to a log line instead of Msgf.
//
//	flog.InfoEvt().Str("pipeline", name).Int("steps", n).Msg("started")

// DebugEvt returns a Debug-level event pre-configured with caller info.
func DebugEvt() *zerolog.Event {
	stateMu.RLock()
	evt := l.Debug()
	stateMu.RUnlock()
	if mustCaller() {
		evt = evt.Caller(1)
	}
	return evt
}

// InfoEvt returns an Info-level event pre-configured with caller info.
func InfoEvt() *zerolog.Event {
	stateMu.RLock()
	evt := l.Info()
	stateMu.RUnlock()
	if mustCaller() {
		evt = evt.Caller(1)
	}
	return evt
}

// WarnEvt returns a Warn-level event pre-configured with caller info.
func WarnEvt() *zerolog.Event {
	stateMu.RLock()
	evt := l.Warn()
	stateMu.RUnlock()
	if mustCaller() {
		evt = evt.Caller(1)
	}
	return evt
}

// ErrorEvt returns an Error-level event pre-configured with caller and stack info.
func ErrorEvt() *zerolog.Event {
	stateMu.RLock()
	evt := l.Error()
	stateMu.RUnlock()
	if mustCaller() {
		evt = evt.Caller(1)
	}
	if mustStack() {
		evt = evt.Stack()
	}
	return evt
}

// Field helpers for attaching a batch of key-value pairs to a log event.

func addFields(evt *zerolog.Event, fields map[string]any) {
	for k, v := range fields {
		evt.Any(k, v)
	}
}

// DebugFields logs a debug message with structured fields.
func DebugFields(msg string, fields map[string]any) {
	stateMu.RLock()
	evt := l.Debug()
	stateMu.RUnlock()
	if mustCaller() {
		evt = evt.Caller(1)
	}
	addFields(evt, fields)
	evt.Msg(msg)
}

// InfoFields logs an info message with structured fields.
func InfoFields(msg string, fields map[string]any) {
	stateMu.RLock()
	evt := l.Info()
	stateMu.RUnlock()
	if mustCaller() {
		evt = evt.Caller(1)
	}
	addFields(evt, fields)
	evt.Msg(msg)
}

// WarnFields logs a warning message with structured fields.
func WarnFields(msg string, fields map[string]any) {
	stateMu.RLock()
	evt := l.Warn()
	stateMu.RUnlock()
	if mustCaller() {
		evt = evt.Caller(1)
	}
	addFields(evt, fields)
	evt.Msg(msg)
}

// ErrFields logs an error with structured fields, without triggering alarm.
func ErrFields(err error, msg string, fields map[string]any) {
	stateMu.RLock()
	evt := l.Error().Err(err)
	stateMu.RUnlock()
	if mustCaller() {
		evt = evt.Caller(1)
	}
	if mustStack() {
		evt = evt.Stack()
	}
	addFields(evt, fields)
	evt.Msg(msg)
}

// Package-level convenience functions.

// Debug logs a formatted debug message.
func Debug(format string, a ...any) {
	stateMu.RLock()
	evt := l.Debug()
	stateMu.RUnlock()
	if mustCaller() {
		evt = evt.Caller(1)
	}
	evt.Msgf(format, a...)
}

// Info logs a formatted info message.
func Info(format string, a ...any) {
	stateMu.RLock()
	evt := l.Info()
	stateMu.RUnlock()
	if mustCaller() {
		evt = evt.Caller(1)
	}
	evt.Msgf(format, a...)
}

// Warn logs a formatted warning message.
func Warn(format string, a ...any) {
	stateMu.RLock()
	evt := l.Warn()
	stateMu.RUnlock()
	if mustCaller() {
		evt = evt.Caller(1)
	}
	evt.Msgf(format, a...)
}

// Error logs an error.
func Error(err error) {
	Err(err)
}

// Err logs an error without triggering alarm.
func Err(err error) {
	stateMu.RLock()
	evt := l.Error().Err(err)
	stateMu.RUnlock()
	if mustCaller() {
		evt = evt.Caller(1)
	}
	if mustStack() {
		evt = evt.Stack()
	}
	evt.Msg("error occurred")
}

// Fatal logs a formatted fatal message and exits the program.
func Fatal(format string, a ...any) {
	stateMu.RLock()
	evt := l.Fatal()
	stateMu.RUnlock()
	if mustCaller() {
		evt = evt.Caller(1)
	}
	evt.Msgf(format, a...)
}

// Panic logs a formatted panic message and panics.
func Panic(format string, a ...any) {
	stateMu.RLock()
	evt := l.Panic()
	stateMu.RUnlock()
	if mustCaller() {
		evt = evt.Caller(1)
	}
	evt.Msgf(format, a...)
}

// zerologLevel converts a level string to a zerolog.Level.
func zerologLevel(level string) zerolog.Level {
	switch level {
	case DebugLevel:
		return zerolog.DebugLevel
	case InfoLevel:
		return zerolog.InfoLevel
	case WarnLevel:
		return zerolog.WarnLevel
	case ErrorLevel:
		return zerolog.ErrorLevel
	case FatalLevel:
		return zerolog.FatalLevel
	case PanicLevel:
		return zerolog.PanicLevel
	default:
		return zerolog.InfoLevel
	}
}
