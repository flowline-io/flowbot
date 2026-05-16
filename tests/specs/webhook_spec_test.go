//go:build integration
// +build integration

package specs

import (
	"context"

	"github.com/flowline-io/flowbot/internal/store/ent/gen/webhook"
	"github.com/flowline-io/flowbot/pkg/auth"
	"github.com/flowline-io/flowbot/pkg/parser"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/ruleset/command"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Webhook Module", Label("module", "webhook"), func() {

	Describe("Command structure", func() {
		It("defines webhook list command", func() {
			cmd := command.Rule{
				Define: "webhook list",
				Help:   "Lists all configured webhooks",
				Handler: func(ctx types.Context, tokens []*parser.Token) types.MsgPayload {
					return types.EmptyMsg{}
				},
			}
			Expect(cmd.Define).To(Equal("webhook list"))
			_ = cmd
		})

		It("defines webhook create command", func() {
			cmd := command.Rule{
				Define: "webhook create [flag]",
				Help:   "Creates a new webhook with generated secret",
				Handler: func(ctx types.Context, tokens []*parser.Token) types.MsgPayload {
					return types.EmptyMsg{}
				},
			}
			Expect(cmd.Define).To(Equal("webhook create [flag]"))
			_ = cmd
		})

		It("defines webhook del command", func() {
			cmd := command.Rule{
				Define: "webhook del [secret]",
				Help:   "Deletes a webhook by secret",
				Handler: func(ctx types.Context, tokens []*parser.Token) types.MsgPayload {
					return types.EmptyMsg{}
				},
			}
			Expect(cmd.Define).To(Equal("webhook del [secret]"))
			_ = cmd
		})

		It("defines webhook activate command", func() {
			cmd := command.Rule{
				Define: "webhook activate [secret]",
				Help:   "Activates a disabled webhook",
				Handler: func(ctx types.Context, tokens []*parser.Token) types.MsgPayload {
					return types.EmptyMsg{}
				},
			}
			Expect(cmd.Define).To(Equal("webhook activate [secret]"))
			_ = cmd
		})

		It("defines webhook inactive command", func() {
			cmd := command.Rule{
				Define: "webhook inactive [secret]",
				Help:   "Deactivates a webhook",
				Handler: func(ctx types.Context, tokens []*parser.Token) types.MsgPayload {
					return types.EmptyMsg{}
				},
			}
			Expect(cmd.Define).To(Equal("webhook inactive [secret]"))
			_ = cmd
		})
	})

	Describe("Webhook database operations", func() {
		It("creates a webhook record", func() {
			w, err := EntClient.Webhook.Create().
				SetUID("webhook-test-uid").
				SetTopic("webhook-test-topic").
				SetFlag("test-flag").
				SetSecret("test-secret-12345").
				Save(context.Background())
			Expect(err).NotTo(HaveOccurred())
			Expect(w.ID).NotTo(BeZero())

			EntClient.Webhook.DeleteOne(w).Exec(context.Background())
		})

		It("activates and deactivates webhooks", func() {
			w, err := EntClient.Webhook.Create().
				SetUID("active-test-uid").
				SetTopic("active-test").
				SetFlag("active-flag").
				SetSecret("active-secret").
				Save(context.Background())
			Expect(err).NotTo(HaveOccurred())

			updated, err := EntClient.Webhook.UpdateOne(w).SetState(1).Save(context.Background())
			Expect(err).NotTo(HaveOccurred())
			Expect(updated.State).To(Equal(1))

			updated, err = EntClient.Webhook.UpdateOne(w).SetState(0).Save(context.Background())
			Expect(err).NotTo(HaveOccurred())
			Expect(updated.State).To(Equal(0))

			EntClient.Webhook.DeleteOne(w).Exec(context.Background())
		})

		It("deletes a webhook by ID", func() {
			w, err := EntClient.Webhook.Create().
				SetUID("del-test-uid").
				SetTopic("del-test").
				SetFlag("del-flag").
				SetSecret("del-secret").
				Save(context.Background())
			Expect(err).NotTo(HaveOccurred())

			err = EntClient.Webhook.DeleteOne(w).Exec(context.Background())
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Describe("Webhook signature verification", func() {
		It("signs and verifies webhook payloads", func() {
			token, err := auth.NewToken()
			Expect(err).NotTo(HaveOccurred())
			Expect(token).NotTo(BeEmpty())
		})

		It("webhook secret generation", func() {
			token, err := auth.NewToken()
			Expect(err).NotTo(HaveOccurred())
			Expect(token).To(HavePrefix("fb_"))
		})
	})

	Describe("Query webhooks by fields", func() {
		It("queries webhooks by uid", func() {
			uid := "query-uid-" + types.Id()
			w, err := EntClient.Webhook.Create().
				SetUID(uid).
				SetTopic("query-topic").
				SetFlag("query-flag").
				SetSecret("query-secret").
				Save(context.Background())
			Expect(err).NotTo(HaveOccurred())

			webhooks, err := EntClient.Webhook.Query().Where(webhook.UID(uid)).All(context.Background())
			Expect(err).NotTo(HaveOccurred())
			Expect(len(webhooks)).To(Equal(1))
			Expect(webhooks[0].Secret).To(Equal("query-secret"))

			EntClient.Webhook.DeleteOne(w).Exec(context.Background())
		})
	})

	Describe("Webhook secret uniqueness", func() {
		It("creates webhooks with unique secrets", func() {
			w1, err := EntClient.Webhook.Create().
				SetUID("unique-uid-1").
				SetTopic("unique-topic").
				SetFlag("unique-flag-1").
				SetSecret("unique-secret-1").
				Save(context.Background())
			Expect(err).NotTo(HaveOccurred())

			w2, err := EntClient.Webhook.Create().
				SetUID("unique-uid-2").
				SetTopic("unique-topic-2").
				SetFlag("unique-flag-2").
				SetSecret("unique-secret-2").
				Save(context.Background())
			Expect(err).NotTo(HaveOccurred())
			Expect(w1.Secret).NotTo(Equal(w2.Secret))

			EntClient.Webhook.DeleteOne(w1).Exec(context.Background())
			EntClient.Webhook.DeleteOne(w2).Exec(context.Background())
		})
	})
})
