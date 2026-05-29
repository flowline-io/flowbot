//go:build e2e

package e2e

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConfigsPage(t *testing.T) {
	tests := []struct {
		name       string
		seedConfig bool
		wantText   string
	}{
		{"empty configs page shows empty state", false, "No configs found"},
		{"seeded configs page shows table", true, "e2e-seed-key"},
		{"page renders new config button", false, "New Config"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.seedConfig {
				seedConfig(t, "e2e", "e2e-topic", "e2e-seed-key", "hello")
				t.Cleanup(func() {
					ResetDB(t)
				})
			}

			page := loginViaCookie(t)
			page.MustNavigate(URL("/configs"))
			wait := page.MustWaitRequestIdle()
			wait()

			body := page.MustElement("body").MustText()
			assert.Contains(t, body, tt.wantText)
		})
	}
}

func TestConfigCreate(t *testing.T) {
	tests := []struct {
		name  string
		uid   string
		topic string
		key   string
		value string
		want  string
	}{
		{"create string config shows in table", "e2e-c1", "topic1", "key1", `"hello-world"`, "key1"},
		{"create numeric config shows in table", "e2e-c2", "topic2", "num-key", `42`, "42"},
		{"create with empty uid shows error", "", "topic3", "key3", `"val"`, "uid"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ResetDB(t)

			page := loginViaCookie(t)
			page.MustNavigate(URL("/configs"))
			wait := page.MustWaitRequestIdle()
			wait()

			page.MustElement(`[data-testid="configs-new"]`).MustClick()
			wait = page.MustWaitRequestIdle()
			wait()

			page.MustElement(`[data-testid="config-uid"]`).MustInput(tt.uid)
			page.MustElement(`[data-testid="config-topic"]`).MustInput(tt.topic)
			page.MustElement(`[data-testid="config-key"]`).MustInput(tt.key)
			page.MustElement(`[data-testid="config-value"]`).MustInput(tt.value)
			page.MustElement(`[data-testid="config-save"]`).MustClick()
			wait = page.MustWaitRequestIdle()
			wait()

			body := page.MustElement("body").MustText()
			assert.Contains(t, body, tt.want)
		})
	}
}

func TestConfigUpdate(t *testing.T) {
	tests := []struct {
		name      string
		newValue  string
		wantValue string
	}{
		{"update string value", `"updated-value"`, "updated-value"},
		{"update numeric value", `99`, "99"},
		{"update empty value to null", `null`, "null"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ResetDB(t)
			seedConfig(t, "e2e-u", "e2e-topic", "e2e-update-key", "original")

			page := loginViaCookie(t)
			page.MustNavigate(URL("/configs"))
			wait := page.MustWaitRequestIdle()
			wait()

			page.MustElement(`[data-testid="config-edit"]`).MustClick()
			wait = page.MustWaitRequestIdle()
			wait()

			el := page.MustElement(`[data-testid="config-value"]`)
			el.MustSelectAllText()
			el.MustInput(tt.newValue)
			page.MustElement(`[data-testid="config-save"]`).MustClick()
			wait = page.MustWaitRequestIdle()
			wait()

			body := page.MustElement("body").MustText()
			assert.Contains(t, body, tt.wantValue)
		})
	}
}

func TestConfigDelete(t *testing.T) {
	tests := []struct {
		name string
	}{
		{"delete removes row from table"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ResetDB(t)
			seedConfig(t, "e2e-d", "e2e-topic", "e2e-delete-key", "to-be-deleted")

			page := loginViaCookie(t)
			page.MustNavigate(URL("/configs"))
			wait := page.MustWaitRequestIdle()
			wait()

			page.MustElement(`[data-testid="config-delete"]`).MustClick()

			wait = page.MustWaitRequestIdle()
			wait()

			body := page.MustElement("body").MustText()
			assert.Contains(t, body, "No configs found")
		})
	}
}
