package docker

import (
	"encoding/base64"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDecodeBase64Auth(t *testing.T) {
	for _, tc := range base64TestCases() {
		t.Run(tc.name, func(t *testing.T) {
			u, p, err := decodeBase64Auth(tc.config)
			if tc.expErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tc.expUser, u)
			assert.Equal(t, tc.expPass, p)
		})
	}
}

func TestGetRegistryCredentials(t *testing.T) {
	for _, tc := range base64TestCases() {
		t.Run(tc.name, func(t *testing.T) {
			config := config{
				AuthConfigs: map[string]authConfig{
					"some.domain": tc.config,
				},
			}
			u, p, err := config.getRegistryCredentials("some.domain")
			if tc.expErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tc.expUser, u)
			assert.Equal(t, tc.expPass, p)
		})
	}
}

type base64TestCase struct {
	name    string
	config  authConfig
	expUser string
	expPass string
	expErr  bool
}

func base64TestCases() []base64TestCase {
	cases := []base64TestCase{
		{name: "empty"},
		{name: "not base64", expErr: true, config: authConfig{Auth: "not base64"}},
		{name: "invalid format", expErr: true, config: authConfig{
			Auth: base64.StdEncoding.EncodeToString([]byte("invalid format")),
		}},
		{name: "happy case", expUser: "user", expPass: "pass", config: authConfig{
			Auth: base64.StdEncoding.EncodeToString([]byte("user:pass")),
		}},
	}

	return cases
}
