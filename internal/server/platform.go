package server

import "github.com/flowline-io/flowbot/internal/platforms/slack"

func hookPlatform() {
	// slack
	go slack.HandleSlack()
}
