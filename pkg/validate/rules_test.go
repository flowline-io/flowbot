package validate_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/pkg/validate"
)

func TestValidateVar(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		value   any
		tag     string
		wantErr bool
	}{
		{
			name:    "required title passes",
			value:   "Hello",
			tag:     validate.TagTitle,
			wantErr: false,
		},
		{
			name:    "empty required title fails",
			value:   "",
			tag:     validate.TagTitle,
			wantErr: true,
		},
		{
			name:    "title exceeding max length fails",
			value:   strings.Repeat("a", validate.TitleMaxLen+1),
			tag:     validate.TagTitle,
			wantErr: true,
		},
		{
			name:    "optional title empty passes",
			value:   "",
			tag:     validate.TagTitleOptional,
			wantErr: false,
		},
		{
			name:    "valid url passes",
			value:   "https://example.com/path",
			tag:     validate.TagURL,
			wantErr: false,
		},
		{
			name:    "invalid url fails",
			value:   "not-a-url",
			tag:     validate.TagURL,
			wantErr: true,
		},
		{
			name:    "positive id passes",
			value:   1,
			tag:     validate.TagID,
			wantErr: false,
		},
		{
			name:    "zero id fails",
			value:   0,
			tag:     validate.TagID,
			wantErr: true,
		},
		{
			name:    "non-negative zero passes",
			value:   0,
			tag:     validate.TagNonNegative,
			wantErr: false,
		},
		{
			name:    "negative number fails non-negative tag",
			value:   -1,
			tag:     validate.TagNonNegative,
			wantErr: true,
		},
		{
			name:    "required string passes",
			value:   "x",
			tag:     validate.TagRequired,
			wantErr: false,
		},
		{
			name:    "empty required string fails",
			value:   "",
			tag:     validate.TagRequired,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := validate.ValidateVar(tt.value, tt.tag)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestValidationConstants(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		got  int
		want int
	}{
		{name: "title max length", got: validate.TitleMaxLen, want: 200},
		{name: "description max length", got: validate.DescMaxLen, want: 2000},
		{name: "url max length", got: validate.URLMaxLen, want: 2048},
		{name: "max file size bytes", got: validate.MaxFileSizeBytes, want: validate.MaxFileSizeMB * 1024 * 1024},
		{name: "max tags count", got: validate.MaxTagsCount, want: 50},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, tt.got)
		})
	}
}
