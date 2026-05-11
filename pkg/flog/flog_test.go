package flog

import (
	"errors"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
)

func TestSetLevel(t *testing.T) {
	// Clear stale module levels from other tests that would cause
	// syncGlobalLevel to compute a lower minimum.
	moduleLvls.Clear()

	tests := []struct {
		name          string
		level         string
		expectedLevel zerolog.Level
	}{
		{name: "debug level", level: DebugLevel, expectedLevel: zerolog.DebugLevel},
		{name: "info level", level: InfoLevel, expectedLevel: zerolog.InfoLevel},
		{name: "warn level", level: WarnLevel, expectedLevel: zerolog.WarnLevel},
		{name: "error level", level: ErrorLevel, expectedLevel: zerolog.ErrorLevel},
		{name: "fatal level", level: FatalLevel, expectedLevel: zerolog.FatalLevel},
		{name: "panic level", level: PanicLevel, expectedLevel: zerolog.PanicLevel},
		{name: "invalid level defaults to info", level: "invalid", expectedLevel: zerolog.InfoLevel},
		{name: "empty level defaults to info", level: "", expectedLevel: zerolog.InfoLevel},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			SetLevel(tt.level)
			assert.Equal(t, tt.expectedLevel, zerolog.GlobalLevel())
		})
	}
}

func TestLevelConstants(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		got  string
		want string
	}{
		{name: "DebugLevel", got: DebugLevel, want: "debug"},
		{name: "InfoLevel", got: InfoLevel, want: "info"},
		{name: "WarnLevel", got: WarnLevel, want: "warn"},
		{name: "ErrorLevel", got: ErrorLevel, want: "error"},
		{name: "FatalLevel", got: FatalLevel, want: "fatal"},
		{name: "PanicLevel", got: PanicLevel, want: "panic"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.got)
		})
	}
}

func TestGetLogger(t *testing.T) {
	tests := []struct {
		name   string
		config Config
	}{
		{name: "info level", config: Config{Level: "info"}},
		{name: "debug level", config: Config{Level: "debug"}},
		{name: "with sampling", config: Config{Level: "info", Sampling: &SamplingConfig{Burst: 3, Period: 0}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			Init(tt.config)
			logger := GetLogger()
			assert.NotNil(t, logger)
		})
	}
}

func TestInit(t *testing.T) {
	tests := []struct {
		name   string
		config Config
	}{
		{name: "info level", config: Config{Level: "info"}},
		{name: "debug with file log", config: Config{Level: "debug", FileLog: true, AlarmEnabled: true}},
		{name: "info with json", config: Config{Level: "info", JSONOutput: true}},
		{
			name: "with rotation",
			config: Config{
				Level:    "info",
				FileLog:  true,
				Rotation: &RotationConfig{MaxSize: 1, MaxAge: 7, MaxBackups: 3, Compress: false},
			},
		},
		{
			name: "with sampling",
			config: Config{
				Level:    "info",
				Sampling: &SamplingConfig{Burst: 5, Period: 0},
			},
		},
		{
			name:   "with module levels",
			config: Config{Level: "info", ModuleLevel: map[string]string{"pipeline": "debug"}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.NotPanics(t, func() {
				Init(tt.config)
			})
		})
	}
}

func TestLogFunctions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		fn   func()
	}{
		{
			name: "Debug",
			fn: func() {
				Init(Config{Level: DebugLevel})
				SetLevel(DebugLevel)
				Debug("test debug message: %s", "arg")
			},
		},
		{
			name: "Info",
			fn: func() {
				Init(Config{Level: InfoLevel})
				SetLevel(InfoLevel)
				Info("test info message: %s", "arg")
			},
		},
		{
			name: "Warn",
			fn: func() {
				Init(Config{Level: WarnLevel})
				SetLevel(WarnLevel)
				Warn("test warn message: %s", "arg")
			},
		},
		{
			name: "Error",
			fn: func() {
				Init(Config{Level: ErrorLevel})
				SetLevel(ErrorLevel)
				Error(assert.AnError)
			},
		},
		{
			name: "Err",
			fn: func() {
				Init(Config{Level: ErrorLevel})
				SetLevel(ErrorLevel)
				Err(errors.New("test error without alarm"))
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.NotPanics(t, tt.fn)
		})
	}
}

