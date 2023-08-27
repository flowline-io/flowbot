package types

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math"
	"reflect"
	"strconv"
	"strings"
	"unicode/utf8"
)

var commonHead = map[string]interface{}{
	"mime": "text/x-drafty",
}

type FmtMessage struct {
	At  int    `json:"at,omitempty"`
	Len int    `json:"len,omitempty"`
	Tp  string `json:"tp,omitempty"`
	Key int    `json:"key,omitempty"`
}

type EntMessage struct {
	Tp   string  `json:"tp,omitempty"`
	Data EntData `json:"data"`
}

type EntData struct {
	Mime   string      `json:"mime,omitempty"`
	Val    interface{} `json:"val,omitempty"`
	Url    string      `json:"url,omitempty"`
	Ref    string      `json:"ref,omitempty"`
	Width  int         `json:"width,omitempty"`
	Height int         `json:"height,omitempty"`
	Name   string      `json:"name,omitempty"`
	Size   int         `json:"size,omitempty"`
	Act    string      `json:"act,omitempty"`
}

type ChatMessage struct {
	Text        string       `json:"txt,omitempty"`
	Fmt         []FmtMessage `json:"fmt,omitempty"`
	Ent         []EntMessage `json:"ent,omitempty"`
	IsPlainText bool         `json:"-"`
	MessageType string       `json:"-"`

	Src MsgPayload `json:"src,omitempty"`
	Tye string     `json:"tye,omitempty"`
}

// GetFormattedText Get original text message, inlude original '\n'
func (c ChatMessage) GetFormattedText() string {
	if c.Text == "" {
		return ""
	}
	t := []byte(c.Text)
	for _, item := range c.Fmt {
		if item.Tp == "BR" {
			t[item.At] = '\n'
		}
	}
	return string(t)
}

// GetEntDatas get entity from chat message by entity type
func (c ChatMessage) GetEntDatas(tp string) []EntData {
	var ret []EntData
	for _, item := range c.Ent {
		if item.Tp == tp {
			ret = append(ret, item.Data)
		}
	}
	return ret
}

// GetMentions get mentioned users
func (c ChatMessage) GetMentions() []EntData {
	return c.GetEntDatas("MN")
}

// GetImages get images
func (c ChatMessage) GetImages() []EntData {
	return c.GetEntDatas("IM")
}

// GetHashTags get hashtags
func (c ChatMessage) GetHashTags() []EntData {
	return c.GetEntDatas("HT")
}

// GetLinks get links
func (c ChatMessage) GetLinks() []EntData {
	return c.GetEntDatas("LN")
}

// GetGenericAttachment get generic attachment
func (c ChatMessage) GetGenericAttachment() []EntData {
	return c.GetEntDatas("EX")
}

func (c ChatMessage) Content() (map[string]interface{}, interface{}) {
	if c.IsPlainText {
		return nil, c.Text
	}
	d, err := json.Marshal(c)
	if err != nil {
		return nil, ""
	}

	var res map[string]interface{}
	err = json.Unmarshal(d, &res)
	if err != nil {
		return nil, ""
	}

	return map[string]interface{}{
		"mime": "text/x-drafty",
	}, res
}

type MsgBuilder struct {
	Payload MsgPayload
	Message ChatMessage
}

