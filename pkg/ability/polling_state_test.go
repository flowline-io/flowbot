package ability

import (
	"context"
	"testing"
)

type mockPersistence struct {
	data map[string]PollingEntry
}

func (m *mockPersistence) LoadAll(_ context.Context) (map[string]PollingEntry, error) {
	return m.data, nil
}

func (m *mockPersistence) Save(_ context.Context, resourceName, cursor string, knownHashes map[string]string) error {
	if m.data == nil {
		m.data = make(map[string]PollingEntry)
	}
	m.data[resourceName] = PollingEntry{
		Cursor:      cursor,
		KnownHashes: knownHashes,
	}
	return nil
}

func TestPollingState_Get_Empty(t *testing.T) {
	state := NewPollingState(&mockPersistence{})
	tests := []struct {
		name     string
		resource string
	}{
		{name: "unknown resource returns empty", resource: "test/unknown"},
		{name: "another unknown resource", resource: "other/rsrc"},
		{name: "slash resource", resource: "a/b/c"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entry := state.Get(tt.resource)
			if entry.Cursor != "" {
				t.Errorf("cursor = %q, want empty", entry.Cursor)
			}
			if len(entry.KnownHashes) != 0 {
				t.Errorf("knownHashes len = %d, want 0", len(entry.KnownHashes))
			}
		})
	}
}

func TestPollingState_UpdateAndGet(t *testing.T) {
	state := NewPollingState(&mockPersistence{})
	state.Update("test/rsrc", PollingEntry{
		Cursor:      "cursor-1",
		KnownHashes: map[string]string{"k1": "h1"},
	})

	tests := []struct {
		name     string
		resource string
		wantCur  string
		wantLen  int
	}{
		{
			name:     "get after update",
			resource: "test/rsrc",
			wantCur:  "cursor-1",
			wantLen:  1,
		},
		{
			name:     "still unknown returns empty",
			resource: "test/other",
			wantCur:  "",
			wantLen:  0,
		},
		{
			name:     "get same resource again",
			resource: "test/rsrc",
			wantCur:  "cursor-1",
			wantLen:  1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entry := state.Get(tt.resource)
			if entry.Cursor != tt.wantCur {
				t.Errorf("cursor = %q, want %q", entry.Cursor, tt.wantCur)
			}
			if len(entry.KnownHashes) != tt.wantLen {
				t.Errorf("knownHashes len = %d, want %d", len(entry.KnownHashes), tt.wantLen)
			}
		})
	}
}

func TestPollingState_Flush(t *testing.T) {
	persist := &mockPersistence{}
	state := NewPollingState(persist)

	state.Update("test/rsrc", PollingEntry{
		Cursor:      "cursor-flush",
		KnownHashes: map[string]string{"k1": "h1", "k2": "h2"},
	})
	state.MarkDirty("test/rsrc")

	err := state.Flush(context.Background())
	if err != nil {
		t.Fatalf("Flush: %v", err)
	}

	if _, ok := persist.data["test/rsrc"]; !ok {
		t.Fatal("expected data to be persisted after Flush")
	}
	if persist.data["test/rsrc"].Cursor != "cursor-flush" {
		t.Errorf("persisted cursor = %q, want %q", persist.data["test/rsrc"].Cursor, "cursor-flush")
	}
}
