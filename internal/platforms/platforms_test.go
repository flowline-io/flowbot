package platforms

import (
	"testing"

	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/protocol"
)

func assertSingleSegment(t *testing.T, msg protocol.Message, wantType string) {
	t.Helper()
	if len(msg) != 1 {
		t.Fatalf("expected 1 segment, got %d", len(msg))
	}
	if msg[0].Type != wantType {
		t.Errorf("expected type %s, got %s", wantType, msg[0].Type)
	}
}

func assertSegmentCount(t *testing.T, msg protocol.Message, want int) {
	t.Helper()
	if len(msg) != want {
		t.Fatalf("expected %d segments, got %d", want, len(msg))
	}
}

func assertSegmentData(t *testing.T, data map[string]any, key string, want any) {
	t.Helper()
	got, ok := data[key]
	if !ok {
		t.Errorf("key %q not found in segment data", key)
		return
	}
	if got != want {
		t.Errorf("key %q: expected %v, got %v", key, want, got)
	}
}

func assertNilMessage(t *testing.T, msg protocol.Message) {
	t.Helper()
	if msg != nil {
		t.Errorf("expected nil message, got %v", msg)
	}
}

func TestMessageConvertText(t *testing.T) {
	tests := []struct {
		name  string
		input types.TextMsg
		want  string
	}{
		{name: "with content", input: types.TextMsg{Text: "hello world"}, want: "hello world"},
		{name: "with empty text", input: types.TextMsg{Text: ""}, want: ""},
		{name: "with special chars", input: types.TextMsg{Text: "hi!"}, want: "hi!"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := MessageConvert(tt.input)
			assertSingleSegment(t, msg, "text")
			assertSegmentData(t, msg[0].Data, "text", tt.want)
		})
	}
}

func TestMessageConvertLink(t *testing.T) {
	tests := []struct {
		name  string
		input types.LinkMsg
	}{
		{name: "with all fields", input: types.LinkMsg{Title: "GH", Url: "https://gh.com", Cover: "icon.png"}},
		{name: "with empty fields", input: types.LinkMsg{}},
		{name: "title only", input: types.LinkMsg{Title: "Link"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := MessageConvert(tt.input)
			assertSingleSegment(t, msg, "link")
		})
	}
}

func TestMessageConvertTable(t *testing.T) {
	tests := []struct {
		name  string
		input types.TableMsg
	}{
		{name: "with data", input: types.TableMsg{Title: "Stats", Header: []string{"N", "V"}, Row: [][]any{{"CPU", "80%"}, {"Mem", "60%"}}}},
		{name: "empty rows", input: types.TableMsg{Title: "Empty", Header: []string{"Col"}}},
		{name: "no title", input: types.TableMsg{Header: []string{"A"}, Row: [][]any{{1}}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := MessageConvert(tt.input)
			assertSingleSegment(t, msg, "table")
		})
	}
}

func TestMessageConvertInfo(t *testing.T) {
	tests := []struct {
		name  string
		input types.InfoMsg
	}{
		{name: "map[string]any model", input: types.InfoMsg{Title: "Info", Model: map[string]any{"k": "v"}}},
		{name: "map[string]string model", input: types.InfoMsg{Title: "S", Model: map[string]string{"cpu": "10%"}}},
		{name: "nil model", input: types.InfoMsg{Title: "Empty"}},
		{name: "struct model", input: types.InfoMsg{Title: "Struct", Model: struct{ Key string }{Key: "val"}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := MessageConvert(tt.input)
			assertSingleSegment(t, msg, "action_card")
			assertSegmentData(t, msg[0].Data, "title", tt.input.Title)
		})
	}
}

func TestMessageConvertChart(t *testing.T) {
	tests := []struct {
		name  string
		input types.ChartMsg
	}{
		{name: "full chart", input: types.ChartMsg{Title: "CPU", SubTitle: "24h", XAxis: []string{"M", "T"}, Series: []float64{10, 20}}},
		{name: "no subtitle", input: types.ChartMsg{Title: "Mem", XAxis: []string{"A"}, Series: []float64{5}}},
		{name: "empty chart", input: types.ChartMsg{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := MessageConvert(tt.input)
			assertSingleSegment(t, msg, "chart")
			assertSegmentData(t, msg[0].Data, "chart_type", "bar")
		})
	}
}

