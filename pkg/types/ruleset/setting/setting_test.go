package setting

import (
	"testing"

	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/stretchr/testify/assert"
)

func TestRow_ID(t *testing.T) {
	r := Row{Key: "token"}
	assert.Equal(t, "token", r.ID())
}

func TestRow_TYPE(t *testing.T) {
	r := Row{Key: "token"}
	assert.Equal(t, types.SettingRule, r.TYPE())
}

func TestRow_EmptyKey(t *testing.T) {
	r := Row{}
	assert.Equal(t, "", r.ID())
}

func TestRule_Creation(t *testing.T) {
	rules := Rule{
		{Key: "token", Type: types.FormFieldText, Title: "API Token", Detail: "Your API token"},
		{Key: "zone_id", Type: types.FormFieldText, Title: "Zone ID", Detail: "Cloudflare zone ID"},
		{Key: "account_id", Type: types.FormFieldText, Title: "Account ID", Detail: "Cloudflare account ID"},
	}
	assert.Len(t, rules, 3)
}

func TestRule_FieldTypes(t *testing.T) {
	tests := []struct {
		name     string
		row      Row
		wantType types.FormFieldType
	}{
		{
			name:     "text field",
			row:      Row{Key: "secret", Type: types.FormFieldText, Title: "Secret"},
			wantType: types.FormFieldText,
		},
		{
			name:     "number field",
			row:      Row{Key: "number", Type: types.FormFieldNumber, Title: "Number"},
			wantType: types.FormFieldNumber,
		},
		{
			name:     "password field",
			row:      Row{Key: "pass", Type: types.FormFieldPassword, Title: "Password"},
			wantType: types.FormFieldPassword,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.wantType, tt.row.Type)
			assert.Equal(t, types.SettingRule, tt.row.TYPE())
		})
	}
}

func TestRule_Empty(t *testing.T) {
	rules := Rule{}
	assert.Len(t, rules, 0)
}

func TestRow_AllFields(t *testing.T) {
	r := Row{
		Key:    "api_key",
		Type:   types.FormFieldText,
		Title:  "API Key",
		Detail: "Enter your API key for authentication",
	}
	assert.Equal(t, "api_key", r.Key)
	assert.Equal(t, types.FormFieldText, r.Type)
	assert.Equal(t, "API Key", r.Title)
	assert.Equal(t, "Enter your API key for authentication", r.Detail)
}
