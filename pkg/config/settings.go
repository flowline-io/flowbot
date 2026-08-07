package config

import (
	"fmt"
	"reflect"
	"regexp"
	"strings"
	"time"

	"github.com/bytedance/sonic"
)

const (
	// MaskedSecret is the display value for configured secrets on the Settings page.
	MaskedSecret = "••••••••"
	// EmptyDisplay is the display value for empty non-secret values.
	EmptyDisplay = "(empty)"
	// NotSetDisplay is the display value for unset / nil values (including unset secrets).
	NotSetDisplay = "(not set)"

	maskedSecret  = MaskedSecret
	emptyDisplay  = EmptyDisplay
	notSetDisplay = NotSetDisplay
)

// SettingEntry is one flattened runtime config row for the Settings page.
type SettingEntry struct {
	// Path is the yaml-style dotted path (e.g. postgres.dsn, models[0].api_key).
	Path string
	// Value is the display value (secrets redacted).
	Value string
	// Description is the field godoc when available.
	Description string
	// Sensitive reports whether the value was redacted or treated as a secret.
	Sensitive bool
}

// SettingGroup is a top-level section of setting rows.
type SettingGroup struct {
	// Name is the top-level group id (root for top-level scalars, else first path segment).
	Name string
	// Entries are flattened rows in this group.
	Entries []SettingEntry
}

var indexSegmentPattern = regexp.MustCompile(`\[\d+]`)

// SettingsCatalog builds redacted, schema-complete setting groups from cfg.
func SettingsCatalog(cfg Type) []SettingGroup {
	var entries []SettingEntry
	walkValue(reflect.ValueOf(cfg), reflect.TypeOf(cfg), "", false, &entries)

	order := make([]string, 0)
	grouped := make(map[string][]SettingEntry)
	for _, e := range entries {
		e.Description = FieldDocs[normalizeDocPath(e.Path)]
		group := settingGroupName(e.Path)
		if _, ok := grouped[group]; !ok {
			order = append(order, group)
		}
		grouped[group] = append(grouped[group], e)
	}

	out := make([]SettingGroup, 0, len(order))
	for _, name := range order {
		out = append(out, SettingGroup{Name: name, Entries: grouped[name]})
	}
	return out
}

func settingGroupName(path string) string {
	if path == "" || !strings.Contains(path, ".") {
		return "root"
	}
	base := strings.SplitN(path, ".", 2)[0]
	return indexSegmentPattern.ReplaceAllString(base, "")
}

func normalizeDocPath(path string) string {
	return indexSegmentPattern.ReplaceAllString(path, "")
}

func isSensitivePath(path string, explicit bool) bool {
	if explicit {
		return true
	}
	for _, seg := range pathSegments(path) {
		if sensitiveSegment(seg) {
			return true
		}
	}
	return false
}

func pathSegments(path string) []string {
	path = indexSegmentPattern.ReplaceAllString(path, "")
	if path == "" {
		return nil
	}
	return strings.Split(path, ".")
}

func sensitiveSegment(seg string) bool {
	s := strings.ToLower(strings.TrimSpace(seg))
	switch s {
	case "password", "secret", "token", "dsn", "key", "api_key", "apikey",
		"access_token", "bearer_token", "client_secret", "signing_secret",
		"verification_token", "app_token", "bot_token", "app_secret",
		"webhook_token", "sign_secret", "hmac_secret", "ssh_password",
		"private_key", "public_key", "consumer_key", "encryption_key",
		"secret_access_key", "credentials":
		return true
	}
	for _, suffix := range []string{
		"_password", "_secret", "_token", "_api_key", "_access_key", "_encryption_key", "_key",
	} {
		if strings.HasSuffix(s, suffix) {
			return true
		}
	}
	return false
}

