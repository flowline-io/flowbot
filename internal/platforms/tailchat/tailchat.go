package tailchat

import (
	"fmt"
	"github.com/gofiber/fiber/v2"
)

const ID = "tailchat"

func HandleWebsocket(stop <-chan bool) {
}

func HandleHttp(c *fiber.Ctx) error {
	var payload Payload
	err := c.BodyParser(&payload)
	if err != nil {
		return err
	}

	// Ignore all messages created by the bot itself
	if payload.UserID == payload.Payload.MessageAuthor {
		return nil
	}

	fmt.Println(payload.Payload.MessagePlainContent)

	cli := newClient()
	err = cli.auth()
	if err != nil {
		return err
	}

	err = cli.sendMessage(SendMessageData{
		ConverseId: payload.Payload.ConverseID,
		GroupId:    payload.Payload.GroupID,
		Content:    "hi from tailchat",
		Meta: SendMessageMeta{
			Mentions: []string{
				payload.UserID,
			},
			Reply: SendMessageReply{
				Id:      payload.ID,
				Author:  payload.UserID,
				Content: payload.Payload.MessagePlainContent,
			},
		},
	})
	if err != nil {
		return err
	}

	return nil
}
