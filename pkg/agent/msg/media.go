package msg

import (
	"strings"
)

// KindFromMIME maps a MIME type to a MediaKind.
func KindFromMIME(mimeType string) (MediaKind, bool) {
	base := strings.ToLower(strings.TrimSpace(mimeType))
	if i := strings.IndexByte(base, ';'); i >= 0 {
		base = strings.TrimSpace(base[:i])
	}
	switch {
	case strings.HasPrefix(base, "image/"):
		return MediaKindImage, true
	case strings.HasPrefix(base, "audio/"):
		return MediaKindAudio, true
	case strings.HasPrefix(base, "video/"):
		return MediaKindVideo, true
	default:
		return "", false
	}
}

// MediaPlaceholder returns a short text stub for media-only turns (titles, previews).
func MediaPlaceholder(kind MediaKind) string {
	switch kind {
	case MediaKindImage:
		return "[image]"
	case MediaKindAudio:
		return "[audio]"
	case MediaKindVideo:
		return "[video]"
	default:
		return "[media]"
	}
}

// StripMediaParts returns a copy of parts with MediaPart entries removed.
func StripMediaParts(parts []ContentPart) []ContentPart {
	return FilterMediaParts(parts, func(MediaKind) bool { return false })
}

// FilterMediaParts returns a copy of parts keeping only MediaPart entries for which keep is true.
func FilterMediaParts(parts []ContentPart, keep func(MediaKind) bool) []ContentPart {
	if len(parts) == 0 {
		return parts
	}
	out := make([]ContentPart, 0, len(parts))
	for _, part := range parts {
		if mp, ok := part.(MediaPart); ok {
			if keep != nil && keep(mp.Kind) {
				out = append(out, part)
			}
			continue
		}
		out = append(out, part)
	}
	return out
}

// StripMediaFromMessages removes MediaPart from user (and custom) messages for tool-model turns.
func StripMediaFromMessages(messages []AgentMessage) []AgentMessage {
	return FilterMediaFromMessages(messages, func(MediaKind) bool { return false })
}

// FilterMediaFromMessages keeps MediaPart entries for which keep returns true; dropped
// media-only user turns are replaced with a text stub.
func FilterMediaFromMessages(messages []AgentMessage, keep func(MediaKind) bool) []AgentMessage {
	if len(messages) == 0 {
		return messages
	}
	out := make([]AgentMessage, len(messages))
	for i, message := range messages {
		switch m := message.(type) {
		case UserMessage:
			filtered := FilterMediaParts(m.Parts, keep)
			if len(filtered) == 0 {
				filtered = []ContentPart{TextPart{Text: mediaStubFromParts(m.Parts)}}
			}
			m.Parts = filtered
			out[i] = m
		case CustomMessage:
			m.Parts = FilterMediaParts(m.Parts, keep)
			out[i] = m
		default:
			out[i] = message
		}
	}
	return out
}

func mediaStubFromParts(parts []ContentPart) string {
	var stubs []string
	for _, part := range parts {
		if mp, ok := part.(MediaPart); ok {
			stubs = append(stubs, MediaPlaceholder(mp.Kind))
		}
	}
	if len(stubs) == 0 {
		return "[media]"
	}
	return strings.Join(stubs, " ")
}
