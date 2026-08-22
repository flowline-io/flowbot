package platforms

import (
	"fmt"
	"reflect"
	"slices"
	"strings"

	"github.com/bytedance/sonic"

	"github.com/flowline-io/flowbot/internal/store/ent/schema"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/protocol"
)

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
	if strings.TrimSpace(v.Text) == "" {
		return nil
	}
	return protocol.Message{
		protocol.Text(v.Text),
	}
}

func convertLink(v types.LinkMsg) protocol.Message {
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
	if v.Model == nil {
		return protocol.Message{
			{
				Type: "action_card",
				Data: map[string]any{
					"title":       v.Title,
					"description": "",
					"fields":      map[string]any{},
				},
			},
		}
	}

	if msg := convertModelAsTable(v.Title, v.Model); msg != nil {
		return msg
	}

	fields := flattenModelFields(v.Model)
	return protocol.Message{
		{
			Type: "action_card",
			Data: map[string]any{
				"title":       v.Title,
				"description": "",
				"fields":      fields,
			},
		},
	}
}

// convertModelAsTable turns a slice model into a table segment when possible.
func convertModelAsTable(title string, model any) protocol.Message {
	rv := reflect.ValueOf(model)
	if !rv.IsValid() {
		return nil
	}
	for rv.Kind() == reflect.Pointer {
		if rv.IsNil() {
			return nil
		}
		rv = rv.Elem()
	}
	if rv.Kind() != reflect.Slice && rv.Kind() != reflect.Array {
		return nil
	}
	if rv.Len() == 0 {
		return protocol.Message{
			{
				Type: "action_card",
				Data: map[string]any{
					"title":       title,
					"description": "_No items_",
					"fields":      map[string]any{},
				},
			},
		}
	}

	headers, rows := sliceToTable(rv)
	if len(headers) == 0 {
		return nil
	}
	rowAny := make([]any, 0, len(rows))
	for _, row := range rows {
		rowAny = append(rowAny, row)
	}
	return protocol.Message{
		{
			Type: "table",
			Data: map[string]any{
				"title":   title,
				"headers": headers,
				"rows":    rowAny,
			},
		},
	}
}

func sliceToTable(rv reflect.Value) ([]string, [][]any) {
	first := rv.Index(0)
	for first.Kind() == reflect.Interface || first.Kind() == reflect.Pointer {
		if first.IsNil() {
			return nil, nil
		}
		first = first.Elem()
	}

	switch first.Kind() {
	case reflect.Map:
		return sliceOfMapsToTable(rv)
	case reflect.Struct:
		return sliceOfStructsToTable(rv)
	default:
		rows := make([][]any, 0, rv.Len())
		for i := 0; i < rv.Len(); i++ {
			rows = append(rows, []any{fmt.Sprintf("%v", rv.Index(i).Interface())})
		}
		return []string{"Value"}, rows
	}
}

func sliceOfMapsToTable(rv reflect.Value) ([]string, [][]any) {
	keySet := make(map[string]struct{})
	items := make([]map[string]any, 0, rv.Len())
	for i := 0; i < rv.Len(); i++ {
		item := rv.Index(i)
		for item.Kind() == reflect.Interface || item.Kind() == reflect.Pointer {
			if item.IsNil() {
				item = reflect.Value{}
				break
			}
			item = item.Elem()
		}
		if !item.IsValid() || item.Kind() != reflect.Map {
			continue
		}
		m := make(map[string]any)
		for _, k := range item.MapKeys() {
			ks := fmt.Sprintf("%v", k.Interface())
			keySet[ks] = struct{}{}
			m[ks] = item.MapIndex(k).Interface()
		}
		items = append(items, m)
	}
	if len(items) == 0 {
		return nil, nil
	}
	headers := sortedKeys(keySet)
	rows := make([][]any, 0, len(items))
	for _, m := range items {
		row := make([]any, len(headers))
		for i, h := range headers {
			row[i] = m[h]
		}
		rows = append(rows, row)
	}
	return headers, rows
}

func sliceOfStructsToTable(rv reflect.Value) ([]string, [][]any) {
	first := derefValue(rv.Index(0))
	if !first.IsValid() || first.Kind() != reflect.Struct {
		return nil, nil
	}

	headers, fieldIdx := exportedStructFields(first.Type())
	if len(headers) == 0 {
		return nil, nil
	}

	rows := make([][]any, 0, rv.Len())
	for i := 0; i < rv.Len(); i++ {
		item := derefValue(rv.Index(i))
		if !item.IsValid() || item.Kind() != reflect.Struct {
			continue
		}
		row := make([]any, len(fieldIdx))
		for j, idx := range fieldIdx {
			row[j] = item.Field(idx).Interface()
		}
		rows = append(rows, row)
	}
	return headers, rows
}

