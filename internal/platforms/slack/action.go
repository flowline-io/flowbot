package slack

import (
	"fmt"
	"strings"

	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/protocol"
	"github.com/slack-go/slack"
)

type Action struct {
	api *slack.Client
}

func (a *Action) GetLatestEvents(_ protocol.Request) protocol.Response {
	return protocol.NewFailedResponse(protocol.ErrUnsupportedAction.New("unsupported action"))
}

func (a *Action) GetSupportedActions(_ protocol.Request) protocol.Response {
	return protocol.NewFailedResponse(protocol.ErrUnsupportedAction.New("unsupported action"))
}

func (a *Action) GetStatus(_ protocol.Request) protocol.Response {
	return protocol.NewFailedResponse(protocol.ErrUnsupportedAction.New("unsupported action"))
}

func (a *Action) GetVersion(_ protocol.Request) protocol.Response {
	return protocol.NewFailedResponse(protocol.ErrUnsupportedAction.New("unsupported action"))
}

func (a *Action) SendMessage(req protocol.Request) protocol.Response {
	channel, _ := types.KV(req.Params).String("topic")
	message, _ := types.KV(req.Params).Any("message")
	content, ok := message.(protocol.Message)
	if !ok {
		return protocol.NewFailedResponse(protocol.ErrBadSegmentType.New("message type error"))
	}
	if len(content) == 0 {
		return protocol.NewSuccessResponse(nil)
	}

	ts, err := a.postRichMessage(channel, content)
	if err != nil {
		flog.Error(fmt.Errorf("failed to send message to %s, %w", channel, err))
		return protocol.NewFailedResponse(protocol.ErrInternalHandler.New("send message error"))
	}

	return protocol.NewSuccessResponse(map[string]string{
		"message_id": ts,
		"channel":    channel,
	})
}

// UpdateMessage updates an existing message (e.g. replace a "thinking…" indicator with the final result).
func (a *Action) UpdateMessage(req protocol.Request) protocol.Response {
	channel, _ := types.KV(req.Params).String("topic")
	timestamp, _ := types.KV(req.Params).String("message_id")
	message, _ := types.KV(req.Params).Any("message")
	content, ok := message.(protocol.Message)
	if !ok {
		return protocol.NewFailedResponse(protocol.ErrBadSegmentType.New("message type error"))
	}
	if timestamp == "" {
		return protocol.NewFailedResponse(protocol.ErrBadParam.New("message_id required"))
	}

	msgOptions, _ := a.buildMsgOptions(content)
	if len(msgOptions) == 0 {
		return protocol.NewFailedResponse(protocol.ErrBadSegmentData.New("no valid message content"))
	}

	_, _, _, err := a.api.UpdateMessage(channel, timestamp, msgOptions...)
	if err != nil {
		flog.Error(fmt.Errorf("failed to update message %s in %s, %w", timestamp, channel, err))
		return protocol.NewFailedResponse(protocol.ErrInternalHandler.New("update message error"))
	}

	return protocol.NewSuccessResponse(nil)
}

// DeleteMessage deletes an existing message.
func (a *Action) DeleteMessage(req protocol.Request) protocol.Response {
	channel, _ := types.KV(req.Params).String("topic")
	timestamp, _ := types.KV(req.Params).String("message_id")
	if timestamp == "" {
		return protocol.NewFailedResponse(protocol.ErrBadParam.New("message_id required"))
	}

	_, _, err := a.api.DeleteMessage(channel, timestamp)
	if err != nil {
		flog.Error(fmt.Errorf("failed to delete message %s in %s, %w", timestamp, channel, err))
		return protocol.NewFailedResponse(protocol.ErrInternalHandler.New("delete message error"))
	}

	return protocol.NewSuccessResponse(nil)
}

// SendStatusMessage posts a temporary status/thinking indicator and returns
// the message timestamp so it can later be updated with the final result.
func (a *Action) SendStatusMessage(channel, statusText string) (string, error) {
	blocks := statusBlocks(statusText)
	_, ts, err := a.api.PostMessage(channel,
		slack.MsgOptionBlocks(blocks...),
		slack.MsgOptionText(statusText, false),
	)
	if err != nil {
		return "", fmt.Errorf("failed to send status message to %s: %w", channel, err)
	}
	return ts, nil
}