func TestMessageConvertHtml(t *testing.T) {
	tests := []struct {
		name  string
		input types.HtmlMsg
	}{
		{name: "with content", input: types.HtmlMsg{Raw: "<b>bold</b>"}},
		{name: "empty raw", input: types.HtmlMsg{Raw: ""}},
		{name: "multiline html", input: types.HtmlMsg{Raw: "<p>\nhello\n</p>"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := MessageConvert(tt.input)
			assertSingleSegment(t, msg, "markdown")
		})
	}
}

func TestMessageConvertMarkdown(t *testing.T) {
	tests := []struct {
		name    string
		input   types.MarkdownMsg
		wantNil bool
	}{
		{name: "with title and raw", input: types.MarkdownMsg{Title: "T", Raw: "**bold**"}, wantNil: false},
		{name: "raw only", input: types.MarkdownMsg{Raw: "**bold**"}, wantNil: false},
		{name: "empty returns nil", input: types.MarkdownMsg{Title: "", Raw: ""}, wantNil: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := MessageConvert(tt.input)
			if tt.wantNil {
				assertNilMessage(t, msg)
			} else {
				assertSingleSegment(t, msg, "markdown")
			}
		})
	}
}

func TestMessageConvertKV(t *testing.T) {
	tests := []struct {
		name    string
		input   types.KVMsg
		wantNil bool
	}{
		{name: "with pairs", input: types.KVMsg{"k1": "v1", "k2": "v2"}, wantNil: false},
		{name: "single key", input: types.KVMsg{"k": "v"}, wantNil: false},
		{name: "empty returns nil", input: types.KVMsg{}, wantNil: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := MessageConvert(tt.input)
			if tt.wantNil {
				assertNilMessage(t, msg)
			} else {
				assertSingleSegment(t, msg, "kv")
			}
		})
	}
}

func TestMessageConvertForm(t *testing.T) {
	tests := []struct {
		name  string
		input types.FormMsg
	}{
		{name: "with fields", input: types.FormMsg{Title: "F", ID: "f1", Field: []types.FormField{{Label: "N", Key: "n", Type: types.FormFieldText}}}},
		{name: "multiple fields", input: types.FormMsg{Title: "F2", ID: "f2", Field: []types.FormField{{Label: "A"}, {Label: "B"}}}},
		{name: "no fields", input: types.FormMsg{Title: "F3", ID: "f3"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := MessageConvert(tt.input)
			assertSingleSegment(t, msg, "form")
			assertSegmentData(t, msg[0].Data, "title", tt.input.Title)
		})
	}
}

func TestMessageConvertEmpty(t *testing.T) {
	tests := []struct {
		name    string
		input   types.MsgPayload
		wantNil bool
	}{
		{name: "EmptyMsg", input: types.EmptyMsg{}, wantNil: true},
		{name: "non-MsgPayload string returns error text", input: nil, wantNil: false},
		{name: "int input uses default converter", input: nil, wantNil: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var input any = tt.input
			if tt.name == "non-MsgPayload string returns error text" {
				input = "plain string"
			} else if tt.name == "int input uses default converter" {
				input = 42
			}
			msg := MessageConvert(input)
			if tt.wantNil {
				assertNilMessage(t, msg)
			} else {
				assertSegmentCount(t, msg, 1)
			}
		})
	}
}

type mockAction struct {
	sendMessageResp   protocol.Response
	updateMessageResp protocol.Response
	deleteMessageResp protocol.Response
}

