package partials

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEnabledStatusIcon(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		enabled bool
		want    StatusIconSpec
	}{
		{
			name:    "enabled uses success check",
			enabled: true,
			want:    StatusIconSpec{Label: "Enabled", ToneClass: "text-success", Glyph: StatusGlyphCheck},
		},
		{
			name:    "disabled uses muted x",
			enabled: false,
			want:    StatusIconSpec{Label: "Disabled", ToneClass: "text-base-content/40", Glyph: StatusGlyphX},
		},
		{
			name:    "enabled label stays human readable",
			enabled: true,
			want:    StatusIconSpec{Label: "Enabled", ToneClass: "text-success", Glyph: StatusGlyphCheck},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, EnabledStatusIcon(tt.enabled))
		})
	}
}

func TestSessionStateStatusIcon(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		state string
		want  StatusIconSpec
	}{
		{
			name:  "active session",
			state: "Active",
			want:  StatusIconSpec{Label: "Active", ToneClass: "text-success", Glyph: StatusGlyphCheck},
		},
		{
			name:  "closed session",
			state: "Closed",
			want:  StatusIconSpec{Label: "Closed", ToneClass: "text-base-content/40", Glyph: StatusGlyphStop},
		},
		{
			name:  "unknown session state",
			state: "Pending",
			want:  StatusIconSpec{Label: "Pending", ToneClass: "text-warning", Glyph: StatusGlyphAlert},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, SessionStateStatusIcon(tt.state))
		})
	}
}

func TestScheduledTaskStateStatusIcon(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		state string
		want  StatusIconSpec
	}{
		{
			name:  "active task",
			state: "active",
			want:  StatusIconSpec{Label: "active", ToneClass: "text-success", Glyph: StatusGlyphPlay},
		},
		{
			name:  "paused task",
			state: "paused",
			want:  StatusIconSpec{Label: "paused", ToneClass: "text-warning", Glyph: StatusGlyphPause},
		},
		{
			name:  "failed task",
			state: "failed",
			want:  StatusIconSpec{Label: "failed", ToneClass: "text-error", Glyph: StatusGlyphX},
		},
		{
			name:  "completed task",
			state: "completed",
			want:  StatusIconSpec{Label: "completed", ToneClass: "text-info", Glyph: StatusGlyphCheck},
		},
		{
			name:  "unknown task state",
			state: "queued",
			want:  StatusIconSpec{Label: "queued", ToneClass: "text-base-content/40", Glyph: StatusGlyphMinus},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, ScheduledTaskStateStatusIcon(tt.state))
		})
	}
}

func TestScheduledTaskRunStateStatusIcon(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		state string
		want  StatusIconSpec
	}{
		{
			name:  "completed run",
			state: "completed",
			want:  StatusIconSpec{Label: "completed", ToneClass: "text-success", Glyph: StatusGlyphCheck},
		},
		{
			name:  "running run",
			state: "running",
			want:  StatusIconSpec{Label: "running", ToneClass: "text-warning", Glyph: StatusGlyphPlay},
		},
		{
			name:  "failed run",
			state: "failed",
			want:  StatusIconSpec{Label: "failed", ToneClass: "text-error", Glyph: StatusGlyphX},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, ScheduledTaskRunStateStatusIcon(tt.state))
		})
	}
}

func TestPipelineRuntimeStatusIcon(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		enabled bool
		want    StatusIconSpec
	}{
		{
			name:    "active pipeline",
			enabled: true,
			want:    StatusIconSpec{Label: "Active", ToneClass: "text-success", Glyph: StatusGlyphPlay},
		},
		{
			name:    "paused pipeline",
			enabled: false,
			want:    StatusIconSpec{Label: "Paused", ToneClass: "text-warning", Glyph: StatusGlyphPause},
		},
		{
			name:    "active uses play glyph",
			enabled: true,
			want:    StatusIconSpec{Label: "Active", ToneClass: "text-success", Glyph: StatusGlyphPlay},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, PipelineRuntimeStatusIcon(tt.enabled))
		})
	}
}
