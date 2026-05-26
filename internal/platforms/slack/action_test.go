package slack

import (
	"testing"

	"github.com/flowline-io/flowbot/pkg/types/protocol"
)

func TestSegHandlersDispatch(t *testing.T) {
	tests := []struct {
		name        string
		segmentType string
		wantText    string
		wantBlocks  int
		wantFiles   int
		segment     protocol.MessageSegment
	}{
		{
			name:        "text segment with content",
			segmentType: "text",
			wantText:    "hello",
			wantBlocks:  0,
			wantFiles:   0,
			segment:     protocol.MessageSegment{Type: "text", Data: map[string]any{"text": "hello"}},
		},
		{
			name:        "text segment with empty content",
			segmentType: "text",
			wantText:    "",
			wantBlocks:  0,
			wantFiles:   0,
			segment:     protocol.MessageSegment{Type: "text", Data: map[string]any{}},
		},
		{
			name:        "url segment",
			segmentType: "url",
			wantText:    "https://example.com",
			wantBlocks:  0,
			wantFiles:   0,
			segment:     protocol.MessageSegment{Type: "url", Data: map[string]any{"url": "https://example.com"}},
		},
		{
			name:        "mention segment",
			segmentType: "mention",
			wantText:    "<@U123>",
			wantBlocks:  0,
			wantFiles:   0,
			segment:     protocol.MessageSegment{Type: "mention", Data: map[string]any{"user_id": "U123"}},
		},
		{
			name:        "mention_all segment",
			segmentType: "mention_all",
			wantText:    "<!channel>",
			wantBlocks:  0,
			wantFiles:   0,
			segment:     protocol.MessageSegment{Type: "mention_all", Data: map[string]any{}},
		},
		{
			name:        "link segment with title and url",
			segmentType: "link",
			wantText:    "",
			wantBlocks:  1,
			wantFiles:   0,
			segment:     protocol.MessageSegment{Type: "link", Data: map[string]any{"title": "GitHub", "url": "https://github.com"}},
		},
		{
			name:        "markdown segment with title and text",
			segmentType: "markdown",
			wantText:    "",
			wantBlocks:  2,
			wantFiles:   0,
			segment:     protocol.MessageSegment{Type: "markdown", Data: map[string]any{"title": "Title", "text": "content"}},
		},
		{
			name:        "markdown segment text only",
			segmentType: "markdown",
			wantText:    "",
			wantBlocks:  1,
			wantFiles:   0,
			segment:     protocol.MessageSegment{Type: "markdown", Data: map[string]any{"text": "content"}},
		},
		{
			name:        "file segment",
			segmentType: "file",
			wantText:    "",
			wantBlocks:  0,
			wantFiles:   1,
			segment:     protocol.MessageSegment{Type: "file", Data: map[string]any{"file_id": "/tmp/test.txt"}},
		},
		{
			name:        "unknown segment type is ignored",
			segmentType: "unknown_xyz",
			wantText:    "",
			wantBlocks:  0,
			wantFiles:   0,
			segment:     protocol.MessageSegment{Type: "unknown_xyz", Data: map[string]any{"key": "val"}},
		},
		{
			name:        "kv segment with fields",
			segmentType: "kv",
			wantText:    "",
			wantBlocks:  1,
			wantFiles:   0,
			segment:     protocol.MessageSegment{Type: "kv", Data: map[string]any{"fields": map[string]string{"k": "v"}}},
		},
		{
			name:        "status segment",
			segmentType: "status",
			wantText:    "",
			wantBlocks:  1,
			wantFiles:   0,
			segment:     protocol.MessageSegment{Type: "status", Data: map[string]any{"text": "thinking..."}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler, ok := segHandlers[tt.segmentType]
			if !ok {
				if tt.wantText == "" && tt.wantBlocks == 0 && tt.wantFiles == 0 {
					return // expected to not exist
				}
				t.Fatalf("expected handler for segment type %s", tt.segmentType)
			}
			txt, blocks, files := handler(tt.segment)
			if txt != tt.wantText {
				t.Errorf("expected text %q, got %q", tt.wantText, txt)
			}
			if len(blocks) != tt.wantBlocks {
				t.Errorf("expected %d blocks, got %d", tt.wantBlocks, len(blocks))
			}
			if len(files) != tt.wantFiles {
				t.Errorf("expected %d files, got %d", tt.wantFiles, len(files))
			}
		})
	}
}

