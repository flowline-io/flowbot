//go:build integration
// +build integration

package specs

import (
	"context"
	"time"

	"github.com/flowline-io/flowbot/pkg/types"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Database Core Models", Label("database", "integration"), func() {
	ctx := context.Background()

	Describe("User", func() {
		It("creates a new user with valid data", func() {
			u, err := EntClient.User.Create().SetFlag("test-flag-" + types.Id()).SetName("Test User").Save(ctx)
			Expect(err).NotTo(HaveOccurred())
			Expect(u.ID).NotTo(BeZero())
			Expect(u.Name).To(Equal("Test User"))

			EntClient.User.DeleteOne(u).Exec(ctx)
		})

		It("retrieves a user by ID", func() {
			u, err := EntClient.User.Create().SetFlag("get-test-" + types.Id()).SetName("Get Test").Save(ctx)
			Expect(err).NotTo(HaveOccurred())

			got, err := EntClient.User.Get(ctx, u.ID)
			Expect(err).NotTo(HaveOccurred())
			Expect(got.Name).To(Equal("Get Test"))

			EntClient.User.DeleteOne(u).Exec(ctx)
		})

		It("updates user fields", func() {
			u, err := EntClient.User.Create().SetFlag("upd-test-" + types.Id()).SetName("Original").Save(ctx)
			Expect(err).NotTo(HaveOccurred())

			updated, err := EntClient.User.UpdateOne(u).SetName("Updated").Save(ctx)
			Expect(err).NotTo(HaveOccurred())
			Expect(updated.Name).To(Equal("Updated"))

			EntClient.User.DeleteOne(u).Exec(ctx)
		})

		It("deletes a user", func() {
			u, err := EntClient.User.Create().SetFlag("del-test-" + types.Id()).SetName("Delete Me").Save(ctx)
			Expect(err).NotTo(HaveOccurred())

			err = EntClient.User.DeleteOne(u).Exec(ctx)
			Expect(err).NotTo(HaveOccurred())

			_, err = EntClient.User.Get(ctx, u.ID)
			Expect(err).To(HaveOccurred())
		})

		It("rejects creation with empty flag", func() {
			_, err := EntClient.User.Create().SetFlag("").SetName("No Flag").Save(ctx)
			Expect(err).To(HaveOccurred())
		})
	})

	Describe("Bot", func() {
		It("creates a new bot with valid data", func() {
			b, err := EntClient.Bot.Create().SetName("test-bot-" + types.Id()).Save(ctx)
			Expect(err).NotTo(HaveOccurred())
			Expect(b.ID).NotTo(BeZero())

			EntClient.Bot.DeleteOne(b).Exec(ctx)
		})

		It("retrieves a bot by ID", func() {
			b, err := EntClient.Bot.Create().SetName("get-bot-" + types.Id()).Save(ctx)
			Expect(err).NotTo(HaveOccurred())

			got, err := EntClient.Bot.Get(ctx, b.ID)
			Expect(err).NotTo(HaveOccurred())
			Expect(got.Name).To(Equal(b.Name))

			EntClient.Bot.DeleteOne(b).Exec(ctx)
		})

		It("updates bot fields", func() {
			b, err := EntClient.Bot.Create().SetName("upd-bot-" + types.Id()).Save(ctx)
			Expect(err).NotTo(HaveOccurred())

			updated, err := EntClient.Bot.UpdateOne(b).SetState(1).Save(ctx)
			Expect(err).NotTo(HaveOccurred())
			Expect(updated.State).To(Equal(1))

			EntClient.Bot.DeleteOne(b).Exec(ctx)
		})

		It("deletes a bot", func() {
			b, err := EntClient.Bot.Create().SetName("del-bot-" + types.Id()).Save(ctx)
			Expect(err).NotTo(HaveOccurred())

			err = EntClient.Bot.DeleteOne(b).Exec(ctx)
			Expect(err).NotTo(HaveOccurred())
		})

		It("rejects creation with empty name", func() {
			_, err := EntClient.Bot.Create().SetName("").Save(ctx)
			Expect(err).To(HaveOccurred())
		})
	})

	Describe("Platform", func() {
		It("creates a new platform", func() {
			p, err := EntClient.Platform.Create().SetName("test-platform-" + types.Id()).Save(ctx)
			Expect(err).NotTo(HaveOccurred())
			Expect(p.ID).NotTo(BeZero())

			EntClient.Platform.DeleteOne(p).Exec(ctx)
		})

		It("retrieves a platform by ID", func() {
			p, err := EntClient.Platform.Create().SetName("get-platform-" + types.Id()).Save(ctx)
			Expect(err).NotTo(HaveOccurred())

			got, err := EntClient.Platform.Get(ctx, p.ID)
			Expect(err).NotTo(HaveOccurred())
			Expect(got.Name).To(Equal(p.Name))

			EntClient.Platform.DeleteOne(p).Exec(ctx)
		})

		It("updates platform fields", func() {
			p, err := EntClient.Platform.Create().SetName("upd-platform-" + types.Id()).Save(ctx)
			Expect(err).NotTo(HaveOccurred())

			updated, err := EntClient.Platform.UpdateOne(p).SetName("renamed-platform").Save(ctx)
			Expect(err).NotTo(HaveOccurred())
			Expect(updated.Name).To(Equal("renamed-platform"))

			EntClient.Platform.DeleteOne(p).Exec(ctx)
		})

		It("deletes a platform", func() {
			p, err := EntClient.Platform.Create().SetName("del-platform-" + types.Id()).Save(ctx)
			Expect(err).NotTo(HaveOccurred())

			err = EntClient.Platform.DeleteOne(p).Exec(ctx)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Describe("Webhook", func() {
		It("creates a new webhook with valid data", func() {
			w, err := EntClient.Webhook.Create().
				SetUID("uid-" + types.Id()).
				SetTopic("test-topic").
				SetFlag("test-flag").
				SetSecret("secret-123").
				Save(ctx)
			Expect(err).NotTo(HaveOccurred())
			Expect(w.ID).NotTo(BeZero())

			EntClient.Webhook.DeleteOne(w).Exec(ctx)
		})

		It("retrieves a webhook by ID", func() {
			w, err := EntClient.Webhook.Create().
				SetUID("uid-" + types.Id()).
				SetTopic("get-topic").
				SetFlag("get-flag").
				SetSecret("get-secret").
				Save(ctx)
			Expect(err).NotTo(HaveOccurred())

			got, err := EntClient.Webhook.Get(ctx, w.ID)
			Expect(err).NotTo(HaveOccurred())
			Expect(got.Secret).To(Equal("get-secret"))

			EntClient.Webhook.DeleteOne(w).Exec(ctx)
		})

		It("updates webhook fields", func() {
			w, err := EntClient.Webhook.Create().
				SetUID("uid-" + types.Id()).
				SetTopic("upd-topic").
				SetFlag("upd-flag").
				SetSecret("upd-secret").
				Save(ctx)
			Expect(err).NotTo(HaveOccurred())

			updated, err := EntClient.Webhook.UpdateOne(w).SetState(1).Save(ctx)
			Expect(err).NotTo(HaveOccurred())
			Expect(updated.State).To(Equal(1))

			EntClient.Webhook.DeleteOne(w).Exec(ctx)
		})

		It("deletes a webhook", func() {
			w, err := EntClient.Webhook.Create().
				SetUID("uid-" + types.Id()).
				SetTopic("del-topic").
				SetFlag("del-flag").
				SetSecret("del-secret").
				Save(ctx)
			Expect(err).NotTo(HaveOccurred())

			err = EntClient.Webhook.DeleteOne(w).Exec(ctx)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Describe("Data", func() {
		It("creates a new data record", func() {
			d, err := EntClient.Data.Create().
				SetUID("uid-" + types.Id()).
				SetTopic("test-topic").
				SetKey("test-key").
				SetValue(map[string]any{"val": 1}).
				Save(ctx)
			Expect(err).NotTo(HaveOccurred())
			Expect(d.ID).NotTo(BeZero())

			EntClient.Data.DeleteOne(d).Exec(ctx)
		})

		It("retrieves a data record by ID", func() {
			d, err := EntClient.Data.Create().
				SetUID("uid-" + types.Id()).
				SetTopic("get-data").
				SetKey("get-key").
				SetValue(map[string]any{"val": "hello"}).
				Save(ctx)
			Expect(err).NotTo(HaveOccurred())

			got, err := EntClient.Data.Get(ctx, d.ID)
			Expect(err).NotTo(HaveOccurred())
			Expect(got.Key).To(Equal("get-key"))

			EntClient.Data.DeleteOne(d).Exec(ctx)
		})

		It("updates data fields", func() {
			d, err := EntClient.Data.Create().
				SetUID("uid-" + types.Id()).
				SetTopic("upd-data").
				SetKey("upd-key").
				SetValue(map[string]any{"val": 1}).
				Save(ctx)
			Expect(err).NotTo(HaveOccurred())

			updated, err := EntClient.Data.UpdateOne(d).SetValue(map[string]any{"val": 2}).Save(ctx)
			Expect(err).NotTo(HaveOccurred())
			Expect(updated.Value["val"]).To(Equal(float64(2)))

			EntClient.Data.DeleteOne(d).Exec(ctx)
		})

		It("deletes a data record", func() {
			d, err := EntClient.Data.Create().
				SetUID("uid-" + types.Id()).
				SetTopic("del-data").
				SetKey("del-key").
				SetValue(map[string]any{}).
				Save(ctx)
			Expect(err).NotTo(HaveOccurred())

			err = EntClient.Data.DeleteOne(d).Exec(ctx)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Describe("ConfigData", func() {
		It("creates a new configuration entry", func() {
			c, err := EntClient.ConfigData.Create().
				SetUID("uid-" + types.Id()).
				SetTopic("cfg-topic").
				SetKey("cfg-key").
				SetValue(map[string]any{"enabled": true}).
				Save(ctx)
			Expect(err).NotTo(HaveOccurred())
			Expect(c.ID).NotTo(BeZero())

			EntClient.ConfigData.DeleteOne(c).Exec(ctx)
		})

		It("retrieves configuration by key", func() {
			c, err := EntClient.ConfigData.Create().
				SetUID("uid-" + types.Id()).
				SetTopic("cfg-get").
				SetKey("cfg-get-key").
				SetValue(map[string]any{"mode": "auto"}).
				Save(ctx)
			Expect(err).NotTo(HaveOccurred())

			got, err := EntClient.ConfigData.Get(ctx, c.ID)
			Expect(err).NotTo(HaveOccurred())
			Expect(got.Key).To(Equal("cfg-get-key"))

			EntClient.ConfigData.DeleteOne(c).Exec(ctx)
		})

		It("updates configuration value", func() {
			c, err := EntClient.ConfigData.Create().
				SetUID("uid-" + types.Id()).
				SetTopic("cfg-upd").
				SetKey("cfg-upd-key").
				SetValue(map[string]any{"val": "old"}).
				Save(ctx)
			Expect(err).NotTo(HaveOccurred())

			updated, err := EntClient.ConfigData.UpdateOne(c).SetValue(map[string]any{"val": "new"}).Save(ctx)
			Expect(err).NotTo(HaveOccurred())
			Expect(updated.Value["val"]).To(Equal("new"))

			EntClient.ConfigData.DeleteOne(c).Exec(ctx)
		})

		It("deletes a configuration entry", func() {
			c, err := EntClient.ConfigData.Create().
				SetUID("uid-" + types.Id()).
				SetTopic("cfg-del").
				SetKey("cfg-del-key").
				SetValue(map[string]any{}).
				Save(ctx)
			Expect(err).NotTo(HaveOccurred())

			err = EntClient.ConfigData.DeleteOne(c).Exec(ctx)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Describe("Counter", func() {
		It("creates a new counter", func() {
			c, err := EntClient.Counter.Create().
				SetUID("uid-" + types.Id()).
				SetTopic("cnt-topic").
				SetFlag("cnt-flag").
				SetDigit(0).
				Save(ctx)
			Expect(err).NotTo(HaveOccurred())
			Expect(c.ID).NotTo(BeZero())

			EntClient.Counter.DeleteOne(c).Exec(ctx)
		})

		It("increments a counter value", func() {
			c, err := EntClient.Counter.Create().
				SetUID("uid-" + types.Id()).
				SetTopic("cnt-inc").
				SetFlag("cnt-inc-flag").
				SetDigit(5).
				Save(ctx)
			Expect(err).NotTo(HaveOccurred())

			updated, err := EntClient.Counter.UpdateOne(c).SetDigit(10).Save(ctx)
			Expect(err).NotTo(HaveOccurred())
			Expect(updated.Digit).To(Equal(int64(10)))

			EntClient.Counter.DeleteOne(c).Exec(ctx)
		})

		It("deletes a counter", func() {
			c, err := EntClient.Counter.Create().
				SetUID("uid-" + types.Id()).
				SetTopic("cnt-del").
				SetFlag("cnt-del-flag").
				Save(ctx)
			Expect(err).NotTo(HaveOccurred())

			err = EntClient.Counter.DeleteOne(c).Exec(ctx)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Describe("Form", func() {
		It("creates a new form", func() {
			f, err := EntClient.Form.Create().
				SetFormID("form-" + types.Id()).
				SetUID("uid-" + types.Id()).
				SetTopic("form-topic").
				SetSchema(map[string]any{"fields": []string{"name", "email"}}).
				Save(ctx)
			Expect(err).NotTo(HaveOccurred())
			Expect(f.ID).NotTo(BeZero())

			EntClient.Form.DeleteOne(f).Exec(ctx)
		})

		It("retrieves a form by ID", func() {
			f, err := EntClient.Form.Create().
				SetFormID("form-get-" + types.Id()).
				SetUID("uid-" + types.Id()).
				SetTopic("form-get").
				SetSchema(map[string]any{}).
				Save(ctx)
			Expect(err).NotTo(HaveOccurred())

			got, err := EntClient.Form.Get(ctx, f.ID)
			Expect(err).NotTo(HaveOccurred())
			Expect(got.FormID).To(Equal(f.FormID))

			EntClient.Form.DeleteOne(f).Exec(ctx)
		})

		It("deletes a form", func() {
			f, err := EntClient.Form.Create().
				SetFormID("form-del-" + types.Id()).
				SetUID("uid-" + types.Id()).
				SetTopic("form-del").
				SetSchema(map[string]any{}).
				Save(ctx)
			Expect(err).NotTo(HaveOccurred())

			err = EntClient.Form.DeleteOne(f).Exec(ctx)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Describe("Page", func() {
		It("creates a new page", func() {
			p, err := EntClient.Page.Create().
				SetPageID("page-" + types.Id()).
				SetUID("uid-" + types.Id()).
				SetTopic("page-topic").
				SetType("dashboard").
				SetSchema(map[string]any{}).
				Save(ctx)
			Expect(err).NotTo(HaveOccurred())
			Expect(p.ID).NotTo(BeZero())

			EntClient.Page.DeleteOne(p).Exec(ctx)
		})

		It("retrieves a page by ID", func() {
			p, err := EntClient.Page.Create().
				SetPageID("page-get-" + types.Id()).
				SetUID("uid-" + types.Id()).
				SetTopic("page-get").
				SetType("view").
				SetSchema(map[string]any{}).
				Save(ctx)
			Expect(err).NotTo(HaveOccurred())

			got, err := EntClient.Page.Get(ctx, p.ID)
			Expect(err).NotTo(HaveOccurred())
			Expect(got.PageID).To(Equal(p.PageID))

			EntClient.Page.DeleteOne(p).Exec(ctx)
		})

		It("deletes a page", func() {
			p, err := EntClient.Page.Create().
				SetPageID("page-del-" + types.Id()).
				SetUID("uid-" + types.Id()).
				SetTopic("page-del").
				SetType("del").
				SetSchema(map[string]any{}).
				Save(ctx)
			Expect(err).NotTo(HaveOccurred())

			err = EntClient.Page.DeleteOne(p).Exec(ctx)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Describe("Behavior", func() {
		It("creates a new behavior rule", func() {
			b, err := EntClient.Behavior.Create().
				SetUID("uid-" + types.Id()).
				SetFlag("behavior-" + types.Id()).
				Save(ctx)
			Expect(err).NotTo(HaveOccurred())
			Expect(b.ID).NotTo(BeZero())

			EntClient.Behavior.DeleteOne(b).Exec(ctx)
		})

		It("retrieves a behavior by ID", func() {
			b, err := EntClient.Behavior.Create().
				SetUID("uid-" + types.Id()).
				SetFlag("behavior-get-" + types.Id()).
				Save(ctx)
			Expect(err).NotTo(HaveOccurred())

			got, err := EntClient.Behavior.Get(ctx, b.ID)
			Expect(err).NotTo(HaveOccurred())
			Expect(got.Flag).To(Equal(b.Flag))

			EntClient.Behavior.DeleteOne(b).Exec(ctx)
		})

		It("deletes a behavior", func() {
			b, err := EntClient.Behavior.Create().
				SetUID("uid-" + types.Id()).
				SetFlag("behavior-del-" + types.Id()).
				Save(ctx)
			Expect(err).NotTo(HaveOccurred())

			err = EntClient.Behavior.DeleteOne(b).Exec(ctx)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Describe("Instruct", func() {
		It("creates a new instruct record", func() {
			now := time.Now().Add(time.Hour)
			i, err := EntClient.Instruct.Create().
				SetNo("instr-" + types.Id()).
				SetUID("uid-" + types.Id()).
				SetObject("test").
				SetBot("test-bot").
				SetFlag("test-flag").
				SetContent(map[string]any{"cmd": "test"}).
				SetExpireAt(now).
				Save(ctx)
			Expect(err).NotTo(HaveOccurred())
			Expect(i.ID).NotTo(BeZero())

			EntClient.Instruct.DeleteOne(i).Exec(ctx)
		})

		It("retrieves an instruct by ID", func() {
			now := time.Now().Add(time.Hour)
			i, err := EntClient.Instruct.Create().
				SetNo("instr-get-" + types.Id()).
				SetUID("uid-" + types.Id()).
				SetObject("test").
				SetBot("test-bot").
				SetFlag("get-flag").
				SetContent(map[string]any{}).
				SetExpireAt(now).
				Save(ctx)
			Expect(err).NotTo(HaveOccurred())

			got, err := EntClient.Instruct.Get(ctx, i.ID)
			Expect(err).NotTo(HaveOccurred())
			Expect(got.No).To(Equal(i.No))

			EntClient.Instruct.DeleteOne(i).Exec(ctx)
		})

		It("deletes an instruct", func() {
			now := time.Now().Add(time.Hour)
			i, err := EntClient.Instruct.Create().
				SetNo("instr-del-" + types.Id()).
				SetUID("uid-" + types.Id()).
				SetObject("test").
				SetBot("test-bot").
				SetFlag("del-flag").
				SetContent(map[string]any{}).
				SetExpireAt(now).
				Save(ctx)
			Expect(err).NotTo(HaveOccurred())

			err = EntClient.Instruct.DeleteOne(i).Exec(ctx)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Describe("Agent", func() {
		It("creates a new agent", func() {
			a, err := EntClient.Agent.Create().
				SetUID("uid-" + types.Id()).
				SetTopic("agent-topic").
				SetHostid("host-1").
				SetHostname("agent-host").
				SetLastOnlineAt(time.Now()).
				Save(ctx)
			Expect(err).NotTo(HaveOccurred())
			Expect(a.ID).NotTo(BeZero())

			EntClient.Agent.DeleteOne(a).Exec(ctx)
		})

		It("retrieves an agent by ID", func() {
			a, err := EntClient.Agent.Create().
				SetUID("uid-" + types.Id()).
				SetTopic("agent-get").
				SetHostid("host-get").
				SetHostname("agent-get-host").
				SetLastOnlineAt(time.Now()).
				Save(ctx)
			Expect(err).NotTo(HaveOccurred())

			got, err := EntClient.Agent.Get(ctx, a.ID)
			Expect(err).NotTo(HaveOccurred())
			Expect(got.Hostname).To(Equal("agent-get-host"))

			EntClient.Agent.DeleteOne(a).Exec(ctx)
		})

		It("deletes an agent", func() {
			a, err := EntClient.Agent.Create().
				SetUID("uid-" + types.Id()).
				SetTopic("agent-del").
				SetHostid("host-del").
				SetHostname("agent-del-host").
				SetLastOnlineAt(time.Now()).
				Save(ctx)
			Expect(err).NotTo(HaveOccurred())

			err = EntClient.Agent.DeleteOne(a).Exec(ctx)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Describe("Transaction Support", func() {
		It("commits multiple operations in a single transaction", func() {
			tx, err := EntClient.Tx(ctx)
			Expect(err).NotTo(HaveOccurred())

			u, err := tx.User.Create().SetFlag("tx-user-" + types.Id()).SetName("Tx User").Save(ctx)
			Expect(err).NotTo(HaveOccurred())

			b, err := tx.Bot.Create().SetName("tx-bot-" + types.Id()).Save(ctx)
			Expect(err).NotTo(HaveOccurred())

			err = tx.Commit()
			Expect(err).NotTo(HaveOccurred())

			gotUser, err := EntClient.User.Get(ctx, u.ID)
			Expect(err).NotTo(HaveOccurred())
			Expect(gotUser.Name).To(Equal("Tx User"))

			gotBot, err := EntClient.Bot.Get(ctx, b.ID)
			Expect(err).NotTo(HaveOccurred())
			Expect(gotBot.Name).To(Equal(b.Name))

			err = EntClient.User.DeleteOne(u).Exec(ctx)
			Expect(err).NotTo(HaveOccurred())
			err = EntClient.Bot.DeleteOne(b).Exec(ctx)
			Expect(err).NotTo(HaveOccurred())
		})

		It("rolls back all operations on failure", func() {
			tx, err := EntClient.Tx(ctx)
			Expect(err).NotTo(HaveOccurred())

			u, err := tx.User.Create().SetFlag("rollback-user-" + types.Id()).SetName("Rollback").Save(ctx)
			Expect(err).NotTo(HaveOccurred())

			err = tx.Rollback()
			Expect(err).NotTo(HaveOccurred())

			_, err = EntClient.User.Get(ctx, u.ID)
			Expect(err).To(HaveOccurred())
		})
	})
})
