package flog

import (
	"errors"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
)

func TestSetLevel(t *testing.T) {
	tests := []struct {
		name          string
		level         string
		expectedLevel zerolog.Level
	}{
		{
			name:          "debug level",
			level:         DebugLevel,
			expectedLevel: zerolog.DebugLevel,
		},
		{
			name:          "info level",
			level:         InfoLevel,
			expectedLevel: zerolog.InfoLevel,
		},
		{
			name:          "warn level",
			level:         WarnLevel,
			expectedLevel: zerolog.WarnLevel,
		},
		{
			name:          "error level",
			level:         ErrorLevel,
			expectedLevel: zerolog.ErrorLevel,
		},
		{
			name:          "fatal level",
			level:         FatalLevel,
			expectedLevel: zerolog.FatalLevel,
		},
		{
			name:          "panic level",
			level:         PanicLevel,
			expectedLevel: zerolog.PanicLevel,
		},
		{
			name:          "invalid level defaults to info",
			level:         "invalid",
			expectedLevel: zerolog.InfoLevel,
		},
		{
			name:          "empty level defaults to info",
			level:         "",
			expectedLevel: zerolog.InfoLevel,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			SetLevel(tt.level)
			assert.Equal(t, tt.expectedLevel, zerolog.GlobalLevel())
		})
	}
}

func TestLevelConstants(t *testing.T) {
	assert.Equal(t, "debug", DebugLevel)
	assert.Equal(t, "info", InfoLevel)
	assert.Equal(t, "warn", WarnLevel)
	assert.Equal(t, "error", ErrorLevel)
	assert.Equal(t, "fatal", FatalLevel)
	assert.Equal(t, "panic", PanicLevel)
}

func TestGetLogger(t *testing.T) {
	Init(Config{Level: "info"})

	logger := GetLogger()
	assert.NotNil(t, logger)
}

func TestInit(t *testing.T) {
	assert.NotPanics(t, func() {
		Init(Config{Level: "info"})
	})

	assert.NotPanics(t, func() {
		Init(Config{Level: "debug", FileLog: true, AlarmEnabled: true})
	})

	assert.NotPanics(t, func() {
		Init(Config{Level: "info", JSONOutput: true})
	})

	assert.NotPanics(t, func() {
		Init(Config{
			Level:   "info",
			FileLog: true,
			Rotation: &RotationConfig{
				MaxSize:    1,
				MaxAge:     7,
				MaxBackups: 3,
				Compress:   false,
			},
		})
	})

	assert.NotPanics(t, func() {
		Init(Config{
			Level: "info",
			Sampling: &SamplingConfig{
				Burst:  5,
				Period: 0,
			},
		})
	})

	assert.NotPanics(t, func() {
		Init(Config{
			Level: "info",
			ModuleLevel: map[string]string{
				"pipeline": "debug",
			},
		})
	})
}

func TestDebug(t *testing.T) {
	Init(Config{Level: DebugLevel})
	SetLevel(DebugLevel)

	assert.NotPanics(t, func() {
		Debug("test debug message: %s", "arg")
	})
}

func TestInfo(t *testing.T) {
	Init(Config{Level: InfoLevel})
	SetLevel(InfoLevel)

	assert.NotPanics(t, func() {
		Info("test info message: %s", "arg")
	})
}

func TestWarn(t *testing.T) {
	Init(Config{Level: WarnLevel})
	SetLevel(WarnLevel)

	assert.NotPanics(t, func() {
		Warn("test warn message: %s", "arg")
	})
}

func TestError(t *testing.T) {
	Init(Config{Level: ErrorLevel})
	SetLevel(ErrorLevel)

	assert.NotPanics(t, func() {
		Error(assert.AnError)
	})
}

func TestErr(t *testing.T) {
	Init(Config{Level: ErrorLevel})
	SetLevel(ErrorLevel)

	assert.NotPanics(t, func() {
		Err(errors.New("test error without alarm"))
	})
}

func TestFatal(t *testing.T) {
	Init(Config{Level: InfoLevel})

	assert.NotPanics(t, func() {
		_ = Fatal
	})
}

func TestPanic(t *testing.T) {
	Init(Config{Level: InfoLevel})

	assert.Panics(t, func() {
		Panic("test panic message: %s", "arg")
	})
}

func TestStructuredFields(t *testing.T) {
	Init(Config{Level: DebugLevel})

	assert.NotPanics(t, func() {
		DebugFields("debug with fields", map[string]any{"key": "value"})
		InfoFields("info with fields", map[string]any{"key": "value"})
		WarnFields("warn with fields", map[string]any{"key": "value"})
	})
}

func TestErrFields(t *testing.T) {
	Init(Config{Level: ErrorLevel})

	assert.NotPanics(t, func() {
		ErrFields(errors.New("test"), "error with fields", map[string]any{"key": "value"})
	})
}

func TestEventHelpers(t *testing.T) {
	Init(Config{Level: DebugLevel})

	assert.NotPanics(t, func() {
		DebugEvt().Str("key", "val").Msg("debug event")
		InfoEvt().Str("key", "val").Msg("info event")
		WarnEvt().Str("key", "val").Msg("warn event")
	})
}

func TestErrorEvt(t *testing.T) {
	Init(Config{Level: ErrorLevel})

	assert.NotPanics(t, func() {
		ErrorEvt().Err(errors.New("test")).Str("key", "val").Msg("error event")
	})
}

func TestModule(t *testing.T) {
	Init(Config{
		Level: DebugLevel,
		ModuleLevel: map[string]string{
			"testmodule": "warn",
		},
	})

	m := Module("testmodule")
	assert.NotNil(t, m)

	m2 := Module("nonexistent")
	assert.NotNil(t, m2)
}

func TestSampled(t *testing.T) {
	Init(Config{
		Level: InfoLevel,
		Sampling: &SamplingConfig{
			Burst:  3,
			Period: 0,
		},
	})

	s := Sampled()
	assert.NotNil(t, s)

	assert.NotPanics(t, func() {
		s.Info().Msg("sampled info")
	})
}

func TestSetModuleLevel(t *testing.T) {
	Init(Config{Level: InfoLevel})

	SetModuleLevel("m1", "debug")
	m := Module("m1")
	assert.NotNil(t, m)
}
