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
		store   *notifyTestStore
		want    string
		wantErr error
	}{
		{
			name: "returns default enabled channel name",
			store: &notifyTestStore{
				defaultChannel: &model.NotifyChannel{Name: "phone", Enabled: true, IsDefault: true},
			},
			want: "phone",
		},
		{
			name:    "missing default",
			store:   &notifyTestStore{},
			wantErr: ErrNoDefaultChannel,
		},
		{
			name: "disabled default treated as missing",
			store: &notifyTestStore{
				defaultChannel: &model.NotifyChannel{Name: "phone", Enabled: false, IsDefault: true},
			},
			wantErr: ErrNoDefaultChannel,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			replaceDatabaseForTest(t, tt.store)
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
		store   *notifyTestStore
		want    string
		wantErr error
	}{
		{
			name: "returns default template id",
			store: &notifyTestStore{
				defaultTemplate: &model.NotifyTemplate{TemplateID: "agent.notify", IsDefault: true},
			},
			want: "agent.notify",
		},
		{
			name:    "missing default",
			store:   &notifyTestStore{},
			wantErr: ErrNoDefaultTemplate,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			replaceDatabaseForTest(t, tt.store)
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
				replaceDatabaseForTest(t, &notifyTestStore{
					defaultTemplate: &model.NotifyTemplate{TemplateID: AgentNotifyTemplateID, IsDefault: true},
				})
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
				replaceDatabaseForTest(t, &notifyTestStore{
					defaultChannel: &model.NotifyChannel{Name: "phone", Enabled: true, IsDefault: true},
				})
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
				replaceDatabaseForTest(t, &notifyTestStore{
					defaultChannel:  &model.NotifyChannel{Name: "phone", Protocol: "testdefaults", URI: "testdefaults://chan/tok", Enabled: true, IsDefault: true},
					defaultTemplate: &model.NotifyTemplate{TemplateID: AgentNotifyTemplateID, IsDefault: true},
					globalChannels: map[string]model.NotifyChannel{
						"phone": {Name: "phone", Protocol: "testdefaults", URI: "testdefaults://chan/tok", Enabled: true},
					},
				})
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
		store      *notifyTestStore
		wantCreate bool
	}{
		{
			name:       "creates when missing",
			store:      &notifyTestStore{},
			wantCreate: true,
		},
		{
			name: "skips when present",
			store: &notifyTestStore{
				templatesByID: map[string]model.NotifyTemplate{
					AgentNotifyTemplateID: {TemplateID: AgentNotifyTemplateID},
				},
			},
			wantCreate: false,
		},
		{
			name:       "nil database is no-op",
			store:      nil,
			wantCreate: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.store == nil {
				var db store.Adapter
				replaceDatabaseForTest(t, db)
			} else {
				replaceDatabaseForTest(t, tt.store)
			}
			require.NoError(t, SeedAgentNotifyTemplate(context.Background()))
			if tt.store == nil {
				return
			}
			if tt.wantCreate {
				require.Len(t, tt.store.createdTemplates, 1)
				assert.Equal(t, AgentNotifyTemplateID, tt.store.createdTemplates[0].TemplateID)
				return
			}
			assert.Empty(t, tt.store.createdTemplates)
		})
	}
}
