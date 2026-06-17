//go:build integration
// +build integration

package specs

import (
	"context"

	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/internal/store/ent/gen"
	"github.com/flowline-io/flowbot/pkg/types"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("AgentSubagent Store", Label("database", "chatagent", "integration"), func() {
	ctx := context.Background()

	newSubagent := func() *gen.AgentSubagent {
		flag := "subagent-" + types.Id()
		return &gen.AgentSubagent{
			Flag:         flag,
			Name:         flag,
			Description:  "A specialized subagent for tests",
			SystemPrompt: "You are a test subagent.",
			Tools:        []string{"read_file", "run_terminal"},
			Source:       "test",
			Enabled:      true,
		}
	}

	It("creates and retrieves a subagent by flag", func() {
		s := newSubagent()
		Expect(store.Database.CreateAgentSubagent(ctx, s)).To(Succeed())
		Expect(s.ID).NotTo(BeZero())
		DeferCleanup(func() { _ = store.Database.DeleteAgentSubagent(ctx, s.Flag) })

		got, err := store.Database.GetAgentSubagentByFlag(ctx, s.Flag)
		Expect(err).NotTo(HaveOccurred())
		Expect(got.Name).To(Equal(s.Name))
		Expect(got.Tools).To(ConsistOf("read_file", "run_terminal"))
		Expect(got.Enabled).To(BeTrue())
	})

	It("retrieves a subagent by name", func() {
		s := newSubagent()
		Expect(store.Database.CreateAgentSubagent(ctx, s)).To(Succeed())
		DeferCleanup(func() { _ = store.Database.DeleteAgentSubagent(ctx, s.Flag) })

		got, err := store.Database.GetAgentSubagentByName(ctx, s.Name)
		Expect(err).NotTo(HaveOccurred())
		Expect(got.Flag).To(Equal(s.Flag))
	})

	It("lists only enabled subagents when enabledOnly is set", func() {
		enabled := newSubagent()
		Expect(store.Database.CreateAgentSubagent(ctx, enabled)).To(Succeed())
		DeferCleanup(func() { _ = store.Database.DeleteAgentSubagent(ctx, enabled.Flag) })

		disabled := newSubagent()
		disabled.Enabled = false
		Expect(store.Database.CreateAgentSubagent(ctx, disabled)).To(Succeed())
		DeferCleanup(func() { _ = store.Database.DeleteAgentSubagent(ctx, disabled.Flag) })

		rows, err := store.Database.ListAgentSubagents(ctx, true)
		Expect(err).NotTo(HaveOccurred())
		flags := make([]string, 0, len(rows))
		for _, r := range rows {
			flags = append(flags, r.Flag)
		}
		Expect(flags).To(ContainElement(enabled.Flag))
		Expect(flags).NotTo(ContainElement(disabled.Flag))
	})

	It("updates subagent fields", func() {
		s := newSubagent()
		Expect(store.Database.CreateAgentSubagent(ctx, s)).To(Succeed())
		DeferCleanup(func() { _ = store.Database.DeleteAgentSubagent(ctx, s.Flag) })

		s.Description = "Updated description"
		s.Model = "gpt-4o"
		Expect(store.Database.UpdateAgentSubagent(ctx, s)).To(Succeed())

		got, err := store.Database.GetAgentSubagentByFlag(ctx, s.Flag)
		Expect(err).NotTo(HaveOccurred())
		Expect(got.Description).To(Equal("Updated description"))
		Expect(got.Model).To(Equal("gpt-4o"))
	})

	It("deletes a subagent and reports not found afterwards", func() {
		s := newSubagent()
		Expect(store.Database.CreateAgentSubagent(ctx, s)).To(Succeed())

		Expect(store.Database.DeleteAgentSubagent(ctx, s.Flag)).To(Succeed())

		_, err := store.Database.GetAgentSubagentByFlag(ctx, s.Flag)
		Expect(err).To(HaveOccurred())
	})

	It("advances the max updated-at watermark after a write", func() {
		before, err := store.Database.GetAgentSubagentsMaxUpdatedAt(ctx)
		Expect(err).NotTo(HaveOccurred())

		s := newSubagent()
		Expect(store.Database.CreateAgentSubagent(ctx, s)).To(Succeed())
		DeferCleanup(func() { _ = store.Database.DeleteAgentSubagent(ctx, s.Flag) })

		after, err := store.Database.GetAgentSubagentsMaxUpdatedAt(ctx)
		Expect(err).NotTo(HaveOccurred())
		Expect(after.Before(before)).To(BeFalse())
	})
})
