package chatagent

import (
	"context"
	"testing"

	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/internal/store/ent/gen"
	"github.com/flowline-io/flowbot/internal/store/ent/schema"
)

func TestTruncateInboxLabel(t *testing.T) {
	t.Parallel()
	tests := []struct {
		in   string
		max  int
		want string
	}{
		{in: "short", max: 48, want: "short"},
		{in: "  padded  ", max: 48, want: "padded"},
		{in: "abcdefghijkl", max: 12, want: "abcdefghijkl"},
		{in: "abcdefghijklm", max: 12, want: "abcdefghijk…"},
		{in: "你好世界测试标题很长", max: 4, want: "你好世…"},
		{in: "", max: 12, want: ""},
	}
	for _, tt := range tests {
		if got := truncateInboxLabel(tt.in, tt.max); got != tt.want {
			t.Fatalf("truncateInboxLabel(%q, %d) = %q, want %q", tt.in, tt.max, got, tt.want)
		}
	}
}

func TestSessionInboxSourceLabelWithoutDB(t *testing.T) {
	t.Parallel()
	got := sessionInboxSourceLabel(context.Background(), "session-id-abcdefgh")
	want := truncateInboxLabel("session-id-abcdefgh", 12)
	if got != want {
		t.Fatalf("sessionInboxSourceLabel without DB = %q, want %q", got, want)
	}
}

// TestApprovalNotifyDrainsBeforeDatabaseRestore ensures fire-and-forget approval
// notify work finishes before store.Database is swapped (CI -race regression).
func TestApprovalNotifyDrainsBeforeDatabaseRestore(t *testing.T) {
	installSQLiteTestDatabase(t)

	requireSession(t, &gen.ChatSession{
		Flag:  "sess-approval-notify",
		UID:   "user-1",
		State: int(schema.ChatSessionActive),
	})

	gate := NewConfirmGate("sess-approval-notify", NewChannelPublisher(1), nil)
	gate.notifyApprovalPending("confirm-1", "command: ls")
	markApprovalInboxRead("sess-approval-notify", "confirm-1")
	WaitApprovalNotifyForTest()
}

func requireSession(t *testing.T, sess *gen.ChatSession) {
	t.Helper()
	if err := store.ChatStoreFromDB().CreateChatSession(context.Background(), sess); err != nil {
		t.Fatalf("CreateChatSession: %v", err)
	}
}