// AppendText Append text message to build message
func (m *MsgBuilder) AppendText(text string, opt TextOption) {
	baseLen := utf8.RuneCountInString(m.Message.Text)
	m.Message.Text += text
	if strings.Contains(text, "\n") {
		for i := 0; i < utf8.RuneCountInString(text); i++ {
			if text[i] == '\n' {
				_fmt := FmtMessage{
					At:  baseLen + i,
					Tp:  "BR",
					Len: 1,
				}
				m.Message.Fmt = append(m.Message.Fmt, _fmt)
			}
		}
	}

	leftLen := baseLen + (utf8.RuneCountInString(text) - utf8.RuneCountInString(strings.TrimLeft(text, "\t\n\v\f\r ")))
	subLen := utf8.RuneCountInString(text) - utf8.RuneCountInString(strings.TrimRight(text, "\t\n\v\f\r "))
	validLen := utf8.RuneCountInString(m.Message.Text) - leftLen - subLen

	if opt.IsBold {
		_fmt := FmtMessage{
			Tp:  "ST",
			At:  leftLen,
			Len: validLen,
		}
		m.Message.Fmt = append(m.Message.Fmt, _fmt)
	}
	if opt.IsItalic {
		_fmt := FmtMessage{
			Tp:  "EM",
			At:  leftLen,
			Len: validLen,
		}
		m.Message.Fmt = append(m.Message.Fmt, _fmt)
	}
	if opt.IsDeleted {
		_fmt := FmtMessage{
			Tp:  "DL",
			At:  leftLen,
			Len: validLen,
		}
		m.Message.Fmt = append(m.Message.Fmt, _fmt)
	}
	if opt.IsCode {
		_fmt := FmtMessage{
			Tp:  "CO",
			At:  leftLen,
			Len: validLen,
		}
		m.Message.Fmt = append(m.Message.Fmt, _fmt)
	}
	if opt.IsLink {
		_fmt := FmtMessage{
			At:  leftLen,
			Len: validLen,
			Key: len(m.Message.Ent),
		}
		m.Message.Fmt = append(m.Message.Fmt, _fmt)
		url := strings.ToLower(strings.TrimSpace(text))
		if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
			url = "http://" + strings.TrimSpace(text)
		}
		ent := EntMessage{
			Tp: "LN",
			Data: EntData{
				Url: url,
			},
		}
		m.Message.Ent = append(m.Message.Ent, ent)
	}
	if opt.IsMention {
		_fmt := FmtMessage{
			At:  leftLen,
			Len: validLen,
			Key: len(m.Message.Ent),
		}
		m.Message.Fmt = append(m.Message.Fmt, _fmt)
		mentionName := substr(strings.TrimSpace(text), 1, len(strings.TrimSpace(text))-1)
		ent := EntMessage{
			Tp: "MN",
			Data: EntData{
				Val: mentionName,
			},
		}
		m.Message.Ent = append(m.Message.Ent, ent)
	}
	if opt.IsHashTag {
		_fmt := FmtMessage{
			At:  leftLen,
			Len: validLen,
			Key: len(m.Message.Ent),
		}
		m.Message.Fmt = append(m.Message.Fmt, _fmt)
		hashTag := strings.TrimSpace(text)
		ent := EntMessage{
			Tp: "HT",
			Data: EntData{
				Val: hashTag,
			},
		}
		m.Message.Ent = append(m.Message.Ent, ent)
	}
	if opt.IsForm {
		_fmt := FmtMessage{
			Tp:  "FM",
			At:  leftLen,
			Len: validLen,
		}
		m.Message.Fmt = append(m.Message.Fmt, _fmt)
	}
	if opt.IsButton {
		var btnName = opt.ButtonDataName
		if btnName == "" {
			opt.ButtonDataName = strings.ToLower(strings.TrimSpace(text))
		}
		_fmt := FmtMessage{
			At:  leftLen,
			Len: validLen,
			Key: len(m.Message.Ent),
		}
		m.Message.Fmt = append(m.Message.Fmt, _fmt)
		//btnText := strings.TrimSpace(text)
		ent := EntMessage{
			Tp: "BN",
			Data: EntData{
				Name: opt.ButtonDataName,
				Act:  opt.ButtonDataAct,
				Val:  opt.ButtonDataVal,
				Ref:  opt.ButtonDataRef,
			},
		}
		m.Message.Ent = append(m.Message.Ent, ent)
	}
}

// AppendTextLine Append text message and line break to build message
func (m *MsgBuilder) AppendTextLine(text string, opt TextOption) {
	m.AppendText(text+"\n", opt)
}

// AppendImage Append image to build message
func (m *MsgBuilder) AppendImage(imageName string, opt ImageOption) {
	m.Message.Text = " "
	m.Message.Fmt = append(m.Message.Fmt, FmtMessage{
		At:  0,
		Len: 1,
		Key: len(m.Message.Ent),
	})
	m.Message.Ent = append(m.Message.Ent, EntMessage{
		Tp: "IM",
		Data: EntData{
			Mime:   opt.Mime,
			Width:  opt.Width,
			Height: opt.Height,
			Name:   imageName,
			Val:    opt.ImageBase64,
			Size:   opt.Size,
		},
	})
}

// AppendFile Append file to build message
func (m *MsgBuilder) AppendFile(fileName string, opt FileOption) {
	m.Message.Fmt = append(m.Message.Fmt, FmtMessage{
		At:  utf8.RuneCountInString(m.Message.Text),
		Len: 0,
		Key: len(m.Message.Ent),
	})
	m.Message.Ent = append(m.Message.Ent, EntMessage{
		Tp: "EX",
		Data: EntData{
			Mime: opt.Mime,
			Name: fileName,
			Val:  opt.ContentBase64,
		},
	})
}

// AppendAttachment append a attachment file to chat message
func (m *MsgBuilder) AppendAttachment(fileName string, opt AttachmentOption) {
	m.Message.Fmt = append(m.Message.Fmt, FmtMessage{
		At:  utf8.RuneCountInString(m.Message.Text),
		Len: 1,
		Key: len(m.Message.Ent),
	})
	m.Message.Ent = append(m.Message.Ent, EntMessage{
		Tp: "EX",
		Data: EntData{
			Mime: opt.Mime,
			Name: fileName,
			Ref:  opt.RelativeUrl,
			Size: opt.Size,
		},
	})
}

