package utils

import (
	"io"

	"github.com/bytedance/sonic"
)

func EncodeJSON(w io.Writer, v any) error {
	return EncodeJSONEscapeHTML(w, v, true)
}

func EncodeJSONEscapeHTML(w io.Writer, v any, esc bool) error {
	enc := sonic.Config{EscapeHTML: esc}.Froze().NewEncoder(w)
	return enc.Encode(v)
}

func EncodeJSONEscapeHTMLIndent(w io.Writer, v any, esc bool, indent string) error {
	enc := sonic.Config{EscapeHTML: esc}.Froze().NewEncoder(w)
	enc.SetIndent("", indent)
	return enc.Encode(v)
}

func DecodeJSON(r io.Reader, v any) error {
	dec := sonic.ConfigStd.NewDecoder(r)
	for {
		if err := dec.Decode(v); err == io.EOF {
			break
		} else if err != nil {
			return err
		}
	}
	return nil
}