func (a *Action) GetUserInfo(_ protocol.Request) protocol.Response {
	return protocol.NewFailedResponse(protocol.ErrUnsupportedAction.New("unsupported action"))
}

func (a *Action) CreateChannel(_ protocol.Request) protocol.Response {
	return protocol.NewFailedResponse(protocol.ErrUnsupportedAction.New("unsupported action"))
}

func (a *Action) GetChannelInfo(_ protocol.Request) protocol.Response {
	return protocol.NewFailedResponse(protocol.ErrUnsupportedAction.New("unsupported action"))
}

func (a *Action) GetChannelList(_ protocol.Request) protocol.Response {
	return protocol.NewFailedResponse(protocol.ErrUnsupportedAction.New("unsupported action"))
}

func (a *Action) RegisterChannels(_ protocol.Request) protocol.Response {
	return protocol.NewFailedResponse(protocol.ErrUnsupportedAction.New("unsupported action"))
}

func (a *Action) RegisterSlashCommands(_ protocol.Request) protocol.Response {
	return protocol.NewFailedResponse(protocol.ErrUnsupportedAction.New("unsupported action"))
}

// ──────────────────────────────────────────
// Rich message rendering
// ──────────────────────────────────────────

// postRichMessage converts protocol.Message segments into Slack Block Kit blocks
// and posts them. Returns the message timestamp (used as message_id on Slack).
func (a *Action) postRichMessage(channel string, content protocol.Message) (string, error) {
	msgOptions, fileIDs := a.buildMsgOptions(content)
	if len(msgOptions) == 0 {
		return "", fmt.Errorf("no valid message content")
	}

	_, ts, err := a.api.PostMessage(channel, msgOptions...)
	if err != nil {
		return "", fmt.Errorf("failed to send message to %s: %w", channel, err)
	}

	// Upload and share files after posting the message
	if len(fileIDs) > 0 {
		a.uploadAndShareFiles(channel, fileIDs)
	}

	return ts, nil
}