func (m *mockAction) SendMessage(protocol.Request) protocol.Response   { return m.sendMessageResp }
func (m *mockAction) UpdateMessage(protocol.Request) protocol.Response { return m.updateMessageResp }
func (m *mockAction) DeleteMessage(protocol.Request) protocol.Response { return m.deleteMessageResp }
func (*mockAction) GetLatestEvents(protocol.Request) protocol.Response { return protocol.Response{} }
func (*mockAction) GetSupportedActions(protocol.Request) protocol.Response {
	return protocol.Response{}
}
func (*mockAction) GetStatus(protocol.Request) protocol.Response        { return protocol.Response{} }
func (*mockAction) GetVersion(protocol.Request) protocol.Response       { return protocol.Response{} }
func (*mockAction) GetUserInfo(protocol.Request) protocol.Response      { return protocol.Response{} }
func (*mockAction) CreateChannel(protocol.Request) protocol.Response    { return protocol.Response{} }
func (*mockAction) GetChannelInfo(protocol.Request) protocol.Response   { return protocol.Response{} }
func (*mockAction) GetChannelList(protocol.Request) protocol.Response   { return protocol.Response{} }
func (*mockAction) RegisterChannels(protocol.Request) protocol.Response { return protocol.Response{} }
func (*mockAction) RegisterSlashCommands(protocol.Request) protocol.Response {
	return protocol.Response{}
}

var _ protocol.Action = &mockAction{}

type mockAdapter struct{}

func (*mockAdapter) MessageConvert(_ any) protocol.Message { return nil }
func (*mockAdapter) EventConvert(_ any) protocol.Event     { return protocol.Event{} }

func TestCallerDo(t *testing.T) {
	tests := []struct {
		name   string
		action protocol.Action
		req    protocol.Request
		expect func(t *testing.T, resp protocol.Response)
	}{
		{
			name:   "SendMessage delegates to action",
			action: &mockAction{sendMessageResp: protocol.NewSuccessResponse("msg-id")},
			req:    protocol.Request{Action: protocol.SendMessageAction},
			expect: func(t *testing.T, resp protocol.Response) {
				if resp.Status != protocol.Success || resp.Data != "msg-id" {
					t.Errorf("unexpected response: %+v", resp)
				}
			},
		},
		{
			name:   "UpdateMessage delegates to action",
			action: &mockAction{updateMessageResp: protocol.NewSuccessResponse("updated")},
			req:    protocol.Request{Action: protocol.UpdateMessageAction},
			expect: func(t *testing.T, resp protocol.Response) {
				if resp.Status != protocol.Success || resp.Data != "updated" {
					t.Errorf("unexpected response: %+v", resp)
				}
			},
		},
		{
			name:   "DeleteMessage delegates to action",
			action: &mockAction{deleteMessageResp: protocol.NewSuccessResponse("deleted")},
			req:    protocol.Request{Action: protocol.DeleteMessageAction},
			expect: func(t *testing.T, resp protocol.Response) {
				if resp.Status != protocol.Success || resp.Data != "deleted" {
					t.Errorf("unexpected response: %+v", resp)
				}
			},
		},
		{
			name:   "unknown action returns unsupported error",
			action: &mockAction{},
			req:    protocol.Request{Action: "unknown_action"},
			expect: func(t *testing.T, resp protocol.Response) {
				if resp.Status != protocol.Failed || resp.RetCode != "10002" {
					t.Errorf("expected failed/10002, got %+v", resp)
				}
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			caller := &Caller{Action: tt.action, Adapter: &mockAdapter{}}
			tt.expect(t, caller.Do(tt.req))
		})
	}
}

func TestGetCaller(t *testing.T) {
	tests := []struct {
		name    string
		prep    func()
		lookup  string
		wantErr bool
	}{
		{
			name: "returns caller for registered platform",
			prep: func() {
				callers = make(map[string]*Caller)
				callers["test-platform"] = &Caller{Action: &mockAction{}, Adapter: &mockAdapter{}}
			},
			lookup:  "test-platform",
			wantErr: false,
		},
		{
			name: "returns error for unregistered platform",
			prep: func() {
				callers = make(map[string]*Caller)
			},
			lookup:  "nonexistent",
			wantErr: true,
		},
		{
			name: "returns error when no platforms registered",
			prep: func() {
				callers = make(map[string]*Caller)
			},
			lookup:  "any-platform",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.prep()
			c, err := GetCaller(tt.lookup)
			if tt.wantErr {
				if err == nil || c != nil {
					t.Error("expected error and nil caller")
				}
			} else {
				if err != nil || c == nil {
					t.Error("expected no error and non-nil caller")
				}
			}
		})
	}
}
