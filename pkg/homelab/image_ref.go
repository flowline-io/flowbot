package homelab

import "strings"

// WatchImageRef returns the image reference and tag to watch. Digest-pinned
// references and empty image names are not watchable.
func WatchImageRef(image string) (ref, tag string, ok bool) {
	image = strings.TrimSpace(image)
	if image == "" || strings.Contains(image, "@") {
		return "", "", false
	}
	tag = ParseImageVersion(image)
	if tag != "" {
		return image, tag, true
	}
	return image + ":latest", "latest", true
}

// ServiceWatchesImage reports whether a compose service is included in image
// update checks. Digest-pinned images and flowbot.image.watch=false are excluded.
func ServiceWatchesImage(svc ComposeService) bool {
	if v, ok := svc.Labels[LabelImageWatch]; ok && strings.EqualFold(strings.TrimSpace(v), "false") {
		return false
	}
	_, _, ok := WatchImageRef(svc.Image)
	return ok
}

// ImageUpdateIdempotencyKey is DataEvent.IdempotencyKey for one pending digest.
func ImageUpdateIdempotencyKey(app, service, remoteDigest string) string {
	return app + "/" + service + "/" + remoteDigest
}

// ImageUpdateEntityID is DataEvent.EntityID for one service.
func ImageUpdateEntityID(app, service string) string {
	return app + "/" + service
}
