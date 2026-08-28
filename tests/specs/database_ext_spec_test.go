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

			row, err := store.AgentStoreFromDB().GetAgentKnowledgeByPath(ctx, path)
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

			rows, err := store.AgentStoreFromDB().SearchAgentKnowledge(ctx, store.AgentKnowledgeSearchParams{
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
