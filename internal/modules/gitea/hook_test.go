package gitea

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHookFunctions(t *testing.T) {
	tests := []struct {
		name string
		fn   any
	}{
		{name: "hookIssueOpened should be defined", fn: hookIssueOpened},
		{name: "hookIssueCreated should be defined", fn: hookIssueCreated},
		{name: "hookIssueClosed should be defined", fn: hookIssueClosed},
		{name: "hookPush should be defined", fn: hookPush},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.NotNil(t, tt.fn)
		})
	}
}
