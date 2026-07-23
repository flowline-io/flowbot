package transform

import (
	"github.com/flowline-io/flowbot/pkg/agent/msg"
	"github.com/tmc/langchaingo/llms"
)

// Attachment is an input artifact that may become multimodal LLM content.
type Attachment struct {
	MIMEType string
	FileID   string
	Kind     msg.MediaKind
	Data     []byte
	URL      string
}

// ProcessAttachments converts attachments into agent MediaPart values.
func ProcessAttachments(attachments []Attachment) []msg.ContentPart {
	parts := make([]msg.ContentPart, 0, len(attachments))
	for _, attachment := range attachments {
		kind := attachment.Kind
		if kind == "" {
			if inferred, ok := msg.KindFromMIME(attachment.MIMEType); ok {
				kind = inferred
			} else {
				kind = msg.MediaKindImage
			}
		}
		parts = append(parts, msg.MediaPart{
			Kind:     kind,
			MIMEType: attachment.MIMEType,
			FileID:   attachment.FileID,
			Data:     attachment.Data,
			URL:      attachment.URL,
		})
	}
	return parts
}

// AttachmentsToLLM converts attachments directly into langchaingo content parts.
func AttachmentsToLLM(attachments []Attachment) ([]llms.ContentPart, error) {
	parts := ProcessAttachments(attachments)
	return partsToLLM(parts)
}