func walkValue(v reflect.Value, t reflect.Type, path string, parentNil bool, out *[]SettingEntry) {
	for v.Kind() == reflect.Interface || v.Kind() == reflect.Pointer {
		if v.IsNil() {
			if path != "" && t != nil && t.Kind() == reflect.Struct {
				walkStructSchema(t, path, true, out)
				return
			}
			if path != "" {
				*out = append(*out, SettingEntry{
					Path:      path,
					Value:     notSetDisplay,
					Sensitive: isSensitivePath(path, false),
				})
			}
			return
		}
		v = v.Elem()
		t = v.Type()
	}

	if !v.IsValid() {
		if path != "" {
			*out = append(*out, SettingEntry{Path: path, Value: notSetDisplay})
		}
		return
	}

	switch v.Kind() {
	case reflect.Struct:
		if v.Type() == reflect.TypeOf(time.Duration(0)) {
			appendLeaf(path, v, false, parentNil, out)
			return
		}
		walkStruct(v, path, parentNil, out)
	case reflect.Map:
		walkMap(v, path, out)
	case reflect.Slice, reflect.Array:
		walkSlice(v, path, out)
	default:
		appendLeaf(path, v, false, parentNil, out)
	}
}

func walkStruct(v reflect.Value, path string, parentNil bool, out *[]SettingEntry) {
	t := v.Type()
	for i := 0; i < t.NumField(); i++ {
		sf := t.Field(i)
		if sf.PkgPath != "" { // unexported
			continue
		}
		key := fieldPathKey(sf)
		if key == "-" {
			continue
		}
		walkStructField(v.Field(i), sf, joinPath(path, key), parentNil, out)
	}
}

func walkStructField(fv reflect.Value, sf reflect.StructField, childPath string, parentNil bool, out *[]SettingEntry) {
	explicit := sf.Tag.Get("sensitive") == "true"
	ft := sf.Type

	if ft.Kind() == reflect.Pointer {
		if fv.IsNil() || parentNil {
			if deref := ft.Elem(); deref.Kind() == reflect.Struct && deref != reflect.TypeOf(time.Time{}) {
				walkStructSchema(deref, childPath, true, out)
				return
			}
			*out = append(*out, SettingEntry{
				Path:      childPath,
				Value:     notSetDisplay,
				Sensitive: isSensitivePath(childPath, explicit),
			})
			return
		}
	}

	switch {
	case ft.Kind() == reflect.Struct && ft != reflect.TypeOf(time.Duration(0)) && ft != reflect.TypeOf(time.Time{}):
		walkValue(fv, ft, childPath, parentNil, out)
	case ft.Kind() == reflect.Slice || ft.Kind() == reflect.Array:
		walkSliceField(fv, ft, childPath, explicit, out)
	case ft.Kind() == reflect.Map:
		walkMapField(fv, childPath, explicit, out)
	case ft.Kind() == reflect.Interface || ft.Kind() == reflect.Pointer:
		walkValue(fv, ft, childPath, false, out)
	default:
		appendLeaf(childPath, fv, explicit, parentNil, out)
	}
}

func walkStructSchema(t reflect.Type, path string, asNotSet bool, out *[]SettingEntry) {
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct {
		*out = append(*out, SettingEntry{
			Path:      path,
			Value:     notSetDisplay,
			Sensitive: isSensitivePath(path, false),
		})
		return
	}
	for i := 0; i < t.NumField(); i++ {
		sf := t.Field(i)
		if sf.PkgPath != "" {
			continue
		}
		key := fieldPathKey(sf)
		if key == "-" {
			continue
		}
		childPath := joinPath(path, key)
		explicit := sf.Tag.Get("sensitive") == "true"
		ft := sf.Type
		for ft.Kind() == reflect.Pointer {
			ft = ft.Elem()
		}
		switch {
		case ft == reflect.TypeOf(time.Duration(0)) || ft == reflect.TypeOf(time.Time{}):
			*out = append(*out, SettingEntry{
				Path:      childPath,
				Value:     notSetDisplay,
				Sensitive: isSensitivePath(childPath, explicit),
			})
		case ft.Kind() == reflect.Struct:
			walkStructSchema(ft, childPath, asNotSet, out)
		case ft.Kind() == reflect.Slice || ft.Kind() == reflect.Array || ft.Kind() == reflect.Map || ft.Kind() == reflect.Interface:
			*out = append(*out, SettingEntry{
				Path:      childPath,
				Value:     notSetDisplay,
				Sensitive: isSensitivePath(childPath, explicit),
			})
		default:
			*out = append(*out, SettingEntry{
				Path:      childPath,
				Value:     notSetDisplay,
				Sensitive: isSensitivePath(childPath, explicit),
			})
		}
	}
}

