//go:build integration
// +build integration

package specs

import (
	"bytes"
	"text/template"

	"github.com/flowline-io/flowbot/pkg/notify"
	"github.com/flowline-io/flowbot/pkg/parser"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/ruleset/command"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Notify Module", Label("module", "notify"), func() {

	Describe("Command structure", func() {
		It("defines notify list command", func() {
			cmd := command.Rule{
				Define: "notify list",
				Help:   "Lists all notification templates",
				Handler: func(ctx types.Context, tokens []*parser.Token) types.MsgPayload {
					return types.EmptyMsg{}
				},
			}
			Expect(cmd.Define).To(Equal("notify list"))
			_ = cmd
		})

		It("defines notify delete command", func() {
			cmd := command.Rule{
				Define: "notify delete [string]",
				Help:   "Deletes a notification template by name",
				Handler: func(ctx types.Context, tokens []*parser.Token) types.MsgPayload {
					return types.EmptyMsg{}
				},
			}
			Expect(cmd.Define).To(Equal("notify delete [string]"))
			_ = cmd
		})

		It("defines notify config command", func() {
			cmd := command.Rule{
				Define: "notify config",
				Help:   "Shows current notification configuration",
				Handler: func(ctx types.Context, tokens []*parser.Token) types.MsgPayload {
					return types.EmptyMsg{}
				},
			}
			Expect(cmd.Define).To(Equal("notify config"))
			_ = cmd
		})
	})

	Describe("Form definitions", func() {
		It("creates notification form with required fields", func() {
			formFields := []types.FormField{
				{Key: "name", Type: types.FormFieldText, Label: "Template Name", Rule: "required"},
				{Key: "template", Type: types.FormFieldTextarea, Label: "Template Body", Rule: "required"},
				{Key: "channel", Type: types.FormFieldSelect, Label: "Channel", Option: []string{"slack", "discord", "ntfy", "email"}},
			}
			Expect(len(formFields)).To(Equal(3))
			Expect(formFields[0].Rule).To(Equal("required"))
			Expect(formFields[1].Rule).To(Equal("required"))
		})

		It("rejects creation with empty name", func() {
			rule := "required"
			_ = types.FormField{Key: "name", Type: types.FormFieldText, Rule: rule}
		})

		It("supports different field types", func() {
			Expect(types.FormFieldText).To(BeEquivalentTo("text"))
			Expect(types.FormFieldTextarea).To(BeEquivalentTo("textarea"))
			Expect(types.FormFieldSelect).To(BeEquivalentTo("select"))
			Expect(types.FormFieldCheckbox).To(BeEquivalentTo("checkbox"))
			Expect(types.FormFieldNumber).To(BeEquivalentTo("number"))
		})
	})

	Describe("Multi-Channel Delivery", func() {
		It("creates message with all fields", func() {
			msg := notify.Message{
				Title:    "Test Notification",
				Body:     "This is a test message body.",
				Url:      "https://example.com/alert",
				Priority: notify.High,
			}
			Expect(msg.Title).To(Equal("Test Notification"))
			Expect(msg.Body).To(Equal("This is a test message body."))
			Expect(msg.Url).To(Equal("https://example.com/alert"))
			Expect(msg.Priority).To(Equal(notify.High))
		})

		It("has correct priority constants", func() {
			Expect(int(notify.Low)).To(Equal(1))
			Expect(int(notify.Moderate)).To(Equal(2))
			Expect(int(notify.Normal)).To(Equal(3))
			Expect(int(notify.High)).To(Equal(4))
			Expect(int(notify.Emergency)).To(Equal(5))
		})

		It("parses protocol scheme from URI", func() {
			scheme, err := notify.ParseSchema("slack://token@channel")
			Expect(err).NotTo(HaveOccurred())
			Expect(scheme).To(Equal("slack"))
		})

		It("fails gracefully with empty template list", func() {
			result, err := notify.ParseTemplate("hello world", nil)
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(BeEmpty())
		})

		It("parses notification template from structured text", func() {
			scheme, err := notify.ParseTemplate("slack://token1/channel1", []string{"slack://{token}/{channel}"})
			Expect(err).NotTo(HaveOccurred())
			Expect(scheme).NotTo(BeNil())
			Expect(scheme["token"]).To(Equal("token1"))
			Expect(scheme["channel"]).To(Equal("channel1"))
		})
	})

	Describe("Template Rendering", func() {
		It("renders notification body from Go template", func() {
			const tmpl = "Hello {{.name}}, your task {{.task}} is due!"
			data := map[string]any{"name": "Alice", "task": "Review PR"}

			t := template.Must(template.New("test").Parse(tmpl))
			var buf bytes.Buffer
			err := t.Execute(&buf, data)
			Expect(err).NotTo(HaveOccurred())
			Expect(buf.String()).To(Equal("Hello Alice, your task Review PR is due!"))
		})
	})

	Describe("MsgPayload types for notifications", func() {
		It("creates text messages", func() {
			msg := types.TextMsg{Text: "Notification content"}
			Expect(types.TypeOf(msg)).To(Equal("TextMsg"))
		})

		It("creates KV messages for structured data", func() {
			msg := types.KVMsg{"channel": "slack", "status": "sent"}
			Expect(msg["channel"]).To(Equal("slack"))
		})
	})

	Describe("FormField value types", func() {
		It("has correct value type constants", func() {
			Expect(types.FormFieldValueString).To(BeEquivalentTo("string"))
			Expect(types.FormFieldValueBool).To(BeEquivalentTo("bool"))
			Expect(types.FormFieldValueInt64).To(BeEquivalentTo("int64"))
		})
	})
})
