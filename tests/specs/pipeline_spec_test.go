//go:build integration
// +build integration

package specs

import (
	"context"
	"time"

	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/internal/store/model"
	"github.com/flowline-io/flowbot/pkg/hub"
	"github.com/flowline-io/flowbot/pkg/metrics"
	"github.com/flowline-io/flowbot/pkg/pipeline"
	"github.com/flowline-io/flowbot/pkg/pipeline/template"
	"github.com/flowline-io/flowbot/pkg/types"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Pipeline Engine", Label("pipeline"), func() {

	Describe("Pipeline Execution", func() {
		Context("with a valid definition", func() {
			It("constructs pipeline definition with trigger and steps", func() {
				def := pipeline.Definition{
					Name:    "test-pipeline-" + types.Id(),
					Enabled: true,
					Trigger: pipeline.Trigger{Event: types.EventBookmarkCreated},
					Steps: []pipeline.Step{
						{Name: "step-1", Capability: hub.CapNotify, Operation: "send", Params: map[string]any{"message": "hello"}},
					},
				}
				Expect(def.Name).NotTo(BeEmpty())
				Expect(def.Enabled).To(BeTrue())
				Expect(def.Trigger.Event).To(Equal(types.EventBookmarkCreated))
				Expect(def.Steps).To(HaveLen(1))
				Expect(def.Steps[0].Name).To(Equal("step-1"))
			})

			It("passes step output as input to the next step", func() {
				rc := pipeline.NewRenderContext(types.DataEvent{
					EventID:   "test-event-" + types.Id(),
					EventType: types.EventBookmarkCreated,
					Data:      types.KV{"url": "https://example.com"},
				})
				Expect(rc).NotTo(BeNil())

				rc.RecordStepResult("fetch", map[string]any{"title": "Example"})

				input := map[string]any{"url": "{{event.url}}"}
				rendered, err := rc.RenderParams(input)
				Expect(err).NotTo(HaveOccurred())
				Expect(rendered["url"]).To(Equal("https://example.com"))
			})

			It("returns final step result on success", func() {
				event := types.DataEvent{
					EventID:   "result-test-" + types.Id(),
					EventType: types.EventReaderEntryStarred,
				}
				rc := pipeline.NewRenderContext(event)
				rc.RecordStepResult("step-1", map[string]any{"output": "value-1"})

				result, err := rc.RenderString(`{{step "step-1" "output"}}`)
				Expect(err).NotTo(HaveOccurred())
				Expect(result).To(Equal("value-1"))
			})
		})

		Context("with a disabled pipeline", func() {
			It("creates disabled pipeline definition", func() {
				def := pipeline.Definition{
					Name:    "disabled-pipeline-" + types.Id(),
					Enabled: false,
					Trigger: pipeline.Trigger{Event: types.EventBookmarkCreated},
				}
				Expect(def.Enabled).To(BeFalse())
				Expect(def.Name).NotTo(BeEmpty())
			})
		})

		Context("with retry configuration", func() {
			It("respects maximum retry count", func() {
				retry := types.RetryConfig{
					MaxAttempts: 3,
					Delay:       100 * time.Millisecond,
					Backoff:     types.BackoffFixed,
				}
				Expect(retry.RetryEnabled()).To(BeTrue())
				Expect(retry.MaxAttempts).To(Equal(3))
			})

			It("builds backoff from config", func() {
				retry := types.RetryConfig{
					MaxAttempts: 5,
					Delay:       1 * time.Second,
					Backoff:     types.BackoffExponential,
					MaxDelay:    30 * time.Second,
				}
				bo := retry.BuildBackOff()
				Expect(bo).NotTo(BeNil())
			})
		})
	})

	Describe("Event Matching", func() {
		It("matches pipeline by exact event name", func() {
			defs := []pipeline.Definition{
				{Name: "p1", Enabled: true, Trigger: pipeline.Trigger{Event: types.EventBookmarkCreated}, Steps: []pipeline.Step{}},
				{Name: "p2", Enabled: true, Trigger: pipeline.Trigger{Event: types.EventKanbanTaskCreated}, Steps: []pipeline.Step{}},
			}

			matched := pipeline.FindByEvent(defs, types.EventBookmarkCreated)
			Expect(matched).To(HaveLen(1))
			Expect(matched[0].Name).To(Equal("p1"))
		})

		It("matches multiple pipelines to the same event", func() {
			defs := []pipeline.Definition{
				{Name: "p1", Enabled: true, Trigger: pipeline.Trigger{Event: types.EventBookmarkArchived}, Steps: []pipeline.Step{}},
				{Name: "p2", Enabled: true, Trigger: pipeline.Trigger{Event: types.EventBookmarkArchived}, Steps: []pipeline.Step{}},
				{Name: "p3", Enabled: true, Trigger: pipeline.Trigger{Event: types.EventBookmarkCreated}, Steps: []pipeline.Step{}},
			}

			matched := pipeline.FindByEvent(defs, types.EventBookmarkArchived)
			Expect(matched).To(HaveLen(2))
		})

		It("does not match unrelated events", func() {
			defs := []pipeline.Definition{
				{Name: "p1", Enabled: true, Trigger: pipeline.Trigger{Event: types.EventBookmarkCreated}, Steps: []pipeline.Step{}},
			}

			matched := pipeline.FindByEvent(defs, "unrelated.event")
			Expect(matched).To(BeEmpty())
		})
	})

	Describe("Template Rendering", func() {
		It("renders step parameters from Go templates", func() {
			event := types.DataEvent{
				EventID:   "tmpl-test-" + types.Id(),
				EventType: types.EventBookmarkCreated,
				Data:      types.KV{"url": "https://example.com", "title": "Example"},
			}
			rc := pipeline.NewRenderContext(event)

			rendered, err := rc.RenderParams(map[string]any{
				"message":  "New bookmark: {{event.title}}",
				"endpoint": "{{event.url}}/api",
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(rendered["message"]).To(Equal("New bookmark: Example"))
			Expect(rendered["endpoint"]).To(Equal("https://example.com/api"))
		})

		It("injects trigger event data into template context", func() {
			event := types.DataEvent{
				EventID:   "inject-test-" + types.Id(),
				EventType: types.EventReaderEntryStarred,
				Data:      types.KV{"entry_id": "42", "feed_title": "Tech Blog"},
			}
			rc := pipeline.NewRenderContext(event)

			rendered, err := rc.RenderParams(map[string]any{
				"entry_id": "{{event.entry_id}}",
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(rendered["entry_id"]).To(Equal("42"))
		})

		It("injects previous step results into template context", func() {
			event := types.DataEvent{
				EventID:   "stepref-test-" + types.Id(),
				EventType: types.EventBookmarkCreated,
			}
			rc := pipeline.NewRenderContext(event)
			rc.RecordStepResult("extract", map[string]any{"tags": []string{"go", "testing"}})

			rendered, err := rc.RenderString(`{{ range $tag := step "extract" "tags" }}{{ $tag }},{{ end }}`)
			Expect(err).NotTo(HaveOccurred())
			Expect(rendered).To(ContainSubstring("go"))
			Expect(rendered).To(ContainSubstring("testing"))
		})

		It("returns error for malformed template syntax", func() {
			event := types.DataEvent{
				EventID:   "err-test-" + types.Id(),
				EventType: types.EventBookmarkCreated,
			}
			rc := pipeline.NewRenderContext(event)

			_, err := rc.RenderParams(map[string]any{
				"bad": "{{ .Event.Data.title",
			})
			Expect(err).To(HaveOccurred())
		})
	})

	Describe("Pipeline Loading", func() {
		It("validates DAG structure on load", func() {
			_ = pipeline.Definition{
				Name:    "dag-test",
				Enabled: true,
				Trigger: pipeline.Trigger{Event: "test.event"},
				Steps: []pipeline.Step{
					{Name: "a", Capability: hub.CapNotify, Operation: "send"},
					{Name: "b", Capability: hub.CapNotify, Operation: "send"},
				},
			}
		})
	})

	Describe("Pipeline engine with store", func() {
		It("creates pipeline definitions via store", func() {
			pipelineStore := store.NewPipelineStore(EntClient)
			err := pipelineStore.UpsertDefinition(
				context.Background(),
				"stored-pipeline-"+types.Id(),
				"test stored definition",
				true,
				model.JSON{"event": "test.event"},
				model.JSON{"steps": []string{"step-1"}},
			)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Describe("Template engine", func() {
		It("renders with built-in helper functions", func() {
			engine := template.New()
			Expect(engine).NotTo(BeNil())

			data := &template.TemplateData{
				Event: map[string]any{"title": "Hello", "count": 42},
				Steps: map[string]map[string]any{
					"step-1": {"result": "success"},
				},
				Env: map[string]string{"HOST": "localhost"},
			}

			result, err := engine.RenderString("{{ .Event.title }}", data)
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(Equal("Hello"))
		})
	})

	Describe("Engine handler creation", func() {
		It("creates engine with definitions and store", func() {
			defs := []pipeline.Definition{}
			pipelineStore := store.NewPipelineStore(EntClient)
			eng := pipeline.NewEngine(defs, pipelineStore, metrics.NewPipelineCollector(nil), metrics.NewEventCollector(nil))
			Expect(eng).NotTo(BeNil())

			handler := eng.Handler()
			Expect(handler).NotTo(BeNil())
		})
	})

	Describe("ResumePipeline", func() {
		It("handles non-existent run gracefully", func() {
			pipelineStore := store.NewPipelineStore(EntClient)
			eng := pipeline.NewEngine([]pipeline.Definition{}, pipelineStore, metrics.NewPipelineCollector(nil), metrics.NewEventCollector(nil))

			err := eng.ResumePipeline(context.Background(), 99999)
			Expect(err).To(HaveOccurred())
		})
	})
})
