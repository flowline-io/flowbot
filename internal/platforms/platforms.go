package platforms

import (
	"errors"
	"fmt"

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
	case protocol.UpdateMessageAction:
		return c.Action.UpdateMessage(req)
	case protocol.DeleteMessageAction:
		return c.Action.DeleteMessage(req)
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
		// Rich link segment with title, URL, and optional cover image
		return protocol.Message{
			{
				Type: "link",
				Data: map[string]any{
					"title": v.Title,
					"url":   v.Url,
					"cover": v.Cover,
				},
			},
		}
	case types.TableMsg:
		// Produce a rich "table" segment so platforms with Block Kit can render it natively
		var rows []any
		for _, row := range v.Row {
			rows = append(rows, row)
		}
		return protocol.Message{
			{
				Type: "table",
				Data: map[string]any{
					"title":   v.Title,
					"headers": v.Header,
					"rows":    rows,
				},
			},
		}
	case types.InfoMsg:
		// Rich action card segment with key-value fields
		var description string
		structuredFields := make(map[string]any)
		if v.Model != nil {
			// Try to extract structured fields from the model
			switch m := v.Model.(type) {
			case map[string]any:
				for k, val := range m {
					structuredFields[k] = val
				}
			case map[string]string:
				for k, val := range m {
					structuredFields[k] = val
				}
			default:
				// Fallback: marshal to YAML for display as description
				s, err := yaml.Marshal(v.Model)
				if err == nil {
					description = utils.BytesToString(s)
				}
			}
		}
		return protocol.Message{
			{
				Type: "action_card",
				Data: map[string]any{
					"title":       v.Title,
					"description": description,
					"fields":      structuredFields,
				},
			},
		}
	case types.ChartMsg:
		// Rich chart segment
		labels := make([]any, 0, len(v.XAxis))
		for _, l := range v.XAxis {
			labels = append(labels, l)
		}
		values := make([]any, 0, len(v.Series))
		for _, s := range v.Series {
			values = append(values, s)
		}
		return protocol.Message{
			{
				Type: "chart",
				Data: map[string]any{
					"chart_type": "bar",
					"title":      v.Title,
					"subtitle":   v.SubTitle,
					"labels":     labels,
					"values":     values,
				},
			},
		}
	case types.HtmlMsg:
		// Rich HTML/markdown segment
		return protocol.Message{
			{
				Type: "markdown",
				Data: map[string]any{
					"text": v.Raw,
				},
			},
		}
	case types.MarkdownMsg:
		if v.Title == "" && v.Raw == "" {
			return nil
		}
		// Rich markdown segment
		return protocol.Message{
			{
				Type: "markdown",
				Data: map[string]any{
					"title": v.Title,
					"text":  v.Raw,
				},
			},
		}
	case types.InstructMsg:
		// Rich instruct card segment
		fields := map[string]any{
			"No":       v.No,
			"State":    string(v.State),
			"Priority": string(v.Priority),
		}
		if v.Bot != "" {
			fields["Bot"] = v.Bot
		}
		if v.Flag != "" {
			fields["Flag"] = v.Flag
		}
		if !v.ExpireAt.IsZero() {
			fields["ExpireAt"] = v.ExpireAt.Format("2006-01-02 15:04")
		}
		var description string
		if len(v.Content) > 0 {
			s, err := yaml.Marshal(v.Content)
			if err == nil {
				description = utils.BytesToString(s)
			}
		}
		return protocol.Message{
			{
				Type: "action_card",
				Data: map[string]any{
					"title":       fmt.Sprintf("Instruction: %s", v.No),
					"description": description,
					"fields":      fields,
				},
			},
		}
	case types.KVMsg:
		if len(v) == 0 {
			return nil
		}
		// Rich key-value fields segment
		fields := make(map[string]any, len(v))
		for k, val := range v {
			fields[k] = val
		}
		return protocol.Message{
			{
				Type: "kv",
				Data: map[string]any{
					"fields": fields,
				},
			},
		}
	case types.FormMsg:
		// Rich form segment for platforms that support interactive forms
		var fields []any
		for _, field := range v.Field {
			f := map[string]any{
				"label":       field.Label,
				"key":         field.Key,
				"type":        string(field.Type),
				"placeholder": field.Placeholder,
			}
			if field.Value != nil {
				f["initial_value"] = fmt.Sprintf("%v", field.Value)
			}
			if len(field.Option) > 0 {
				opts := make([]any, 0, len(field.Option))
				for _, o := range field.Option {
					opts = append(opts, o)
				}
				f["options"] = opts
			}
			fields = append(fields, f)
		}
		return protocol.Message{
			{
				Type: "form",
				Data: map[string]any{
					"title":  v.Title,
					"id":     v.ID,
					"fields": fields,
				},
			},
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
