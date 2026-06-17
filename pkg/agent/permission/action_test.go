package permission_test

import (
	"testing"

	"github.com/flowline-io/flowbot/pkg/agent/permission"
	"github.com/stretchr/testify/assert"
)

func TestParseAction(t *testing.T) {
	tests := []struct {
		name  string
		raw   string
		want  permission.Action
		valid bool
	}{
		{name: "allow", raw: "allow", want: permission.ActionAllow, valid: true},
		{name: "ask", raw: "ask", want: permission.ActionAsk, valid: true},
		{name: "deny", raw: "deny", want: permission.ActionDeny, valid: true},
		{name: "invalid", raw: "maybe", valid: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := permission.ParseAction(tt.raw)
			assert.Equal(t, tt.valid, ok)
			if tt.valid {
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestStricter(t *testing.T) {
	tests := []struct {
		name string
		a, b permission.Action
		want permission.Action
	}{
		{name: "deny wins", a: permission.ActionAllow, b: permission.ActionDeny, want: permission.ActionDeny},
		{name: "ask over allow", a: permission.ActionAllow, b: permission.ActionAsk, want: permission.ActionAsk},
		{name: "equal", a: permission.ActionAsk, b: permission.ActionAsk, want: permission.ActionAsk},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, permission.Stricter(tt.a, tt.b))
		})
	}
}