// buildMsgOptions converts protocol.Message segments into slack.MsgOption slice
// and collects file IDs for separate upload.
func (a *Action) buildMsgOptions(content protocol.Message) ([]slack.MsgOption, []string) {
	var textParts []string
	var blocks []slack.Block
	var fileIDs []string

	for _, segment := range content {
		switch segment.Type {
		case "text":
			if text, ok := segment.Data["text"].(string); ok {
				textParts = append(textParts, text)
			}
		case "url":
			if url, ok := segment.Data["url"].(string); ok {
				textParts = append(textParts, url)
			}
		case "mention":
			if userID, ok := segment.Data["user_id"].(string); ok {
				textParts = append(textParts, fmt.Sprintf("<@%s>", userID))
			}
		case "mention_all":
			textParts = append(textParts, "<!channel>")
		case "image":
			if fileID, ok := segment.Data["file_id"].(string); ok {
				blocks = append(blocks, imageBlock(fileID, "image", ""))
			}
		case "file", "video", "audio", "voice":
			if fileID, ok := segment.Data["file_id"].(string); ok {
				fileIDs = append(fileIDs, fileID)
			}
		case "location":
			lat, latOk := segment.Data["latitude"].(float64)
			lon, lonOk := segment.Data["longitude"].(float64)
			if latOk && lonOk {
				title, _ := segment.Data["title"].(string)
				locContent, _ := segment.Data["content"].(string)
				locText := fmt.Sprintf("📍 *%s*\nLat: %.6f, Lon: %.6f", title, lat, lon)
				if locContent != "" {
					locText += "\n" + locContent
				}
				blocks = append(blocks, section(locText))
			}
		case "reply":
			if userID, ok := segment.Data["user_id"].(string); ok {
				if msgID, ok2 := segment.Data["message_id"].(string); ok2 {
					blocks = append(blocks, contextBlock(fmt.Sprintf("↩️ Replying to <@%s> (msg: %s)", userID, msgID)))
				}
			}

		// ── Rich UI component segments ──

		case "chart":
			chartType, _ := segment.Data["chart_type"].(string) // "bar" or "pie"
			title, _ := segment.Data["title"].(string)
			subtitle, _ := segment.Data["subtitle"].(string)
			labels := toStringSlice(segment.Data["labels"])
			values := toFloat64Slice(segment.Data["values"])

			switch chartType {
			case "pie":
				blocks = append(blocks, renderPieChart(title, labels, values)...)
			default: // bar (default)
				blocks = append(blocks, renderBarChart(title, subtitle, labels, values)...)
			}

		case "table":
			title, _ := segment.Data["title"].(string)
			headers := toStringSlice(segment.Data["headers"])
			rows := toRowSlice(segment.Data["rows"])
			blocks = append(blocks, buildTableBlocks(title, headers, rows)...)

		case "form":
			// Forms are rendered as inline input blocks in the message
			// For full modal forms, use the "form_modal" type instead
			title, _ := segment.Data["title"].(string)
			fields := toFormFieldDefs(segment.Data["fields"])
			if title != "" {
				blocks = append(blocks, header(title))
			}
			blocks = append(blocks, divider())
			for _, f := range fields {
				fieldText := fmt.Sprintf("*%s*", f.Label)
				if f.Placeholder != "" {
					fieldText += fmt.Sprintf("  _%s_", f.Placeholder)
				}
				if f.InitialVal != "" {
					fieldText += fmt.Sprintf("\nCurrent: `%s`", f.InitialVal)
				}
				blocks = append(blocks, section(fieldText))
			}

		case "action_card":
			title, _ := segment.Data["title"].(string)
			description, _ := segment.Data["description"].(string)
			imageURL, _ := segment.Data["image_url"].(string)
			footer, _ := segment.Data["footer"].(string)
			fields := toStringMap(segment.Data["fields"])
			buttons := toButtonDefs(segment.Data["buttons"])

			blocks = append(blocks, buildActionCard(ActionCardDef{
				Title:       title,
				Description: description,
				Fields:      fields,
				ImageURL:    imageURL,
				Buttons:     buttons,
				Footer:      footer,
			})...)

		case "status":
			statusText, _ := segment.Data["text"].(string)
			if statusText == "" {
				statusText = "Processing…"
			}
			blocks = append(blocks, statusBlocks(statusText)...)

		case "link":
			title, _ := segment.Data["title"].(string)
			url, _ := segment.Data["url"].(string)
			cover, _ := segment.Data["cover"].(string)
			if url == "" {
				break
			}
			if title == "" {
				title = url
			}
			linkText := fmt.Sprintf("<%s|%s>", url, title)
			if cover != "" {
				blocks = append(blocks, imageSection(linkText, cover, title))
			} else {
				blocks = append(blocks, sectionWithButton(linkText, "Open Link", "link_open", url, slack.StylePrimary))
			}

		case "markdown":
			title, _ := segment.Data["title"].(string)
			text, _ := segment.Data["text"].(string)
			if title != "" {
				blocks = append(blocks, header(title))
			}
			if text != "" {
				// Slack supports mrkdwn; render as-is
				blocks = append(blocks, section(text))
			}

		case "kv":
			fieldsRaw := toStringMap(segment.Data["fields"])
			if len(fieldsRaw) > 0 {
				blocks = append(blocks, sectionFields(fieldsRaw))
			}
		}
	}

	var msgOptions []slack.MsgOption

	// Combine text parts into a section block when we also have other blocks
	if len(textParts) > 0 {
		fallbackText := strings.Join(textParts, "\n")
		msgOptions = append(msgOptions, slack.MsgOptionText(fallbackText, false))
		// When there are blocks, also prepend a text section so text renders in Block Kit mode
		if len(blocks) > 0 {
			blocks = append([]slack.Block{section(fallbackText)}, blocks...)
		}
	}

	if len(blocks) > 0 {
		msgOptions = append(msgOptions, slack.MsgOptionBlocks(blocks...))
	}

	return msgOptions, fileIDs
}

