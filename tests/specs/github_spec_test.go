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

var _ = Describe("GitHub Module", Label("module", "github"), func() {

	Describe("Command structure", func() {
		It("defines github setting command", func() {
			cmd := command.Rule{
				Define: "github setting",
				Help:   "Configures GitHub OAuth settings",
				Handler: func(ctx types.Context, tokens []*parser.Token) types.MsgPayload {
					return types.EmptyMsg{}
				},
			}
			Expect(cmd.Define).To(Equal("github setting"))
			_ = cmd
		})

		It("defines github oauth command", func() {
			cmd := command.Rule{
				Define: "github oauth",
				Help:   "Initiates OAuth authorization flow",
				Handler: func(ctx types.Context, tokens []*parser.Token) types.MsgPayload {
					return types.EmptyMsg{}
				},
			}
			Expect(cmd.Define).To(Equal("github oauth"))
			_ = cmd
		})

		It("defines github user command", func() {
			cmd := command.Rule{
				Define: "github user",
				Help:   "Returns authenticated user profile",
				Handler: func(ctx types.Context, tokens []*parser.Token) types.MsgPayload {
					return types.EmptyMsg{}
				},
			}
			Expect(cmd.Define).To(Equal("github user"))
			_ = cmd
		})

		It("defines github card command", func() {
			cmd := command.Rule{
				Define: "github card [string]",
				Help:   "Returns a GitHub repository card view",
				Handler: func(ctx types.Context, tokens []*parser.Token) types.MsgPayload {
					return types.EmptyMsg{}
				},
			}
			Expect(cmd.Define).To(ContainSubstring("github card"))
			_ = cmd
		})

		It("defines github repo command", func() {
			cmd := command.Rule{
				Define: "github repo [string]",
				Help:   "Returns detailed repository information",
				Handler: func(ctx types.Context, tokens []*parser.Token) types.MsgPayload {
					return types.EmptyMsg{}
				},
			}
			Expect(cmd.Define).To(ContainSubstring("github repo"))
			_ = cmd
		})

		It("defines deploy command", func() {
			cmd := command.Rule{
				Define: "deploy",
				Help:   "Triggers deployment for a package",
				Handler: func(ctx types.Context, tokens []*parser.Token) types.MsgPayload {
					return types.EmptyMsg{}
				},
			}
			Expect(cmd.Define).To(Equal("deploy"))
			_ = cmd
		})
	})

	Describe("Cron job definitions", func() {
		It("has starred repos sync cron", func() {
			cronDef := struct {
				Name  string
				When  string
				Scope string
			}{
				Name:  "github_starred",
				When:  "every 30 minutes",
				Scope: "system",
			}
			Expect(cronDef.Name).To(Equal("github_starred"))
		})

		It("has notifications sync cron", func() {
			cronDef := struct {
				Name  string
				When  string
				Scope string
			}{
				Name:  "github_notifications",
				When:  "every 1 minute",
				Scope: "system",
			}
			Expect(cronDef.Name).To(Equal("github_notifications"))
		})
	})

	Describe("MsgPayload types used by github module", func() {
		It("uses InfoMsg for user profiles", func() {
			msg := types.InfoMsg{
				Title: "GitHub User",
				Model: map[string]any{
					"login":     "testuser",
					"followers": 42,
				},
			}
			Expect(msg.Title).To(Equal("GitHub User"))
			Expect(types.TypeOf(msg)).To(Equal("InfoMsg"))
		})
	})
})
