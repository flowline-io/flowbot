package slack

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/bytedance/sonic"
	"github.com/slack-go/slack"

	"github.com/flowline-io/flowbot/pkg/types/protocol"
)

func newTestSlackClient(t *testing.T, handler http.HandlerFunc) *slack.Client {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	return slack.New("test-token", slack.OptionAPIURL(srv.URL+"/"))
}

func slackOK(w http.ResponseWriter, data any) {
	w.Header().Set("Content-Type", "application/json")
	if data != nil {
		_ = sonic.ConfigDefault.NewEncoder(w).Encode(data)
	} else {
		w.WriteHeader(http.StatusOK)
		_ = sonic.ConfigDefault.NewEncoder(w).Encode(map[string]bool{"ok": true})
	}
}

func slackError(w http.ResponseWriter, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = sonic.ConfigDefault.NewEncoder(w).Encode(map[string]any{"ok": false, "error": msg})
}

func TestPostRichMessageTextOnly(t *testing.T) {
	var receivedCall bool
	api := newTestSlackClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/chat.postMessage" {
			receivedCall = true
			slackOK(w, map[string]any{
				"ok":      true,
				"ts":      "1700000000.000100",
				"channel": "C123",
			})
			return
		}
		http.Error(w, "unexpected call", 500)
	})

	action := &Action{api: api}
	content := protocol.Message{
		{Type: "text", Data: map[string]any{"text": "hello world"}},
	}

	ts, err := action.postRichMessage("C123", "", content)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ts != "1700000000.000100" {
		t.Errorf("expected ts '1700000000.000100', got %q", ts)
	}
	if !receivedCall {
		t.Error("expected PostMessage to be called")
	}
}

func TestPostRichMessageWithThread(t *testing.T) {
	var receivedTS bool
	api := newTestSlackClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/chat.postMessage" {
			_ = r.ParseForm()
			if r.FormValue("thread_ts") == "1700000000.000200" {
				receivedTS = true
			}
			slackOK(w, map[string]any{"ok": true, "ts": "1700000000.000300"})
			return
		}
		http.Error(w, "unexpected call", 500)
	})

	action := &Action{api: api}
	content := protocol.Message{
		{Type: "text", Data: map[string]any{"text": "reply"}},
	}

	_, err := action.postRichMessage("C123", "1700000000.000200", content)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !receivedTS {
		t.Error("expected thread_ts to be set")
	}
}

func TestPostRichMessageEmptyContent(t *testing.T) {
	api := newTestSlackClient(t, func(w http.ResponseWriter, _ *http.Request) {
		t.Error("should not call API for empty content")
		http.Error(w, "unexpected", 500)
	})

	action := &Action{api: api}
	_, err := action.postRichMessage("C123", "", protocol.Message{})
	if err == nil {
		t.Error("expected error for empty content")
	}
}

func TestPostRichMessageAPIError(t *testing.T) {
	api := newTestSlackClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/chat.postMessage" {
			slackError(w, "channel_not_found")
			return
		}
		http.Error(w, "unexpected call", 500)
	})

	action := &Action{api: api}
	content := protocol.Message{
		{Type: "text", Data: map[string]any{"text": "hello"}},
	}

	_, err := action.postRichMessage("C123", "", content)
	if err == nil {
		t.Error("expected error from API")
	}
}

func TestPostRichMessageWithMarkdown(t *testing.T) {
	var blocksSent bool
	api := newTestSlackClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/chat.postMessage" {
			_ = r.ParseForm()
			if r.FormValue("blocks") != "" {
				blocksSent = true
			}
			slackOK(w, map[string]any{"ok": true, "ts": "1700000000.000400"})
			return
		}
		http.Error(w, "unexpected call", 500)
	})

	action := &Action{api: api}
	content := protocol.Message{
		{Type: "markdown", Data: map[string]any{"title": "Title", "text": "body"}},
	}

	_, err := action.postRichMessage("C123", "", content)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !blocksSent {
		t.Error("expected blocks to be sent for markdown content")
	}
}