func TestPanic(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		level string
	}{
		{name: "panic at info level", level: InfoLevel},
		{name: "panic at debug level", level: DebugLevel},
		{name: "panic at warn level", level: WarnLevel},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			Init(Config{Level: tt.level})
			assert.Panics(t, func() {
				Panic("test panic message: %s", "arg")
			})
		})
	}
}

func TestStructuredFields(t *testing.T) {
	t.Parallel()

	Init(Config{Level: DebugLevel})

	tests := []struct {
		name string
		fn   func()
	}{
		{
			name: "DebugFields",
			fn:   func() { DebugFields("debug with fields", map[string]any{"key": "value"}) },
		},
		{
			name: "InfoFields",
			fn:   func() { InfoFields("info with fields", map[string]any{"key": "value"}) },
		},
		{
			name: "WarnFields",
			fn:   func() { WarnFields("warn with fields", map[string]any{"key": "value"}) },
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.NotPanics(t, tt.fn)
		})
	}

	t.Run("ErrFields", func(t *testing.T) {
		Init(Config{Level: ErrorLevel})
		assert.NotPanics(t, func() {
			ErrFields(errors.New("test"), "error with fields", map[string]any{"key": "value"})
		})
	})
}

func TestEventHelpers(t *testing.T) {
	t.Parallel()

	Init(Config{Level: DebugLevel})

	tests := []struct {
		name string
		fn   func()
	}{
		{
			name: "DebugEvt",
			fn:   func() { DebugEvt().Str("key", "val").Msg("debug event") },
		},
		{
			name: "InfoEvt",
			fn:   func() { InfoEvt().Str("key", "val").Msg("info event") },
		},
		{
			name: "WarnEvt",
			fn:   func() { WarnEvt().Str("key", "val").Msg("warn event") },
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.NotPanics(t, tt.fn)
		})
	}

	t.Run("ErrorEvt", func(t *testing.T) {
		Init(Config{Level: ErrorLevel})
		assert.NotPanics(t, func() {
			ErrorEvt().Err(errors.New("test")).Str("key", "val").Msg("error event")
		})
	})
}

func TestModule(t *testing.T) {
	t.Parallel()

	Init(Config{Level: DebugLevel, ModuleLevel: map[string]string{"testmodule": "warn"}})

	tests := []struct {
		name       string
		moduleName string
		wantNil    bool
	}{
		{name: "configured module", moduleName: "testmodule", wantNil: false},
		{name: "nonexistent module", moduleName: "nonexistent", wantNil: false},
		{name: "empty module name falls back to parent", moduleName: "", wantNil: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := Module(tt.moduleName)
			if tt.wantNil {
				assert.Nil(t, m)
			} else {
				assert.NotNil(t, m)
			}
		})
	}
}

func TestSampled(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		config  Config
		wantNil bool
	}{
		{
			name:    "with burst sampling",
			config:  Config{Level: InfoLevel, Sampling: &SamplingConfig{Burst: 3, Period: 0}},
			wantNil: false,
		},
		{
			name:    "with higher burst sampling",
			config:  Config{Level: InfoLevel, Sampling: &SamplingConfig{Burst: 10, Period: 0}},
			wantNil: false,
		},
		{
			name:    "without sampling falls back to logger",
			config:  Config{Level: InfoLevel},
			wantNil: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			Init(tt.config)
			s := Sampled()
			if tt.wantNil {
				assert.Nil(t, s)
			} else {
				assert.NotNil(t, s)
			}
			assert.NotPanics(t, func() {
				s.Info().Msg("sampled info")
			})
		})
	}
}

func TestSetModuleLevel(t *testing.T) {
	tests := []struct {
		name       string
		moduleName string
		level      string
	}{
		{name: "set module to debug", moduleName: "m1", level: DebugLevel},
		{name: "set module to warn", moduleName: "m2", level: WarnLevel},
		{name: "set module to error", moduleName: "m3", level: ErrorLevel},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			Init(Config{Level: InfoLevel})
			SetModuleLevel(tt.moduleName, tt.level)
			m := Module(tt.moduleName)
			assert.NotNil(t, m)
		})
	}
}

func TestFatal(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		config Config
	}{
		{name: "fatal reference at info", config: Config{Level: InfoLevel}},
		{name: "fatal reference at debug", config: Config{Level: DebugLevel}},
		{name: "fatal reference at warn", config: Config{Level: WarnLevel}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			Init(tt.config)
			assert.NotPanics(t, func() {
				_ = Fatal
			})
		})
	}
}