func derefValue(v reflect.Value) reflect.Value {
	for v.Kind() == reflect.Pointer {
		if v.IsNil() {
			return reflect.Value{}
		}
		v = v.Elem()
	}
	return v
}

func exportedStructFields(rt reflect.Type) ([]string, []int) {
	var headers []string
	var fieldIdx []int
	for i := 0; i < rt.NumField(); i++ {
		f := rt.Field(i)
		if !f.IsExported() || !isScalarTableField(f.Type) {
			continue
		}
		name := f.Name
		if tag := f.Tag.Get("json"); tag != "" && tag != "-" {
			name = strings.Split(tag, ",")[0]
			if name == "" || name == "-" {
				continue
			}
		}
		headers = append(headers, name)
		fieldIdx = append(fieldIdx, i)
	}
	return headers, fieldIdx
}

func isScalarTableField(t reflect.Type) bool {
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	switch t.Kind() {
	case reflect.Bool, reflect.String,
		reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64:
		return true
	default:
		// Named string-like types (e.g. AppStatus) are still Kind()==String after unwrap
		// of pointer; for defined types based on string, Kind is still String.
		return false
	}
}

func flattenModelFields(model any) map[string]any {
	fields := make(map[string]any)
	switch m := model.(type) {
	case types.KV:
		for k, v := range m {
			fields[k] = formatFieldValue(v)
		}
	case map[string]any:
		for k, v := range m {
			fields[k] = formatFieldValue(v)
		}
	case map[string]string:
		for k, v := range m {
			fields[k] = v
		}
	default:
		flat, err := structToMap(model)
		if err != nil {
			fields["value"] = fmt.Sprintf("%v", model)
			return fields
		}
		for k, v := range flat {
			fields[k] = formatFieldValue(v)
		}
	}
	return fields
}

func structToMap(model any) (map[string]any, error) {
	b, err := sonic.Marshal(model)
	if err != nil {
		return nil, err
	}
	var out map[string]any
	if err := sonic.Unmarshal(b, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func formatFieldValue(v any) string {
	if v == nil {
		return ""
	}
	switch t := v.(type) {
	case string:
		return t
	case []byte:
		return string(t)
	case fmt.Stringer:
		return t.String()
	case map[string]any, []any, types.KV:
		b, err := sonic.Marshal(t)
		if err != nil {
			return fmt.Sprintf("%v", t)
		}
		return string(b)
	default:
		rv := reflect.ValueOf(v)
		if rv.Kind() == reflect.Slice || rv.Kind() == reflect.Array || rv.Kind() == reflect.Map || rv.Kind() == reflect.Struct {
			b, err := sonic.Marshal(v)
			if err != nil {
				return fmt.Sprintf("%v", v)
			}
			return string(b)
		}
		return fmt.Sprintf("%v", v)
	}
}

func sortedKeys(set map[string]struct{}) []string {
	keys := make([]string, 0, len(set))
	for k := range set {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	return keys
}

func convertChart(v types.ChartMsg) protocol.Message {
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
	if strings.TrimSpace(v.Raw) == "" {
		return nil
	}
	return protocol.Message{
		{
			Type: "html",
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
	fields := map[string]any{
		"No":       v.No,
		"State":    instructStateLabel(v.State),
		"Priority": instructPriorityLabel(v.Priority),
	}
	if v.Object != "" {
		fields["Object"] = string(v.Object)
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
	for k, val := range v.Content {
		fields[k] = formatFieldValue(val)
	}
	return protocol.Message{
		{
			Type: "action_card",
			Data: map[string]any{
				"title":       fmt.Sprintf("Instruction: %s", v.No),
				"description": "",
				"fields":      fields,
			},
		},
	}
}

func instructStateLabel(s schema.InstructState) string {
	switch s {
	case schema.InstructCreate:
		return "create"
	case schema.InstructDone:
		return "done"
	case schema.InstructCancel:
		return "cancel"
	default:
		return "unknown"
	}
}

func instructPriorityLabel(p schema.InstructPriority) string {
	switch p {
	case schema.InstructPriorityHigh:
		return "high"
	case schema.InstructPriorityLow:
		return "low"
	case schema.InstructPriorityDefault:
		return "default"
	default:
		return fmt.Sprintf("%d", p)
	}
}

func convertKV(v types.KVMsg) protocol.Message {
	if len(v) == 0 {
		return nil
	}
	fields := make(map[string]any, len(v))
	for k, val := range v {
		fields[k] = formatFieldValue(val)
	}
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
	fields := flattenModelFields(data)
	if len(fields) == 0 {
		return nil
	}
	return protocol.Message{
		{
			Type: "kv",
			Data: map[string]any{
				"fields": fields,
			},
		},
	}
}
