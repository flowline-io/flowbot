package discord

import (
	"testing"

	"github.com/bwmarrin/discordgo"

	"github.com/flowline-io/flowbot/pkg/types/protocol"
)

func TestAdapterEventConvertMessageCreate(t *testing.T) {
	adapter := &Adapter{}
	tests := []struct {
		name       string
		msg        *discordgo.MessageCreate
		wantDetail string
		wantTopic  string
	}{
		{
			name: "DM produces MessageDirectEvent",
			msg: &discordgo.MessageCreate{Message: &discordgo.Message{
				ID: "msg-001", Content: "hello", ChannelID: "D123", GuildID: "",
				Author: &discordgo.User{ID: "U456"},
			}},
			wantDetail: protocol.MessageDirectEvent,
			wantTopic:  "dm",
		},
		{
			name: "guild produces MessageGroupEvent",
			msg: &discordgo.MessageCreate{Message: &discordgo.Message{
				ID: "msg-002", Content: "group", ChannelID: "C789", GuildID: "G111",
				Author: &discordgo.User{ID: "U222"},
			}},
			wantDetail: protocol.MessageGroupEvent,
			wantTopic:  "text",
		},
		{
			name: "bot message returns empty",
			msg: &discordgo.MessageCreate{Message: &discordgo.Message{
				ID: "msg-bot", Content: "bot", ChannelID: "D999",
				Author: &discordgo.User{ID: "B333", Bot: true},
			}},
			wantDetail: "",
			wantTopic:  "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			evt := adapter.EventConvert(tt.msg)
			if evt.DetailType != tt.wantDetail {
				t.Errorf("expected DetailType %q, got %q", tt.wantDetail, evt.DetailType)
			}
			if tt.wantDetail != "" {
				data, ok := evt.Data.(protocol.MessageEventData)
				if !ok {
					t.Fatal("expected MessageEventData")
				}
				if evt.Id == "" || evt.Type != protocol.MessageEventType {
					t.Error("expected Id and MessageEventType")
				}
				if data.Self.Platform != ID {
					t.Errorf("expected platform %s, got %s", ID, data.Self.Platform)
				}
				if data.TopicType != tt.wantTopic {
					t.Errorf("expected topicType %s, got %s", tt.wantTopic, data.TopicType)
				}
			}
		})
	}
}

func TestAdapterEventConvertReady(t *testing.T) {
	adapter := &Adapter{}
	tests := []struct {
		name    string
		version int
	}{
		{name: "Ready produces MetaConnectEvent", version: 10},
		{name: "Ready with version 8", version: 8},
		{name: "Ready with version 0", version: 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			evt := adapter.EventConvert(&discordgo.Ready{Version: tt.version})
			if evt.Id == "" {
				t.Error("expected non-empty Id")
			}
			if evt.DetailType != protocol.MetaConnectEvent {
				t.Errorf("expected MetaConnectEvent, got %s", evt.DetailType)
			}
			if evt.Type != protocol.MetaEventType {
				t.Errorf("expected MetaEventType, got %s", evt.Type)
			}
		})
	}
}

func TestAdapterEventConvertInteraction(t *testing.T) {
	adapter := &Adapter{}
	tests := []struct {
		name       string
		interact   *discordgo.InteractionCreate
		wantDetail string
		wantCmd    string
	}{
		{
			name: "command produces MessageCommandEvent",
			interact: &discordgo.InteractionCreate{Interaction: &discordgo.Interaction{
				Type: discordgo.InteractionApplicationCommand,
				Data: discordgo.ApplicationCommandInteractionData{Name: "ping"},
			}},
			wantDetail: protocol.MessageCommandEvent,
			wantCmd:    "ping",
		},
		{
			name: "empty command name returns empty",
			interact: &discordgo.InteractionCreate{Interaction: &discordgo.Interaction{
				Type: discordgo.InteractionApplicationCommand,
				Data: discordgo.ApplicationCommandInteractionData{Name: ""},
			}},
			wantDetail: "",
			wantCmd:    "",
		},
		{
			name: "ping command",
			interact: &discordgo.InteractionCreate{Interaction: &discordgo.Interaction{
				Type: discordgo.InteractionApplicationCommand,
				Data: discordgo.ApplicationCommandInteractionData{Name: "help"},
			}},
			wantDetail: protocol.MessageCommandEvent,
			wantCmd:    "help",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			evt := adapter.EventConvert(tt.interact)
			if evt.DetailType != tt.wantDetail {
				t.Errorf("expected DetailType %q, got %q", tt.wantDetail, evt.DetailType)
			}
			if tt.wantDetail != "" {
				cmdData, ok := evt.Data.(protocol.CommandEventData)
				if !ok {
					t.Fatal("expected CommandEventData")
				}
				if cmdData.Command != tt.wantCmd {
					t.Errorf("expected command %q, got %q", tt.wantCmd, cmdData.Command)
				}
			}
		})
	}
}

func TestAdapterEventConvertEdgeCasesDiscord(t *testing.T) {
	adapter := &Adapter{}
	tests := []struct {
		name string
		data any
	}{
		{name: "nil input returns empty", data: nil},
		{name: "non-discord type", data: "not a discord event"},
		{name: "int input", data: 42},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			evt := adapter.EventConvert(tt.data)
			if evt.DetailType != "" || evt.Id != "" {
				t.Error("expected empty event")
			}
		})
	}
}

func TestAdapterMessageConvertDiscord(t *testing.T) {
	adapter := &Adapter{}
	tests := []struct {
		name  string
		input any
	}{
		{name: "nil input", input: nil},
		{name: "string input", input: "plain text"},
		{name: "int input", input: 42},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := adapter.MessageConvert(tt.input)
			if len(result) != 1 {
				t.Fatalf("expected 1 segment, got %d", len(result))
			}
		})
	}
}