func TestPostRichMessageWithFileUpload(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "test-upload.txt")
	if err := os.WriteFile(tmpFile, []byte("test content"), 0644); err != nil {
		t.Fatal(err)
	}

	var (
		uploadURLCalled      bool
		uploadFileCalled     bool
		completeUploadCalled bool
	)

	api := newTestSlackClient(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/chat.postMessage":
			slackOK(w, map[string]any{"ok": true, "ts": "1700000000.000500", "channel": "C123"})
		case "/files.getUploadURLExternal":
			uploadURLCalled = true
			slackOK(w, map[string]any{
				"ok":         true,
				"upload_url": fmt.Sprintf("http://%s/upload", r.Host),
				"file_id":    "F123",
			})
		case "/upload":
			uploadFileCalled = true
			w.WriteHeader(http.StatusOK)
		case "/files.completeUploadExternal":
			completeUploadCalled = true
			slackOK(w, map[string]any{
				"ok":    true,
				"files": []map[string]any{{"id": "F123"}},
			})
		default:
			http.Error(w, "unexpected call: "+r.URL.Path, 500)
		}
	})

	action := &Action{api: api}
	content := protocol.Message{
		{Type: "text", Data: map[string]any{"text": "check this file"}},
		{Type: "file", Data: map[string]any{"file_id": tmpFile}},
	}

	_, err := action.postRichMessage("C123", "", content)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !uploadURLCalled {
		t.Error("expected GetUploadURLExternal to be called")
	}
	if !uploadFileCalled {
		t.Error("expected UploadToURL to be called")
	}
	if !completeUploadCalled {
		t.Error("expected CompleteUploadExternal to be called")
	}
}

func TestSendStatusMessage(t *testing.T) {
	var receivedText string
	api := newTestSlackClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/chat.postMessage" {
			_ = r.ParseForm()
			receivedText = r.FormValue("text")
			slackOK(w, map[string]any{"ok": true, "ts": "1700000000.000600"})
			return
		}
		http.Error(w, "unexpected call", 500)
	})

	action := &Action{api: api}
	ts, err := action.SendStatusMessage("C123", "Thinking...")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ts != "1700000000.000600" {
		t.Errorf("expected ts '1700000000.000600', got %q", ts)
	}
	if receivedText != "Thinking..." {
		t.Errorf("expected text 'Thinking...', got %q", receivedText)
	}
}

func TestSendStatusMessageAPIError(t *testing.T) {
	api := newTestSlackClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/chat.postMessage" {
			slackError(w, "not_in_channel")
			return
		}
		http.Error(w, "unexpected call", 500)
	})

	action := &Action{api: api}
	_, err := action.SendStatusMessage("C123", "Thinking...")
	if err == nil {
		t.Error("expected error from API")
	}
}

func TestUpdateMessageSuccess(t *testing.T) {
	var updatedTimestamp string
	api := newTestSlackClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/chat.update" {
			_ = r.ParseForm()
			updatedTimestamp = r.FormValue("ts")
			slackOK(w, map[string]any{"ok": true, "ts": updatedTimestamp})
			return
		}
		http.Error(w, "unexpected call", 500)
	})

	action := &Action{api: api}
	resp := action.UpdateMessage(protocol.Request{
		Action: protocol.UpdateMessageAction,
		Params: map[string]any{
			"topic":      "C123",
			"message_id": "1700000000.000700",
			"message": protocol.Message{
				{Type: "text", Data: map[string]any{"text": "updated content"}},
			},
		},
	})
	if resp.Status != protocol.Success {
		t.Errorf("expected success, got %s: %s", resp.Status, resp.Message)
	}
	if updatedTimestamp != "1700000000.000700" {
		t.Errorf("expected ts '1700000000.000700', got %q", updatedTimestamp)
	}
}

