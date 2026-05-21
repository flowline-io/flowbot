// Package platforms provides multi-platform integration for chat and messaging.
package platforms

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"reflect"

	"github.com/goccy/go-yaml"

	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/internal/store/model"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/protocol"
	"github.com/flowline-io/flowbot/pkg/utils"
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

var msgConverters = map[reflect.Type]func(types.MsgPayload) protocol.Message{
	reflect.TypeFor[types.TextMsg]():     func(p types.MsgPayload) protocol.Message { return convertText(p.(types.TextMsg)) },
	reflect.TypeFor[types.LinkMsg]():     func(p types.MsgPayload) protocol.Message { return convertLink(p.(types.LinkMsg)) },
	reflect.TypeFor[types.TableMsg]():    func(p types.MsgPayload) protocol.Message { return convertTable(p.(types.TableMsg)) },
	reflect.TypeFor[types.InfoMsg]():     func(p types.MsgPayload) protocol.Message { return convertInfo(p.(types.InfoMsg)) },
	reflect.TypeFor[types.ChartMsg]():    func(p types.MsgPayload) protocol.Message { return convertChart(p.(types.ChartMsg)) },
	reflect.TypeFor[types.HtmlMsg]():     func(p types.MsgPayload) protocol.Message { return convertHtml(p.(types.HtmlMsg)) },
	reflect.TypeFor[types.MarkdownMsg](): func(p types.MsgPayload) protocol.Message { return convertMarkdown(p.(types.MarkdownMsg)) },
	reflect.TypeFor[types.InstructMsg](): func(p types.MsgPayload) protocol.Message { return convertInstruct(p.(types.InstructMsg)) },
	reflect.TypeFor[types.KVMsg]():       func(p types.MsgPayload) protocol.Message { return convertKV(p.(types.KVMsg)) },
	reflect.TypeFor[types.FormMsg]():     func(p types.MsgPayload) protocol.Message { return convertForm(p.(types.FormMsg)) },
	reflect.TypeFor[types.EmptyMsg]():    func(p types.MsgPayload) protocol.Message { return convertEmpty(p.(types.EmptyMsg)) },
}

// MessageConvert converts a generic payload into a platform-agnostic protocol.Message.
func MessageConvert(data any) protocol.Message {
	d, ok := data.(types.MsgPayload)
	if !ok {
		return protocol.Message{
			protocol.Text("error message payload"),
		}
	}
	typ := reflect.TypeOf(d)
	if fn, ok := msgConverters[typ]; ok {
		return fn(d)
	}
	return convertDefault(data)
}

func convertText(v types.TextMsg) protocol.Message {
	return protocol.Message{
		protocol.Text(v.Text),
	}
}

func convertLink(v types.LinkMsg) protocol.Message {
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
}

func convertTable(v types.TableMsg) protocol.Message {
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
}

func convertInfo(v types.InfoMsg) protocol.Message {
	// Rich action card segment with key-value fields
	var description string
	structuredFields := make(map[string]any)
	if v.Model != nil {
		// Try to extract structured fields from the model
		switch m := v.Model.(type) {
		case map[string]any:
			maps.Copy(structuredFields, m)
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
}

func convertChart(v types.ChartMsg) protocol.Message {
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
}

func convertHtml(v types.HtmlMsg) protocol.Message {
	// Rich HTML/markdown segment
	return protocol.Message{
		{
			Type: "markdown",
			Data: map[string]any{
				"text": v.Raw,
			},
		},
	}
}

func convertMarkdown(v types.MarkdownMsg) protocol.Message {
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
}

func convertInstruct(v types.InstructMsg) protocol.Message {
	// Rich instruct card segment
	fields := map[string]any{
		"No":       v.No,
		"State":    fmt.Sprintf("%d", v.State),
		"Priority": fmt.Sprintf("%d", v.Priority),
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
}

func convertKV(v types.KVMsg) protocol.Message {
	if len(v) == 0 {
		return nil
	}
	// Rich key-value fields segment
	fields := make(map[string]any, len(v))
	maps.Copy(fields, v)
	return protocol.Message{
		{
			Type: "kv",
			Data: map[string]any{
				"fields": fields,
			},
		},
	}
}

func convertForm(v types.FormMsg) protocol.Message {
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
}

func convertEmpty(_ types.EmptyMsg) protocol.Message {
	return nil
}

func convertDefault(data any) protocol.Message {
	s, err := yaml.Marshal(data)
	if err != nil {
		flog.Error(err)
		return nil
	}

	return protocol.Message{
		protocol.Text(utils.BytesToString(s)),
	}
}

func PlatformRegister(name string, caller *Caller) error {
	_, err := store.Database.GetPlatformByName(context.Background(), name)
	if err != nil && !errors.Is(err, types.ErrNotFound) {
		return err
	}
	if errors.Is(err, types.ErrNotFound) {
		_, err = store.Database.CreatePlatform(context.Background(), &model.Platform{
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
