package partials

import (
	"net/url"
	"strconv"
	"strings"

	"github.com/flowline-io/flowbot/internal/store/ent/gen"
)

// NotificationHistoryParams holds History tab list state for the notifications table.
type NotificationHistoryParams struct {
	Records    []*gen.NotificationRecord
	NextCursor string
	Group      string // "", "channel", "rule", or "unread"
	Channel    string
	RuleID     string
	Channels   []string
	RuleIDs    []string
}

// NotificationGroup is a labeled section of delivery records for History grouping.
type NotificationGroup struct {
	Key     string
	Label   string
	Records []*gen.NotificationRecord
}

// NormalizeNotifyHistoryGroup returns a known History group mode or empty (flat list).
func NormalizeNotifyHistoryGroup(group string) string {
	switch strings.ToLower(strings.TrimSpace(group)) {
	case "channel", "rule", "unread":
		return strings.ToLower(strings.TrimSpace(group))
	default:
		return ""
	}
}

// NotifyHistoryListURL builds the History fragment URL preserving group and filters.
func NotifyHistoryListURL(group, channel, ruleID, cursor string) string {
	q := url.Values{}
	if g := NormalizeNotifyHistoryGroup(group); g != "" {
		q.Set("group", g)
	}
	if channel != "" {
		q.Set("channel", channel)
	}
	if ruleID != "" {
		q.Set("rule_id", ruleID)
	}
	if cursor != "" {
		q.Set("cursor", cursor)
	}
	path := "/service/web/notifications/list"
	if encoded := q.Encode(); encoded != "" {
		return path + "?" + encoded
	}
	return path
}

// NotifyChannelJumpURL links History rows to the Channels tab focused on a channel name.
func NotifyChannelJumpURL(channel string) string {
	q := url.Values{}
	q.Set("tab", "channels")
	q.Set("channel", channel)
	return "/service/web/notifications?" + q.Encode()
}

// NotifyRuleJumpURL links History rows to the Rules tab focused on a rule id.
func NotifyRuleJumpURL(ruleID string) string {
	q := url.Values{}
	q.Set("tab", "rules")
	q.Set("rule_id", ruleID)
	return "/service/web/notifications?" + q.Encode()
}

// NotifyRecordReadURL is the mark-read endpoint for a delivery record.
func NotifyRecordReadURL(id int64) string {
	return "/service/web/notifications/" + strconv.FormatInt(id, 10) + "/read"
}

// NotifyRecordRowClass returns row classes; failed deliveries are highlighted.
func NotifyRecordRowClass(status string) string {
	if status == "failed" {
		return "bg-error/5 border-l-4 border-error"
	}
	return ""
}

// GroupNotificationRecords builds section headers for channel/rule History grouping.
// Unread and flat modes return a single unlabeled group.
func GroupNotificationRecords(group string, records []*gen.NotificationRecord) []NotificationGroup {
	group = NormalizeNotifyHistoryGroup(group)
	if group != "channel" && group != "rule" {
		return []NotificationGroup{{Key: "", Label: "", Records: records}}
	}

	order := make([]string, 0)
	labels := make(map[string]string)
	buckets := make(map[string][]*gen.NotificationRecord)
	for _, r := range records {
		key := r.Channel
		label := r.Channel
		if group == "rule" {
			key = r.RuleID
			if key == "" {
				label = "No rule"
			} else {
				label = key
			}
		}
		if _, ok := buckets[key]; !ok {
			order = append(order, key)
			labels[key] = label
		}
		buckets[key] = append(buckets[key], r)
	}

	out := make([]NotificationGroup, 0, len(order))
	for _, key := range order {
		out = append(out, NotificationGroup{Key: key, Label: labels[key], Records: buckets[key]})
	}
	return out
}
