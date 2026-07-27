package webauth_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/flowline-io/flowbot/pkg/webauth"
)

func TestLooksLikeTOTPCode(t *testing.T) {
	assert.True(t, webauth.LooksLikeTOTPCode("123456"))
	assert.False(t, webauth.LooksLikeTOTPCode("ABCDEFGH"))
	assert.False(t, webauth.LooksLikeTOTPCode("12345"))
	assert.False(t, webauth.LooksLikeTOTPCode("1234567"))
}
