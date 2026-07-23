package chatagent

import (
	"context"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	"github.com/bytedance/sonic"

	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/pkg/agent/llm"
	"github.com/flowline-io/flowbot/pkg/agent/model"
	"github.com/flowline-io/flowbot/pkg/agent/msg"
	"github.com/flowline-io/flowbot/pkg/cache"
	"github.com/flowline-io/flowbot/pkg/config"
	"github.com/flowline-io/flowbot/pkg/media"
	"github.com/flowline-io/flowbot/pkg/types"
)

const (
	maxAttachmentsPerMessage = 8
)

var allowedMIMETypes = map[string]msg.MediaKind{
	"image/jpeg":      msg.MediaKindImage,
	"image/png":       msg.MediaKindImage,
	"image/webp":      msg.MediaKindImage,
	"image/gif":       msg.MediaKindImage,
	"audio/mpeg":      msg.MediaKindAudio,
	"audio/mp3":       msg.MediaKindAudio,
	"audio/wav":       msg.MediaKindAudio,
	"audio/x-wav":     msg.MediaKindAudio,
	"audio/ogg":       msg.MediaKindAudio,
	"audio/mp4":       msg.MediaKindAudio,
	"video/mp4":       msg.MediaKindVideo,
	"video/webm":      msg.MediaKindVideo,
	"video/quicktime": msg.MediaKindVideo,
}

// AttachmentRef is a session-scoped media reference on a user turn.
type AttachmentRef struct {
	FileID   string `json:"file_id"`
	MIMEType string `json:"mime_type,omitempty"`
	Kind     string `json:"kind,omitempty"`
}

// MediaUploadResult is returned after a successful session media upload.
type MediaUploadResult struct {
	FileID   string        `json:"file_id"`
	MIMEType string        `json:"mime_type"`
	Kind     msg.MediaKind `json:"kind"`
	Name     string        `json:"name,omitempty"`
	Size     int64         `json:"size"`
}

type sessionMediaBinding struct {
	FileID   string        `json:"file_id"`
	Session  string        `json:"session_id"`
	Owner    string        `json:"owner"`
	MIMEType string        `json:"mime_type"`
	Kind     msg.MediaKind `json:"kind"`
}

// ValidateMIMEAllowlist checks MIME against the strict multimodal allowlist.
func ValidateMIMEAllowlist(mimeType string) (msg.MediaKind, error) {
	base := strings.ToLower(strings.TrimSpace(mimeType))
	if i := strings.IndexByte(base, ';'); i >= 0 {
		base = strings.TrimSpace(base[:i])
	}
	kind, ok := allowedMIMETypes[base]
	if !ok {
		return "", types.Errorf(types.ErrInvalidArgument, "unsupported media type %q", mimeType)
	}
	return kind, nil
}

// UploadSessionMedia stores a file via the configured media handler and binds it to the session.
func UploadSessionMedia(ctx context.Context, sessionID, ownerUID, filename, mimeType string, r io.ReadSeeker, size int64) (MediaUploadResult, error) {
	sessionID = strings.TrimSpace(sessionID)
	ownerUID = strings.TrimSpace(ownerUID)
	if sessionID == "" || ownerUID == "" {
		return MediaUploadResult{}, types.Errorf(types.ErrInvalidArgument, "session and owner are required")
	}
	kind, err := ValidateMIMEAllowlist(mimeType)
	if err != nil {
		return MediaUploadResult{}, err
	}
	if store.FileSystem == nil {
		return MediaUploadResult{}, types.Errorf(types.ErrInvalidArgument, "media handler is not configured")
	}
	if size <= 0 {
		return MediaUploadResult{}, types.Errorf(types.ErrInvalidArgument, "empty file")
	}
	fdef := &types.FileDef{
		ObjHeader: types.ObjHeader{Id: types.Id()},
		Name:      filename,
		MimeType:  mimeType,
		Size:      size,
		User:      ownerUID,
	}
	_, written, err := store.FileSystem.Upload(fdef, r)
	if err != nil {
		return MediaUploadResult{}, fmt.Errorf("upload media: %w", err)
	}
	binding := sessionMediaBinding{
		FileID:   fdef.Id,
		Session:  sessionID,
		Owner:    ownerUID,
		MIMEType: mimeType,
		Kind:     kind,
	}
	if err := saveSessionMediaBinding(ctx, binding); err != nil {
		return MediaUploadResult{}, err
	}
	return MediaUploadResult{
		FileID:   fdef.Id,
		MIMEType: mimeType,
		Kind:     kind,
		Name:     filename,
		Size:     written,
	}, nil
}

