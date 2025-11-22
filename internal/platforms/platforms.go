package platforms

import (
	"errors"
	"fmt"
	"strings"

	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/internal/store/model"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/protocol"
	"github.com/flowline-io/flowbot/pkg/utils"
	"github.com/goccy/go-yaml"
	"gorm.io/gorm"
)

var callers = make(map[string]*Caller)

type Caller struct {
	Action  protocol.Action
	Adapter protocol.Adapter
}

func (c *Caller) Do(req protocol.Request) protocol.Response {
	switch req.Action {
	case protocol.SendMessageAction:
		return c.Action.SendMessage(req)
	}
	return protocol.NewFailedResponse(protocol.ErrUnsupportedAction.New("error action"))
}

func MessageConvert(data any) protocol.Message {
	d, ok := data.(types.MsgPayload)
	if !ok {
		return protocol.Message{
			protocol.Text("error message payload"),
		}
	}
	switch v := d.(type) {
	case types.TextMsg:
		return protocol.Message{
			protocol.Text(v.Text),
		}
	case types.LinkMsg:
		msg := protocol.Message{
			protocol.Text(v.Title),
			protocol.Url(v.Url),
		}
		return msg
	case types.TableMsg:
		var parts []string
		if v.Title != "" {
			parts = append(parts, fmt.Sprintf("*%s*", v.Title))
		}
		if len(v.Header) > 0 {
			headerRow := strings.Join(v.Header, " | ")
			parts = append(parts, headerRow)
			separator := strings.Repeat("-", len(headerRow))
			parts = append(parts, separator)
		}
		for _, row := range v.Row {
			var rowParts []string
			for _, cell := range row {
				rowParts = append(rowParts, fmt.Sprintf("%v", cell))
			}
			parts = append(parts, strings.Join(rowParts, " | "))
		}
		return protocol.Message{
			protocol.Text(strings.Join(parts, "\n")),
		}
	case types.InfoMsg:
		var parts []string
		if v.Title != "" {
			parts = append(parts, fmt.Sprintf("*%s*", v.Title))
		}
		if v.Model != nil {
			s, err := yaml.Marshal(v.Model)
			if err == nil {
				parts = append(parts, utils.BytesToString(s))
			}
		}
		if len(parts) == 0 {
			return nil
		}
		return protocol.Message{
			protocol.Text(strings.Join(parts, "\n")),
		}
	case types.ChartMsg:
		var parts []string
		if v.Title != "" {
			parts = append(parts, fmt.Sprintf("*%s*", v.Title))
		}
		if v.SubTitle != "" {
			parts = append(parts, fmt.Sprintf("_%s_", v.SubTitle))
		}
		if len(v.XAxis) > 0 && len(v.Series) > 0 {
			parts = append(parts, "Chart Data:")
			for i, label := range v.XAxis {
				if i < len(v.Series) {
					parts = append(parts, fmt.Sprintf("  %s: %.2f", label, v.Series[i]))
				}
			}
		}
		if len(parts) == 0 {
			return nil
		}
		return protocol.Message{
			protocol.Text(strings.Join(parts, "\n")),
		}
	case types.HtmlMsg:
		// Convert HTML to plain text (basic conversion)
		// For better results, consider using a HTML parser
		return protocol.Message{
			protocol.Text(v.Raw),
		}
	case types.MarkdownMsg:
		var parts []string
		if v.Title != "" {
			parts = append(parts, fmt.Sprintf("*%s*", v.Title))
		}
		if v.Raw != "" {
			parts = append(parts, v.Raw)
		}
		if len(parts) == 0 {
			return nil
		}
		return protocol.Message{
			protocol.Text(strings.Join(parts, "\n")),
		}
	case types.InstructMsg:
		var parts []string
		parts = append(parts, fmt.Sprintf("*Instruction: %s*", v.No))
		if v.Bot != "" {
			parts = append(parts, fmt.Sprintf("Bot: %s", v.Bot))
		}
		if v.Flag != "" {
			parts = append(parts, fmt.Sprintf("Flag: %s", v.Flag))
		}
		if len(v.Content) > 0 {
			s, err := yaml.Marshal(v.Content)
			if err == nil {
				parts = append(parts, fmt.Sprintf("Content:\n%s", utils.BytesToString(s)))
			}
		}
		return protocol.Message{
			protocol.Text(strings.Join(parts, "\n")),
		}
	case types.KVMsg:
		var parts []string
		for k, val := range v {
			parts = append(parts, fmt.Sprintf("%s: %v", k, val))
		}
		if len(parts) == 0 {
			return nil
		}
		return protocol.Message{
			protocol.Text(strings.Join(parts, "\n")),
		}
	case types.FormMsg:
		var parts []string
		if v.Title != "" {
			parts = append(parts, fmt.Sprintf("*%s*", v.Title))
		}
		if v.ID != "" {
			parts = append(parts, fmt.Sprintf("Form ID: %s", v.ID))
		}
		if len(v.Field) > 0 {
			parts = append(parts, "Fields:")
			for _, field := range v.Field {
				fieldText := fmt.Sprintf("  - %s (%s)", field.Label, field.Type)
				if field.Value != nil {
					fieldText += fmt.Sprintf(": %v", field.Value)
				}
				parts = append(parts, fieldText)
			}
		}
		if len(parts) == 0 {
			return nil
		}
		return protocol.Message{
			protocol.Text(strings.Join(parts, "\n")),
		}
	case types.EmptyMsg:
		return nil
	default:
		s, err := yaml.Marshal(data)
		if err != nil {
			flog.Error(err)
			return nil
		}

		return protocol.Message{
			protocol.Text(utils.BytesToString(s)),
		}
	}
}

func PlatformRegister(name string, caller *Caller) error {
	_, err := store.Database.GetPlatformByName(name)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		_, err = store.Database.CreatePlatform(&model.Platform{
			Name: name,
		})
		if err != nil {
			return fmt.Errorf("failed to create platform %s, %w", name, err)
		}
	}
	callers[name] = caller
	return nil
}

func GetCaller(name string) (*Caller, error) {
	if c, ok := callers[name]; ok {
		return c, nil
	}
	return nil, errors.New("caller not found")
}
