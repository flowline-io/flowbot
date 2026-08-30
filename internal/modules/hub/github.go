package hub

import (
	"fmt"

	"github.com/flowline-io/flowbot/pkg/config"
	"github.com/flowline-io/flowbot/pkg/notify"
	"github.com/flowline-io/flowbot/pkg/providers/drone"
	"github.com/flowline-io/flowbot/pkg/providers/gitea"
	"github.com/flowline-io/flowbot/pkg/types"
)

func deploy(ctx types.Context) error {
	client, err := gitea.GetClient()
	if err != nil {
		return err
	}

	// get namespace
	user, err := client.GetMyUserInfo()
	if err != nil {
		return err
	}

	// create build
	dClient := drone.GetClient()
	build, err := dClient.CreateBuild(user.UserName, drone.DefaultDeployRepoName)
	if err != nil {
		return err
	}

	err = notify.GatewaySendDefaults(ctx.Context(), ctx.AsUser, deployNotifyPayload(
		user.UserName,
		drone.DefaultDeployRepoName,
		build.Number,
		config.App.Search.UrlBaseMap[drone.ID],
	))
	if notify.WarnSkipNoDefault(err, "deploy") {
		return nil
	}
	if err != nil {
		return err
	}

	return nil
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