// ResolveAttachments validates ownership and builds MediaPart values for a run.
func ResolveAttachments(ctx context.Context, sessionID, ownerUID string, refs []AttachmentRef) ([]msg.ContentPart, error) {
	if len(refs) == 0 {
		return nil, nil
	}
	if len(refs) > maxAttachmentsPerMessage {
		return nil, types.Errorf(types.ErrInvalidArgument, "at most %d attachments per message", maxAttachmentsPerMessage)
	}
	parts := make([]msg.ContentPart, 0, len(refs))
	for _, ref := range refs {
		fileID := strings.TrimSpace(ref.FileID)
		if fileID == "" {
			return nil, types.Errorf(types.ErrInvalidArgument, "attachment file_id is required")
		}
		binding, err := loadSessionMediaBinding(ctx, sessionID, fileID)
		if err != nil {
			return nil, err
		}
		if binding.Owner != "" && ownerUID != "" && binding.Owner != ownerUID {
			return nil, types.Errorf(types.ErrForbidden, "media file does not belong to caller")
		}
		kind := binding.Kind
		mimeType := binding.MIMEType
		if ref.MIMEType != "" {
			mimeType = ref.MIMEType
		}
		if kind == "" {
			kind, err = ValidateMIMEAllowlist(mimeType)
			if err != nil {
				return nil, err
			}
		}
		parts = append(parts, msg.MediaPart{
			Kind:     kind,
			MIMEType: mimeType,
			FileID:   fileID,
		})
	}
	return parts, nil
}

// RejectUnsupportedModalities fails when the chat model cannot accept attachment kinds.
func RejectUnsupportedModalities(modelName string, parts []msg.ContentPart) error {
	for _, part := range parts {
		mp, ok := part.(msg.MediaPart)
		if !ok {
			continue
		}
		if model.SupportsModality(modelName, mp.Kind) {
			continue
		}
		return types.Errorf(types.ErrInvalidArgument,
			"model %q does not support %s input; choose a model with the required modality", modelName, mp.Kind)
	}
	return nil
}

// PrepareMediaForProvider fills MediaPart URL or Data for ConvertToLLM based on provider capabilities.
func PrepareMediaForProvider(ctx context.Context, provider string, messages []msg.AgentMessage) ([]msg.AgentMessage, error) {
	accessor, ok := media.AsAccessor(store.FileSystem)
	if !ok || accessor == nil {
		return messages, nil
	}
	hydrate := providerNeedsBinary(provider)
	ttl := config.App.ChatAgent.Media.SignedURLTTL
	out := make([]msg.AgentMessage, len(messages))
	for i, message := range messages {
		user, ok := message.(msg.UserMessage)
		if !ok {
			out[i] = message
			continue
		}
		parts := make([]msg.ContentPart, len(user.Parts))
		copy(parts, user.Parts)
		for j, part := range parts {
			mp, ok := part.(msg.MediaPart)
			if !ok || mp.FileID == "" {
				continue
			}
			if hydrate || mp.Kind != msg.MediaKindImage {
				fd, data, err := media.ReadAll(ctx, accessor, mp.FileID)
				if err != nil {
					return nil, fmt.Errorf("hydrate media %s: %w", mp.FileID, err)
				}
				if mp.MIMEType == "" && fd != nil {
					mp.MIMEType = fd.MimeType
				}
				mp.Data = data
				mp.URL = ""
			} else {
				signed, err := accessor.SignGetURL(ctx, mp.FileID, ttl)
				if err != nil {
					return nil, fmt.Errorf("sign media %s: %w", mp.FileID, err)
				}
				mp.URL = signed
				mp.Data = nil
			}
			parts[j] = mp
		}
		user.Parts = parts
		out[i] = user
	}
	return out, nil
}

