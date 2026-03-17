package github

import (
	"fmt"

	"github.com/flowline-io/flowbot/pkg/config"
	"github.com/flowline-io/flowbot/pkg/event"
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
	err = event.SendMessage(ctx, types.TextMsg{Text: fmt.Sprintf("Deployment triggered: [%s/%s/deploy/%d](%s/%s/deploy/%d)\n\n*Repository:* %s\n*Build #:* %d",
		user.UserName, drone.DefaultDeployRepoName, build.Number,
		config.App.Search.UrlBaseMap[drone.ID], user.UserName, build.Number,
		drone.DefaultDeployRepoName,
		build.Number,
	)})
	if err != nil {
		return err
	}

	return nil
}
