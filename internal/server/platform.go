package server

import (
	"fmt"
	"github.com/flowline-io/flowbot/pkg/config"
	"github.com/flowline-io/flowbot/pkg/logs"
	"github.com/slack-go/slack"
	"github.com/slack-go/slack/slackevents"
	"github.com/slack-go/slack/socketmode"
	"strconv"
	"time"
)

var channelid = ""

func hookPlatform() {
	api := slack.New(config.App.Platform.Slack.BotToken, slack.OptionDebug(true), slack.OptionAppLevelToken(config.App.Platform.Slack.AppToken))

	client := socketmode.New(api, socketmode.OptionDebug(true))

	go func() {
		for {
			select {
			case event := <-client.Events:
				switch event.Type {
				case socketmode.EventTypeEventsAPI:
					logs.Info.Println(event.Data)
					//err := makeRequest(&SlackRequest{
					//	StatusCode: 200,
					//	Content:    "flowbot is up and running!",
					//}, api)
					//if err != nil {
					//	logs.Err.Println(err)
					//}
					apiEvent := event.Data.(slackevents.EventsAPIEvent)
					messageEvent := apiEvent.InnerEvent.Data.(*slackevents.MessageEvent)
					fmt.Println(messageEvent.Text)
				}
			}
		}
	}()

	go func() {
		err := client.Run()
		if err != nil {
			logs.Err.Println(err)
		}
	}()
}

// SlackRequest takes in the StatusCode and Content from other functions to display to the user's slack.
type SlackRequest struct {
	// StatusCode is the http code that will be returned back to the user.
	StatusCode int `json:"statusCode"`
	// Content will contain the presigned url, error messages, or success messages.
	Content string `json:"body"`
}

func makeRequest(in *SlackRequest, api *slack.Client) error {
	code := strconv.Itoa(in.StatusCode)
	attachment := slack.Attachment{
		Color: "#0069ff",
		Fields: []slack.AttachmentField{
			{
				Title: in.Content,
				Value: fmt.Sprintf("Response: %s", code),
			},
		},
		Footer: "FlowBot " + " | " + time.Now().Format("01-02-2006 3:4:5 MST"),
	}
	_, _, err := api.PostMessage(
		channelid,
		slack.MsgOptionAttachments(attachment),
		slack.MsgOptionAsUser(true),
	)
	if err != nil {
		return err
	}
	return nil
}
