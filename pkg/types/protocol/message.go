package protocol

import (
	"bytes"
	"fmt"
)

type Message []MessageSegment

type MessageSegment struct {
	Type string         `json:"type"`
	Data map[string]any `json:"data"`
}

// String impls the interface Stringer
func (m Message) String() string {
	b := &bytes.Buffer{}
	for _, segment := range m {
		_, _ = fmt.Fprintf(b, "%s\n", segment.String())
	}
	return b.String()
}

// String impls the interface Stringer
func (s MessageSegment) String() string {
	b := &bytes.Buffer{}
	_, _ = fmt.Fprintf(b, "%s\n", s.Type)
	for k, v := range s.Data {
		_, _ = fmt.Fprintf(b, "%s: %v\n", k, v)
	}
	return b.String()
}

func Text(text ...interface{}) MessageSegment {
	return MessageSegment{
		Type: "text",
		Data: map[string]any{
			"text": fmt.Sprint(text...),
		},
	}
}

func Url(url string) MessageSegment {
	return MessageSegment{
		Type: "url",
		Data: map[string]any{
			"url": url,
		},
	}
}

func Mention(userId string) MessageSegment {
	if userId == "" {
		return MentionAll()
	}
	return MessageSegment{
		Type: "mention",
		Data: map[string]any{
			"user_id": userId,
		},
	}
}

func MentionAll() MessageSegment {
	return MessageSegment{
		Type: "mention_all",
		Data: map[string]any{
			"user_id": "all",
		},
	}
}

func Image(fileId string) MessageSegment {
	return MessageSegment{
		Type: "image",
		Data: map[string]any{
			"file_id": fileId,
		},
	}
}

func Voice(fileId string) MessageSegment {
	return MessageSegment{
		Type: "voice",
		Data: map[string]any{
			"file_id": fileId,
		},
	}
}

func Audio(fileId string) MessageSegment {
	return MessageSegment{
		Type: "audio",
		Data: map[string]any{
			"file_id": fileId,
		},
	}
}

func Video(fileId string) MessageSegment {
	return MessageSegment{
		Type: "video",
		Data: map[string]any{
			"file_id": fileId,
		},
	}
}

func File(fileId string) MessageSegment {
	return MessageSegment{
		Type: "file",
		Data: map[string]any{
			"file_id": fileId,
		},
	}
}

func Location(latitude, longitude float64, title string, content string) MessageSegment {
	return MessageSegment{
		Type: "location",
		Data: map[string]any{
			"latitude":  latitude,
			"longitude": longitude,
			"title":     title,
			"content":   content,
		},
	}
}

func Reply(userId, messageId string) MessageSegment {
	return MessageSegment{
		Type: "reply",
		Data: map[string]any{
			"user_id":    userId,
			"message_id": messageId,
		},
	}
}
