package lineage

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/pkg/types"
)

func TestVisibleSubject(t *testing.T) {
	t.Parallel()
	got, err := VisibleSubject("user-a", false, "")
	require.NoError(t, err)
	assert.Equal(t, "user-a", got)

	_, err = VisibleSubject("user-a", false, "user-b")
	require.ErrorIs(t, err, types.ErrForbidden)

	got, err = VisibleSubject("user-a", true, "")
	require.NoError(t, err)
	assert.Empty(t, got)

	got, err = VisibleSubject("user-a", true, "user-b")
	require.NoError(t, err)
	assert.Equal(t, "user-b", got)
}
