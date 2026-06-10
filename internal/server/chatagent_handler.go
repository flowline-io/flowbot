package server

import (
	"context"
	"time"

	"github.com/flowline-io/flowbot/internal/platforms"
	"github.com/flowline-io/flowbot/internal/server/chatagent"
	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/internal/store/ent/gen"
	"github.com/flowline-io/flowbot/internal/store/ent/schema"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/protocol"
)

var chatAgentService = chatagent.NewService()

func runChatAgent(
	eventCtx context.Context,
	caller *platforms.Caller,
	msg protocol.MessageEventData,
	_ types.Uid,
	sessionID string,
	platformID int64,
	topic string,
) {
	ctx, cancel := context.WithTimeout(eventCtx, chatagent.DefaultRunTimeout)
	defer cancel()

	reply, err := chatAgentService.Run(ctx, chatagent.RunRequest{
		SessionID: sessionID,
		Text:      msg.AltMessage,
	})
	if err != nil {
		flog.Error(err)
		sendChatReply(caller, msg, types.TextMsg{Text: "Chat agent error: " + err.Error()})
		return
	}

	now := time.Now().UTC()
	err = store.Database.CreateMessage(ctx, gen.Message{
		Flag:          types.Id(),
		PlatformID:    platformID,
		PlatformMsgID: "",
		Topic:         topic,
		Role:          types.Assistant,
		Session:       sessionID,
		Content:       schema.JSON{"text": reply},
		State:         int(schema.MessageCreated),
		CreatedAt:     now,
		UpdatedAt:     now,
	})
	if err != nil {
		flog.Error(err)
	}

	sendChatReply(caller, msg, types.TextMsg{Text: reply})
}

func sendChatReply(caller *platforms.Caller, msg protocol.MessageEventData, payload types.MsgPayload) {
	resp := caller.Do(protocol.Request{
		Action: protocol.SendMessageAction,
		Params: types.KV{
			"topic":   msg.TopicId,
			"message": caller.Adapter.MessageConvert(payload),
		},
	})
	flog.Info("[chat-agent] response: %+v", resp)
}
