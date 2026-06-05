package modules_test

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/rs/zerolog"

	"github.com/flowline-io/flowbot/pkg/plugin"
	"github.com/flowline-io/flowbot/pkg/plugin/adapter"
	"github.com/flowline-io/flowbot/pkg/plugin/manager"
	"github.com/flowline-io/flowbot/pkg/types"
)

func TestPluginSystem(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Plugin System Suite")
}

// stubRunner implements plugin.Runner for BDD tests.
type stubRunner struct {
	callResult json.RawMessage
	callError  error
}

func (s *stubRunner) Load(_ context.Context, m *plugin.Manifest) (*plugin.PluginInfo, error) {
	return &plugin.PluginInfo{Name: m.Name, Version: m.Version}, nil
}
func (s *stubRunner) Start(_ context.Context, _ json.RawMessage) error { return nil }
func (s *stubRunner) Stop(_ context.Context) error                     { return nil }
func (s *stubRunner) Call(_ context.Context, _ string, _ json.RawMessage) (json.RawMessage, error) {
	if s.callError != nil {
		return nil, s.callError
	}
	return s.callResult, nil
}
func (s *stubRunner) Health(_ context.Context) (*plugin.HealthStatus, error) {
	return &plugin.HealthStatus{Ready: true}, nil
}

var _ = Describe("Plugin System", func() {
	var runner *stubRunner

	BeforeEach(func() {
		runner = &stubRunner{}
	})

	Describe("Module Adapter", func() {
		It("responds to commands through the adapter", func() {
			runner.callResult = json.RawMessage(`{"_type": "KVMsg", "text": "hello from plugin"}`)

			m := &plugin.Manifest{Name: "test", Runtime: plugin.RuntimeGRPC}
			a := adapter.NewModuleAdapter(m, runner)

			payload, err := a.Command(types.Context{}, "hello")
			Expect(err).NotTo(HaveOccurred())
			result := payload.Convert()
			Expect(result).To(HaveKey("text"))
		})

		It("handles plugin errors gracefully", func() {
			runner.callError = fmt.Errorf("plugin error")

			m := &plugin.Manifest{Name: "test", Runtime: plugin.RuntimeGRPC}
			a := adapter.NewModuleAdapter(m, runner)

			_, err := a.Command(types.Context{}, "hello")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("plugin error"))
		})
	})

	Describe("Plugin Manager", func() {
		It("lists empty when no plugins loaded", func() {
			mgr := manager.NewPluginManager(manager.DefaultPluginConfig(), zerolog.Nop())
			Expect(mgr.List()).To(BeEmpty())
		})

		It("rejects unload of unknown plugin", func() {
			mgr := manager.NewPluginManager(manager.DefaultPluginConfig(), zerolog.Nop())
			err := mgr.UnloadPlugin(context.Background(), "nonexistent")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("not found"))
		})

		It("rejects reload of unknown plugin", func() {
			mgr := manager.NewPluginManager(manager.DefaultPluginConfig(), zerolog.Nop())
			err := mgr.ReloadPlugin(context.Background(), "nonexistent", &plugin.Manifest{}, nil)
			Expect(err).To(HaveOccurred())
		})
	})
})
