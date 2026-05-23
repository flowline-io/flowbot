package provider_event_source_test

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/flowline-io/flowbot/pkg/ability"
)

type stubPollingResource struct {
	name     string
	interval time.Duration
}

func (r *stubPollingResource) ResourceName() string              { return r.name }
func (r *stubPollingResource) DefaultInterval() time.Duration    { return r.interval }
func (r *stubPollingResource) DiffKey(item any) string           { return item.(string) }
func (r *stubPollingResource) ContentHash(item any) string       { return "h_" + item.(string) }
func (r *stubPollingResource) CursorField() string               { return "id" }
func (r *stubPollingResource) List(_ context.Context, _ string) (ability.PollResult, error) {
	return ability.PollResult{}, nil
}

type stubPersistence struct {
	data map[string]ability.PollingEntry
}

func (s *stubPersistence) LoadAll(_ context.Context) (map[string]ability.PollingEntry, error) {
	return s.data, nil
}

func (s *stubPersistence) Save(_ context.Context, resourceName, cursor string, knownHashes map[string]string) error {
	if s.data == nil {
		s.data = make(map[string]ability.PollingEntry)
	}
	s.data[resourceName] = ability.PollingEntry{
		Cursor:      cursor,
		KnownHashes: knownHashes,
	}
	return nil
}

var _ = Describe("Cron Polling Lifecycle", func() {
	Context("RegisterPolling and Start/Stop", func() {
		It("registers and starts without error", func() {
			mgr := ability.NewEventSourceManager(nil, nil, nil)
			mgr.RegisterPolling(&stubPollingResource{
				name:     "bdd/bookmarks",
				interval: time.Hour,
			}, time.Hour)

			Expect(mgr.Start(context.Background())).To(Succeed())
			Expect(mgr.Stop(context.Background())).To(Succeed())
		})

		It("starts with empty pollers without error", func() {
			mgr := ability.NewEventSourceManager(nil, nil, nil)
			Expect(mgr.Start(context.Background())).To(Succeed())
			Expect(mgr.Stop(context.Background())).To(Succeed())
		})

		It("registers multiple polling resources and starts", func() {
			mgr := ability.NewEventSourceManager(nil, nil, nil)
			mgr.RegisterPolling(&stubPollingResource{
				name:     "bdd/resource1",
				interval: time.Minute,
			}, time.Minute)
			mgr.RegisterPolling(&stubPollingResource{
				name:     "bdd/resource2",
				interval: 5 * time.Minute,
			}, 5*time.Minute)
			mgr.RegisterPolling(&stubPollingResource{
				name:     "bdd/resource3",
				interval: time.Hour,
			}, time.Hour)

			Expect(mgr.Start(context.Background())).To(Succeed())
			Expect(mgr.Stop(context.Background())).To(Succeed())
		})
	})

	Context("PollingState persistence", func() {
		It("persists cursor and recovers after state load", func() {
			persist := &stubPersistence{}
			state := ability.NewPollingState(persist)
			state.Update("bdd/recovery", ability.PollingEntry{
				Cursor:      "cursor-42",
				KnownHashes: map[string]string{"k1": "h1"},
			})
			state.MarkDirty("bdd/recovery")
			Expect(state.Flush(context.Background())).To(Succeed())

			entry := state.Get("bdd/recovery")
			Expect(entry.Cursor).To(Equal("cursor-42"))
			Expect(entry.KnownHashes).To(HaveKeyWithValue("k1", "h1"))
		})

		It("returns empty entry for unknown resource", func() {
			state := ability.NewPollingState(&stubPersistence{})
			entry := state.Get("bdd/unknown")
			Expect(entry.Cursor).To(BeEmpty())
			Expect(entry.KnownHashes).To(BeEmpty())
		})

		It("updates existing entry with new cursor and hashes", func() {
			state := ability.NewPollingState(&stubPersistence{})
			state.Update("bdd/rsrc", ability.PollingEntry{
				Cursor:      "cursor-v1",
				KnownHashes: map[string]string{"a": "hash-a"},
			})
			state.Update("bdd/rsrc", ability.PollingEntry{
				Cursor:      "cursor-v2",
				KnownHashes: map[string]string{"b": "hash-b"},
			})

			entry := state.Get("bdd/rsrc")
			Expect(entry.Cursor).To(Equal("cursor-v2"))
			Expect(entry.KnownHashes).To(HaveKeyWithValue("b", "hash-b"))
		})
	})

	Context("PollingState load from persistence", func() {
		It("loads persisted entries into memory", func() {
			persist := &stubPersistence{
				data: map[string]ability.PollingEntry{
					"a/rsrc": {Cursor: "cur-a", KnownHashes: map[string]string{"ka": "ha"}},
					"b/rsrc": {Cursor: "cur-b", KnownHashes: map[string]string{"kb": "hb"}},
				},
			}
			state := ability.NewPollingState(persist)
			Expect(state.Load(context.Background())).To(Succeed())

			a := state.Get("a/rsrc")
			Expect(a.Cursor).To(Equal("cur-a"))
			Expect(a.KnownHashes).To(HaveKeyWithValue("ka", "ha"))

			b := state.Get("b/rsrc")
			Expect(b.Cursor).To(Equal("cur-b"))
			Expect(b.KnownHashes).To(HaveKeyWithValue("kb", "hb"))
		})

		It("load overwrites pre-existing in-memory entries", func() {
			persist := &stubPersistence{
				data: map[string]ability.PollingEntry{
					"x/rsrc": {Cursor: "persisted-cursor", KnownHashes: map[string]string{"kx": "hx"}},
				},
			}
			state := ability.NewPollingState(persist)
			state.Update("x/rsrc", ability.PollingEntry{Cursor: "stale-cursor"})
			Expect(state.Load(context.Background())).To(Succeed())

			entry := state.Get("x/rsrc")
			Expect(entry.Cursor).To(Equal("persisted-cursor"))
		})
	})
})
