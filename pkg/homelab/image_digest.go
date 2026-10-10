package homelab

import (
	"fmt"
	"slices"
	"strings"

	"github.com/bytedance/sonic"
)

func normalizeDigest(s string) string {
	s = strings.TrimSpace(s)
	if i := strings.LastIndex(s, "@"); i >= 0 {
		s = strings.TrimSpace(s[i+1:])
	}
	return s
}

func parseManifestDigest(output string) string {
	lines := strings.Split(output, "\n")
	for _, line := range slices.Backward(lines) {
		d := normalizeDigest(line)
		if strings.HasPrefix(d, "sha256:") && len(d) > len("sha256:") {
			return d
		}
	}
	return ""
}

func parseRepoDigestsJSON(output string) ([]string, error) {
	s := strings.TrimSpace(output)
	if s == "" || s == "null" {
		return nil, nil
	}
	var digests []string
	if err := sonic.Unmarshal([]byte(s), &digests); err != nil {
		return nil, fmt.Errorf("parse RepoDigests: %w", err)
	}
	return digests, nil
}

func imageNameWithoutTag(image string) string {
	image = strings.TrimSpace(image)
	if i := strings.Index(image, "@"); i >= 0 {
		image = image[:i]
	}
	if tag := ParseImageVersion(image); tag != "" {
		return strings.TrimSuffix(image, ":"+tag)
	}
	return image
}

func canonicalImageName(name string) string {
	name = strings.TrimPrefix(name, "docker.io/")
	name = strings.TrimPrefix(name, "library/")
	return name
}

func pickRepoDigest(digests []string, image string) string {
	want := canonicalImageName(imageNameWithoutTag(image))
	for _, raw := range digests {
		name, dig, ok := strings.Cut(strings.TrimSpace(raw), "@")
		if !ok {
			continue
		}
		dig = strings.TrimSpace(dig)
		if !strings.HasPrefix(dig, "sha256:") {
			continue
		}
		if canonicalImageName(name) == want {
			return dig
		}
	}
	return ""
}

func repoDigestFromInspect(output, image string) (string, error) {
	digests, err := parseRepoDigestsJSON(output)
	if err != nil {
		return "", err
	}
	return pickRepoDigest(digests, image), nil
}

func firstNonEmptyLine(s string) string {
	for line := range strings.SplitSeq(s, "\n") {
		if t := strings.TrimSpace(line); t != "" {
			return t
		}
	}
	return ""
}
