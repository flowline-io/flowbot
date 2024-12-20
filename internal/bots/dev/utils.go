package dev

import (
	"context"
	"crypto/sha1"
	"fmt"
	"hash"
	"reflect"
	"regexp"
	"sort"
	"strconv"

	"github.com/flowline-io/flowbot/pkg/cache"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/types"
)

func unique(ctx context.Context, id string, latest []any) ([]types.KV, error) {
	result := make([]types.KV, 0)
	uniqueKey := fmt.Sprintf("unique:%s", id)

	for _, item := range latest {
		val, err := kvHash(item)
		if err != nil {
			return nil, fmt.Errorf("failed to hash kv: %w", err)
		}
		if len(val) == 0 {
			continue
		}
		b, err := cache.DB.SAdd(ctx, uniqueKey, val).Result()
		if err != nil {
			return nil, fmt.Errorf("failed to set unique key: %w", err)
		}
		if b == 1 {
			kv, ok := item.(map[string]any)
			if !ok {
				continue
			}
			result = append(result, kv)
			flog.Info("[unique] key: %s added: %s", id, val)
		}
	}

	return result, nil
}

func kvHash(item any) (string, error) {
	h := sha1.New()
	if err := writeHash(h, reflect.ValueOf(item)); err != nil {
		return "", fmt.Errorf("failed to hash: %w", err)
	}
	return fmt.Sprintf("%x", h.Sum(nil)), nil
}

func writeHash(h hash.Hash, v reflect.Value) error {
	if !v.IsValid() {
		_, _ = h.Write([]byte("null"))
		return nil
	}

	switch v.Kind() {
	case reflect.Bool:
		if v.Bool() {
			_, _ = h.Write([]byte("t"))
		} else {
			_, _ = h.Write([]byte("f"))
		}

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		_, _ = h.Write([]byte(strconv.FormatInt(v.Int(), 10)))

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		_, _ = h.Write([]byte(strconv.FormatUint(v.Uint(), 10)))

	case reflect.Float32, reflect.Float64:
		_, _ = h.Write([]byte(strconv.FormatFloat(v.Float(), 'f', -1, 64)))

	case reflect.String:
		_, _ = h.Write([]byte(v.String()))

	case reflect.Slice, reflect.Array:
		_, _ = h.Write([]byte("["))
		for i := 0; i < v.Len(); i++ {
			if i > 0 {
				_, _ = h.Write([]byte(","))
			}
			if err := writeHash(h, v.Index(i)); err != nil {
				return err
			}
		}
		_, _ = h.Write([]byte("]"))

	case reflect.Map:
		keys := v.MapKeys()
		sortedKeys := make([]string, len(keys))
		for i, k := range keys {
			sortedKeys[i] = fmt.Sprint(k.Interface())
		}
		sort.Strings(sortedKeys)

		_, _ = h.Write([]byte("{"))
		for i, keyStr := range sortedKeys {
			if i > 0 {
				_, _ = h.Write([]byte(","))
			}
			_, _ = h.Write([]byte(keyStr))
			_, _ = h.Write([]byte(":"))
			for _, k := range keys {
				if fmt.Sprint(k.Interface()) == keyStr {
					if err := writeHash(h, v.MapIndex(k)); err != nil {
						return err
					}
					break
				}
			}
		}
		_, _ = h.Write([]byte("}"))

	case reflect.Struct:
		t := v.Type()
		_, _ = h.Write([]byte("{"))
		for i := 0; i < v.NumField(); i++ {
			if i > 0 {
				_, _ = h.Write([]byte(","))
			}
			field := t.Field(i)
			_, _ = h.Write([]byte(field.Name))
			_, _ = h.Write([]byte(":"))
			if err := writeHash(h, v.Field(i)); err != nil {
				return err
			}
		}
		_, _ = h.Write([]byte("}"))

	case reflect.Ptr, reflect.Interface:
		if v.IsNil() {
			_, _ = h.Write([]byte("null"))
			return nil
		}
		return writeHash(h, v.Elem())

	default:
		return fmt.Errorf("unsupported type: %v", v.Kind())
	}

	return nil
}

func kvGrep(pattern string, input types.KV) (types.KV, error) {
	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, fmt.Errorf("invalid pattern: %w", err)
	}

	var data = make(map[string][]types.KV)

	for k, v := range input {
		list, ok := v.([]any)
		if !ok {
			continue
		}

		for _, item := range list {
			kv, ok := item.(map[string]any)
			if !ok {
				continue
			}

			for _, value := range kv {
				valueStr, ok := value.(string)
				if !ok {
					continue
				}

				if re.MatchString(valueStr) {
					data[k] = append(data[k], kv)
				}
			}
		}
	}

	result := make(types.KV)
	for k := range data {
		result[k] = data[k]
	}

	return result, nil
}
