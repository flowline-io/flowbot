package transform

import (
	"github.com/flowline-io/flowbot/pkg/agent/msg"
	"github.com/tmc/langchaingo/llms"
)

// Attachment is an input artifact that may become multimodal LLM content.
type Attachment struct {
	MIMEType string
	Data     []byte
	URL      string
}

// ProcessAttachments converts attachments into agent content parts.
func ProcessAttachments(attachments []Attachment) []msg.ContentPart {
	parts := make([]msg.ContentPart, 0, len(attachments))
	for _, attachment := range attachments {
		parts = append(parts, msg.ImagePart{
			MIMEType: attachment.MIMEType,
			Data:     attachment.Data,
			URL:      attachment.URL,
		})
	}
	return parts
}

// AttachmentsToLLM converts attachments directly into langchaingo content parts.
func AttachmentsToLLM(attachments []Attachment) []llms.ContentPart {
	parts := ProcessAttachments(attachments)
	return partsToLLM(parts)
}
