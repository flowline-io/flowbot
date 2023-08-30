package server

import (
	"github.com/flowline-io/flowbot/internal/platforms/discord"
	"github.com/flowline-io/flowbot/internal/platforms/slack"
)

func hookPlatform(stop <-chan bool) {
	// slack
	go slack.HandleSlack(stop)
	// discord
	go discord.HandleDiscord(stop)
}