// Parse a raw ServerData to friendly ChatMessage
func (m *MsgBuilder) Parse(message ServerData) (ChatMessage, error) {
	var chatMsg ChatMessage
	if strings.Contains(message.Head, "mime") {
		err := json.Unmarshal([]byte(message.Content), &chatMsg)
		if err != nil {
			return ChatMessage{}, err
		}
		chatMsg.IsPlainText = false
	} else {
		err := json.Unmarshal([]byte(message.Content), &chatMsg)
		if err != nil {
			return ChatMessage{}, err
		}
		chatMsg.IsPlainText = true
	}
	if strings.HasPrefix(message.Topic, "usr") {
		chatMsg.MessageType = "user"
	}
	if strings.HasPrefix(message.Topic, "grp") {
		chatMsg.MessageType = "group"
	}
	return chatMsg, nil
}

// BuildTextMessage build text chat message with formatted
func (m *MsgBuilder) BuildTextMessage(text string) ChatMessage {
	msg := ChatMessage{}
	msg.Text = text
	msg.Ent = []EntMessage{}
	msg.Fmt = []FmtMessage{}
	if strings.Contains(text, "\n") {
		for i := 0; i < len(text); i++ {
			if text[i] == '\n' {
				_fmt := FmtMessage{
					At:  i,
					Tp:  "BR",
					Len: 1,
				}
				msg.Fmt = append(msg.Fmt, _fmt)
			}
		}
	}
	return msg
}

// BuildImageMessage build a image chat message
func (m *MsgBuilder) BuildImageMessage(imageName string, text string, opt ImageOption) ChatMessage {
	msg := ChatMessage{}
	msg.Text = text
	msg.Ent = []EntMessage{}
	msg.Fmt = []FmtMessage{}
	msg.Ent = append(msg.Ent, EntMessage{
		Tp: "IM",
		Data: EntData{
			Mime:   opt.Mime,
			Width:  opt.Width,
			Height: opt.Height,
			Name:   imageName,
			Val:    opt.ImageBase64,
		},
	})
	msg.Fmt = append(msg.Fmt, FmtMessage{
		At:  len(text),
		Len: 1,
		Key: 0,
	})
	if strings.Contains(text, "\n") {
		for i := 0; i < len(text); i++ {
			if text[i] == '\n' {
				_fmt := FmtMessage{
					At:  i,
					Tp:  "BR",
					Len: 1,
				}
				msg.Fmt = append(msg.Fmt, _fmt)
			}
		}
	}
	return msg
}

// BuildFileMessage build a file chat message
func (m *MsgBuilder) BuildFileMessage(fileName string, text string, opt FileOption) ChatMessage {
	msg := ChatMessage{}
	msg.Text = text
	msg.Ent = []EntMessage{}
	msg.Fmt = []FmtMessage{}
	msg.Ent = append(msg.Ent, EntMessage{
		Tp: "EX",
		Data: EntData{
			Mime: opt.Mime,
			Name: fileName,
			Val:  opt.ContentBase64,
		},
	})
	msg.Fmt = append(msg.Fmt, FmtMessage{
		At:  len(text),
		Len: 0,
		Key: 0,
	})
	if strings.Contains(text, "\n") {
		for i := 0; i < len(text); i++ {
			if text[i] == '\n' {
				_fmt := FmtMessage{
					At:  i,
					Tp:  "BR",
					Len: 1,
				}
				msg.Fmt = append(msg.Fmt, _fmt)
			}
		}
	}
	return msg
}

// BuildAttachmentMessage build a attachment message
func (m *MsgBuilder) BuildAttachmentMessage(fileName string, text string, opt AttachmentOption) ChatMessage {
	msg := ChatMessage{}
	msg.Text = text
	msg.Ent = []EntMessage{}
	msg.Fmt = []FmtMessage{}
	msg.Ent = append(msg.Ent, EntMessage{
		Tp: "EX",
		Data: EntData{
			Mime: opt.Mime,
			Name: fileName,
			Ref:  opt.RelativeUrl,
			Size: opt.Size,
		},
	})
	msg.Fmt = append(msg.Fmt, FmtMessage{
		At:  len(text),
		Len: 1,
		Key: 0,
	})
	if strings.Contains(text, "\n") {
		for i := 0; i < len(text); i++ {
			if text[i] == '\n' {
				_fmt := FmtMessage{
					At:  i,
					Tp:  "BR",
					Len: 1,
				}
				msg.Fmt = append(msg.Fmt, _fmt)
			}
		}
	}
	return msg
}

