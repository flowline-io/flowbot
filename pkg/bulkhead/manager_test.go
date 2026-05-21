package bulkhead

import (
	"runtime"
	"testing"
)

func TestGetReturnsSingleton(t *testing.T) {
	a := Get("foo")
	b := Get("foo")
	if a != b {
		t.Error("Get should return the same instance for the same name")
	}
}

func TestGetDifferentNamesReturnDifferentInstances(t *testing.T) {
	a := Get("bar")
	b := Get("baz")
	if a == b {
		t.Error("Get should return different instances for different names")
	}
}

func TestGetDefaultConfig(t *testing.T) {
	b := Get("default-test")
	n := runtime.GOMAXPROCS(0)
	expected := n * 4
	if cap(b.sem) != expected {
		t.Errorf("default sem cap: want %d, got %d", expected, cap(b.sem))
	}
	if cap(b.queue) != expected {
		t.Errorf("default queue cap: want %d, got %d", expected, cap(b.queue))
	}
	if b.config.timeout.String() != "30s" {
		t.Errorf("default timeout: want 30s, got %s", b.config.timeout)
	}
}

func TestSetDefaultsAppliesToNewInstances(t *testing.T) {
	SetDefaults(
		WithMaxConcurrent(5),
		WithMaxQueue(3),
	)
	b := Get("set-defaults-test")
	if cap(b.sem) != 5 {
		t.Errorf("sem cap: want 5, got %d", cap(b.sem))
	}
	if cap(b.queue) != 3 {
		t.Errorf("queue cap: want 3, got %d", cap(b.queue))
	}
}
