package partials

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"

	"github.com/flowline-io/flowbot/internal/store/ent/gen"
	"github.com/flowline-io/flowbot/internal/store/ent/gen/notificationrecord"
)

func TestNormalizeNotifyHistoryGroup(t *testing.T) {
	tests := []struct {
		name  string
		group string
		want  string
	}{
		{name: "channel", group: "channel", want: "channel"},
		{name: "rule mixed case", group: "Rule", want: "rule"},
		{name: "unread", group: "unread", want: "unread"},
		{name: "empty defaults flat", group: "", want: ""},
		{name: "unknown ignored", group: "status", want: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NormalizeNotifyHistoryGroup(tt.group); got != tt.want {
				t.Errorf("NormalizeNotifyHistoryGroup(%q) = %q, want %q", tt.group, got, tt.want)
			}
		})
	}
}

func TestNotifyHistoryListURL(t *testing.T) {
	tests := []struct {
		name    string
		group   string
		channel string
		ruleID  string
		cursor  string
		want    string
	}{
		{name: "no filters", want: "/service/web/notifications/list"},
		{
			name:    "group channel with cursor",
			group:   "channel",
			channel: "slack",
			cursor:  "42",
			want:    "/service/web/notifications/list?channel=slack&cursor=42&group=channel",
		},
		{
			name:  "unread group",
			group: "unread",
			want:  "/service/web/notifications/list?group=unread",
		},
		{
			name:   "rule filter",
			group:  "rule",
			ruleID: "night_mute",
			want:   "/service/web/notifications/list?group=rule&rule_id=night_mute",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NotifyHistoryListURL(tt.group, tt.channel, tt.ruleID, tt.cursor)
			if got != tt.want {
				t.Errorf("NotifyHistoryListURL() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestNotifyJumpURLs(t *testing.T) {
	tests := []struct {
		name string
		fn   func() string
		want string
	}{
		{
			name: "channel jump",
			fn:   func() string { return NotifyChannelJumpURL("ops-slack") },
			want: "/service/web/notifications?channel=ops-slack&tab=channels",
		},
		{
			name: "rule jump",
			fn:   func() string { return NotifyRuleJumpURL("night_mute") },
			want: "/service/web/notifications?rule_id=night_mute&tab=rules",
		},
		{
			name: "mark read",
			fn:   func() string { return NotifyRecordReadURL(7) },
			want: "/service/web/notifications/7/read",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.fn(); got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestNotifyRecordRowClass(t *testing.T) {
	tests := []struct {
		name   string
		status string
		want   string
	}{
		{name: "failed highlighted", status: "failed", want: "bg-error/5 border-l-4 border-error"},
		{name: "success plain", status: "success", want: ""},
		{name: "dropped plain", status: "dropped", want: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NotifyRecordRowClass(tt.status); got != tt.want {
				t.Errorf("NotifyRecordRowClass(%q) = %q, want %q", tt.status, got, tt.want)
			}
		})
	}
}

func TestGroupNotificationRecords(t *testing.T) {
	recs := []*gen.NotificationRecord{
		{ID: 3, Channel: "slack", RuleID: "r1", Status: notificationrecord.StatusFailed, CreatedAt: time.Now()},
		{ID: 2, Channel: "ntfy", RuleID: "", Status: notificationrecord.StatusSuccess, CreatedAt: time.Now()},
		{ID: 1, Channel: "slack", RuleID: "r1", Status: notificationrecord.StatusSuccess, CreatedAt: time.Now()},
	}
	tests := []struct {
		name      string
		group     string
		wantKeys  []string
		wantLens  []int
		wantLabel string
	}{
		{name: "flat single group", group: "", wantKeys: []string{""}, wantLens: []int{3}},
		{name: "by channel", group: "channel", wantKeys: []string{"slack", "ntfy"}, wantLens: []int{2, 1}},
		{name: "by rule with empty", group: "rule", wantKeys: []string{"r1", ""}, wantLens: []int{2, 1}, wantLabel: "No rule"},
		{name: "unread flat", group: "unread", wantKeys: []string{""}, wantLens: []int{3}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GroupNotificationRecords(tt.group, recs)
			if len(got) != len(tt.wantKeys) {
				t.Fatalf("len = %d, want %d", len(got), len(tt.wantKeys))
			}
			for i := range tt.wantKeys {
				if got[i].Key != tt.wantKeys[i] {
					t.Errorf("key[%d] = %q, want %q", i, got[i].Key, tt.wantKeys[i])
				}
				if len(got[i].Records) != tt.wantLens[i] {
					t.Errorf("len[%d] = %d, want %d", i, len(got[i].Records), tt.wantLens[i])
				}
			}
			if tt.wantLabel != "" {
				found := false
				for _, g := range got {
					if g.Label == tt.wantLabel {
						found = true
					}
				}
				if !found {
					t.Errorf("missing label %q", tt.wantLabel)
				}
			}
		})
	}
}

func TestNotificationsTableRuleFilterSelected(t *testing.T) {
	tests := []struct {
		name       string
		ruleID     string
		wantSelect string
		wantAbsent []string
	}{
		{
			name:       "selects matching rule only",
			ruleID:     "b",
			wantSelect: `<option value="b" selected>`,
			wantAbsent: []string{`<option value="c" selected>`, `<option value="a" selected>`},
		},
		{
			name:       "all rules when empty",
			ruleID:     "",
			wantSelect: `<option value="" selected>`,
			wantAbsent: []string{`<option value="b" selected>`},
		},
		{
			name:       "selects last facet without sticking",
			ruleID:     "c",
			wantSelect: `<option value="c" selected>`,
			wantAbsent: []string{`<option value="b" selected>`},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			err := NotificationsTable(NotificationHistoryParams{
				Group:   "rule",
				RuleID:  tt.ruleID,
				RuleIDs: []string{"d", "e", "b", "a", "c"},
			}).Render(context.Background(), &buf)
			if err != nil {
				t.Fatalf("Render() error = %v", err)
			}
			body := buf.String()
			if !strings.Contains(body, tt.wantSelect) {
				t.Fatalf("missing %q in:\n%s", tt.wantSelect, body)
			}
			for _, absent := range tt.wantAbsent {
				if strings.Contains(body, absent) {
					t.Fatalf("unexpected %q in:\n%s", absent, body)
				}
			}
			if strings.Count(body, " selected>") != 1 {
				t.Fatalf("want exactly one selected option, body:\n%s", body)
			}
		})
	}
}
