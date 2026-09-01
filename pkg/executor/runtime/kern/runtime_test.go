package kern

import (
	"context"
	"testing"
	"time"

	"github.com/flowline-io/flowbot/pkg/types"
)

func TestBoxName(t *testing.T) {
	t.Parallel()
	tests := []struct {
		id   string
		want string
	}{
		{id: "abc-123", want: "fb-abc-123"},
		{id: "task/with\\slashes", want: "fb-task-with-slashes"},
		{id: "", want: "fb-task"},
	}
	for _, tt := range tests {
		t.Run(tt.id, func(t *testing.T) {
			t.Parallel()
			if got := BoxName(tt.id); got != tt.want {
				t.Fatalf("BoxName(%q) = %q, want %q", tt.id, got, tt.want)
			}
		})
	}
}

func TestBoxOptionsBuildArgs(t *testing.T) {
	t.Parallel()
	opts := BoxOptions{
		Name:            "fb-test",
		Image:           "alpine:3.20",
		SecurityProfile: "untrusted",
		RequireLimits:   true,
		Memory:          "256m",
		CPUs:            "0.5",
		Network:         "host",
		Binds:           []string{"/tmp/w:/flowbot"},
		Env:             []string{"FOO=bar"},
		Workdir:         "/flowbot",
		Command:         []string{"echo", "hi"},
	}
	args := opts.buildArgs("kern")
	want := []string{
		"kern", "box", "job", "--name", "fb-test", "--image", "alpine:3.20",
		"--security-profile", "untrusted", "--require-limits",
		"--memory", "256m", "--cpus", "0.5", "--net", "host",
		"-v", "/tmp/w:/flowbot", "-e", "FOO=bar", "--workdir", "/flowbot",
		"--", "echo", "hi",
	}
	if len(args) != len(want) {
		t.Fatalf("len(args)=%d want %d\nargs=%v", len(args), len(want), args)
	}
	for i := range want {
		if args[i] != want[i] {
			t.Fatalf("args[%d]=%q want %q\nfull=%v", i, args[i], want[i], args)
		}
	}
}

func TestValidateTaskRejectsUnsupported(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		task *types.Task
	}{
		{name: "registry", task: &types.Task{Image: "alpine", Registry: &types.Registry{Username: "u"}}},
		{name: "gpus", task: &types.Task{Image: "alpine", GPUs: "all"}},
		{name: "networks", task: &types.Task{Image: "alpine", Networks: []string{"net"}}},
		{name: "volume", task: &types.Task{Image: "alpine", Mounts: []types.Mount{{Type: types.MountTypeVolume}}}},
		{name: "bind disallowed", task: &types.Task{Image: "alpine", Mounts: []types.Mount{{Type: types.MountTypeBind, Source: "/a", Target: "/b"}}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if err := validateTask(tt.task, false); err == nil {
				t.Fatal("expected error")
			}
		})
	}
}

func TestValidateBindMount(t *testing.T) {
	t.Parallel()
	err := ValidateBindMount(true, types.Mount{Type: types.MountTypeBind, Source: "/src", Target: "/dst"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := ValidateBindMount(false, types.Mount{Type: types.MountTypeBind, Source: "/src", Target: "/dst"}); err == nil {
		t.Fatal("expected bind disallowed error")
	}
}

func TestSetupWorkdirRejectsPathEscape(t *testing.T) {
	t.Parallel()
	_, err := setupWorkdir(&types.Task{
		ID:    "task-1",
		Image: "alpine",
		Files: map[string]string{"../escape": "x"},
	})
	if err == nil {
		t.Fatal("expected path escape error")
	}
}

func skipIfNoKern(t *testing.T) {
	t.Helper()
	ctx, cancel := contextWithTimeout(t, 5*time.Second)
	defer cancel()
	client := NewClient(Config{})
	if err := LookPath(client.Binary()); err != nil {
		t.Skipf("skipping: kern binary not found: %v", err)
	}
	if err := client.Doctor(ctx); err != nil {
		t.Skipf("skipping: kern doctor failed: %v", err)
	}
}

func contextWithTimeout(t *testing.T, d time.Duration) (context.Context, func()) {
	t.Helper()
	return context.WithTimeout(context.Background(), d)
}