func TestSegHandlersAllSegTypesRegistered(t *testing.T) {
	expectedTypes := []string{
		"text", "url", "mention", "mention_all",
		"image", "file", "video", "audio", "voice",
		"location", "reply", "chart", "table", "form",
		"action_card", "status", "link", "markdown", "kv",
	}

	for _, segType := range expectedTypes {
		if _, ok := segHandlers[segType]; !ok {
			t.Errorf("missing handler for segment type: %s", segType)
		}
	}
}

func TestBuildMsgOptions(t *testing.T) {
	action := &Action{}

	tests := []struct {
		name        string
		content     protocol.Message
		wantOptsMin int
		wantOptsMax int
	}{
		{
			name: "single text segment",
			content: protocol.Message{
				{Type: "text", Data: map[string]any{"text": "hello"}},
			},
			wantOptsMin: 1,
			wantOptsMax: 1,
		},
		{
			name: "single markdown segment with title",
			content: protocol.Message{
				{Type: "markdown", Data: map[string]any{"title": "Title", "text": "body"}},
			},
			wantOptsMin: 1,
			wantOptsMax: 1,
		},
		{
			name: "text and markdown together produce both blocks",
			content: protocol.Message{
				{Type: "text", Data: map[string]any{"text": "hello"}},
				{Type: "markdown", Data: map[string]any{"text": "markdown body"}},
			},
			wantOptsMin: 2,
			wantOptsMax: 2,
		},
		{
			name:        "empty content produces no output",
			content:     protocol.Message{},
			wantOptsMin: 0,
			wantOptsMax: 0,
		},
		{
			name: "unknown segment type is silently ignored",
			content: protocol.Message{
				{Type: "fancy_type", Data: map[string]any{}},
			},
			wantOptsMin: 0,
			wantOptsMax: 0,
		},
		{
			name: "link segment produces block",
			content: protocol.Message{
				{Type: "link", Data: map[string]any{"title": "Link", "url": "https://a.com"}},
			},
			wantOptsMin: 1,
			wantOptsMax: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts, _ := action.buildMsgOptions(tt.content)
			if len(opts) < tt.wantOptsMin || len(opts) > tt.wantOptsMax {
				t.Errorf("expected %d-%d opts, got %d", tt.wantOptsMin, tt.wantOptsMax, len(opts))
			}
		})
	}
}

func TestSendMessageInputValidation(t *testing.T) {
	action := &Action{}

	tests := []struct {
		name      string
		req       protocol.Request
		wantFail  bool
		wantCode  string
	}{
		{
			name: "empty message returns success",
			req: protocol.Request{
				Action: protocol.SendMessageAction,
				Params: map[string]any{
					"topic":   "C123",
					"message": protocol.Message{},
				},
			},
			wantFail: false,
		},
		{
			name: "nil message returns error",
			req: protocol.Request{
				Action: protocol.SendMessageAction,
				Params: map[string]any{
					"topic":   "C123",
					"message": nil,
				},
			},
			wantFail: true,
			wantCode: "10006",
		},
		{
			name: "wrong message type returns error",
			req: protocol.Request{
				Action: protocol.SendMessageAction,
				Params: map[string]any{
					"topic":   "C123",
					"message": "not a protocol.Message",
				},
			},
			wantFail: true,
			wantCode: "10006",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := action.SendMessage(tt.req)
			if tt.wantFail {
				if resp.Status != protocol.Failed {
					t.Errorf("expected Failed, got %s", resp.Status)
				}
				if tt.wantCode != "" && resp.RetCode != tt.wantCode {
					t.Errorf("expected retcode %s, got %s", tt.wantCode, resp.RetCode)
				}
			} else {
				if resp.Status != protocol.Success {
					t.Errorf("expected Success, got %s (%s)", resp.Status, resp.Message)
				}
			}
		})
	}
}

func TestUnsupportedActions(t *testing.T) {
	action := &Action{}

	tests := []struct {
		name string
		call func() protocol.Response
	}{
		{name: "GetLatestEvents", call: func() protocol.Response { return action.GetLatestEvents(protocol.Request{}) }},
		{name: "GetSupportedActions", call: func() protocol.Response { return action.GetSupportedActions(protocol.Request{}) }},
		{name: "GetStatus", call: func() protocol.Response { return action.GetStatus(protocol.Request{}) }},
		{name: "GetVersion", call: func() protocol.Response { return action.GetVersion(protocol.Request{}) }},
		{name: "GetUserInfo", call: func() protocol.Response { return action.GetUserInfo(protocol.Request{}) }},
		{name: "CreateChannel", call: func() protocol.Response { return action.CreateChannel(protocol.Request{}) }},
		{name: "GetChannelInfo", call: func() protocol.Response { return action.GetChannelInfo(protocol.Request{}) }},
		{name: "GetChannelList", call: func() protocol.Response { return action.GetChannelList(protocol.Request{}) }},
		{name: "RegisterChannels", call: func() protocol.Response { return action.RegisterChannels(protocol.Request{}) }},
		{name: "RegisterSlashCommands", call: func() protocol.Response { return action.RegisterSlashCommands(protocol.Request{}) }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := tt.call()
			if resp.Status != protocol.Failed {
				t.Errorf("expected Failed, got %s", resp.Status)
			}
			if resp.RetCode != "10002" {
				t.Errorf("expected retcode 10002 (unsupported), got %s", resp.RetCode)
			}
		})
	}
}

