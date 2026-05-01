package types

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTypeOf(t *testing.T) {
	assert.Equal(t, "TextMsg", TypeOf(TextMsg{Text: "hello"}))
	assert.Equal(t, "LinkMsg", TypeOf(LinkMsg{Title: "t", Url: "u"}))
	assert.Equal(t, "TableMsg", TypeOf(TableMsg{}))
	assert.Equal(t, "InfoMsg", TypeOf(InfoMsg{}))
	assert.Equal(t, "ChartMsg", TypeOf(ChartMsg{}))
	assert.Equal(t, "HtmlMsg", TypeOf(HtmlMsg{}))
	assert.Equal(t, "MarkdownMsg", TypeOf(MarkdownMsg{}))
	assert.Equal(t, "InstructMsg", TypeOf(InstructMsg{}))
	assert.Equal(t, "FormMsg", TypeOf(FormMsg{}))
	assert.Equal(t, "KVMsg", TypeOf(KVMsg{}))
	assert.Equal(t, "EmptyMsg", TypeOf(EmptyMsg{}))
}

func TestToPayload_TextMsg(t *testing.T) {
	src, _ := json.Marshal(TextMsg{Text: "hello"})
	payload := ToPayload("TextMsg", src)
	msg, ok := payload.(TextMsg)
	assert.True(t, ok)
	assert.Equal(t, "hello", msg.Text)
}

func TestToPayload_LinkMsg(t *testing.T) {
	src, _ := json.Marshal(LinkMsg{Title: "t", Url: "http://x"})
	payload := ToPayload("LinkMsg", src)
	msg, ok := payload.(LinkMsg)
	assert.True(t, ok)
	assert.Equal(t, "t", msg.Title)
}

func TestToPayload_TableMsg(t *testing.T) {
	src, _ := json.Marshal(TableMsg{Title: "tbl"})
	payload := ToPayload("TableMsg", src)
	msg, ok := payload.(TableMsg)
	assert.True(t, ok)
	assert.Equal(t, "tbl", msg.Title)
}

func TestToPayload_InfoMsg(t *testing.T) {
	src, _ := json.Marshal(InfoMsg{Title: "info"})
	payload := ToPayload("InfoMsg", src)
	msg, ok := payload.(InfoMsg)
	assert.True(t, ok)
	assert.Equal(t, "info", msg.Title)
}

func TestToPayload_ChartMsg(t *testing.T) {
	src, _ := json.Marshal(ChartMsg{Title: "chart"})
	payload := ToPayload("ChartMsg", src)
	msg, ok := payload.(ChartMsg)
	assert.True(t, ok)
	assert.Equal(t, "chart", msg.Title)
}

func TestToPayload_KVMsg(t *testing.T) {
	src, _ := json.Marshal(KVMsg{"key": "val"})
	payload := ToPayload("KVMsg", src)
	msg, ok := payload.(KVMsg)
	assert.True(t, ok)
	assert.Equal(t, "val", msg["key"])
}

func TestToPayload_Unknown(t *testing.T) {
	payload := ToPayload("UnknownMsg", []byte(`{}`))
	assert.Nil(t, payload)
}

func TestMsgPayload_Convert(t *testing.T) {
	txt := TextMsg{Text: "hello"}
	assert.Equal(t, txt, txt.Convert())

	link := LinkMsg{Title: "t", Url: "u"}
	assert.Equal(t, link, link.Convert())

	kv := KVMsg{"a": "b"}
	assert.Equal(t, kv, kv.Convert())

	empty := EmptyMsg{}
	assert.Equal(t, empty, empty.Convert())
}

func TestFormFieldConstants(t *testing.T) {
	assert.Equal(t, FormFieldType("text"), FormFieldText)
	assert.Equal(t, FormFieldType("password"), FormFieldPassword)
	assert.Equal(t, FormFieldType("number"), FormFieldNumber)
	assert.Equal(t, FormFieldType("radio"), FormFieldRadio)
	assert.Equal(t, FormFieldType("checkbox"), FormFieldCheckbox)
	assert.Equal(t, FormFieldType("select"), FormFieldSelect)
	assert.Equal(t, FormFieldType("textarea"), FormFieldTextarea)
}

func TestFormFieldValueTypeConstants(t *testing.T) {
	assert.Equal(t, FormFieldValueType("string"), FormFieldValueString)
	assert.Equal(t, FormFieldValueType("bool"), FormFieldValueBool)
	assert.Equal(t, FormFieldValueType("int64"), FormFieldValueInt64)
	assert.Equal(t, FormFieldValueType("float64"), FormFieldValueFloat64)
	assert.Equal(t, FormFieldValueType("string_slice"), FormFieldValueStringSlice)
}
