package bulkhead

import (
	"runtime"
	"testing"
)

func TestGetReturnsSingleton(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "same name returns same instance"},
		{name: "repeated Get returns identical pointer"},
		{name: "singleton across calls"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := Get("foo-" + tt.name)
			b := Get("foo-" + tt.name)
			if a != b {
				t.Error("Get should return the same instance for the same name")
			}
		})
	}
}

func TestGetDifferentNamesReturnDifferentInstances(t *testing.T) {
	tests := []struct {
		name string
		a, b string
	}{
		{name: "bar vs baz", a: "bar", b: "baz"},
		{name: "llm vs bookmark", a: "llm", b: "bookmark"},
		{name: "kanban vs homelab", a: "kanban", b: "homelab"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := Get(tt.a)
			b := Get(tt.b)
			if a == b {
				t.Error("Get should return different instances for different names")
			}
		})
	}
}

func TestGetDefaultConfig(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "sem cap equals GOMAXPROCS*4"},
		{name: "queue cap equals GOMAXPROCS*4"},
		{name: "timeout is 30s"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := Get("default-test-" + tt.name)
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
		})
	}
}

func TestSetDefaultsAppliesToNewInstances(t *testing.T) {
	save := defaultManager.defaults
	saveOpts := defaultManager.extraOpts
	t.Cleanup(func() {
		defaultManager.defaults = save
		defaultManager.extraOpts = saveOpts
		Reset()
	})

	tests := []struct {
		name       string
		concurrent int
		queue      int
	}{
		{name: "custom sizes 5/3", concurrent: 5, queue: 3},
		{name: "custom sizes 10/5", concurrent: 10, queue: 5},
		{name: "custom sizes 2/1", concurrent: 2, queue: 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			Reset()
			SetDefaults(
				WithMaxConcurrent(tt.concurrent),
				WithMaxQueue(tt.queue),
			)
			b := Get("set-defaults-" + tt.name)
			if cap(b.sem) != tt.concurrent {
				t.Errorf("sem cap: want %d, got %d", tt.concurrent, cap(b.sem))
			}
			if cap(b.queue) != tt.queue {
				t.Errorf("queue cap: want %d, got %d", tt.queue, cap(b.queue))
			}
		})
	}
}
