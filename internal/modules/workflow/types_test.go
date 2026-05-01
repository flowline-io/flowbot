package workflow

import (
	"testing"

	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/stretchr/testify/assert"
)

func TestRuleType_Fields(t *testing.T) {
	r := rule{
		Bot:   "test_bot",
		Id:    "test_id",
		Title: "Test Title",
		Desc:  "Test Description",
		InputSchema: []types.FormField{
			{Key: "input1", Type: types.FormFieldText},
		},
		OutputSchema: []types.FormField{
			{Key: "output1", Type: types.FormFieldText},
		},
	}

	assert.Equal(t, "test_bot", r.Bot)
	assert.Equal(t, "test_id", r.Id)
	assert.Equal(t, "Test Title", r.Title)
	assert.Equal(t, "Test Description", r.Desc)
	assert.Len(t, r.InputSchema, 1)
	assert.Len(t, r.OutputSchema, 1)
}

func TestRuleType_EmptyFields(t *testing.T) {
	r := rule{}
	assert.Empty(t, r.Bot)
	assert.Empty(t, r.Id)
	assert.Empty(t, r.Title)
	assert.Empty(t, r.Desc)
	assert.Nil(t, r.InputSchema)
	assert.Nil(t, r.OutputSchema)
}