func walkSliceField(v reflect.Value, t reflect.Type, path string, explicit bool, out *[]SettingEntry) {
	if !v.IsValid() || v.IsNil() || v.Len() == 0 {
		*out = append(*out, SettingEntry{
			Path:      path,
			Value:     emptyDisplay,
			Sensitive: isSensitivePath(path, explicit),
		})
		return
	}
	elem := t.Elem()
	for elem.Kind() == reflect.Pointer {
		elem = elem.Elem()
	}
	// Expand composite elements so nested secrets (e.g. modules[].auth.password) are redacted by path.
	switch elem.Kind() {
	case reflect.Struct:
		if elem != reflect.TypeOf(time.Time{}) {
			for i := 0; i < v.Len(); i++ {
				walkValue(v.Index(i), v.Index(i).Type(), fmt.Sprintf("%s[%d]", path, i), false, out)
			}
			return
		}
	case reflect.Map, reflect.Interface, reflect.Slice, reflect.Array:
		for i := 0; i < v.Len(); i++ {
			walkValue(v.Index(i), v.Index(i).Type(), fmt.Sprintf("%s[%d]", path, i), false, out)
		}
		return
	}
	appendLeaf(path, v, explicit, false, out)
}

func walkSlice(v reflect.Value, path string, out *[]SettingEntry) {
	walkSliceField(v, v.Type(), path, false, out)
}

func walkMapField(v reflect.Value, path string, explicit bool, out *[]SettingEntry) {
	if !v.IsValid() || v.IsNil() || v.Len() == 0 {
		*out = append(*out, SettingEntry{
			Path:      path,
			Value:     emptyDisplay,
			Sensitive: isSensitivePath(path, explicit),
		})
		return
	}
	walkMap(v, path, out)
}

func walkMap(v reflect.Value, path string, out *[]SettingEntry) {
	if !v.IsValid() || v.IsNil() || v.Len() == 0 {
		*out = append(*out, SettingEntry{
			Path:  path,
			Value: emptyDisplay,
		})
		return
	}
	keys := v.MapKeys()
	// Stable-ish order by string key.
	type keyed struct {
		key string
		val reflect.Value
	}
	items := make([]keyed, 0, len(keys))
	for _, k := range keys {
		items = append(items, keyed{key: fmt.Sprint(k.Interface()), val: v.MapIndex(k)})
	}
	for i := 0; i < len(items); i++ {
		for j := i + 1; j < len(items); j++ {
			if items[j].key < items[i].key {
				items[i], items[j] = items[j], items[i]
			}
		}
	}
	for _, item := range items {
		child := joinPath(path, item.key)
		walkValue(item.val, item.val.Type(), child, false, out)
	}
}

func appendLeaf(path string, v reflect.Value, explicit, forceNotSet bool, out *[]SettingEntry) {
	if path == "" {
		return
	}
	sensitive := isSensitivePath(path, explicit)
	if forceNotSet {
		*out = append(*out, SettingEntry{Path: path, Value: notSetDisplay, Sensitive: sensitive})
		return
	}
	if sensitive {
		if isEmptyValue(v) {
			*out = append(*out, SettingEntry{Path: path, Value: notSetDisplay, Sensitive: true})
			return
		}
		*out = append(*out, SettingEntry{Path: path, Value: maskedSecret, Sensitive: true})
		return
	}
	*out = append(*out, SettingEntry{Path: path, Value: formatDisplayValue(v), Sensitive: false})
}

