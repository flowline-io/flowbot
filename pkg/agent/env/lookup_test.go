package env

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLookupExecutableSkipsWindowsAppsAndEmptyFiles(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	aliasDir := filepath.Join(root, "WindowsApps")
	emptyDir := filepath.Join(root, "empty")
	realDir := filepath.Join(root, "bin")
	for _, dir := range []string{aliasDir, emptyDir, realDir} {
		if err := os.MkdirAll(dir, 0o750); err != nil {
			t.Fatal(err)
		}
	}
	alias := filepath.Join(aliasDir, "python.exe")
	empty := filepath.Join(emptyDir, "python.exe")
	want := filepath.Join(realDir, "python.exe")
	if err := os.WriteFile(alias, []byte("alias"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(empty, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(want, []byte("interpreter"), 0o644); err != nil {
		t.Fatal(err)
	}

	pathEnv := aliasDir + string(os.PathListSeparator) + emptyDir + string(os.PathListSeparator) + realDir
	got := lookupExecutable("python", pathEnv, ".exe")
	if got != want {
		t.Fatalf("lookupExecutable() = %q, want %q", got, want)
	}
}

func TestLookupExecutableUsesFirstRealFile(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	first := filepath.Join(root, "first")
	second := filepath.Join(root, "second")
	for _, dir := range []string{first, second} {
		if err := os.MkdirAll(dir, 0o750); err != nil {
			t.Fatal(err)
		}
	}
	want := filepath.Join(first, "python.exe")
	if err := os.WriteFile(want, []byte("first"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(second, "python.exe"), []byte("second"), 0o644); err != nil {
		t.Fatal(err)
	}
	pathEnv := first + string(os.PathListSeparator) + second
	got := lookupExecutable("python", pathEnv, ".exe")
	if got != want {
		t.Fatalf("lookupExecutable() = %q, want %q", got, want)
	}
}

func TestLookupExecutableMisses(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	aliasDir := filepath.Join(root, "WindowsApps")
	if err := os.MkdirAll(aliasDir, 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(aliasDir, "python.exe"), []byte("alias"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := lookupExecutable("python", aliasDir, ".exe"); got != "" {
		t.Fatalf("lookupExecutable() = %q, want empty", got)
	}
}

func TestResolveHostExecutableLeavesExplicitPath(t *testing.T) {
	t.Parallel()
	name := filepath.Join("bin", "python")
	if got := resolveHostExecutable(name); got != name {
		t.Fatalf("resolveHostExecutable() = %q, want %q", got, name)
	}
}
