package eval

import "testing"

func TestCloneSandbox(t *testing.T) {
	t.Parallel()

	ws := &WorkspaceSandbox{root: "x"}
	wsClone, ok := cloneSandbox(ws).(*WorkspaceSandbox)
	if !ok {
		t.Fatal("expected WorkspaceSandbox clone")
	}
	if wsClone == ws {
		t.Fatal("expected distinct WorkspaceSandbox instance")
	}

	dk := NewDockerSandbox(DockerSandboxConfig{Image: "img"})
	dk.root = "x"
	dkClone, ok := cloneSandbox(dk).(*DockerSandbox)
	if !ok {
		t.Fatal("expected DockerSandbox clone")
	}
	if dkClone == dk {
		t.Fatal("expected distinct DockerSandbox instance")
	}
	if dkClone.root != "" {
		t.Fatal("expected cloned DockerSandbox root reset")
	}
}

func TestNewDockerSandbox_DefaultImage(t *testing.T) {
	t.Parallel()
	sb := NewDockerSandbox(DockerSandboxConfig{})
	if sb.cfg.Image == "" {
		t.Fatal("expected default docker image")
	}
}
