package app

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/flowline-io/flowbot/pkg/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRunControlHint(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		okHint     string
		failLabel  string
		wantSubstr string
	}{
		{name: "success returns ok hint", okHint: "Canceled — ready for input", wantSubstr: "Canceled — ready for input"},
		{name: "cancel failure surfaces error", err: errors.New("timeout"), okHint: "Canceled — ready for input", failLabel: "Cancel failed — server may still be running", wantSubstr: "Cancel failed"},
		{name: "confirm failure surfaces error", err: errors.New("conflict"), okHint: "Ctrl+C cancel run", failLabel: "Confirm failed — server may still be waiting", wantSubstr: "Confirm failed"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := runControlHint(tt.err, tt.okHint, tt.failLabel)
			assert.Contains(t, got, tt.wantSubstr)
			if tt.err == nil {
				assert.Equal(t, tt.okHint, got)
			}
		})
	}
}

func TestSessionCacheHint(t *testing.T) {
	tests := []struct {
		name    string
		err     error
		want    string
		wantSub string
	}{
		{name: "nil error is silent", err: nil, want: ""},
		{name: "write failure warns", err: errors.New("permission denied"), wantSub: "persist session locally"},
		{name: "clear failure warns", err: errors.New("missing file"), wantSub: "missing file"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := sessionCacheHint(tt.err)
			if tt.want != "" {
				assert.Equal(t, tt.want, got)
				return
			}
			if tt.wantSub != "" {
				assert.Contains(t, got, tt.wantSub)
			}
		})
	}
}

func TestSubmitConfirmChoiceReportsAPIError(t *testing.T) {
	tests := []struct {
		name    string
		status  int
		wantSub string
	}{
		{name: "server error on confirm", status: http.StatusInternalServerError, wantSub: "Confirm failed"},
		{name: "conflict on confirm", status: http.StatusConflict, wantSub: "Confirm failed"},
		{name: "success keeps streaming hint", status: http.StatusNoContent, wantSub: "Ctrl+C cancel run"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/confirm") {
					if tt.status >= 300 {
						w.WriteHeader(tt.status)
						_, _ = w.Write([]byte(`{"error":"boom"}`))
						return
					}
					w.WriteHeader(http.StatusNoContent)
					return
				}
				http.NotFound(w, r)
			}))
			t.Cleanup(srv.Close)

			cl := client.NewClient(srv.URL, "token")
			m := NewModel(cl, "default")
			m.sessionID = "sess-1"
			m.confirm.id = "confirm-1"
			m.phase = PhaseConfirming

			cmd := m.submitConfirmChoice(confirmChoice{approved: true, mode: client.ConfirmModeOnce})
			require.NotNil(t, cmd)
			msg := cmd()
			rc, ok := msg.(runControlMsg)
			require.True(t, ok)
			_, focus := m.updateRunControl(rc)
			assert.Contains(t, m.hint, tt.wantSub)
			assert.Equal(t, PhaseStreaming, m.phase)
			assert.NotNil(t, focus)
		})
	}
}
