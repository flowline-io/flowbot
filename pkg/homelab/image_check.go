package homelab

import (
	"context"
	"strings"
	"time"

	"github.com/flowline-io/flowbot/pkg/flog"
)

// DefaultImageCheckInterval is used when runtime can inspect images and
// homelab.image_check.interval is empty.
const DefaultImageCheckInterval = 6 * time.Hour

// ImageCheckSource is DataEvent.Source for image update detections.
const ImageCheckSource = "homelab_image_check"

// ImageInspectTimeout bounds one local or remote digest query.
const ImageInspectTimeout = 30 * time.Second

// ImageUpdate is a confirmed digest mismatch for one compose service.
type ImageUpdate struct {
	AppName       string
	Service       string
	Image         string
	Tag           string
	CurrentDigest string
	RemoteDigest  string
	Capability    string
}

// ImageCheckInterval returns the inspect period when image checks should run.
// Empty interval defaults to DefaultImageCheckInterval. Zero or negative
// durations, unusable runtime modes, and invalid duration strings disable checks.
func ImageCheckInterval(mode RuntimeMode, interval string) (time.Duration, bool) {
	switch mode {
	case RuntimeModeDockerSocket, RuntimeModeSSH:
	default:
		return 0, false
	}
	s := strings.TrimSpace(interval)
	if s == "" {
		return DefaultImageCheckInterval, true
	}
	if s == "0" {
		return 0, false
	}
	d, err := time.ParseDuration(s)
	if err != nil || d <= 0 {
		return 0, false
	}
	return d, true
}

// CheckImageUpdates inspects each watchable service serially and returns
// confirmed remote-vs-local digest mismatches. Per-service failures are logged
// and skipped.
func CheckImageUpdates(ctx context.Context, rt Runtime, apps []App) []ImageUpdate {
	if rt == nil {
		return nil
	}
	var out []ImageUpdate
	for _, app := range apps {
		capID := appCapabilityID(app)
		for _, svc := range app.Services {
			if err := ctx.Err(); err != nil {
				return out
			}
			if !ServiceWatchesImage(svc) {
				continue
			}
			ref, tag, ok := WatchImageRef(svc.Image)
			if !ok {
				continue
			}
			local, err := inspectLocalDigest(ctx, rt, app, svc)
			if err != nil {
				flog.Warn("homelab image check: local digest %s/%s: %v", app.Name, svc.Name, err)
				continue
			}
			if local == "" {
				flog.Info("homelab image check: skip %s/%s: no RepoDigest", app.Name, svc.Name)
				continue
			}
			remote, err := inspectRemoteDigest(ctx, rt, app, ref)
			if err != nil {
				flog.Warn("homelab image check: remote digest %s/%s: %v", app.Name, svc.Name, err)
				continue
			}
			if remote == "" {
				flog.Info("homelab image check: skip %s/%s: empty remote digest", app.Name, svc.Name)
				continue
			}
			if local == remote {
				continue
			}
			out = append(out, ImageUpdate{
				AppName:       app.Name,
				Service:       svc.Name,
				Image:         ref,
				Tag:           tag,
				CurrentDigest: local,
				RemoteDigest:  remote,
				Capability:    capID,
			})
		}
	}
	return out
}

func appCapabilityID(app App) string {
	caps := ParseLabels(app.Labels)
	if len(caps) == 0 {
		return ""
	}
	return strings.TrimSpace(caps[0].Capability)
}

func inspectLocalDigest(ctx context.Context, rt Runtime, app App, svc ComposeService) (string, error) {
	cctx, cancel := context.WithTimeout(ctx, ImageInspectTimeout)
	defer cancel()
	return rt.ImageRepoDigest(cctx, app, svc)
}

func inspectRemoteDigest(ctx context.Context, rt Runtime, app App, ref string) (string, error) {
	cctx, cancel := context.WithTimeout(ctx, ImageInspectTimeout)
	defer cancel()
	return rt.RemoteManifestDigest(cctx, app, ref)
}
