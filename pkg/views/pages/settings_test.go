package pages

import (
	"testing"

	"github.com/flowline-io/flowbot/pkg/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewSettingsPageData(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name         string
		groups       []config.SettingGroup
		wantTitles   []string
		wantKind     SettingsValueKind
		wantFilterIn []string
	}{
		{
			name: "maps groups titles kinds and filter text",
			groups: []config.SettingGroup{
				{
					Name: "postgres",
					Entries: []config.SettingEntry{
						{Path: "postgres.dsn", Value: config.MaskedSecret, Description: "DSN", Sensitive: true},
					},
				},
				{
					Name: "root",
					Entries: []config.SettingEntry{
						{Path: "listen", Value: ":80", Description: "Listen addr"},
					},
				},
			},
			wantTitles:   []string{"Postgres", "Root"},
			wantKind:     SettingsValueSecret,
			wantFilterIn: []string{"postgres.dsn", "DSN"},
		},
		{
			name: "empty placeholder uses empty kind",
			groups: []config.SettingGroup{
				{
					Name: "root",
					Entries: []config.SettingEntry{
						{Path: "listen", Value: config.EmptyDisplay},
					},
				},
			},
			wantTitles: []string{"Root"},
			wantKind:   SettingsValueEmpty,
		},
		{
			name: "json value uses code kind",
			groups: []config.SettingGroup{
				{
					Name: "profiling",
					Entries: []config.SettingEntry{
						{Path: "profiling.profile_types", Value: `["cpu"]`},
					},
				},
			},
			wantTitles: []string{"Profiling"},
			wantKind:   SettingsValueCode,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			data := NewSettingsPageData(tt.groups)
			require.Len(t, data.Sections, len(tt.wantTitles))
			for i, title := range tt.wantTitles {
				assert.Equal(t, title, data.Sections[i].Title)
			}
			require.NotEmpty(t, data.Sections[0].Entries)
			assert.Equal(t, tt.wantKind, data.Sections[0].Entries[0].ValueKind)
			for _, want := range tt.wantFilterIn {
				assert.Contains(t, data.Sections[0].Entries[0].FilterText, want)
			}
		})
	}
}

func TestSettingsGroupTitle(t *testing.T) {
	t.Parallel()
	tests := []struct {
		in, want string
	}{
		{in: "root", want: "Root"},
		{in: "chat_agent", want: "Chat Agent"},
		{in: "http", want: "HTTP"},
		{in: "api_path", want: "API Path"},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, settingsGroupTitle(tt.in))
		})
	}
}
