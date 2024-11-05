package types

import (
	"fmt"
	"sort"
	"time"

	"github.com/flowline-io/flowbot/internal/store/model"
	jsoniter "github.com/json-iterator/go"
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
	builder := MsgBuilder{Payload: a}
	if a.Title != "" {
		builder.AppendTextLine(a.Title, TextOption{IsBold: true})
		builder.AppendText(a.Url, TextOption{IsLink: true})
	} else {
		builder.AppendText(a.Url, TextOption{IsLink: true})
	}

	return builder.Content()
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
	builder := MsgBuilder{Payload: i}
	// title
	builder.AppendTextLine(i.Title, TextOption{})
	// model
	var m map[string]interface{}
	switch v := i.Model.(type) {
	case map[string]interface{}:
		m = v
	default:
		d, _ := jsoniter.Marshal(i.Model)
		_ = jsoniter.Unmarshal(d, &m)
	}

	// sort keys
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		builder.AppendText(fmt.Sprintf("%s: ", k), TextOption{IsBold: true})
		builder.AppendText(toString(m[k]), TextOption{})
		builder.AppendText("\n", TextOption{})
	}

	return builder.Content()
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
