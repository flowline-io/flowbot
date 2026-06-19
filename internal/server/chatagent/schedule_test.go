package chatagent

import (
	"testing"
	"time"

	"github.com/flowline-io/flowbot/internal/store/ent/schema"
)

func TestValidateScheduleInput(t *testing.T) {
	t.Parallel()
	future := time.Now().UTC().Add(time.Hour)

	tests := []struct {
		name    string
		kind    string
		cron    string
		runAt   *time.Time
		wantErr bool
	}{
		{name: "valid cron", kind: string(schema.ChatScheduledTaskKindCron), cron: "0 9 * * *"},
		{name: "valid once", kind: string(schema.ChatScheduledTaskKindOnce), runAt: &future},
		{name: "cron missing expr", kind: string(schema.ChatScheduledTaskKindCron), cron: "", wantErr: true},
		{name: "once missing run_at", kind: string(schema.ChatScheduledTaskKindOnce), wantErr: true},
		{name: "once past run_at", kind: string(schema.ChatScheduledTaskKindOnce), runAt: new(time.Now().UTC().Add(-time.Minute)), wantErr: true},
		{name: "cron with run_at conflict", kind: string(schema.ChatScheduledTaskKindCron), cron: "0 9 * * *", runAt: &future, wantErr: true},
		{name: "invalid kind", kind: "weekly", cron: "0 9 * * *", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := ValidateScheduleInput(tt.kind, tt.cron, tt.runAt)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("ValidateScheduleInput() error = %v", err)
			}
		})
	}
}

func TestParseRunAt(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		raw     string
		wantErr bool
	}{
		{name: "valid RFC3339", raw: "2026-06-20T09:00:00Z"},
		{name: "empty", raw: "", wantErr: true},
		{name: "invalid", raw: "tomorrow", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := ParseRunAt(tt.raw)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("ParseRunAt() error = %v", err)
			}
		})
	}
}

func TestIsScheduleWriteTool(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		tool string
		want bool
	}{
		{name: "schedule write", tool: scheduleToolName, want: true},
		{name: "update write", tool: updateScheduleToolName, want: true},
		{name: "list read", tool: listScheduleToolName, want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := IsScheduleWriteTool(tt.tool); got != tt.want {
				t.Fatalf("IsScheduleWriteTool(%q) = %v, want %v", tt.tool, got, tt.want)
			}
		})
	}
}

func TestValidScheduledTaskState(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		state string
		want  bool
	}{
		{name: "active", state: string(schema.ChatScheduledTaskStateActive), want: true},
		{name: "paused", state: string(schema.ChatScheduledTaskStatePaused), want: true},
		{name: "failed", state: string(schema.ChatScheduledTaskStateFailed), want: true},
		{name: "unknown", state: "archived", want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := ValidScheduledTaskState(tt.state); got != tt.want {
				t.Fatalf("ValidScheduledTaskState(%q) = %v, want %v", tt.state, got, tt.want)
			}
		})
	}
}

func TestEditableScheduledTaskState(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		state string
		want  bool
	}{
		{name: "active editable", state: string(schema.ChatScheduledTaskStateActive), want: true},
		{name: "paused editable", state: string(schema.ChatScheduledTaskStatePaused), want: true},
		{name: "completed not editable", state: string(schema.ChatScheduledTaskStateCompleted), want: false},
		{name: "failed not editable", state: string(schema.ChatScheduledTaskStateFailed), want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := EditableScheduledTaskState(tt.state); got != tt.want {
				t.Fatalf("EditableScheduledTaskState(%q) = %v, want %v", tt.state, got, tt.want)
			}
		})
	}
}
