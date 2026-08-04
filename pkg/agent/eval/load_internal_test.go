package eval

import "testing"

func TestCloneSandbox(t *testing.T) {
	t.Parallel()

	ws := &WorkspaceSandbox{root: "x"}
	wsClone, ok := cloneSandbox(ws).(*WorkspaceSandbox)
	if !ok {
		t.Fatalf("expected WorkspaceSandbox clone")
	}
	if wsClone == ws {
		t.Fatalf("expected distinct WorkspaceSandbox instance")
	}

	dk := NewDockerSandbox(DockerSandboxConfig{Image: "img"})
	dk.root = "x"
	dkClone, ok := cloneSandbox(dk).(*DockerSandbox)
	if !ok {
		t.Fatalf("expected DockerSandbox clone")
	}
	if dkClone == dk {
		t.Fatalf("expected distinct DockerSandbox instance")
	}
	if dkClone.root != "" {
		t.Fatalf("expected cloned DockerSandbox root reset")
	}
}

func TestNewDockerSandbox_DefaultImage(t *testing.T) {
	t.Parallel()
	sb := NewDockerSandbox(DockerSandboxConfig{})
	if sb.cfg.Image == "" {
		t.Fatalf("expected default docker image")
	}
}
