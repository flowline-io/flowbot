package template

import (
	"testing"

	"github.com/flowline-io/flowbot/internal/types"
	"github.com/stretchr/testify/require"
)

func TestParse(t *testing.T) {
	txt, err := Parse(types.Context{}, "Welcome $1 $2", "user", "user2")
	if err != nil {
		t.Fatal(err)
	}
	require.Equal(t, "Welcome user user2", txt)
}
