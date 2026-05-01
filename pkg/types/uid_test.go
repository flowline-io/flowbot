package types

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUid_IsZero(t *testing.T) {
	assert.True(t, ZeroUid.IsZero())
	assert.True(t, Uid("").IsZero())
	assert.False(t, Uid("user-1").IsZero())
	assert.False(t, Uid("x").IsZero())
}

func TestUid_String(t *testing.T) {
	assert.Equal(t, "test-id", Uid("test-id").String())
	assert.Equal(t, "", Uid("").String())
	assert.Equal(t, string(ZeroUid), ZeroUid.String())
}
