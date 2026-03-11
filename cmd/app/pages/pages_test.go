package pages

import (
	"testing"

	"github.com/flowline-io/flowbot/pkg/types/admin"
)

func TestUsers_AvatarInitial(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "empty name",
			input:    "",
			expected: "U",
		},
		{
			name:     "single char",
			input:    "A",
			expected: "A",
		},
		{
			name:     "regular name",
			input:    "John",
			expected: "J",
		},
		{
			name:     "unicode character",
			input:    "张三",
			expected: "张",
		},
		{
			name:     "lowercase",
			input:    "alice",
			expected: "a",
		},
	}

	u := &Users{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := u.avatarInitial(tt.input)
			if result != tt.expected {
				t.Errorf("avatarInitial(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestLogs_TruncateMessage(t *testing.T) {
	tests := []struct {
		name     string
		message  string
		maxLen   int
		expected string
	}{
		{
			name:     "short message",
			message:  "Hello",
			maxLen:   100,
			expected: "Hello",
		},
		{
			name:     "exact length",
			message:  "Hello World",
			maxLen:   11,
			expected: "Hello World",
		},
		{
			name:     "long message",
			message:  "This is a very long log message that needs to be truncated",
			maxLen:   20,
			expected: "This is a very lo...",
		},
		{
			name:     "empty message",
			message:  "",
			maxLen:   10,
			expected: "",
		},
	}

	l := &Logs{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := l.truncateMessage(tt.message, tt.maxLen)
			if result != tt.expected {
				t.Errorf("truncateMessage(%q, %d) = %q, want %q", tt.message, tt.maxLen, result, tt.expected)
			}
		})
	}
}

func TestWorkflows_TruncateDescription(t *testing.T) {
	tests := []struct {
		name        string
		description string
		expected    string
	}{
		{
			name:        "short description",
			description: "A short desc",
			expected:    "A short desc",
		},
		{
			name:        "exact length",
			description: "This is exactly fifty characters long description",
			expected:    "This is exactly fifty characters long description",
		},
		{
			name:        "long description",
			description: "This is a very long workflow description that exceeds the maximum allowed length",
			expected:    "This is a very long workflow description that e...",
		},
		{
			name:        "empty description",
			description: "",
			expected:    "",
		},
	}

	w := &Workflows{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := w.truncateDescription(tt.description)
			if result != tt.expected {
				t.Errorf("truncateDescription(%q) = %q, want %q", tt.description, result, tt.expected)
			}
		})
	}
}

func TestUserStatus_Badge(t *testing.T) {
	tests := []struct {
		status   admin.UserStatus
		contains string
	}{
		{admin.UserActive, "badge-success"},
		{admin.UserInactive, "badge-warning"},
		{admin.UserSuspended, "badge-error"},
	}

	u := &Users{}
	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			result := u.statusBadge(tt.status)
			if result == nil {
				t.Fatal("statusBadge returned nil")
			}
		})
	}
}

func TestLogLevel_Badge(t *testing.T) {
	tests := []struct {
		level admin.LogLevel
	}{
		{admin.LogLevelDebug},
		{admin.LogLevelInfo},
		{admin.LogLevelWarn},
		{admin.LogLevelError},
	}

	l := &Logs{}
	for _, tt := range tests {
		t.Run(string(tt.level), func(t *testing.T) {
			result := l.levelBadge(tt.level)
			if result == nil {
				t.Fatal("levelBadge returned nil")
			}
		})
	}
}

func TestWorkflowStatus_Badge(t *testing.T) {
	tests := []struct {
		status admin.WorkflowStatus
	}{
		{admin.WorkflowPending},
		{admin.WorkflowRunning},
		{admin.WorkflowCompleted},
		{admin.WorkflowFailed},
	}

	w := &Workflows{}
	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			result := w.statusBadge(tt.status)
			if result == nil {
				t.Fatal("statusBadge returned nil")
			}
		})
	}
}

func TestUserRole_Badge(t *testing.T) {
	tests := []struct {
		role admin.UserRole
	}{
		{admin.RoleAdmin},
		{admin.RoleUser},
		{admin.RoleViewer},
	}

	u := &Users{}
	for _, tt := range tests {
		t.Run(string(tt.role), func(t *testing.T) {
			result := u.roleBadge(tt.role)
			if result == nil {
				t.Fatal("roleBadge returned nil")
			}
		})
	}
}
