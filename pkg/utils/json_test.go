package utils

import (
	"bytes"
	"strings"
	"testing"
)

// TestEncodeJSON tests the EncodeJSON function
func TestEncodeJSON(t *testing.T) {
	type testStruct struct {
		Name string `json:"name"`
		Age  int    `json:"age"`
	}

	tests := []struct {
		name    string
		input   any
		wantErr bool
	}{
		{
			name: "simple_struct",
			input: testStruct{
				Name: "John",
				Age:  30,
			},
			wantErr: false,
		},
		{
			name:    "string_input",
			input:   "test string",
			wantErr: false,
		},
		{
			name:    "number_input",
			input:   123,
			wantErr: false,
		},
		{
			name:    "map_input",
			input:   map[string]string{"key": "value"},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			err := EncodeJSON(&buf, tt.input)

			if (err != nil) != tt.wantErr {
				t.Errorf("EncodeJSON() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && buf.Len() == 0 {
				t.Error("EncodeJSON() produced empty output")
			}
		})
	}
}

// TestEncodeJSONEscapeHTML tests the EncodeJSONEscapeHTML function
func TestEncodeJSONEscapeHTML(t *testing.T) {
	type args struct {
		v   any
		esc bool
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
		check   func(string) bool
	}{
		{
			name: "escape_html_true",
			args: args{
				v:   map[string]string{"html": "<script>alert('xss')</script>"},
				esc: true,
			},
			wantErr: false,
			check: func(output string) bool {
				return strings.Contains(output, "\\u003cscript\\u003e")
			},
		},
		{
			name: "escape_html_false",
			args: args{
				v:   map[string]string{"html": "<script>alert('xss')</script>"},
				esc: false,
			},
			wantErr: false,
			check: func(output string) bool {
				return strings.Contains(output, "<script>")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			err := EncodeJSONEscapeHTML(&buf, tt.args.v, tt.args.esc)

			if (err != nil) != tt.wantErr {
				t.Errorf("EncodeJSONEscapeHTML() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				output := buf.String()
				if tt.check != nil && !tt.check(output) {
					t.Errorf("EncodeJSONEscapeHTML() output check failed: %s", output)
				}
			}
		})
	}
}

// TestEncodeJSONEscapeHTMLIndent tests the EncodeJSONEscapeHTMLIndent function
func TestEncodeJSONEscapeHTMLIndent(t *testing.T) {
	type testStruct struct {
		Name string `json:"name"`
		Age  int    `json:"age"`
	}

	tests := []struct {
		name    string
		v       any
		esc     bool
		indent  string
		wantErr bool
	}{
		{
			name: "indent_with_spaces",
			v: testStruct{
				Name: "John",
				Age:  30,
			},
			esc:     true,
			indent:  "  ",
			wantErr: false,
		},
		{
			name: "indent_with_tabs",
			v: map[string]any{
				"key1": "value1",
				"key2": 123,
			},
			esc:     false,
			indent:  "\t",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			err := EncodeJSONEscapeHTMLIndent(&buf, tt.v, tt.esc, tt.indent)

			if (err != nil) != tt.wantErr {
				t.Errorf("EncodeJSONEscapeHTMLIndent() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				output := buf.String()
				if tt.indent == "  " && !strings.Contains(output, "  ") {
					t.Error("EncodeJSONEscapeHTMLIndent() should contain indentation")
				}
			}
		})
	}
}

// TestDecodeJSON tests the DecodeJSON function
func TestDecodeJSON(t *testing.T) {
	type testStruct struct {
		Name string `json:"name"`
		Age  int    `json:"age"`
	}

	tests := []struct {
		name    string
		input   string
		target  any
		wantErr bool
	}{
		{
			name:    "valid_json",
			input:   `{"name":"John","age":30}`,
			target:  &testStruct{},
			wantErr: false,
		},
		{
			name:    "invalid_json",
			input:   `{"name":"John","age":}`,
			target:  &testStruct{},
			wantErr: true,
		},
		{
			name:    "empty_json",
			input:   `{}`,
			target:  &testStruct{},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := strings.NewReader(tt.input)
			err := DecodeJSON(reader, tt.target)

			if (err != nil) != tt.wantErr {
				t.Errorf("DecodeJSON() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
