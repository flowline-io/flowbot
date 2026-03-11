package utils

import (
	"testing"
	"time"
)

func TestDebouncer_New(t *testing.T) {
	d := NewDebouncer(100 * time.Millisecond)
	if d == nil {
		t.Fatal("NewDebouncer returned nil")
	}
	if d.delay != 100*time.Millisecond {
		t.Errorf("delay = %v, want 100ms", d.delay)
	}
	if d.pending {
		t.Error("new debouncer should not be pending")
	}
}

func TestDebouncer_Pending(t *testing.T) {
	d := NewDebouncer(50 * time.Millisecond)
	if d.Pending() {
		t.Error("new debouncer should not be pending")
	}
}
