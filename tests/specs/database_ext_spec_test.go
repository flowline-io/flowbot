//go:build integration
// +build integration

package specs

import (
	"context"
	"time"

	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/pkg/types"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Database Extended Models", Label("database", "integration"), func() {
	ctx := context.Background()

	Describe("Topic", func() {
		It("creates a new topic with valid data", func() {
			t, err := EntClient.Topic.Create().
				SetFlag("topic-" + types.Id()).
				SetPlatform("test").
				SetOwner(0).
				SetName("Test Topic").
				SetType("channel").
				Save(ctx)
			Expect(err).NotTo(HaveOccurred())
			Expect(t.ID).NotTo(BeZero())

			EntClient.Topic.DeleteOne(t).Exec(ctx)
		})

		It("retrieves a topic by ID", func() {
			t, err := EntClient.Topic.Create().
				SetFlag("topic-get-" + types.Id()).
				SetPlatform("test").
				SetOwner(0).
				SetName("Get Topic").
				SetType("channel").
				Save(ctx)
			Expect(err).NotTo(HaveOccurred())

			got, err := EntClient.Topic.Get(ctx, t.ID)
			Expect(err).NotTo(HaveOccurred())
			Expect(got.Name).To(Equal("Get Topic"))

			EntClient.Topic.DeleteOne(t).Exec(ctx)
		})

		It("updates topic fields", func() {
			t, err := EntClient.Topic.Create().
				SetFlag("topic-upd-" + types.Id()).
				SetPlatform("test").
				SetOwner(0).
				SetName("Original").
				SetType("channel").
				Save(ctx)
			Expect(err).NotTo(HaveOccurred())

			updated, err := EntClient.Topic.UpdateOne(t).SetName("Updated").Save(ctx)
			Expect(err).NotTo(HaveOccurred())
			Expect(updated.Name).To(Equal("Updated"))

			EntClient.Topic.DeleteOne(t).Exec(ctx)
		})

		It("hard-deletes a topic", func() {
			t, err := EntClient.Topic.Create().
				SetFlag("topic-del-" + types.Id()).
				SetPlatform("test").
				SetOwner(0).
				SetName("Delete Topic").
				SetType("channel").
				Save(ctx)
			Expect(err).NotTo(HaveOccurred())

			err = EntClient.Topic.DeleteOne(t).Exec(ctx)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Describe("Fileupload", func() {
		It("creates a new file upload record", func() {
			f, err := EntClient.Fileupload.Create().
				SetUID("uid-" + types.Id()).
				SetFid("fid-" + types.Id()).
				SetName("test.txt").
				SetMimetype("text/plain").
				SetSize(100).
				SetLocation("/tmp/test.txt").
				Save(ctx)
			Expect(err).NotTo(HaveOccurred())
			Expect(f.ID).NotTo(BeZero())

			EntClient.Fileupload.DeleteOne(f).Exec(ctx)
		})

		It("retrieves a file upload by ID", func() {
			f, err := EntClient.Fileupload.Create().
				SetUID("uid-" + types.Id()).
				SetFid("fid-get-" + types.Id()).
				SetName("get.txt").
				SetMimetype("text/plain").
				SetSize(200).
				SetLocation("/tmp/get.txt").
				Save(ctx)
			Expect(err).NotTo(HaveOccurred())

			got, err := EntClient.Fileupload.Get(ctx, f.ID)
			Expect(err).NotTo(HaveOccurred())
			Expect(got.Name).To(Equal("get.txt"))

			EntClient.Fileupload.DeleteOne(f).Exec(ctx)
		})

		It("transitions file state", func() {
			f, err := EntClient.Fileupload.Create().
				SetUID("uid-" + types.Id()).
				SetFid("fid-state-" + types.Id()).
				SetName("state.txt").
				SetMimetype("text/plain").
				SetSize(300).
				SetLocation("/tmp/state.txt").
				Save(ctx)
			Expect(err).NotTo(HaveOccurred())

			updated, err := EntClient.Fileupload.UpdateOne(f).SetState(1).Save(ctx)
			Expect(err).NotTo(HaveOccurred())
			Expect(updated.State).To(Equal(1))

			EntClient.Fileupload.DeleteOne(f).Exec(ctx)
		})

		It("deletes a file upload record", func() {
			f, err := EntClient.Fileupload.Create().
				SetUID("uid-" + types.Id()).
				SetFid("fid-del-" + types.Id()).
				SetName("del.txt").
				SetMimetype("text/plain").
				SetSize(400).
				SetLocation("/tmp/del.txt").
				Save(ctx)
			Expect(err).NotTo(HaveOccurred())

			err = EntClient.Fileupload.DeleteOne(f).Exec(ctx)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Describe("URL", func() {
		It("creates a new URL record", func() {
			u, err := EntClient.Url.Create().
				SetFlag("url-" + types.Id()).
				SetURL("https://example.com").
				Save(ctx)
			Expect(err).NotTo(HaveOccurred())
			Expect(u.ID).NotTo(BeZero())

			EntClient.Url.DeleteOne(u).Exec(ctx)
		})

		It("defaults view count to zero on creation", func() {
			u, err := EntClient.Url.Create().
				SetFlag("url-view-" + types.Id()).
				SetURL("https://example.com/page").
				Save(ctx)
			Expect(err).NotTo(HaveOccurred())
			Expect(u.ViewCount).To(Equal(int32(0)))

			EntClient.Url.DeleteOne(u).Exec(ctx)
		})

		It("increments view count", func() {
			u, err := EntClient.Url.Create().
				SetFlag("url-inc-" + types.Id()).
				SetURL("https://example.com/inc").
				Save(ctx)
			Expect(err).NotTo(HaveOccurred())

			updated, err := EntClient.Url.UpdateOne(u).SetViewCount(5).Save(ctx)
			Expect(err).NotTo(HaveOccurred())
			Expect(updated.ViewCount).To(Equal(int32(5)))

			EntClient.Url.DeleteOne(u).Exec(ctx)
		})

		It("updates URL fields", func() {
			u, err := EntClient.Url.Create().
				SetFlag("url-upd-" + types.Id()).
				SetURL("https://example.com/old").
				Save(ctx)
			Expect(err).NotTo(HaveOccurred())

			updated, err := EntClient.Url.UpdateOne(u).SetURL("https://example.com/new").Save(ctx)
			Expect(err).NotTo(HaveOccurred())
			Expect(updated.URL).To(Equal("https://example.com/new"))

			EntClient.Url.DeleteOne(u).Exec(ctx)
		})

		It("deletes a URL record", func() {
			u, err := EntClient.Url.Create().
				SetFlag("url-del-" + types.Id()).
				SetURL("https://example.com/del").
				Save(ctx)
			Expect(err).NotTo(HaveOccurred())

			err = EntClient.Url.DeleteOne(u).Exec(ctx)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Describe("App", func() {
		It("creates a new app registration", func() {
			a, err := EntClient.App.Create().
				SetName("test-app-" + types.Id()).
				SetPath("/apps/test").
				Save(ctx)
			Expect(err).NotTo(HaveOccurred())
			Expect(a.ID).NotTo(BeZero())

			EntClient.App.DeleteOne(a).Exec(ctx)
		})

		It("retrieves an app by ID", func() {
			a, err := EntClient.App.Create().
				SetName("app-get-" + types.Id()).
				SetPath("/apps/get").
				Save(ctx)
			Expect(err).NotTo(HaveOccurred())

			got, err := EntClient.App.Get(ctx, a.ID)
			Expect(err).NotTo(HaveOccurred())
			Expect(got.Name).To(Equal(a.Name))

			EntClient.App.DeleteOne(a).Exec(ctx)
		})

		It("updates app fields", func() {
			a, err := EntClient.App.Create().
				SetName("app-upd-" + types.Id()).
				SetPath("/apps/upd").
				Save(ctx)
			Expect(err).NotTo(HaveOccurred())

			updated, err := EntClient.App.UpdateOne(a).SetStatus("running").Save(ctx)
			Expect(err).NotTo(HaveOccurred())
			Expect(updated.Status).To(Equal("running"))

			EntClient.App.DeleteOne(a).Exec(ctx)
		})

		It("deletes an app", func() {
			a, err := EntClient.App.Create().
				SetName("app-del-" + types.Id()).
				SetPath("/apps/del").
				Save(ctx)
			Expect(err).NotTo(HaveOccurred())

			err = EntClient.App.DeleteOne(a).Exec(ctx)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Describe("CapabilityBinding", func() {
		It("creates a new capability binding", func() {
			cb, err := EntClient.CapabilityBinding.Create().
				SetCapability("karakeep").
				SetApp("test-app").
				Save(ctx)
			Expect(err).NotTo(HaveOccurred())
			Expect(cb.ID).NotTo(BeZero())

			EntClient.CapabilityBinding.DeleteOne(cb).Exec(ctx)
		})

		It("retrieves a binding by ID", func() {
			cb, err := EntClient.CapabilityBinding.Create().
				SetCapability("miniflux").
				SetApp("reader-app").
				Save(ctx)
			Expect(err).NotTo(HaveOccurred())

			got, err := EntClient.CapabilityBinding.Get(ctx, cb.ID)
			Expect(err).NotTo(HaveOccurred())
			Expect(got.Capability).To(Equal("miniflux"))

			EntClient.CapabilityBinding.DeleteOne(cb).Exec(ctx)
		})

		It("updates binding fields", func() {
			cb, err := EntClient.CapabilityBinding.Create().
				SetCapability("kanboard").
				SetApp("kanban-app").
				Save(ctx)
			Expect(err).NotTo(HaveOccurred())

			updated, err := EntClient.CapabilityBinding.UpdateOne(cb).SetHealthy(true).Save(ctx)
			Expect(err).NotTo(HaveOccurred())
			Expect(updated.Healthy).To(BeTrue())

			EntClient.CapabilityBinding.DeleteOne(cb).Exec(ctx)
		})

		It("deletes a binding", func() {
			cb, err := EntClient.CapabilityBinding.Create().
				SetCapability("archive").
				SetApp("archive-app").
				Save(ctx)
			Expect(err).NotTo(HaveOccurred())

			err = EntClient.CapabilityBinding.DeleteOne(cb).Exec(ctx)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Describe("AuditLog", func() {
		It("creates a new audit log entry", func() {
			al, err := EntClient.AuditLog.Create().
				SetAction("test.action").
				SetTargetType("bookmark").
				SetTargetID("123").
				Save(ctx)
			Expect(err).NotTo(HaveOccurred())
			Expect(al.ID).NotTo(BeZero())

			EntClient.AuditLog.DeleteOne(al).Exec(ctx)
		})

		It("retrieves audit logs by actor", func() {
			al, err := EntClient.AuditLog.Create().
				SetAction("user.action").
				SetTargetType("user").
				SetTargetID("456").
				SetActorUID("actor-1").
				Save(ctx)
			Expect(err).NotTo(HaveOccurred())

			got, err := EntClient.AuditLog.Get(ctx, al.ID)
			Expect(err).NotTo(HaveOccurred())
			Expect(got.ActorUID).To(Equal("actor-1"))

			EntClient.AuditLog.DeleteOne(al).Exec(ctx)
		})

		It("retrieves audit logs by action type", func() {
			al, err := EntClient.AuditLog.Create().
				SetAction("delete").
				SetTargetType("task").
				SetTargetID("789").
				Save(ctx)
			Expect(err).NotTo(HaveOccurred())

			got, err := EntClient.AuditLog.Get(ctx, al.ID)
			Expect(err).NotTo(HaveOccurred())
			Expect(got.Action).To(Equal("delete"))

			EntClient.AuditLog.DeleteOne(al).Exec(ctx)
		})

		It("creates audit log with details", func() {
			al, err := EntClient.AuditLog.Create().
				SetAction("update").
				SetTargetType("config").
				SetTargetID("cfg-1").
				SetDetails(map[string]any{"key": "timeout", "old": "30s", "new": "60s"}).
				Save(ctx)
			Expect(err).NotTo(HaveOccurred())

			Expect(al.Details["key"]).To(Equal("timeout"))

			EntClient.AuditLog.DeleteOne(al).Exec(ctx)
		})
	})

	Describe("Parameter", func() {
		It("creates a new parameter", func() {
			p, err := EntClient.Parameter.Create().
				SetFlag("param-" + types.Id()).
				SetParams(map[string]any{"key": "val"}).
				SetExpiredAt(time.Now().Add(time.Hour)).
				Save(ctx)
			Expect(err).NotTo(HaveOccurred())
			Expect(p.ID).NotTo(BeZero())

			EntClient.Parameter.DeleteOne(p).Exec(ctx)
		})

		It("retrieves a parameter by ID", func() {
			p, err := EntClient.Parameter.Create().
				SetFlag("param-get-" + types.Id()).
				SetParams(map[string]any{"mode": "auto"}).
				SetExpiredAt(time.Now().Add(time.Hour)).
				Save(ctx)
			Expect(err).NotTo(HaveOccurred())

			got, err := EntClient.Parameter.Get(ctx, p.ID)
			Expect(err).NotTo(HaveOccurred())
			Expect(got.Flag).To(Equal(p.Flag))

			EntClient.Parameter.DeleteOne(p).Exec(ctx)
		})

		It("updates parameter value", func() {
			p, err := EntClient.Parameter.Create().
				SetFlag("param-upd-" + types.Id()).
				SetParams(map[string]any{"val": "old"}).
				SetExpiredAt(time.Now().Add(time.Hour)).
				Save(ctx)
			Expect(err).NotTo(HaveOccurred())

			updated, err := EntClient.Parameter.UpdateOne(p).SetParams(map[string]any{"val": "new"}).Save(ctx)
			Expect(err).NotTo(HaveOccurred())
			Expect(updated.Params["val"]).To(Equal("new"))

			EntClient.Parameter.DeleteOne(p).Exec(ctx)
		})

		It("deletes a parameter", func() {
			p, err := EntClient.Parameter.Create().
				SetFlag("param-del-" + types.Id()).
				SetParams(map[string]any{}).
				SetExpiredAt(time.Now().Add(time.Hour)).
				Save(ctx)
			Expect(err).NotTo(HaveOccurred())

			err = EntClient.Parameter.DeleteOne(p).Exec(ctx)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Describe("Connection", func() {
		It("creates a new connection", func() {
			c, err := EntClient.Connection.Create().
				SetUID("uid-" + types.Id()).
				SetTopic("conn-topic").
				SetName("test-conn").
				SetType("slack").
				SetConfig(map[string]any{"token": "xxx"}).
				Save(ctx)
			Expect(err).NotTo(HaveOccurred())
			Expect(c.ID).NotTo(BeZero())

			EntClient.Connection.DeleteOne(c).Exec(ctx)
		})

		It("retrieves a connection by ID", func() {
			c, err := EntClient.Connection.Create().
				SetUID("uid-" + types.Id()).
				SetTopic("conn-get").
				SetName("get-conn").
				SetType("discord").
				SetConfig(map[string]any{}).
				Save(ctx)
			Expect(err).NotTo(HaveOccurred())

			got, err := EntClient.Connection.Get(ctx, c.ID)
			Expect(err).NotTo(HaveOccurred())
			Expect(got.Name).To(Equal("get-conn"))

			EntClient.Connection.DeleteOne(c).Exec(ctx)
		})

		It("updates connection fields", func() {
			c, err := EntClient.Connection.Create().
				SetUID("uid-" + types.Id()).
				SetTopic("conn-upd").
				SetName("upd-conn").
				SetType("email").
				SetConfig(map[string]any{}).
				Save(ctx)
			Expect(err).NotTo(HaveOccurred())

			updated, err := EntClient.Connection.UpdateOne(c).SetEnabled(false).Save(ctx)
			Expect(err).NotTo(HaveOccurred())
			Expect(updated.Enabled).To(BeFalse())

			EntClient.Connection.DeleteOne(c).Exec(ctx)
		})

		It("deletes a connection", func() {
			c, err := EntClient.Connection.Create().
				SetUID("uid-" + types.Id()).
				SetTopic("conn-del").
				SetName("del-conn").
				SetType("manual").
				SetConfig(map[string]any{}).
				Save(ctx)
			Expect(err).NotTo(HaveOccurred())

			err = EntClient.Connection.DeleteOne(c).Exec(ctx)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Describe("AgentKnowledge", func() {
		It("creates a knowledge document", func() {
			doc, err := EntClient.AgentKnowledge.Create().
				SetPath("/docs/bdd-" + types.Id() + ".md").
				SetTitle("BDD Doc").
				SetTags([]string{"bdd"}).
				SetSummary("summary").
				SetContent("# Hello").
				Save(ctx)
			Expect(err).NotTo(HaveOccurred())
			Expect(doc.ID).NotTo(BeZero())
			EntClient.AgentKnowledge.DeleteOne(doc).Exec(ctx)
		})

		It("retrieves a knowledge document by path", func() {
			path := "/docs/bdd-get-" + types.Id() + ".md"
			doc, err := EntClient.AgentKnowledge.Create().
				SetPath(path).
				SetTitle("Get Doc").
				SetTags([]string{}).
				SetSummary("").
				SetContent("body").
				Save(ctx)
			Expect(err).NotTo(HaveOccurred())

			row, err := store.Database.GetAgentKnowledgeByPath(ctx, path)
			Expect(err).NotTo(HaveOccurred())
			Expect(row.Title).To(Equal("Get Doc"))

			EntClient.AgentKnowledge.DeleteOne(doc).Exec(ctx)
		})

		It("searches knowledge documents by content", func() {
			path := "/docs/bdd-search-" + types.Id() + ".md"
			doc, err := EntClient.AgentKnowledge.Create().
				SetPath(path).
				SetTitle("Search Doc").
				SetTags([]string{"ops"}).
				SetSummary("meta").
				SetContent("unique-knowledge-token-xyz").
				Save(ctx)
			Expect(err).NotTo(HaveOccurred())

			rows, err := store.Database.SearchAgentKnowledge(ctx, store.AgentKnowledgeSearchParams{
				Query: "unique-knowledge-token-xyz",
				Limit: 10,
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(rows).NotTo(BeEmpty())
			Expect(rows[0].Path).To(Equal(path))

			EntClient.AgentKnowledge.DeleteOne(doc).Exec(ctx)
		})
	})
})
