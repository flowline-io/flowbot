package types

import (
	jsoniter "github.com/json-iterator/go"
	"reflect"
	"time"

	"github.com/flowline-io/flowbot/internal/store/model"
)

type FormFieldType string

const (
	FormFieldText     FormFieldType = "text"
	FormFieldPassword FormFieldType = "password"
	FormFieldNumber   FormFieldType = "number"
	FormFieldColor    FormFieldType = "color"
	FormFieldFile     FormFieldType = "file"
	FormFieldMonth    FormFieldType = "month"
	FormFieldDate     FormFieldType = "date"
	FormFieldTime     FormFieldType = "time"
	FormFieldEmail    FormFieldType = "email"
	FormFieldUrl      FormFieldType = "url"
	FormFieldRadio    FormFieldType = "radio"
	FormFieldCheckbox FormFieldType = "checkbox"
	FormFieldRange    FormFieldType = "range"
	FormFieldSelect   FormFieldType = "select"
	FormFieldTextarea FormFieldType = "textarea"
	FormFieldHidden   FormFieldType = "hidden"
)

type FormFieldValueType string

const (
	FormFieldValueString       FormFieldValueType = "string"
	FormFieldValueBool         FormFieldValueType = "bool"
	FormFieldValueInt64        FormFieldValueType = "int64"
	FormFieldValueFloat64      FormFieldValueType = "float64"
	FormFieldValueStringSlice  FormFieldValueType = "string_slice"
	FormFieldValueInt64Slice   FormFieldValueType = "int64_slice"
	FormFieldValueFloat64Slice FormFieldValueType = "float64_slice"
)

type TextMsg struct {
	Text string `json:"text"`
}

func (t TextMsg) Convert() (KV, interface{}) {
	return nil, t.Text
}

type FormMsg struct {
	ID    string      `json:"id"`
	Title string      `json:"title"`
	Field []FormField `json:"field"`
}

func (a FormMsg) Convert() (KV, interface{}) {
	return nil, nil
}

type FormField struct {
	Type        FormFieldType      `json:"type"`
	Key         string             `json:"key"`
	Value       interface{}        `json:"value"`
	ValueType   FormFieldValueType `json:"value_type"`
	Label       string             `json:"label"`
	Placeholder string             `json:"placeholder"`
	Option      []string           `json:"option"`
	Rule        string             `json:"rule"`
}

type LinkMsg struct {
	Title string `json:"title"`
	Cover string `json:"cover"`
	Url   string `json:"url"`
}

func (a LinkMsg) Convert() (KV, interface{}) {
	return nil, a
}

type TableMsg struct {
	Title  string          `json:"title"`
	Header []string        `json:"header"`
	Row    [][]interface{} `json:"row"`
}

func (t TableMsg) Convert() (KV, interface{}) {
	return nil, nil
}

type InfoMsg struct {
	Title string      `json:"title"`
	Model interface{} `json:"model,omitempty"`
}

func (i InfoMsg) Convert() (KV, interface{}) {
	return nil, i
}

type ChartMsg struct {
	Title    string    `json:"title"`
	SubTitle string    `json:"sub_title"`
	XAxis    []string  `json:"x_axis"`
	Series   []float64 `json:"series"`
}

func (t ChartMsg) Convert() (KV, interface{}) {
	return nil, nil
}

type HtmlMsg struct {
	Raw string
}

func (m HtmlMsg) Convert() (KV, interface{}) {
	return nil, nil
}

type MarkdownMsg struct {
	Title string `json:"title"`
	Raw   string `json:"raw"`
}

func (m MarkdownMsg) Convert() (KV, interface{}) {
	return nil, nil
}

type InstructMsg struct {
	No       string
	Object   model.InstructObject
	Bot      string
	Flag     string
	Content  KV
	Priority model.InstructPriority
	State    model.InstructState
	ExpireAt time.Time
}

func (t InstructMsg) Convert() (KV, interface{}) {
	return nil, nil
}

type KVMsg map[string]any

func (t KVMsg) Convert() (KV, interface{}) {
	return nil, nil
}

func TypeOf(payload MsgPayload) string {
	t := reflect.TypeOf(payload)
	return t.Name()
}

func ToPayload(typ string, src []byte) MsgPayload {
	switch typ {
	case "TextMsg":
		var r TextMsg
		_ = jsoniter.Unmarshal(src, &r)
		return r
	case "LinkMsg":
		var r LinkMsg
		_ = jsoniter.Unmarshal(src, &r)
		return r
	case "TableMsg":
		var r TableMsg
		_ = jsoniter.Unmarshal(src, &r)
		return r
	case "InfoMsg":
		var r InfoMsg
		_ = jsoniter.Unmarshal(src, &r)
		return r
	case "ChartMsg":
		var r ChartMsg
		_ = jsoniter.Unmarshal(src, &r)
		return r
	case "KVMsg":
		var r KVMsg
		_ = jsoniter.Unmarshal(src, &r)
		return r
	}
	return nil
}
