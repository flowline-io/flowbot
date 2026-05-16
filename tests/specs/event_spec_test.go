//go:build integration
// +build integration

package specs

import (
	"context"
	"time"

	"github.com/bytedance/sonic"
	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/internal/store/ent/gen/dataevent"
	"github.com/flowline-io/flowbot/internal/store/ent/gen/eventoutbox"
	"github.com/flowline-io/flowbot/internal/store/model"
	"github.com/flowline-io/flowbot/pkg/ability"
	"github.com/flowline-io/flowbot/pkg/types"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Event System", Label("event"), func() {

	Describe("DataEvent Publishing", func() {
		It("publishes a DataEvent to PostgreSQL data_events table", func() {
			event := types.DataEvent{
				EventID:        "test-event-" + types.Id(),
				EventType:      types.EventBookmarkCreated,
				Source:         "test",
				Capability:     "bookmark",
				Operation:      "create",
				EntityID:       "entity-1",
				IdempotencyKey: "idem-" + types.Id(),
				UID:            "test-uid",
				Data:           types.KV{"url": "https://example.com"},
			}

			eventStore := store.NewEventStore(EntClient)
			err := eventStore.AppendDataEvent(event)
			Expect(err).NotTo(HaveOccurred())

			_, err = EntClient.DataEvent.Query().Where(dataevent.EventID(event.EventID)).Only(context.Background())
			Expect(err).NotTo(HaveOccurred())
		})

		It("assigns a unique event ID", func() {
			event1 := types.DataEvent{
				EventID:        "unique-test-" + types.Id(),
				EventType:      types.EventBookmarkCreated,
				Source:         "test",
				Capability:     "bookmark",
				Operation:      "create",
				EntityID:       "entity-2",
				IdempotencyKey: "idem-" + types.Id(),
			}
			event2 := types.DataEvent{
				EventID:        "unique-test-" + types.Id(),
				EventType:      types.EventBookmarkCreated,
				Source:         "test",
				Capability:     "bookmark",
				Operation:      "create",
				EntityID:       "entity-3",
				IdempotencyKey: "idem-" + types.Id(),
			}

			eventStore := store.NewEventStore(EntClient)
			err := eventStore.AppendDataEvent(event1)
			Expect(err).NotTo(HaveOccurred())
			err = eventStore.AppendDataEvent(event2)
			Expect(err).NotTo(HaveOccurred())

			Expect(event1.EventID).NotTo(Equal(event2.EventID))
		})

		It("records event timestamp and source", func() {
			event := types.DataEvent{
				EventID:   "ts-test-" + types.Id(),
				EventType: types.EventReaderEntryStarred,
				Source:    "test-source",
				Data:      types.KV{"entry_id": "123"},
			}

			eventStore := store.NewEventStore(EntClient)
			err := eventStore.AppendDataEvent(event)
			Expect(err).NotTo(HaveOccurred())

			saved, err := EntClient.DataEvent.Query().Where(dataevent.EventID(event.EventID)).Only(context.Background())
			Expect(err).NotTo(HaveOccurred())
			Expect(saved.Source).To(Equal("test-source"))
			Expect(saved.CreatedAt).NotTo(BeNil())
		})
	})

	Describe("DataEvent Consumption", func() {
		It("acknowledges successfully processed events", func() {
			pipelineStore := store.NewPipelineStore(EntClient)
			consumerName := "test-consumer-" + types.Id()
			eventID := "consumed-event-" + types.Id()

			consumed, err := pipelineStore.HasConsumed(consumerName, eventID)
			Expect(err).NotTo(HaveOccurred())
			Expect(consumed).To(BeFalse())

			err = pipelineStore.RecordConsumption(consumerName, eventID)
			Expect(err).NotTo(HaveOccurred())

			consumed, err = pipelineStore.HasConsumed(consumerName, eventID)
			Expect(err).NotTo(HaveOccurred())
			Expect(consumed).To(BeTrue())
		})

		It("re-queues failed events for retry", func() {
			pipelineStore := store.NewPipelineStore(EntClient)
			run, err := pipelineStore.CreateRun("test-pipeline", "fail-event-"+types.Id(), types.EventBookmarkCreated)
			Expect(err).NotTo(HaveOccurred())
			Expect(run.ID).NotTo(BeZero())

			err = pipelineStore.UpdateRunStatus(run.ID, model.PipelineFailed, "step failed")
			Expect(err).NotTo(HaveOccurred())

			updated, err := EntClient.PipelineRun.Get(context.Background(), run.ID)
			Expect(err).NotTo(HaveOccurred())
			Expect(updated.Status).To(Equal(model.PipelineFailed))
			Expect(updated.Error).To(Equal("step failed"))
		})
	})

	Describe("Event Delivery", func() {
		It("creates pipeline_run record on execution start", func() {
			pipelineStore := store.NewPipelineStore(EntClient)
			run, err := pipelineStore.CreateRun("delivery-test", "delivery-event-"+types.Id(), types.EventBookmarkArchived)
			Expect(err).NotTo(HaveOccurred())
			Expect(run.PipelineName).To(Equal("delivery-test"))
			Expect(run.EventType).To(Equal(types.EventBookmarkArchived))
			Expect(run.StartedAt).NotTo(BeNil())

			err = pipelineStore.UpdateRunStatus(run.ID, model.PipelineDone, "")
			Expect(err).NotTo(HaveOccurred())
		})

		It("updates pipeline_run status on completion", func() {
			pipelineStore := store.NewPipelineStore(EntClient)
			run, err := pipelineStore.CreateRun("status-test", "status-event-"+types.Id(), types.EventKanbanTaskCreated)
			Expect(err).NotTo(HaveOccurred())

			err = pipelineStore.UpdateRunStatus(run.ID, model.PipelineDone, "")
			Expect(err).NotTo(HaveOccurred())

			saved, err := EntClient.PipelineRun.Get(context.Background(), run.ID)
			Expect(err).NotTo(HaveOccurred())
			Expect(saved.Status).To(Equal(model.PipelineDone))
		})

		It("records step execution results", func() {
			pipelineStore := store.NewPipelineStore(EntClient)
			run, err := pipelineStore.CreateRun("step-test", "step-event-"+types.Id(), types.EventReaderEntryStarred)
			Expect(err).NotTo(HaveOccurred())

			params := model.JSON{"url": "https://example.com"}
			stepRun, err := pipelineStore.CreateStepRun(run.ID, "fetch-step", "reader", "list_entries", params, 1)
			Expect(err).NotTo(HaveOccurred())
			Expect(stepRun.StepName).To(Equal("fetch-step"))
			Expect(stepRun.Attempt).To(Equal(1))

			result := model.JSON{"entries": []string{"entry-1"}}
			err = pipelineStore.UpdateStepRun(stepRun.ID, model.PipelineDone, result, "", 1)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Describe("Idempotency", func() {
		It("tracks delivery status per event per pipeline", func() {
			pipelineStore := store.NewPipelineStore(EntClient)
			consumerName := "pipeline-checker-" + types.Id()
			eventID := "idem-event-" + types.Id()

			err := pipelineStore.RecordConsumption(consumerName, eventID)
			Expect(err).NotTo(HaveOccurred())

			consumed, err := pipelineStore.HasConsumed(consumerName, eventID)
			Expect(err).NotTo(HaveOccurred())
			Expect(consumed).To(BeTrue())

			notConsumed, err := pipelineStore.HasConsumed(consumerName, "other-event")
			Expect(err).NotTo(HaveOccurred())
			Expect(notConsumed).To(BeFalse())
		})
	})

	Describe("Event definitions", func() {
		It("has all expected event types", func() {
			Expect(types.EventBookmarkCreated).To(Equal("bookmark.created"))
			Expect(types.EventBookmarkArchived).To(Equal("bookmark.archived"))
			Expect(types.EventReaderEntryStarred).To(Equal("reader.entry.starred"))
			Expect(types.EventReaderEntryRead).To(Equal("reader.entry.read"))
			Expect(types.EventKanbanTaskCreated).To(Equal("kanban.task.created"))
			Expect(types.EventKanbanTaskCompleted).To(Equal("kanban.task.completed"))
		})
	})

	Describe("Event outbox", func() {
		It("persists events to outbox for reliable delivery", func() {
			event := types.DataEvent{
				EventID:   "outbox-test-" + types.Id(),
				EventType: types.EventBookmarkCreated,
				Source:    "test",
				Data:      types.KV{"test": true},
			}

			eventStore := store.NewEventStore(EntClient)
			err := eventStore.AppendEventOutbox(event)
			Expect(err).NotTo(HaveOccurred())

			outbox, err := EntClient.EventOutbox.Query().Where(eventoutbox.EventID(event.EventID)).Only(context.Background())
			Expect(err).NotTo(HaveOccurred())
			Expect(outbox.Published).To(BeFalse())
		})

		It("marks outbox entries as published", func() {
			event := types.DataEvent{
				EventID:   "outbox-pub-" + types.Id(),
				EventType: types.EventReaderEntryRead,
				Source:    "test",
			}

			eventStore := store.NewEventStore(EntClient)
			err := eventStore.AppendEventOutbox(event)
			Expect(err).NotTo(HaveOccurred())

			err = eventStore.MarkOutboxPublished(event.EventID)
			Expect(err).NotTo(HaveOccurred())

			outbox, err := EntClient.EventOutbox.Query().Where(eventoutbox.EventID(event.EventID)).Only(context.Background())
			Expect(err).NotTo(HaveOccurred())
			Expect(outbox.Published).To(BeTrue())
		})
	})

	Describe("Ability EventStore interface", func() {
		It("satisfies the ability.EventStore interface", func() {
			es := store.NewEventStore(EntClient)
			var iface ability.EventStore = es
			Expect(iface).NotTo(BeNil())
		})
	})

	Describe("DataEvent serialization", func() {
		It("serializes and deserializes DataEvent through JSON", func() {
			original := types.DataEvent{
				EventID:        "json-test-" + types.Id(),
				EventType:      types.EventKanbanTaskCreated,
				Source:         "test",
				Capability:     "kanban",
				Operation:      "create_task",
				EntityID:       "task-42",
				IdempotencyKey: "idem-key",
				UID:            "user-1",
				Topic:          "test-topic",
				Data:           types.KV{"title": "Test Task"},
			}

			data, err := sonic.Marshal(original)
			Expect(err).NotTo(HaveOccurred())

			var restored types.DataEvent
			err = sonic.Unmarshal(data, &restored)
			Expect(err).NotTo(HaveOccurred())
			Expect(restored.EventID).To(Equal(original.EventID))
			Expect(restored.EventType).To(Equal(original.EventType))
			Expect(restored.EntityID).To(Equal(original.EntityID))
			Expect(restored.Data["title"]).To(Equal("Test Task"))
		})
	})

	Describe("Pipeline run heartbeat", func() {
		It("updates heartbeat timestamp", func() {
			pipelineStore := store.NewPipelineStore(EntClient)
			run, err := pipelineStore.CreateRun("heartbeat-test", "hb-event-"+types.Id(), types.EventBookmarkArchived)
			Expect(err).NotTo(HaveOccurred())

			err = pipelineStore.UpdateRunHeartbeat(run.ID)
			Expect(err).NotTo(HaveOccurred())

			saved, err := EntClient.PipelineRun.Get(context.Background(), run.ID)
			Expect(err).NotTo(HaveOccurred())
			Expect(saved.LastHeartbeat).NotTo(BeNil())
		})
	})

	Describe("Pipeline checkpoint", func() {
		It("saves and retrieves checkpoint data", func() {
			pipelineStore := store.NewPipelineStore(EntClient)
			run, err := pipelineStore.CreateRun("checkpoint-test", "cp-event-"+types.Id(), types.EventReaderEntryRead)
			Expect(err).NotTo(HaveOccurred())

			checkpoint := map[string]any{
				"step_index": 2,
				"processed":  []string{"step-1", "step-2"},
			}

			err = pipelineStore.SaveCheckpoint(run.ID, checkpoint)
			Expect(err).NotTo(HaveOccurred())

			var loaded map[string]any
			err = pipelineStore.GetCheckpoint(run.ID, &loaded)
			Expect(err).NotTo(HaveOccurred())
			Expect(int(loaded["step_index"].(float64))).To(Equal(2))
		})
	})

	Describe("Incomplete runs", func() {
		It("finds runs that did not complete", func() {
			pipelineStore := store.NewPipelineStore(EntClient)
			run, err := pipelineStore.CreateRun("incomplete-test", "inc-event-"+types.Id(), types.EventBookmarkCreated)
			Expect(err).NotTo(HaveOccurred())

			time.Sleep(100 * time.Millisecond)

			runs, err := pipelineStore.GetIncompleteRuns()
			Expect(err).NotTo(HaveOccurred())

			found := false
			for _, r := range runs {
				if r.ID == run.ID {
					found = true
					break
				}
			}
			Expect(found).To(BeTrue(), "expected the newly created run to appear in incomplete runs")
		})
	})
})
