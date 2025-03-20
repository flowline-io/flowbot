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
	build, err := dClient.CreateBuild(user.LoginName, drone.DefaultDeployRepoName)
	if err != nil {
		return err
	}

	// send message
	err = event.SendMessage(ctx, types.TextMsg{Text: fmt.Sprintf("%s/%d", config.App.Search.UrlBaseMap[drone.ID], build.ID)})
	if err != nil {
		return err
	}

	return nil
}