// uploadAndShareFiles uploads files and shares them to the channel.
// fileIDs here are expected to be file paths or publicly accessible URLs.
func (a *Action) uploadAndShareFiles(channel string, fileIDs []string) {
	for _, fileRef := range fileIDs {
		_, err := a.api.UploadFile(slack.UploadFileParameters{
			File:     fileRef,
			Filename: fileRef,
			Channel:  channel,
		})
		if err != nil {
			flog.Error(fmt.Errorf("failed to share file %s to %s: %w", fileRef, channel, err))
		}
	}
}

// ──────────────────────────────────────────
// Data conversion helpers
// ──────────────────────────────────────────

func toStringSlice(v any) []string {
	if v == nil {
		return nil
	}
	switch s := v.(type) {
	case []string:
		return s
	case []any:
		var result []string
		for _, item := range s {
			if str, ok := item.(string); ok {
				result = append(result, str)
			}
		}
		return result
	}
	return nil
}

func toFloat64Slice(v any) []float64 {
	if v == nil {
		return nil
	}
	switch s := v.(type) {
	case []float64:
		return s
	case []any:
		var result []float64
		for _, item := range s {
			switch n := item.(type) {
			case float64:
				result = append(result, n)
			case int:
				result = append(result, float64(n))
			case int64:
				result = append(result, float64(n))
			}
		}
		return result
	}
	return nil
}

func toRowSlice(v any) [][]any {
	if v == nil {
		return nil
	}
	switch s := v.(type) {
	case [][]any:
		return s
	case []any:
		var result [][]any
		for _, item := range s {
			if row, ok := item.([]any); ok {
				result = append(result, row)
			}
		}
		return result
	}
	return nil
}

func toStringMap(v any) map[string]string {
	if v == nil {
		return nil
	}
	result := make(map[string]string)
	switch m := v.(type) {
	case map[string]string:
		return m
	case map[string]any:
		for k, val := range m {
			result[k] = fmt.Sprintf("%v", val)
		}
	}
	return result
}

func toFormFieldDefs(v any) []FormFieldDef {
	if v == nil {
		return nil
	}
	switch s := v.(type) {
	case []FormFieldDef:
		return s
	case []any:
		var result []FormFieldDef
		for _, item := range s {
			if m, ok := item.(map[string]any); ok {
				f := FormFieldDef{
					Label:       fmt.Sprintf("%v", m["label"]),
					Key:         fmt.Sprintf("%v", m["key"]),
					Type:        fmt.Sprintf("%v", m["type"]),
					Placeholder: fmt.Sprintf("%v", m["placeholder"]),
				}
				if val, ok := m["initial_value"].(string); ok {
					f.InitialVal = val
				}
				if opts, ok := m["options"].([]any); ok {
					for _, o := range opts {
						f.Options = append(f.Options, fmt.Sprintf("%v", o))
					}
				}
				result = append(result, f)
			}
		}
		return result
	}
	return nil
}

func toButtonDefs(v any) []ButtonDef {
	if v == nil {
		return nil
	}
	switch s := v.(type) {
	case []ButtonDef:
		return s
	case []any:
		var result []ButtonDef
		for _, item := range s {
			if m, ok := item.(map[string]any); ok {
				b := ButtonDef{
					Text:     fmt.Sprintf("%v", m["text"]),
					ActionID: fmt.Sprintf("%v", m["action_id"]),
					Value:    fmt.Sprintf("%v", m["value"]),
				}
				if style, ok := m["style"].(string); ok {
					switch style {
					case "primary":
						b.Style = slack.StylePrimary
					case "danger":
						b.Style = slack.StyleDanger
					}
				}
				if url, ok := m["url"].(string); ok {
					b.URL = url
				}
				result = append(result, b)
			}
		}
		return result
	}
	return nil
}
