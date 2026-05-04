package github

import (
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

	// send message
	err = notify.GatewaySend(ctx.Context(), ctx.AsUser, "github.deployment", []string{"slack", "ntfy"}, map[string]any{
		"user":    user.UserName,
		"repo":    drone.DefaultDeployRepoName,
		"build":   build.Number,
		"drone_url": config.App.Search.UrlBaseMap[drone.ID],
	})
	if err != nil {
		return err
	}

	return nil
}
