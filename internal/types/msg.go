package types

import (
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"github.com/flowline-io/flowbot/internal/store/model"
	"github.com/flowline-io/flowbot/pkg/utils"
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

type TextListMsg struct {
	Texts []string
}

func (m TextListMsg) Convert() (KV, interface{}) {
	builder := MsgBuilder{Payload: m}
	for _, text := range m.Texts {
		builder.AppendTextLine(text, TextOption{})
	}
	return builder.Content()
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

type ImageMsg struct {
	Src         string `json:"src"`
	Width       int    `json:"width"`
	Height      int    `json:"height"`
	Alt         string `json:"alt"`
	Mime        string `json:"mime"`
	Size        int    `json:"size"`
	ImageBase64 string `json:"-"`
}

func (i ImageMsg) Convert() (KV, interface{}) {
	builder := MsgBuilder{Payload: i}
	builder.AppendImage(i.Alt, ImageOption{
		Mime:        i.Mime,
		Width:       i.Width,
		Height:      i.Height,
		ImageBase64: i.ImageBase64,
		Size:        i.Size,
	})
	return builder.Content()
}

type FileMsg struct {
	Src string `json:"src"`
	Alt string `json:"alt"`
}

func (i FileMsg) Convert() (KV, interface{}) {
	builder := MsgBuilder{Payload: i}
	builder.AppendAttachment(i.Alt, AttachmentOption{
		Mime:        "application/octet-stream",
		RelativeUrl: i.Src,
	})
	return builder.Content()
}

type VideoMsg struct {
	Src      string  `json:"src"`
	Width    int     `json:"width"`
	Height   int     `json:"height"`
	Alt      string  `json:"alt"`
	Duration float64 `json:"duration"`
}

func (i VideoMsg) Convert() (KV, interface{}) {
	builder := MsgBuilder{Payload: i}
	builder.AppendAttachment(i.Alt, AttachmentOption{
		Mime:        "video/mp4",
		RelativeUrl: i.Src,
	})
	return builder.Content()
}

type AudioMsg struct {
	Src      string  `json:"src"`
	Alt      string  `json:"alt"`
	Duration float64 `json:"duration"`
}

func (i AudioMsg) Convert() (KV, interface{}) {
	builder := MsgBuilder{Payload: i}
	builder.AppendAttachment(i.Alt, AttachmentOption{
		Mime:        "audio/mpeg",
		RelativeUrl: i.Src,
	})
	return builder.Content()
}

type ScriptMsg struct {
	Kind string `json:"kind"`
	Code string `json:"code"`
}

func (a ScriptMsg) Convert() (KV, interface{}) {
	builder := MsgBuilder{Payload: a}
	builder.AppendText(a.Code, TextOption{
		IsCode: true,
	})
	return builder.Content()
}

type ActionMsg struct {
	ID     string   `json:"id"`
	Title  string   `json:"title"`
	Option []string `json:"option"`
	Value  string   `json:"value"`
}

func (a ActionMsg) Convert() (KV, interface{}) {
	builder := MsgBuilder{Payload: a}
	builder.AppendTextLine(a.Title, TextOption{IsBold: true})
	for _, option := range a.Option {
		builder.AppendText(option, TextOption{IsButton: true, ButtonDataAct: "pub", ButtonDataName: option, ButtonDataVal: option})
	}
	return builder.Content()
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

type LocationMsg struct {
	Longitude float64 `json:"longitude"`
	Latitude  float64 `json:"latitude"`
	Address   string  `json:"address"`
}

func (a LocationMsg) Convert() (KV, interface{}) {
	builder := MsgBuilder{Payload: a}
	builder.AppendTextLine(fmt.Sprintf("Location (%f, %f)", a.Longitude, a.Latitude), TextOption{})
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

type DigitMsg struct {
	Title string `json:"title"`
	Digit int    `json:"digit"`
}

func (a DigitMsg) Convert() (KV, interface{}) {
	builder := MsgBuilder{Payload: a}
	builder.AppendText(fmt.Sprintf("Counter %s : %d", a.Title, a.Digit), TextOption{})
	return builder.Content()
}

type OkrMsg struct {
	Title     string             `json:"title"`
	Objective *model.Objective   `json:"objective"`
	KeyResult []*model.KeyResult `json:"key_result"`
}

func (o OkrMsg) Convert() (KV, interface{}) {
	return nil, nil
}

type InfoMsg struct {
	Title string      `json:"title"`
	Model interface{} `json:"model"`
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
		d, _ := json.Marshal(i.Model)
		_ = json.Unmarshal(d, &m)
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

type TodoMsg struct {
	Title string        `json:"title"`
	Todo  []*model.Todo `json:"todo"`
}

func (t TodoMsg) Convert() (KV, interface{}) {
	if len(t.Todo) == 0 {
		return nil, "empty"
	}
	builder := MsgBuilder{Payload: t}
	builder.AppendTextLine("Todo", TextOption{IsBold: true})
	for i, todo := range t.Todo {
		builder.AppendTextLine(fmt.Sprintf("%d: %s", i+1, todo.Content), TextOption{})
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

func Convert(payloads []MsgPayload) ([]KV, []any) {
	var heads []KV
	var contents []interface{}
	for _, item := range payloads {
		head, content := item.Convert()
		heads = append(heads, head)
		contents = append(contents, content)
	}
	return heads, contents
}

type RepoMsg struct {
	ID               *int64     `json:"id,omitempty"`
	NodeID           *string    `json:"node_id,omitempty"`
	Name             *string    `json:"name,omitempty"`
	FullName         *string    `json:"full_name,omitempty"`
	Description      *string    `json:"description,omitempty"`
	Homepage         *string    `json:"homepage,omitempty"`
	CreatedAt        *time.Time `json:"created_at,omitempty"`
	PushedAt         *time.Time `json:"pushed_at,omitempty"`
	UpdatedAt        *time.Time `json:"updated_at,omitempty"`
	HTMLURL          *string    `json:"html_url,omitempty"`
	Language         *string    `json:"language,omitempty"`
	Fork             *bool      `json:"fork,omitempty"`
	ForksCount       *int       `json:"forks_count,omitempty"`
	NetworkCount     *int       `json:"network_count,omitempty"`
	OpenIssuesCount  *int       `json:"open_issues_count,omitempty"`
	StargazersCount  *int       `json:"stargazers_count,omitempty"`
	SubscribersCount *int       `json:"subscribers_count,omitempty"`
	WatchersCount    *int       `json:"watchers_count,omitempty"`
	Size             *int       `json:"size,omitempty"`
	Topics           []string   `json:"topics,omitempty"`
	Archived         *bool      `json:"archived,omitempty"`
	Disabled         *bool      `json:"disabled,omitempty"`
}

func (i RepoMsg) Convert() (KV, interface{}) {
	builder := MsgBuilder{Payload: i}
	// title
	builder.AppendTextLine(*i.FullName, TextOption{IsBold: true})

	var m map[string]interface{}
	d, _ := json.Marshal(i)
	_ = json.Unmarshal(d, &m)

	// sort keys
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		builder.AppendText(fmt.Sprintf("%s: ", k), TextOption{IsBold: true})
		s := toString(m[k])
		builder.AppendText(s, TextOption{IsLink: utils.IsUrl(s)})
		builder.AppendText("\n", TextOption{})
	}

	return builder.Content()
}

type CardMsg struct {
	Name  string
	Image string
	URI   string
	Text  string
}

func (m CardMsg) Convert() (KV, interface{}) {
	builder := MsgBuilder{Payload: m}
	builder.AppendText(m.Name, TextOption{IsBold: true})
	builder.AppendText(" ", TextOption{})
	builder.AppendText(m.URI, TextOption{IsLink: true})
	return builder.Content()
}

type CardListMsg struct {
	Cards []CardMsg
}

func (m CardListMsg) Convert() (KV, interface{}) {
	builder := MsgBuilder{Payload: m}
	for _, card := range m.Cards {
		builder.AppendText(card.Name, TextOption{IsBold: true})
		builder.AppendText(" ", TextOption{})
		builder.AppendTextLine(card.URI, TextOption{IsLink: true})
	}
	return builder.Content()
}

type CrateMsg struct {
	ID            string `json:"id,omitempty"`
	Name          string `json:"name,omitempty"`
	Description   string `json:"description,omitempty"`
	Documentation string `json:"documentation,omitempty"`
	Homepage      string `json:"homepage,omitempty"`
	Repository    string `json:"repository,omitempty"`
	NewestVersion string `json:"newest_version,omitempty"`
	Downloads     int    `json:"downloads,omitempty"`
}

func (c CrateMsg) Convert() (KV, interface{}) {
	builder := MsgBuilder{Payload: c}
	// title
	builder.AppendTextLine(c.Name, TextOption{IsBold: true})

	var m map[string]interface{}
	d, _ := json.Marshal(c)
	_ = json.Unmarshal(d, &m)

	// sort keys
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		builder.AppendText(fmt.Sprintf("%s: ", k), TextOption{IsBold: true})
		s := toString(m[k])
		builder.AppendText(s, TextOption{IsLink: utils.IsUrl(s)})
		builder.AppendText("\n", TextOption{})
	}

	// info page
	builder.AppendText("crates.io page: ", TextOption{IsBold: true})
	builder.AppendText(fmt.Sprintf("https://crates.io/crates/%s", c.ID), TextOption{IsLink: true})
	builder.AppendText("\n", TextOption{})

	return builder.Content()
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

type LinkListMsg struct {
	Links []LinkMsg
}

func (m LinkListMsg) Convert() (KV, interface{}) {
	builder := MsgBuilder{Payload: m}
	for _, card := range m.Links {
		builder.AppendText(card.Title, TextOption{IsBold: true})
		builder.AppendText(" ", TextOption{})
		builder.AppendTextLine(card.Url, TextOption{IsLink: true})
	}
	return builder.Content()
}

type QuestionMsg struct {
	Id         int
	Title      string
	Slug       string
	Difficulty int
	Source     string
}

func (m QuestionMsg) Convert() (KV, interface{}) {
	builder := MsgBuilder{Payload: m}
	builder.AppendTextLine(fmt.Sprintf("[%s:%d] %s", m.Source, m.Id, m.Title), TextOption{})
	if m.Source == "leetcode" {
		builder.AppendText(fmt.Sprintf("https://leetcode.com/problems/%s/", m.Slug), TextOption{IsLink: true})
	}
	return builder.Content()
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
