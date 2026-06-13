package app

import (
	"testing"

	"github.com/flowline-io/flowbot/pkg/client"
	"github.com/stretchr/testify/assert"
)

func TestDisplayVersion(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want string
	}{
		{name: "already prefixed", raw: "v0.92.1", want: "v0.92.1"},
		{name: "bare semver", raw: "0.92.1", want: "v0.92.1"},
		{name: "empty falls back", raw: "  ", want: "dev"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, displayVersion(tt.raw))
		})
	}
}

func TestRenderSplashSkills(t *testing.T) {
	tests := []struct {
		name    string
		skills  []client.ChatSkillInfo
		wantSub string
		notWant string
	}{
		{name: "empty skills shows placeholder", skills: nil, wantSub: splashNoSkills},
		{name: "one skill listed", skills: []client.ChatSkillInfo{{Name: "foo", Description: "bar"}}, wantSub: "foo: bar", notWant: splashNoSkills},
		{name: "skill name only", skills: []client.ChatSkillInfo{{Name: "baz"}}, wantSub: "baz", notWant: splashNoSkills},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info := &client.ChatAgentInfo{
				Version:   "1.0.0",
				ChatModel: "m",
				Provider:  "p",
				Skills:    tt.skills,
			}
			got := RenderSplash(80, info, "sess-1", "http://localhost:6060", NewStyles())
			assert.Contains(t, got, tt.wantSub)
			if tt.notWant != "" {
				assert.NotContains(t, got, tt.notWant)
			}
		})
	}
}

func TestRenderSplashVersionTitle(t *testing.T) {
	tests := []struct {
		name    string
		version string
		wantSub string
	}{
		{name: "no double v prefix", version: "v0.92.1", wantSub: "Flowbot Agent v0.92.1 ·"},
		{name: "bare version", version: "0.92.1", wantSub: "Flowbot Agent v0.92.1 ·"},
		{name: "dev fallback", version: "", wantSub: "Flowbot Agent dev ·"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info := &client.ChatAgentInfo{Version: tt.version, ChatModel: "m", Provider: "p"}
			got := RenderSplash(80, info, "sess-1", "http://localhost:6060", NewStyles())
			assert.Contains(t, got, tt.wantSub)
			assert.NotContains(t, got, "vv")
		})
	}
}
