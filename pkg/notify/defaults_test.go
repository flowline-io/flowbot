package notify

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/model"
)

func TestTemplateReferencesSummary(t *testing.T) {
	tests := []struct {
		name     string
		body     string
		override string
		want     bool
	}{
		{
			name: "body references summary",
			body: "{{ .summary }}",
			want: true,
		},
		{
			name: "body missing summary",
			body: "{{ .message }}",
			want: false,
		},
		{
			name:     "override references summary",
			body:     "{{ .title }}",
			override: `[{"channel":"slack","format":"markdown","template":"Hi {{ .summary }}"}]`,
			want:     true,
		},
		{
			name:     "invalid overrides json",
			body:     "{{ .title }}",
			override: `{`,
			want:     false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, TemplateReferencesSummary(tt.body, tt.override))
		})
	}
}

func TestResolveDefaultChannelName(t *testing.T) {
	tests := []struct {
		name    string
		seed    func(t *testing.T)
		want    string
		wantErr error
	}{
		{
			name: "returns default enabled channel name",
			seed: func(t *testing.T) {
				seedNotifyTestChannel(t, "phone", "test", "test://phone", true, true)
			},
			want: "phone",
		},
		{
			name:    "missing default",
			seed:    func(_ *testing.T) {},
			wantErr: ErrNoDefaultChannel,
		},
		{
			name: "disabled default treated as missing",
			seed: func(t *testing.T) {
				id := seedNotifyTestChannel(t, "phone", "test", "test://phone", true, true)
				require.NoError(t, store.NotifyConfigStoreFromDB().UpdateNotifyChannel(
					context.Background(), id, "phone", "test", "test://phone", false,
				))
			},
			wantErr: ErrNoDefaultChannel,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setupNotifySQLiteDB(t)
			tt.seed(t)
			got, err := ResolveDefaultChannelName(context.Background())
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestResolveDefaultTemplateID(t *testing.T) {
	tests := []struct {
		name    string
		seed    func(t *testing.T)
		want    string
		wantErr error
	}{
		{
			name: "returns default template id",
			seed: func(t *testing.T) {
				seedNotifyTestTemplate(t, model.NotifyTemplate{TemplateID: "agent.notify"}, true)
			},
			want: "agent.notify",
		},
		{
			name:    "missing default",
			seed:    func(_ *testing.T) {},
			wantErr: ErrNoDefaultTemplate,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setupNotifySQLiteDB(t)
			tt.seed(t)
			got, err := ResolveDefaultTemplateID(context.Background())
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestWarnSkipNoDefault(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{name: "nil", err: nil, want: false},
		{name: "no channel", err: ErrNoDefaultChannel, want: true},
		{name: "no template", err: ErrNoDefaultTemplate, want: true},
		{name: "wrapped channel", err: errors.Join(ErrNoDefaultChannel), want: true},
		{name: "other error", err: types.ErrInternal, want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, WarnSkipNoDefault(tt.err, "test"))
		})
	}
}

func TestGatewaySendDefaults(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(t *testing.T)
		wantErr error
	}{
		{
			name: "missing channel",
			setup: func(t *testing.T) {
				setupNotifyTestEnv(t, []Template{{
					ID: AgentNotifyTemplateID, Name: "Agent", DefaultFormat: "markdown",
					DefaultTemplate: AgentNotifyTemplateBody,
				}}, nil, nil)
				setupNotifySQLiteDB(t)
				seedNotifyTestTemplate(t, model.NotifyTemplate{TemplateID: AgentNotifyTemplateID}, true)
			},
			wantErr: ErrNoDefaultChannel,
		},
		{
			name: "missing template",
			setup: func(t *testing.T) {
				setupNotifyTestEnv(t, []Template{{
					ID: AgentNotifyTemplateID, Name: "Agent", DefaultFormat: "markdown",
					DefaultTemplate: AgentNotifyTemplateBody,
				}}, nil, nil)
				setupNotifySQLiteDB(t)
				seedNotifyTestChannel(t, "phone", "testdefaults", "testdefaults://chan/tok", true, true)
			},
			wantErr: ErrNoDefaultTemplate,
		},
		{
			name: "sends via defaults",
			setup: func(t *testing.T) {
				setupNotifyTestEnv(t, []Template{{
					ID: AgentNotifyTemplateID, Name: "Agent", DefaultFormat: "markdown",
					DefaultTemplate: AgentNotifyTemplateBody,
				}}, nil, nil)
				m := &mockNotifyer{
					protocol:  "testdefaults",
					templates: []string{"testdefaults://{channel}/{token}"},
				}
				Register(m.protocol, m)
				t.Cleanup(func() { Unregister(m.protocol) })
				setupNotifySQLiteDB(t)
				seedNotifyTestChannel(t, "phone", "testdefaults", "testdefaults://chan/tok", true, true)
				seedNotifyTestTemplate(t, model.NotifyTemplate{TemplateID: AgentNotifyTemplateID}, true)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setup(t)
			err := GatewaySendDefaults(context.Background(), types.Uid("u1"), map[string]any{PayloadKeySummary: "hi"})
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestSeedAgentNotifyTemplate(t *testing.T) {
	tests := []struct {
		name       string
		seed       func(t *testing.T)
		wantCreate bool
	}{
		{
			name:       "creates when missing",
			seed:       func(t *testing.T) { setupNotifySQLiteDB(t) },
			wantCreate: true,
		},
		{
			name: "skips when present",
			seed: func(t *testing.T) {
				setupNotifySQLiteDB(t)
				seedNotifyTestTemplate(t, model.NotifyTemplate{TemplateID: AgentNotifyTemplateID}, false)
			},
			wantCreate: false,
		},
		{
			name:       "nil database is no-op",
			seed:       func(t *testing.T) { replaceDatabaseForTest(t, nil) },
			wantCreate: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.seed(t)
			require.NoError(t, SeedAgentNotifyTemplate(context.Background()))
			if !tt.wantCreate {
				return
			}
			_, err := store.NotifyConfigStoreFromDB().GetNotifyTemplateByTemplateID(context.Background(), AgentNotifyTemplateID)
			require.NoError(t, err)
		})
	}
}

func TestSeedLifeQuestTemplates(t *testing.T) {
	tests := []struct {
		name       string
		templateID string
		seedFn     func(context.Context) error
	}{
		{
			name:       "completed",
			templateID: LifeQuestCompletedTemplateID,
			seedFn:     SeedLifeQuestCompletedTemplate,
		},
		{
			name:       "failed",
			templateID: LifeQuestFailedTemplateID,
			seedFn:     SeedLifeQuestFailedTemplate,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setupNotifySQLiteDB(t)
			require.NoError(t, tt.seedFn(context.Background()))
			_, err := store.NotifyConfigStoreFromDB().GetNotifyTemplateByTemplateID(context.Background(), tt.templateID)
			require.NoError(t, err)
			require.NoError(t, tt.seedFn(context.Background()))
		})
	}
}