func TestHandleSegImage(t *testing.T) {
	tests := []struct {
		name       string
		fileID     string
		wantBlocks int
		wantFiles  int
		wantText   string
	}{
		{
			name:       "image segment with file_id produces image block",
			fileID:     "F123",
			wantBlocks: 1,
			wantFiles:  0,
			wantText:   "",
		},
		{
			name:       "image segment with empty file_id",
			fileID:     "",
			wantBlocks: 0,
			wantFiles:  0,
			wantText:   "",
		},
		{
			name:       "image segment with missing file_id key",
			fileID:     "",
			wantBlocks: 0,
			wantFiles:  0,
			wantText:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data := map[string]any{}
			if tt.fileID != "" || tt.name == "image segment with file_id produces image block" {
				data["file_id"] = tt.fileID
			}
			txt, blocks, files := handleSegImage(protocol.MessageSegment{Type: "image", Data: data})
			if txt != tt.wantText {
				t.Errorf("expected text %q, got %q", tt.wantText, txt)
			}
			if len(blocks) != tt.wantBlocks {
				t.Errorf("expected %d blocks, got %d", tt.wantBlocks, len(blocks))
			}
			if len(files) != tt.wantFiles {
				t.Errorf("expected %d files, got %d", tt.wantFiles, len(files))
			}
		})
	}
}

func TestHandleSegLocation(t *testing.T) {
	tests := []struct {
		name       string
		segment    protocol.MessageSegment
		wantBlocks int
	}{
		{
			name: "location with valid coordinates produces block",
			segment: protocol.MessageSegment{
				Type: "location",
				Data: map[string]any{
					"latitude":  37.7749,
					"longitude": -122.4194,
					"title":     "SF",
					"content":   "San Francisco",
				},
			},
			wantBlocks: 1,
		},
		{
			name: "location missing latitude returns no blocks",
			segment: protocol.MessageSegment{
				Type: "location",
				Data: map[string]any{
					"longitude": -122.4194,
				},
			},
			wantBlocks: 0,
		},
		{
			name: "location missing longitude returns no blocks",
			segment: protocol.MessageSegment{
				Type: "location",
				Data: map[string]any{
					"latitude": 37.7749,
				},
			},
			wantBlocks: 0,
		},
		{
			name: "location missing all data returns no blocks",
			segment: protocol.MessageSegment{
				Type: "location",
				Data: map[string]any{},
			},
			wantBlocks: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, blocks, _ := handleSegLocation(tt.segment)
			if len(blocks) != tt.wantBlocks {
				t.Errorf("expected %d blocks, got %d", tt.wantBlocks, len(blocks))
			}
		})
	}
}

func TestHandleSegReply(t *testing.T) {
	tests := []struct {
		name       string
		userID     string
		messageID  string
		wantBlocks int
	}{
		{
			name:       "reply with user and message produces context block",
			userID:     "U1",
			messageID:  "msg-1",
			wantBlocks: 1,
		},
		{
			name:       "reply missing user_id returns no blocks",
			userID:     "",
			messageID:  "msg-1",
			wantBlocks: 0,
		},
		{
			name:       "reply missing message_id returns no blocks",
			userID:     "U1",
			messageID:  "",
			wantBlocks: 0,
		},
		{
			name:       "reply missing both returns no blocks",
			userID:     "",
			messageID:  "",
			wantBlocks: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			seg := protocol.MessageSegment{Type: "reply", Data: map[string]any{}}
			if tt.userID != "" {
				seg.Data["user_id"] = tt.userID
			}
			if tt.messageID != "" {
				seg.Data["message_id"] = tt.messageID
			}
			_, blocks, _ := handleSegReply(seg)
			if len(blocks) != tt.wantBlocks {
				t.Errorf("expected %d blocks, got %d", tt.wantBlocks, len(blocks))
			}
		})
	}
}
