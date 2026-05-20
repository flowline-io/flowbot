// Package slack implements the Slack platform integration.
package slack

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/slack-go/slack"

	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/protocol"
)

func safeStr(data map[string]any, key string) string {
	if s, ok := data[key].(string); ok {
		return s
	}
	return ""
}

type Action struct {
	api *slack.Client
}

func (*Action) GetLatestEvents(_ protocol.Request) protocol.Response {
	return protocol.NewFailedResponse(protocol.ErrUnsupportedAction.New("unsupported action"))
}

func (*Action) GetSupportedActions(_ protocol.Request) protocol.Response {
	return protocol.NewFailedResponse(protocol.ErrUnsupportedAction.New("unsupported action"))
}

func (*Action) GetStatus(_ protocol.Request) protocol.Response {
	return protocol.NewFailedResponse(protocol.ErrUnsupportedAction.New("unsupported action"))
}

func (*Action) GetVersion(_ protocol.Request) protocol.Response {
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

	threadId := getThreadContext(channel)
	ts, err := a.postRichMessage(channel, threadId, content)
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

func (*Action) GetUserInfo(_ protocol.Request) protocol.Response {
	return protocol.NewFailedResponse(protocol.ErrUnsupportedAction.New("unsupported action"))
}

func (*Action) CreateChannel(_ protocol.Request) protocol.Response {
	return protocol.NewFailedResponse(protocol.ErrUnsupportedAction.New("unsupported action"))
}

func (*Action) GetChannelInfo(_ protocol.Request) protocol.Response {
	return protocol.NewFailedResponse(protocol.ErrUnsupportedAction.New("unsupported action"))
}

func (*Action) GetChannelList(_ protocol.Request) protocol.Response {
	return protocol.NewFailedResponse(protocol.ErrUnsupportedAction.New("unsupported action"))
}

func (*Action) RegisterChannels(_ protocol.Request) protocol.Response {
	return protocol.NewFailedResponse(protocol.ErrUnsupportedAction.New("unsupported action"))
}

func (*Action) RegisterSlashCommands(_ protocol.Request) protocol.Response {
	return protocol.NewFailedResponse(protocol.ErrUnsupportedAction.New("unsupported action"))
}

// ──────────────────────────────────────────
// Rich message rendering
// ──────────────────────────────────────────

// postRichMessage converts protocol.Message segments into Slack Block Kit blocks
// and posts them. Returns the message timestamp (used as message_id on Slack).
func (a *Action) postRichMessage(channel, threadId string, content protocol.Message) (string, error) {
	msgOptions, fileIDs := a.buildMsgOptions(content)
	if len(msgOptions) == 0 {
		return "", fmt.Errorf("no valid message content")
	}

	if threadId != "" {
		msgOptions = append(msgOptions, slack.MsgOptionTS(threadId))
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

// segHandlers maps segment types to handler functions for buildMsgOptions.
var segHandlers = map[string]func(protocol.MessageSegment) (string, []slack.Block, []string){
	"text":        handleSegText,
	"url":         handleSegURL,
	"mention":     handleSegMention,
	"mention_all": handleSegMentionAll,
	"image":       handleSegImage,
	"file":        handleSegFile,
	"video":       handleSegFile,
	"audio":       handleSegFile,
	"voice":       handleSegFile,
	"location":    handleSegLocation,
	"reply":       handleSegReply,
	"chart":       handleSegChart,
	"table":       handleSegTable,
	"form":        handleSegForm,
	"action_card": handleSegActionCard,
	"status":      handleSegStatus,
	"link":        handleSegLink,
	"markdown":    handleSegMarkdown,
	"kv":          handleSegKV,
}

// buildMsgOptions converts protocol.Message segments into slack.MsgOption slice
// and collects file IDs for separate upload.
func (*Action) buildMsgOptions(content protocol.Message) ([]slack.MsgOption, []string) {
	var textParts []string
	var blocks []slack.Block
	var fileIDs []string

	for _, segment := range content {
		handler, ok := segHandlers[segment.Type]
		if !ok {
			continue
		}
		txt, blks, fids := handler(segment)
		if txt != "" {
			textParts = append(textParts, txt)
		}
		blocks = append(blocks, blks...)
		fileIDs = append(fileIDs, fids...)
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

// ──────────────────────────────────────────
// Segment handler functions
// ──────────────────────────────────────────

func handleSegText(segment protocol.MessageSegment) (string, []slack.Block, []string) {
	if text, ok := segment.Data["text"].(string); ok {
		return text, nil, nil
	}
	return "", nil, nil
}

func handleSegURL(segment protocol.MessageSegment) (string, []slack.Block, []string) {
	if url, ok := segment.Data["url"].(string); ok {
		return url, nil, nil
	}
	return "", nil, nil
}

func handleSegMention(segment protocol.MessageSegment) (string, []slack.Block, []string) {
	if userID, ok := segment.Data["user_id"].(string); ok {
		return fmt.Sprintf("<@%s>", userID), nil, nil
	}
	return "", nil, nil
}

func handleSegMentionAll(_ protocol.MessageSegment) (string, []slack.Block, []string) {
	return "<!channel>", nil, nil
}

func handleSegImage(segment protocol.MessageSegment) (string, []slack.Block, []string) {
	if fileID, ok := segment.Data["file_id"].(string); ok {
		return "", []slack.Block{imageBlock(fileID, "image", "")}, nil
	}
	return "", nil, nil
}

func handleSegFile(segment protocol.MessageSegment) (string, []slack.Block, []string) {
	if fileID, ok := segment.Data["file_id"].(string); ok {
		return "", nil, []string{fileID}
	}
	return "", nil, nil
}

func handleSegLocation(segment protocol.MessageSegment) (string, []slack.Block, []string) {
	lat, latOk := segment.Data["latitude"].(float64)
	lon, lonOk := segment.Data["longitude"].(float64)
	if !latOk || !lonOk {
		return "", nil, nil
	}
	title := safeStr(segment.Data, "title")
	locContent := safeStr(segment.Data, "content")
	locText := fmt.Sprintf("*%s*\nLat: %.6f, Lon: %.6f", title, lat, lon)
	if locContent != "" {
		locText += "\n" + locContent
	}
	return "", []slack.Block{section(locText)}, nil
}

func handleSegReply(segment protocol.MessageSegment) (string, []slack.Block, []string) {
	userID, ok := segment.Data["user_id"].(string)
	if !ok {
		return "", nil, nil
	}
	msgID, ok2 := segment.Data["message_id"].(string)
	if !ok2 {
		return "", nil, nil
	}
	return "", []slack.Block{contextBlock(fmt.Sprintf("Replying to <@%s> (msg: %s)", userID, msgID))}, nil
}

func handleSegChart(segment protocol.MessageSegment) (string, []slack.Block, []string) {
	chartType := safeStr(segment.Data, "chart_type")
	title := safeStr(segment.Data, "title")
	subtitle := safeStr(segment.Data, "subtitle")
	labels := toStringSlice(segment.Data["labels"])
	values := toFloat64Slice(segment.Data["values"])

	switch chartType {
	case "pie":
		return "", renderPieChart(title, labels, values), nil
	default:
		return "", renderBarChart(title, subtitle, labels, values), nil
	}
}

func handleSegTable(segment protocol.MessageSegment) (string, []slack.Block, []string) {
	title := safeStr(segment.Data, "title")
	headers := toStringSlice(segment.Data["headers"])
	rows := toRowSlice(segment.Data["rows"])
	return "", buildTableBlocks(title, headers, rows), nil
}

func handleSegForm(segment protocol.MessageSegment) (string, []slack.Block, []string) {
	title := safeStr(segment.Data, "title")
	fields := toFormFieldDefs(segment.Data["fields"])
	var blocks []slack.Block
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
	return "", blocks, nil
}

func handleSegActionCard(segment protocol.MessageSegment) (string, []slack.Block, []string) {
	title := safeStr(segment.Data, "title")
	description := safeStr(segment.Data, "description")
	imageURL := safeStr(segment.Data, "image_url")
	footer := safeStr(segment.Data, "footer")
	fields := toStringMap(segment.Data["fields"])
	buttons := toButtonDefs(segment.Data["buttons"])
	return "", buildActionCard(ActionCardDef{
		Title:       title,
		Description: description,
		Fields:      fields,
		ImageURL:    imageURL,
		Buttons:     buttons,
		Footer:      footer,
	}), nil
}

func handleSegStatus(segment protocol.MessageSegment) (string, []slack.Block, []string) {
	statusText := safeStr(segment.Data, "text")
	if statusText == "" {
		statusText = "Processing..."
	}
	return "", statusBlocks(statusText), nil
}

func handleSegLink(segment protocol.MessageSegment) (string, []slack.Block, []string) {
	title := safeStr(segment.Data, "title")
	url := safeStr(segment.Data, "url")
	cover := safeStr(segment.Data, "cover")
	if url == "" {
		return "", nil, nil
	}
	if title == "" {
		title = url
	}
	linkText := fmt.Sprintf("<%s|%s>", url, title)
	if cover != "" {
		return "", []slack.Block{imageSection(linkText, cover, title)}, nil
	}
	return "", []slack.Block{sectionWithButton(linkText, "Open Link", "link_open", url, slack.StylePrimary)}, nil
}

func handleSegMarkdown(segment protocol.MessageSegment) (string, []slack.Block, []string) {
	title := safeStr(segment.Data, "title")
	text := safeStr(segment.Data, "text")
	var blocks []slack.Block
	if title != "" {
		blocks = append(blocks, header(title))
	}
	if text != "" {
		blocks = append(blocks, section(text))
	}
	return "", blocks, nil
}

func handleSegKV(segment protocol.MessageSegment) (string, []slack.Block, []string) {
	fieldsRaw := toStringMap(segment.Data["fields"])
	if len(fieldsRaw) > 0 {
		return "", []slack.Block{sectionFields(fieldsRaw)}, nil
	}
	return "", nil, nil
}

// uploadAndShareFiles uploads files and shares them to the channel.
// fileIDs here are expected to be file paths or publicly accessible URLs.
func (a *Action) uploadAndShareFiles(channel string, fileIDs []string) {
	ctx := context.Background()
	for _, fileRef := range fileIDs {
		// Get file info for size
		fileInfo, err := os.Stat(fileRef)
		if err != nil {
			flog.Error(fmt.Errorf("failed to stat file %s: %w", fileRef, err))
			continue
		}

		// Step 1: Get upload URL
		getURLResp, err := a.api.GetUploadURLExternalContext(ctx, slack.GetUploadURLExternalParameters{
			FileName: fileRef,
			FileSize: int(fileInfo.Size()),
		})
		if err != nil {
			flog.Error(fmt.Errorf("failed to get upload URL for %s: %w", fileRef, err))
			continue
		}

		// Step 2: Upload file to the URL
		err = a.api.UploadToURL(ctx, slack.UploadToURLParameters{
			UploadURL: getURLResp.UploadURL,
			File:      fileRef,
			Filename:  fileRef,
		})
		if err != nil {
			flog.Error(fmt.Errorf("failed to upload file %s: %w", fileRef, err))
			continue
		}

		// Step 3: Complete upload and share to channel
		_, err = a.api.CompleteUploadExternalContext(ctx, slack.CompleteUploadExternalParameters{
			Files:   []slack.FileSummary{{ID: getURLResp.FileID, Title: fileRef}},
			Channel: channel,
		})
		if err != nil {
			flog.Error(fmt.Errorf("failed to complete upload for %s to %s: %w", fileRef, channel, err))
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
