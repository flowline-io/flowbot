// Package drone implements the Drone CI capability.
package drone

import (
	"context"
	"fmt"

	"github.com/flowline-io/flowbot/pkg/config"
	"github.com/flowline-io/flowbot/pkg/notify"
	"github.com/flowline-io/flowbot/pkg/providers"
	droneprovider "github.com/flowline-io/flowbot/pkg/providers/drone"
	giteaprovider "github.com/flowline-io/flowbot/pkg/providers/gitea"
	"github.com/flowline-io/flowbot/pkg/types"
)

type buildClient interface {
	CreateBuild(namespace, name string) (*droneprovider.Build, error)
}

type namespaceFunc func(ctx context.Context) (string, error)

type notifyFunc func(ctx context.Context, uid types.Uid, payload map[string]any) error

// Adapter implements Service using the Drone provider and the Gitea user as the namespace.
type Adapter struct {
	builds    buildClient
	namespace namespaceFunc
	notify    notifyFunc
	droneURL  func() string
}

// New creates an Adapter from provider config.
// Returns nil when the Drone endpoint is not configured.
func New() Service {
	endpoint, err := providers.GetConfig(droneprovider.ID, droneprovider.EndpointKey)
	if err != nil || endpoint.String() == "" {
		return nil
	}
	return NewWithDeps(droneprovider.GetClient(), currentGiteaNamespace, sendDeployNotify, func() string {
		return config.App.Search.UrlBaseMap[droneprovider.ID]
	})
}

// NewWithDeps creates an Adapter from explicit dependencies.
// Returns nil when builds or namespace is nil.
func NewWithDeps(builds buildClient, namespace namespaceFunc, send notifyFunc, droneURL func() string) Service {
	if builds == nil || namespace == nil {
		return nil
	}
	if droneURL == nil {
		droneURL = func() string { return "" }
	}
	return &Adapter{
		builds:    builds,
		namespace: namespace,
		notify:    send,
		droneURL:  droneURL,
	}
}

// Deploy resolves the Gitea namespace, creates a build of the deploy repository, and notifies.
func (a *Adapter) Deploy(ctx context.Context, in DeployInput) (*Build, error) {
	if err := ctx.Err(); err != nil {
		return nil, types.WrapError(types.ErrTimeout, "context canceled", err)
	}
	namespace, err := a.namespace(ctx)
	if err != nil {
		return nil, err
	}
	if namespace == "" {
		return nil, types.Errorf(types.ErrProvider, "gitea user has no username")
	}
	repo := droneprovider.DefaultDeployRepoName
	raw, err := a.builds.CreateBuild(namespace, repo)
	if err != nil {
		return nil, types.WrapError(types.ErrProvider, "drone create build", err)
	}
	if raw == nil {
		return nil, types.Errorf(types.ErrProvider, "drone create build returned no build")
	}
	result := &Build{
		Number:    raw.Number,
		Namespace: namespace,
		Repo:      repo,
		URL:       a.droneURL(),
	}
	if a.notify == nil {
		return result, nil
	}
	err = a.notify(ctx, in.UID, deployNotifyPayload(namespace, repo, raw.Number, result.URL))
	if notify.WarnSkipNoDefault(err, "deploy") {
		return result, nil
	}
	if err != nil {
		return nil, err
	}
	return result, nil
}

// HealthCheck reports that the Drone client is configured.
func (*Adapter) HealthCheck(ctx context.Context) (bool, error) {
	if err := ctx.Err(); err != nil {
		return false, types.WrapError(types.ErrTimeout, "context canceled", err)
	}
	return true, nil
}

func currentGiteaNamespace(ctx context.Context) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", types.WrapError(types.ErrTimeout, "context canceled", err)
	}
	client, err := giteaprovider.GetClient()
	if err != nil {
		return "", types.WrapError(types.ErrProvider, "gitea client", err)
	}
	if client == nil {
		return "", types.Errorf(types.ErrUnavailable, "gitea is not configured")
	}
	user, err := client.GetMyUserInfo()
	if err != nil {
		return "", types.WrapError(types.ErrProvider, "gitea get user", err)
	}
	if user == nil || user.UserName == "" {
		return "", types.Errorf(types.ErrProvider, "gitea user has no username")
	}
	return user.UserName, nil
}

func sendDeployNotify(ctx context.Context, uid types.Uid, payload map[string]any) error {
	return notify.GatewaySendDefaults(ctx, uid, payload)
}

func deployNotifyPayload(user, repo string, build int, droneURL string) map[string]any {
	payload := map[string]any{
		notify.PayloadKeyTitle:   "Deployment triggered",
		notify.PayloadKeySummary: fmt.Sprintf("%s/%s #%d", user, repo, build),
	}
	if droneURL != "" {
		payload[notify.PayloadKeyURL] = droneURL
	}
	return payload
}