func isEmptyValue(v reflect.Value) bool {
	if !v.IsValid() {
		return true
	}
	switch v.Kind() {
	case reflect.String:
		return v.String() == ""
	case reflect.Slice, reflect.Map, reflect.Array:
		return v.Len() == 0
	case reflect.Pointer, reflect.Interface:
		return v.IsNil()
	case reflect.Bool, reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr,
		reflect.Float32, reflect.Float64:
		return v.IsZero()
	default:
		return v.IsZero()
	}
}

func formatDisplayValue(v reflect.Value) string {
	if !v.IsValid() {
		return notSetDisplay
	}
	if v.Type() == reflect.TypeOf(time.Duration(0)) {
		d := time.Duration(v.Int())
		if d == 0 {
			return emptyDisplay
		}
		return d.String()
	}
	switch v.Kind() {
	case reflect.String:
		if v.String() == "" {
			return emptyDisplay
		}
		return v.String()
	case reflect.Bool:
		return fmt.Sprintf("%t", v.Bool())
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return fmt.Sprintf("%d", v.Int())
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return fmt.Sprintf("%d", v.Uint())
	case reflect.Float32, reflect.Float64:
		return fmt.Sprintf("%v", v.Float())
	case reflect.Slice, reflect.Array, reflect.Map:
		if v.Len() == 0 {
			return emptyDisplay
		}
		return marshalDisplayJSON(v.Interface())
	default:
		if v.IsZero() {
			return emptyDisplay
		}
		return fmt.Sprint(v.Interface())
	}
}

// marshalDisplayJSON marshals v for display after deep-redacting sensitive map keys.
func marshalDisplayJSON(v any) string {
	b, err := sonic.Marshal(redactDeep(v))
	if err != nil {
		return fmt.Sprint(v)
	}
	return string(b)
}

// redactDeep returns a copy of v with sensitive map keys masked.
func redactDeep(v any) any {
	if v == nil {
		return nil
	}
	rv := reflect.ValueOf(v)
	for rv.Kind() == reflect.Interface || rv.Kind() == reflect.Pointer {
		if rv.IsNil() {
			return nil
		}
		rv = rv.Elem()
	}
	switch rv.Kind() {
	case reflect.Map:
		out := make(map[string]any, rv.Len())
		for _, key := range rv.MapKeys() {
			k := fmt.Sprint(key.Interface())
			val := rv.MapIndex(key).Interface()
			if sensitiveSegment(k) {
				if isEmptyAny(val) {
					out[k] = notSetDisplay
				} else {
					out[k] = maskedSecret
				}
				continue
			}
			out[k] = redactDeep(val)
		}
		return out
	case reflect.Slice, reflect.Array:
		out := make([]any, rv.Len())
		for i := 0; i < rv.Len(); i++ {
			out[i] = redactDeep(rv.Index(i).Interface())
		}
		return out
	default:
		return v
	}
}

func isEmptyAny(v any) bool {
	if v == nil {
		return true
	}
	switch x := v.(type) {
	case string:
		return x == ""
	default:
		return reflect.ValueOf(v).IsZero()
	}
}

func fieldPathKey(sf reflect.StructField) string {
	for _, tag := range []string{"yaml", "mapstructure", "json"} {
		if v := sf.Tag.Get(tag); v != "" {
			name := strings.Split(v, ",")[0]
			if name == "-" {
				// Prefer another tag; if all are "-", fall through to field name.
				continue
			}
			if name != "" {
				return name
			}
		}
	}
	// Store and similar fields use yaml:"-" on every tag; still expose under snake field name.
	return toSnake(sf.Name)
}

func joinPath(parent, key string) string {
	if parent == "" {
		return key
	}
	return parent + "." + key
}

func toSnake(name string) string {
	if name == "" {
		return name
	}
	var b strings.Builder
	for i, r := range name {
		if i > 0 && r >= 'A' && r <= 'Z' {
			_, _ = b.WriteRune('_')
		}
		if r >= 'A' && r <= 'Z' {
			_, _ = b.WriteRune(r - 'A' + 'a')
		} else {
			_, _ = b.WriteRune(r)
		}
	}
	return b.String()
}