func TestUpdateMessageMissingTimestamp(t *testing.T) {
	action := &Action{api: nil}
	resp := action.UpdateMessage(protocol.Request{
		Action: protocol.UpdateMessageAction,
		Params: map[string]any{
			"topic":   "C123",
			"message": protocol.Message{{Type: "text", Data: map[string]any{"text": "content"}}},
		},
	})
	if resp.Status != protocol.Failed || resp.RetCode != "10003" {
		t.Errorf("expected failed/10003, got %s/%s", resp.Status, resp.RetCode)
	}
}

func TestUpdateMessageNilMessage(t *testing.T) {
	action := &Action{api: nil}
	resp := action.UpdateMessage(protocol.Request{
		Action: protocol.UpdateMessageAction,
		Params: map[string]any{
			"topic":      "C123",
			"message_id": "ts-1",
			"message":    nil,
		},
	})
	if resp.Status != protocol.Failed {
		t.Errorf("expected failed, got %s", resp.Status)
	}
}

func TestDeleteMessageSuccess(t *testing.T) {
	var deletedChannel, deletedTimestamp string
	api := newTestSlackClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/chat.delete" {
			_ = r.ParseForm()
			deletedChannel = r.FormValue("channel")
			deletedTimestamp = r.FormValue("ts")
			slackOK(w, map[string]any{"ok": true})
			return
		}
		http.Error(w, "unexpected call", 500)
	})

	action := &Action{api: api}
	resp := action.DeleteMessage(protocol.Request{
		Action: protocol.DeleteMessageAction,
		Params: map[string]any{
			"topic":      "C123",
			"message_id": "1700000000.000800",
		},
	})
	if resp.Status != protocol.Success {
		t.Errorf("expected success, got %s", resp.Status)
	}
	if deletedChannel != "C123" {
		t.Errorf("expected channel 'C123', got %q", deletedChannel)
	}
	if deletedTimestamp != "1700000000.000800" {
		t.Errorf("expected ts '1700000000.000800', got %q", deletedTimestamp)
	}
}

func TestDeleteMessageMissingTimestamp(t *testing.T) {
	action := &Action{api: nil}
	resp := action.DeleteMessage(protocol.Request{
		Action: protocol.DeleteMessageAction,
		Params: map[string]any{
			"topic": "C123",
		},
	})
	if resp.Status != protocol.Failed || resp.RetCode != "10003" {
		t.Errorf("expected failed/10003, got %s/%s", resp.Status, resp.RetCode)
	}
}

func TestSendMessageIntegration(t *testing.T) {
	var msgText, msgChannel string
	api := newTestSlackClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/chat.postMessage" {
			_ = r.ParseForm()
			msgText = r.FormValue("text")
			msgChannel = r.FormValue("channel")
			slackOK(w, map[string]any{"ok": true, "ts": "1700000000.000900", "channel": "C123"})
			return
		}
		http.Error(w, "unexpected call", 500)
	})

	action := &Action{api: api}
	resp := action.SendMessage(protocol.Request{
		Action: protocol.SendMessageAction,
		Params: map[string]any{
			"topic": "C123",
			"message": protocol.Message{
				{Type: "text", Data: map[string]any{"text": "integration test"}},
			},
		},
	})

	if resp.Status != protocol.Success {
		t.Errorf("expected success, got %s", resp.Status)
	}
	respData, ok := resp.Data.(map[string]string)
	if !ok {
		t.Fatal("expected map[string]string response data")
	}
	if respData["message_id"] != "1700000000.000900" {
		t.Errorf("expected message_id '1700000000.000900', got %q", respData["message_id"])
	}
	if respData["channel"] != "C123" {
		t.Errorf("expected channel 'C123', got %q", respData["channel"])
	}
	if msgChannel != "C123" {
		t.Errorf("expected post to channel 'C123', got %q", msgChannel)
	}
	if msgText != "integration test" {
		t.Errorf("expected text 'integration test', got %q", msgText)
	}
}
