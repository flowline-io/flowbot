package types

import (
	"testing"

	"github.com/bytedance/sonic"

	"github.com/stretchr/testify/assert"
)

func TestTypeOf(t *testing.T) {
	tests := []struct {
		name string
		msg  MsgPayload
		want string
	}{
		{
			name: "TextMsg",
			msg:  TextMsg{Text: "hello"},
			want: "TextMsg",
		},
		{
			name: "LinkMsg",
			msg:  LinkMsg{Title: "t", Url: "u"},
			want: "LinkMsg",
		},
		{
			name: "TableMsg",
			msg:  TableMsg{},
			want: "TableMsg",
		},
		{
			name: "InfoMsg",
			msg:  InfoMsg{},
			want: "InfoMsg",
		},
		{
			name: "ChartMsg",
			msg:  ChartMsg{},
			want: "ChartMsg",
		},
		{
			name: "HtmlMsg",
			msg:  HtmlMsg{},
			want: "HtmlMsg",
		},
		{
			name: "MarkdownMsg",
			msg:  MarkdownMsg{},
			want: "MarkdownMsg",
		},
		{
			name: "InstructMsg",
			msg:  InstructMsg{},
			want: "InstructMsg",
		},
		{
			name: "FormMsg",
			msg:  FormMsg{},
			want: "FormMsg",
		},
		{
			name: "KVMsg",
			msg:  KVMsg{},
			want: "KVMsg",
		},
		{
			name: "EmptyMsg",
			msg:  EmptyMsg{},
			want: "EmptyMsg",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, TypeOf(tt.msg))
		})
	}
}

func TestToPayload_TextMsg(t *testing.T) {
	t.Run("TextMsg", func(t *testing.T) {
		src, _ := sonic.Marshal(TextMsg{Text: "hello"})
		payload := ToPayload("TextMsg", src)
		msg, ok := payload.(TextMsg)
		assert.True(t, ok)
		assert.Equal(t, "hello", msg.Text)
	})
}

func TestToPayload_LinkMsg(t *testing.T) {
	t.Run("LinkMsg", func(t *testing.T) {
		src, _ := sonic.Marshal(LinkMsg{Title: "t", Url: "http://x"})
		payload := ToPayload("LinkMsg", src)
		msg, ok := payload.(LinkMsg)
		assert.True(t, ok)
		assert.Equal(t, "t", msg.Title)
	})
}

func TestToPayload_TableMsg(t *testing.T) {
	t.Run("TableMsg", func(t *testing.T) {
		src, _ := sonic.Marshal(TableMsg{Title: "tbl"})
		payload := ToPayload("TableMsg", src)
		msg, ok := payload.(TableMsg)
		assert.True(t, ok)
		assert.Equal(t, "tbl", msg.Title)
	})
}

func TestToPayload_InfoMsg(t *testing.T) {
	t.Run("InfoMsg", func(t *testing.T) {
		src, _ := sonic.Marshal(InfoMsg{Title: "info"})
		payload := ToPayload("InfoMsg", src)
		msg, ok := payload.(InfoMsg)
		assert.True(t, ok)
		assert.Equal(t, "info", msg.Title)
	})
}

func TestToPayload_ChartMsg(t *testing.T) {
	t.Run("ChartMsg", func(t *testing.T) {
		src, _ := sonic.Marshal(ChartMsg{Title: "chart"})
		payload := ToPayload("ChartMsg", src)
		msg, ok := payload.(ChartMsg)
		assert.True(t, ok)
		assert.Equal(t, "chart", msg.Title)
	})
}

func TestToPayload_KVMsg(t *testing.T) {
	t.Run("KVMsg", func(t *testing.T) {
		src, _ := sonic.Marshal(KVMsg{"key": "val"})
		payload := ToPayload("KVMsg", src)
		msg, ok := payload.(KVMsg)
		assert.True(t, ok)
		assert.Equal(t, "val", msg["key"])
	})
}

func TestToPayload_Unknown(t *testing.T) {
	t.Run("unknown message type", func(t *testing.T) {
		payload := ToPayload("UnknownMsg", []byte(`{}`))
		assert.Nil(t, payload)
	})
}

func TestMsgPayload_Convert(t *testing.T) {
	tests := []struct {
		name    string
		payload MsgPayload
		want    MsgPayload
	}{
		{
			name:    "TextMsg",
			payload: TextMsg{Text: "hello"},
			want:    TextMsg{Text: "hello"},
		},
		{
			name:    "LinkMsg",
			payload: LinkMsg{Title: "t", Url: "u"},
			want:    LinkMsg{Title: "t", Url: "u"},
		},
		{
			name:    "KVMsg",
			payload: KVMsg{"a": "b"},
			want:    KVMsg{"a": "b"},
		},
		{
			name:    "EmptyMsg",
			payload: EmptyMsg{},
			want:    EmptyMsg{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.payload.Convert())
		})
	}
}

func TestFormFieldConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant FormFieldType
		want     FormFieldType
	}{
		{
			name:     "text",
			constant: FormFieldText,
			want:     FormFieldType("text"),
		},
		{
			name:     "password",
			constant: FormFieldPassword,
			want:     FormFieldType("password"),
		},
		{
			name:     "number",
			constant: FormFieldNumber,
			want:     FormFieldType("number"),
		},
		{
			name:     "radio",
			constant: FormFieldRadio,
			want:     FormFieldType("radio"),
		},
		{
			name:     "checkbox",
			constant: FormFieldCheckbox,
			want:     FormFieldType("checkbox"),
		},
		{
			name:     "select",
			constant: FormFieldSelect,
			want:     FormFieldType("select"),
		},
		{
			name:     "textarea",
			constant: FormFieldTextarea,
			want:     FormFieldType("textarea"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.constant)
		})
	}
}

func TestFormFieldValueTypeConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant FormFieldValueType
		want     FormFieldValueType
	}{
		{
			name:     "string",
			constant: FormFieldValueString,
			want:     FormFieldValueType("string"),
		},
		{
			name:     "bool",
			constant: FormFieldValueBool,
			want:     FormFieldValueType("bool"),
		},
		{
			name:     "int64",
			constant: FormFieldValueInt64,
			want:     FormFieldValueType("int64"),
		},
		{
			name:     "float64",
			constant: FormFieldValueFloat64,
			want:     FormFieldValueType("float64"),
		},
		{
			name:     "string_slice",
			constant: FormFieldValueStringSlice,
			want:     FormFieldValueType("string_slice"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.constant)
		})
	}
}
