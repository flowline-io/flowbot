package kern

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/utils"
)

func TestIntegrationEcho(t *testing.T) {
	skipIfNoKern(t)
	ctx, cancel := contextWithTimeout(t, 2*time.Minute)
	defer cancel()

	rt, err := NewRuntime(WithConfig(Config{SecurityProfile: "untrusted"}))
	if err != nil {
		t.Fatalf("NewRuntime: %v", err)
	}

	task := &types.Task{
		ID:    utils.NewUUID(),
		Image: "alpine:3.20",
		Run:   "echo -n hello-kern > $OUTPUT",
	}
	if err := rt.Run(ctx, task); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if strings.TrimSpace(task.Result) != "hello-kern" {
		t.Fatalf("result=%q want hello-kern", task.Result)
	}
}

func TestIntegrationUntrustedWritesStdout(t *testing.T) {
	skipIfNoKern(t)
	ctx, cancel := contextWithTimeout(t, 2*time.Minute)
	defer cancel()

	rt, err := NewRuntime(WithConfig(Config{SecurityProfile: "untrusted"}))
	if err != nil {
		t.Fatalf("NewRuntime: %v", err)
	}

	task := &types.Task{
		ID:    utils.NewUUID(),
		Image: "alpine:3.20",
		Run:   "echo wrote > /flowbot/stdout",
	}
	if err := rt.Run(ctx, task); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if strings.TrimSpace(task.Result) != "wrote" {
		t.Fatalf("result=%q want wrote", task.Result)
	}
}

func TestIntegrationBindAllowed(t *testing.T) {
	skipIfNoKern(t)
	ctx, cancel := contextWithTimeout(t, 2*time.Minute)
	defer cancel()

	hostDir := t.TempDir()
	marker := filepath.Join(hostDir, "marker")
	if err := os.WriteFile(marker, []byte("ok"), 0o644); err != nil {
		t.Fatalf("write marker: %v", err)
	}

	rt, err := NewRuntime(WithConfig(Config{
		SecurityProfile: "untrusted",
		BindAllowed:     true,
	}))
	if err != nil {
		t.Fatalf("NewRuntime: %v", err)
	}

	task := &types.Task{
		ID:    utils.NewUUID(),
		Image: "alpine:3.20",
		Run:   "cat /data/marker > /flowbot/stdout",
		Mounts: []types.Mount{{
			Type:   types.MountTypeBind,
			Source: hostDir,
			Target: "/data",
		}},
	}
	if err := rt.Run(ctx, task); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if strings.TrimSpace(task.Result) != "ok" {
		t.Fatalf("result=%q want ok", task.Result)
	}
}
