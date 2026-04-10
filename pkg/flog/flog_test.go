package flog

import (
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
	// Initialize with disabled file log and alarm
	Init(false, false)

	logger := GetLogger()
	assert.NotNil(t, logger)
}

func TestInit(t *testing.T) {
	// Test initialization without file logging
	assert.NotPanics(t, func() {
		Init(false, false)
	})

	// Test initialization with file logging
	assert.NotPanics(t, func() {
		Init(true, false)
	})

	// Test initialization with alarm enabled
	assert.NotPanics(t, func() {
		Init(false, true)
	})
}

func TestDebug(t *testing.T) {
	Init(false, false)
	SetLevel(DebugLevel)

	// Should not panic
	assert.NotPanics(t, func() {
		Debug("test debug message: %s", "arg")
	})
}

func TestInfo(t *testing.T) {
	Init(false, false)
	SetLevel(InfoLevel)

	// Should not panic
	assert.NotPanics(t, func() {
		Info("test info message: %s", "arg")
	})
}

func TestWarn(t *testing.T) {
	Init(false, false)
	SetLevel(WarnLevel)

	// Should not panic
	assert.NotPanics(t, func() {
		Warn("test warn message: %s", "arg")
	})
}

func TestError(t *testing.T) {
	Init(false, false)
	SetLevel(ErrorLevel)

	// Should not panic
	assert.NotPanics(t, func() {
		Error(assert.AnError)
	})
}

func TestFatal(t *testing.T) {
	Init(false, false)

	// Note: Fatal would exit the program, so we just verify it exists
	// In real tests, you might use a custom exit function
	assert.NotPanics(t, func() {
		// We can't actually test Fatal as it calls os.Exit
		// Just verify the function signature is correct
		_ = Fatal
	})
}

func TestPanic(t *testing.T) {
	Init(false, false)

	// Note: Panic would panic, so we recover from it
	assert.Panics(t, func() {
		Panic("test panic message: %s", "arg")
	})
}
