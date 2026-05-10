package docker

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParse(t *testing.T) {
	tests := []struct {
		name      string
		ref       string
		expDomain string
		expPath   string
		expTag    string
	}{
		{
			name:      "ubuntu with tag",
			ref:       "ubuntu:mantic",
			expDomain: "",
			expPath:   "ubuntu",
			expTag:    "mantic",
		},
		{
			name:      "localhost with tag",
			ref:       "localhost:9090/ubuntu:mantic",
			expDomain: "localhost:9090",
			expPath:   "ubuntu",
			expTag:    "mantic",
		},
		{
			name:      "localhost with hyphenated tag",
			ref:       "localhost:9090/ubuntu:mantic-2.7",
			expDomain: "localhost:9090",
			expPath:   "ubuntu",
			expTag:    "mantic-2.7",
		},
		{
			name:      "registry with hyphenated tag",
			ref:       "my-registry/ubuntu:mantic-2.7",
			expDomain: "my-registry",
			expPath:   "ubuntu",
			expTag:    "mantic-2.7",
		},
		{
			name:      "registry without tag",
			ref:       "my-registry/ubuntu",
			expDomain: "my-registry",
			expPath:   "ubuntu",
			expTag:    "",
		},
		{
			name:      "simple name without tag",
			ref:       "ubuntu",
			expDomain: "",
			expPath:   "ubuntu",
			expTag:    "",
		},
		{
			name:      "simple name with latest tag",
			ref:       "ubuntu:latest",
			expDomain: "",
			expPath:   "ubuntu",
			expTag:    "latest",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ref, err := parseRef(tt.ref)
			assert.NoError(t, err)
			assert.Equal(t, tt.expDomain, ref.domain)
			assert.Equal(t, tt.expPath, ref.path)
			assert.Equal(t, tt.expTag, ref.tag)
		})
	}
}
