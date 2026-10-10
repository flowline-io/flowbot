package homelab

import (
	"context"

	"github.com/flowline-io/flowbot/pkg/types"
)

type composeRunner func(ctx context.Context, app App, args ...string) (string, error)

type dockerRunner func(ctx context.Context, args ...string) (string, error)

func fetchImageRepoDigest(ctx context.Context, app App, svc ComposeService, compose composeRunner, docker dockerRunner, inspectOp, imageInspectOp string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", types.WrapError(types.ErrTimeout, "homelab image digest canceled", err)
	}
	ref, _, ok := WatchImageRef(svc.Image)
	if !ok {
		return "", nil
	}
	idOut, psErr := compose(ctx, app, "ps", "-q", svc.Name)
	id := firstNonEmptyLine(idOut)
	if psErr == nil && id != "" {
		out, err := docker(ctx, "inspect", "--format", "{{json .RepoDigests}}", id)
		if err != nil {
			return "", types.WrapError(types.ErrProvider, inspectOp, err)
		}
		digest, err := repoDigestFromInspect(out, ref)
		if err != nil {
			return "", types.WrapError(types.ErrProvider, inspectOp+" RepoDigests", err)
		}
		return digest, nil
	}
	out, err := docker(ctx, "image", "inspect", "--format", "{{json .RepoDigests}}", ref)
	if err != nil {
		return "", types.WrapError(types.ErrProvider, imageInspectOp, err)
	}
	digest, err := repoDigestFromInspect(out, ref)
	if err != nil {
		return "", types.WrapError(types.ErrProvider, imageInspectOp+" RepoDigests", err)
	}
	return digest, nil
}

func fetchRemoteManifestDigest(ctx context.Context, imageRef string, docker dockerRunner, inspectOp string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", types.WrapError(types.ErrTimeout, "homelab remote digest canceled", err)
	}
	out, err := docker(ctx, "buildx", "imagetools", "inspect", "--format", "{{.Manifest.Digest}}", imageRef)
	if err != nil {
		return "", types.WrapError(types.ErrProvider, inspectOp, err)
	}
	return parseManifestDigest(out), nil
}
