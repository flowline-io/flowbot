package ability

import (
	"context"
	"testing"
)

type mockPersistence struct {
	data        map[string]PollingEntry
	saveCalls   []saveCall
	loadAllErr  error
}

type saveCall struct {
	resourceName string
	cursor       string
	knownHashes  map[string]string
}

func (m *mockPersistence) LoadAll(_ context.Context) (map[string]PollingEntry, error) {
	if m.loadAllErr != nil {
		return nil, m.loadAllErr
	}
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
	m.saveCalls = append(m.saveCalls, saveCall{
		resourceName: resourceName,
		cursor:       cursor,
		knownHashes:  knownHashes,
	})
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
	tests := []struct {
		name    string
		setup   func(*PollingState, *mockPersistence)
		want    map[string]PollingEntry
		wantLen int
	}{
		{
			name: "flush single resource",
			setup: func(state *PollingState, _ *mockPersistence) {
				state.Update("test/rsrc", PollingEntry{
					Cursor:      "cursor-1",
					KnownHashes: map[string]string{"k1": "h1"},
				})
			},
			want: map[string]PollingEntry{
				"test/rsrc": {Cursor: "cursor-1", KnownHashes: map[string]string{"k1": "h1"}},
			},
			wantLen: 1,
		},
		{
			name: "flush multiple resources",
			setup: func(state *PollingState, _ *mockPersistence) {
				state.Update("a/rsrc", PollingEntry{
					Cursor:      "cur-a",
					KnownHashes: map[string]string{"ka": "ha"},
				})
				state.Update("b/rsrc", PollingEntry{
					Cursor:      "cur-b",
					KnownHashes: map[string]string{"kb": "hb"},
				})
			},
			want: map[string]PollingEntry{
				"a/rsrc": {Cursor: "cur-a", KnownHashes: map[string]string{"ka": "ha"}},
				"b/rsrc": {Cursor: "cur-b", KnownHashes: map[string]string{"kb": "hb"}},
			},
			wantLen: 2,
		},
		{
			name: "flush empty dirty set is no-op",
			setup: func(_ *PollingState, _ *mockPersistence) {
			},
			want:    map[string]PollingEntry{},
			wantLen: 0,
		},
		{
			name: "flush with updated entry",
			setup: func(state *PollingState, _ *mockPersistence) {
				state.Update("test/rsrc", PollingEntry{
					Cursor:      "cursor-old",
					KnownHashes: map[string]string{"k1": "h1"},
				})
				state.Update("test/rsrc", PollingEntry{
					Cursor:      "cursor-new",
					KnownHashes: map[string]string{"k2": "h2"},
				})
			},
			want: map[string]PollingEntry{
				"test/rsrc": {Cursor: "cursor-new", KnownHashes: map[string]string{"k2": "h2"}},
			},
			wantLen: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			persist := &mockPersistence{}
			state := NewPollingState(persist)
			tt.setup(state, persist)

			err := state.Flush(context.Background())
			if err != nil {
				t.Fatalf("Flush: %v", err)
			}

			if len(persist.data) != tt.wantLen {
				t.Errorf("persisted data len = %d, want %d", len(persist.data), tt.wantLen)
			}
			for key, wantEntry := range tt.want {
				got, ok := persist.data[key]
				if !ok {
					t.Errorf("missing key %q in persisted data", key)
					continue
				}
				if got.Cursor != wantEntry.Cursor {
					t.Errorf("cursor for %q = %q, want %q", key, got.Cursor, wantEntry.Cursor)
				}
			}
		})
	}
}

func TestPollingState_Load(t *testing.T) {
	tests := []struct {
		name      string
		seed      map[string]PollingEntry
		preEntry  string
		preCursor string
	}{
		{
			name:      "load from empty backend",
			seed:      nil,
			preEntry:  "pre/loaded",
			preCursor: "old-cursor",
		},
		{
			name: "load from backend with data",
			seed: map[string]PollingEntry{
				"a/rsrc": {Cursor: "cur-a", KnownHashes: map[string]string{"ka": "ha"}},
			},
			preEntry:  "b/rsrc",
			preCursor: "old-cursor",
		},
		{
			name: "load overwrites existing entries",
			seed: map[string]PollingEntry{
				"x/rsrc": {Cursor: "persisted-cursor", KnownHashes: map[string]string{"kx": "hx"}},
			},
			preEntry:  "x/rsrc",
			preCursor: "stale-cursor",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			persist := &mockPersistence{data: tt.seed}
			state := NewPollingState(persist)

			state.Update(tt.preEntry, PollingEntry{Cursor: tt.preCursor})

			err := state.Load(context.Background())
			if err != nil {
				t.Fatalf("Load: %v", err)
			}

			preEntry := state.Get(tt.preEntry)
			if tt.preEntry == "x/rsrc" && tt.seed["x/rsrc"].Cursor == "persisted-cursor" {
				if preEntry.Cursor != "persisted-cursor" {
					t.Errorf("entry %q cursor after load = %q, want %q (should be overwritten by persisted)", tt.preEntry, preEntry.Cursor, "persisted-cursor")
				}
			} else {
				if preEntry.Cursor != tt.preCursor {
					t.Errorf("entry %q cursor after load = %q, want %q", tt.preEntry, preEntry.Cursor, tt.preCursor)
				}
			}

			for key, wantEntry := range tt.seed {
				got := state.Get(key)
				if got.Cursor != wantEntry.Cursor {
					t.Errorf("loaded entry %q cursor = %q, want %q", key, got.Cursor, wantEntry.Cursor)
				}
			}
		})
	}
}

func TestPollingState_MarkDirty(t *testing.T) {
	tests := []struct {
		name  string
		marks []string
		want  []string
	}{
		{
			name:  "mark single resource dirty",
			marks: []string{"a/rsrc"},
			want:  []string{"a/rsrc"},
		},
		{
			name:  "mark multiple resources dirty",
			marks: []string{"a/rsrc", "b/rsrc", "c/rsrc"},
			want:  []string{"a/rsrc", "b/rsrc", "c/rsrc"},
		},
		{
			name:  "mark already dirty resource",
			marks: []string{"a/rsrc", "a/rsrc", "a/rsrc"},
			want:  []string{"a/rsrc"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			persist := &mockPersistence{}
			state := NewPollingState(persist)

			for _, name := range tt.marks {
				state.Update(name, PollingEntry{Cursor: "c"})
				state.MarkDirty(name)
			}

			err := state.Flush(context.Background())
			if err != nil {
				t.Fatalf("Flush: %v", err)
			}

			if len(persist.data) != len(tt.want) {
				t.Errorf("persisted data len = %d, want %d", len(persist.data), len(tt.want))
			}
			for _, name := range tt.want {
				if _, ok := persist.data[name]; !ok {
					t.Errorf("missing key %q in persisted data", name)
				}
			}
		})
	}
}
