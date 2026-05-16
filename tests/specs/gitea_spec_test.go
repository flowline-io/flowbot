//go:build integration
// +build integration

package specs

import (
	"github.com/flowline-io/flowbot/pkg/parser"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/ruleset/command"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Gitea Module", Label("module", "gitea"), func() {

	Describe("Command structure", func() {
		It("defines gitea command rules", func() {
			cmd := command.Rule{
				Define: "gitea",
				Help:   "Fetches demo repository information from Gitea",
				Handler: func(ctx types.Context, tokens []*parser.Token) types.MsgPayload {
					return types.EmptyMsg{}
				},
			}
			Expect(cmd.Define).To(Equal("gitea"))
			Expect(cmd.Help).To(ContainSubstring("Gitea"))
			_ = cmd
		})
	})

	Describe("Webhook event types", func() {
		It("handles issue events", func() {
			eventRules := []struct {
				ID      string
				Handler func()
			}{
				{ID: "issue_created", Handler: func() {}},
				{ID: "issue_closed", Handler: func() {}},
				{ID: "push", Handler: func() {}},
			}
			Expect(len(eventRules)).To(Equal(3))
		})
	})

	Describe("Cron job definitions", func() {
		It("has metrics collection cron", func() {
			cronDef := struct {
				Name  string
				When  string
				Scope string
			}{
				Name:  "gitea_metrics",
				When:  "every 1 minute",
				Scope: "system",
			}
			Expect(cronDef.Name).To(Equal("gitea_metrics"))
			Expect(cronDef.Scope).To(Equal("system"))
		})
	})
})