func (m *MsgBuilder) Content() (map[string]interface{}, interface{}) {
	if m.Payload != nil {
		m.Message.Tye = Tye(m.Payload)
		m.Message.Src = m.Payload
	}
	return m.Message.Content()
}

type ServerData struct {
	Head    string
	Content string
	Topic   string
}

type TextOption struct {
	IsBold         bool
	IsItalic       bool
	IsDeleted      bool
	IsCode         bool
	IsLink         bool
	IsMention      bool
	IsHashTag      bool
	IsForm         bool
	IsButton       bool
	ButtonDataName string
	ButtonDataAct  string
	ButtonDataVal  string
	ButtonDataRef  string
}

type ImageOption struct {
	Mime        string
	Width       int
	Height      int
	ImageBase64 string
	Size        int
}

type FileOption struct {
	Mime          string
	ContentBase64 string
}

type AttachmentOption struct {
	Mime        string
	RelativeUrl string
	Size        int
}

func substr(input string, start int, length int) string {
	asRunes := []rune(input)

	if start >= len(asRunes) {
		return ""
	}

	if start+length > len(asRunes) {
		length = len(asRunes) - start
	}

	return string(asRunes[start : start+length])
}

func Tye(payload MsgPayload) string {
	t := reflect.TypeOf(payload)
	return t.Name()
}

func toString(v interface{}) string {
	switch v := v.(type) {
	case string:
		return v

	case []byte:
		return string(v)

	case int:
		return strconv.Itoa(v)

	case float64:
		_, frac := math.Modf(v)
		if frac == 0 {
			return strconv.Itoa(int(v))
		}
		return strconv.FormatFloat(v, 'f', 4, 64)

	case bool:
		return strconv.FormatBool(v)

	case nil:
		return ""

	default:
		return fmt.Sprint(v)
	}
}

func ToPayload(typ string, src []byte) MsgPayload {
	switch typ {
	case "TextMsg":
		var r TextMsg
		_ = json.Unmarshal(src, &r)
		return r
	case "ImageMsg":
		var r ImageMsg
		_ = json.Unmarshal(src, &r)
		return r
	case "FileMsg":
		var r FileMsg
		_ = json.Unmarshal(src, &r)
		return r
	case "VideoMsg":
		var r VideoMsg
		_ = json.Unmarshal(src, &r)
		return r
	case "AudioMsg":
		var r AudioMsg
		_ = json.Unmarshal(src, &r)
		return r
	case "ScriptMsg":
		var r ScriptMsg
		_ = json.Unmarshal(src, &r)
		return r
	case "ActionMsg":
		var r ActionMsg
		_ = json.Unmarshal(src, &r)
		return r
	case "LinkMsg":
		var r LinkMsg
		_ = json.Unmarshal(src, &r)
		return r
	case "LocationMsg":
		var r LocationMsg
		_ = json.Unmarshal(src, &r)
		return r
	case "TableMsg":
		var r TableMsg
		_ = json.Unmarshal(src, &r)
		return r
	case "DigitMsg":
		var r DigitMsg
		_ = json.Unmarshal(src, &r)
		return r
	case "OkrMsg":
		var r OkrMsg
		_ = json.Unmarshal(src, &r)
		return r
	case "InfoMsg":
		var r InfoMsg
		_ = json.Unmarshal(src, &r)
		return r
	case "TodoMsg":
		var r TodoMsg
		_ = json.Unmarshal(src, &r)
		return r
	case "ChartMsg":
		var r ChartMsg
		_ = json.Unmarshal(src, &r)
		return r
	case "RepoMsg":
		var r RepoMsg
		_ = json.Unmarshal(src, &r)
		return r
	case "CrateMsg":
		var r CrateMsg
		_ = json.Unmarshal(src, &r)
		return r
	case "QuestionMsg":
		var r QuestionMsg
		_ = json.Unmarshal(src, &r)
		return r
	}
	return nil
}

func ImageConvert(data []byte, name string, width, height int) ImageMsg {
	raw := base64.StdEncoding.EncodeToString(data)
	return ImageMsg{
		Width:       width,
		Height:      height,
		Alt:         fmt.Sprintf("%s.jpg", name),
		Mime:        "image/jpeg",
		Size:        len(data),
		ImageBase64: raw,
	}
}

func ExtractText(content interface{}) string {
	text := ""
	if m, ok := content.(map[string]interface{}); ok {
		if t, ok := m["txt"]; ok {
			if s, ok := t.(string); ok {
				text = s
			}
		}
	} else if s, ok := content.(string); ok {
		text = s
	}
	return text
}
