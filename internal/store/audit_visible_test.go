package store

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/pkg/types"
)

func TestVisibleSubject(t *testing.T) {
	t.Parallel()
	got, err := visibleSubject(AuditCaller{UID: "user-a"})
	require.NoError(t, err)
	assert.Equal(t, "user-a", got)

	_, err = visibleSubject(AuditCaller{UID: "user-a", SubjectID: "user-b"})
	require.ErrorIs(t, err, types.ErrForbidden)

	got, err = visibleSubject(AuditCaller{UID: "user-a", Admin: true})
	require.NoError(t, err)
	assert.Empty(t, got)

	got, err = visibleSubject(AuditCaller{UID: "user-a", Admin: true, SubjectID: "user-b"})
	require.NoError(t, err)
	assert.Equal(t, "user-b", got)
}
