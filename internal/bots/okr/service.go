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

// objective list
//
//	@Summary		objective list
//	@Description	objective list
//	@Tags			okr
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	protocol.Response{data=[]model.Objective}
//	@Router			/okr/v1/objectives [get]
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

// objective detail
//
//	@Summary		objective detail
//	@Description	objective detail
//	@Tags			okr
//	@Accept			json
//	@Produce		json
//	@Param			sequence	path		int	true	"Sequence"
//	@Success		200			{object}	protocol.Response{data=model.Objective}
//	@Router			/okr/v1/objective/{sequence} [get]
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

// objective create
//
//	@Summary		objective create
//	@Description	objective create
//	@Tags			okr
//	@Accept			json
//	@Produce		json
//	@Param			objective	body		model.Objective	true	"objective data"
//	@Success		200			{object}	protocol.Response
//	@Router			/okr/v1/objective [post]
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

// objective update
//
//	@Summary		objective update
//	@Description	objective update
//	@Tags			okr
//	@Accept			json
//	@Produce		json
//	@Param			sequence	path		int				true	"Sequence"
//	@Param			objective	body		model.Objective	true	"objective data"
//	@Success		200			{object}	protocol.Response
//	@Router			/okr/v1/objective/{sequence} [put]
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

// objective delete
//
//	@Summary		objective delete
//	@Description	objective delete
//	@Tags			okr
//	@Accept			json
//	@Produce		json
//	@Param			sequence	path		int	true	"Sequence"
//	@Success		200			{object}	protocol.Response
//	@Router			/okr/v1/objective/{sequence} [delete]
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