func providerNeedsBinary(provider string) bool {
	switch provider {
	case llm.ProviderAnthropic:
		return true
	default:
		return false
	}
}

func sessionMediaCacheKey(sessionID, fileID string) cache.Key {
	return cache.NewKey("chatagent", "session_media", sessionID+":"+fileID)
}

func saveSessionMediaBinding(ctx context.Context, binding sessionMediaBinding) error {
	payload, err := sonic.MarshalString(binding)
	if err != nil {
		return fmt.Errorf("marshal media binding: %w", err)
	}
	key := sessionMediaCacheKey(binding.Session, binding.FileID)
	if rs := cache.DefaultRedisStore(); rs != nil {
		return rs.Set(ctx, key, payload, cache.TTLNone)
	}
	memoryMediaBindings.Store(key.String(), payload)
	return nil
}

func loadSessionMediaBinding(ctx context.Context, sessionID, fileID string) (sessionMediaBinding, error) {
	key := sessionMediaCacheKey(sessionID, fileID)
	var raw string
	if rs := cache.DefaultRedisStore(); rs != nil {
		val, ok, err := rs.Get(ctx, key)
		if err != nil {
			return sessionMediaBinding{}, err
		}
		if !ok || strings.TrimSpace(val) == "" {
			return sessionMediaBinding{}, types.Errorf(types.ErrNotFound, "media file %q is not bound to this session", fileID)
		}
		raw = val
	} else {
		val, ok := memoryMediaBindings.Load(key.String())
		if !ok {
			return sessionMediaBinding{}, types.Errorf(types.ErrNotFound, "media file %q is not bound to this session", fileID)
		}
		rawStr, ok := val.(string)
		if !ok || strings.TrimSpace(rawStr) == "" {
			return sessionMediaBinding{}, types.Errorf(types.ErrNotFound, "media file %q is not bound to this session", fileID)
		}
		raw = rawStr
	}
	var binding sessionMediaBinding
	if err := sonic.UnmarshalString(raw, &binding); err != nil {
		return sessionMediaBinding{}, fmt.Errorf("unmarshal media binding: %w", err)
	}
	return binding, nil
}

var memoryMediaBindings sync.Map

// MediaPlaceholderText builds a title/preview stub from attachments.
func MediaPlaceholderText(parts []msg.ContentPart) string {
	var stubs []string
	for _, part := range parts {
		if mp, ok := part.(msg.MediaPart); ok {
			stubs = append(stubs, msg.MediaPlaceholder(mp.Kind))
		}
	}
	return strings.Join(stubs, " ")
}

// BuildUserMessageParts combines text and media parts for a user turn.
func BuildUserMessageParts(text string, mediaParts []msg.ContentPart) []msg.ContentPart {
	parts := make([]msg.ContentPart, 0, 1+len(mediaParts))
	if strings.TrimSpace(text) != "" {
		parts = append(parts, msg.TextPart{Text: text})
	}
	parts = append(parts, mediaParts...)
	return parts
}

// EnsureMediaPublicConfig validates FS signing prerequisites when using the fs handler.
func EnsureMediaPublicConfig() error {
	if config.App.Media == nil {
		return nil
	}
	handler := strings.TrimSpace(config.App.Media.UseHandler)
	if handler != "" && handler != "fs" {
		return nil
	}
	cfg := config.App.ChatAgent.Media
	if strings.TrimSpace(cfg.PublicBaseURL) == "" {
		return types.Errorf(types.ErrInvalidArgument, "chat_agent.media.public_base_url is required for fs media signing")
	}
	secret := cfg.SignSecret
	if secret == "" {
		secret = config.App.Media.SignSecret
	}
	if strings.TrimSpace(secret) == "" {
		return types.Errorf(types.ErrInvalidArgument, "chat_agent.media.sign_secret (or media.sign_secret) is required for fs media signing")
	}
	return nil
}

// DefaultSignedURLTTL returns the configured TTL or one hour.
func DefaultSignedURLTTL() time.Duration {
	ttl := config.App.ChatAgent.Media.SignedURLTTL
	if ttl <= 0 {
		return time.Hour
	}
	return ttl
}
