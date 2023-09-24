package okr

import (
	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/internal/store/model"
	"github.com/flowline-io/flowbot/internal/types"
	"github.com/flowline-io/flowbot/internal/types/protocol"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/gofiber/fiber/v2"
	"strconv"
)

const serviceVersion = "v1"

func objectiveList(ctx *fiber.Ctx) error {
	uid := types.Uid(0) // fixme
	topic := ""         // fixme
	list, err := store.Chatbot.ListObjectives(uid, topic)
	if err != nil {
		flog.Error(err)
		return ctx.JSON(protocol.NewFailedResponse(protocol.ErrDatabaseReadError))
	}
	return ctx.JSON(protocol.NewSuccessResponse(list))
}

func objectiveDetail(ctx *fiber.Ctx) error {
	uid := types.Uid(0) // fixme
	topic := ""         // fixme
	s := ctx.Params("sequence")
	sequence, _ := strconv.ParseInt(s, 10, 64)

	obj, err := store.Chatbot.GetObjectiveBySequence(uid, topic, sequence)
	if err != nil {
		flog.Error(err)
		return ctx.JSON(protocol.NewFailedResponse(protocol.ErrDatabaseReadError))
	}
	return ctx.JSON(protocol.NewSuccessResponse(obj))
}

func objectiveCreate(ctx *fiber.Ctx) error {
	uid := types.Uid(0) // fixme
	topic := ""         // fixme
	obj := new(model.Objective)
	err := ctx.BodyParser(&obj)
	if err != nil {
		return ctx.JSON(protocol.NewFailedResponse(protocol.ErrBadParam))
	}
	obj.UID = uid.String()
	obj.Topic = topic
	_, err = store.Chatbot.CreateObjective(obj)
	if err != nil {
		flog.Error(err)
		return ctx.JSON(protocol.NewFailedResponse(protocol.ErrDatabaseWriteError))
	}
	return ctx.JSON(protocol.NewSuccessResponse(nil))
}

func objectiveUpdate(ctx *fiber.Ctx) error {
	uid := types.Uid(0) // fixme
	topic := ""         // fixme
	obj := new(model.Objective)
	err := ctx.BodyParser(&obj)
	if err != nil {
		return ctx.JSON(protocol.NewFailedResponse(protocol.ErrBadParam))
	}
	obj.UID = uid.String()
	obj.Topic = topic
	err = store.Chatbot.UpdateObjective(obj)
	if err != nil {
		flog.Error(err)
		return ctx.JSON(protocol.NewFailedResponse(protocol.ErrDatabaseWriteError))
	}
	return ctx.JSON(protocol.NewSuccessResponse(nil))
}

func objectiveDelete(ctx *fiber.Ctx) error {
	uid := types.Uid(0) // fixme
	topic := ""         // fixme
	s := ctx.Params("sequence")
	sequence, _ := strconv.ParseInt(s, 10, 64)

	err := store.Chatbot.DeleteObjectiveBySequence(uid, topic, sequence)
	if err != nil {
		flog.Error(err)
		return ctx.JSON(protocol.NewFailedResponse(protocol.ErrDatabaseWriteError))
	}
	return ctx.JSON(protocol.NewSuccessResponse(nil))
}
