package partials

import (
	"testing"

	"github.com/flowline-io/flowbot/pkg/types/model"
)

func TestRuleActionSummary(t *testing.T) {
	tests := []struct {
		name string
		item model.NotifyRule
		want string
	}{
		{
			name: "mute with condition",
			item: model.NotifyRule{Action: "mute", Condition: "time.hour >= 23 || time.hour < 8"},
			want: "time.hour >= 23 || time.hour < 8",
		},
		{
			name: "throttle with window and limit",
			item: model.NotifyRule{Action: "throttle", ParamsJSON: `{"window":"5m","limit":1}`},
			want: "window 5m · limit 1",
		},
		{
			name: "aggregate with digest and delayed",
			item: model.NotifyRule{Action: "aggregate", ParamsJSON: `{"window":"15m","digest_template_id":"download.digest","delayed_send":true}`},
			want: "window 15m · digest download.digest · delayed",
		},
		{
			name: "drop has no summary",
			item: model.NotifyRule{Action: "drop"},
			want: "",
		},
		{
			name: "mute empty condition",
			item: model.NotifyRule{Action: "mute", Condition: ""},
			want: "",
		},
		{
			name: "throttle empty params",
			item: model.NotifyRule{Action: "throttle", ParamsJSON: ""},
			want: "",
		},
		{
			name: "aggregate window only",
			item: model.NotifyRule{Action: "aggregate", ParamsJSON: `{"window":"15m"}`},
			want: "window 15m",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ruleActionSummary(tt.item)
			if got != tt.want {
				t.Errorf("ruleActionSummary() = %q, want %q", got, tt.want)
			}
		})
	}
}
