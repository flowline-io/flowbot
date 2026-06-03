package homelab

import "strings"

// ParseImageVersion extracts the version tag from a Docker image reference.
// Returns the portion after the last colon if present and represents a tag
// (not a registry port); returns an empty string when there is no valid tag
// or the image uses a digest reference (@sha256:...).
func ParseImageVersion(image string) string {
	if strings.Contains(image, "@") {
		return ""
	}
	if idx := strings.LastIndex(image, ":"); idx != -1 {
		tag := image[idx+1:]
		if strings.Contains(tag, "/") {
			return ""
		}
		return tag
	}
	return ""
}

// AppVersion extracts the application version from the first service
// that has a tagged image. Returns an empty string if no tag is found.
func AppVersion(app App) string {
	for _, svc := range app.Services {
		if v := ParseImageVersion(svc.Image); v != "" {
			return v
		}
	}
	return ""
}
