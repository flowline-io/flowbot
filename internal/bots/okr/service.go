package okr

import (
	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/internal/store/model"
	"github.com/flowline-io/flowbot/internal/types/protocol"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/route"
	"github.com/gofiber/fiber/v2"
	"strconv"
)

// objective list
//
//	@Summary  objective list
//	@Tags     okr
//	@Accept   json
//	@Produce  json
//	@Success  200  {object}  protocol.Response{data=[]model.Objective}
//	@Router   /okr/objectives [get]
func objectiveList(ctx *fiber.Ctx) error {
	uid := route.GetUid(ctx)
	topic := route.GetTopic(ctx)
	list, err := store.Chatbot.ListObjectives(uid, topic)
	if err != nil {
		flog.Error(err)
		return ctx.JSON(protocol.NewFailedResponse(protocol.ErrDatabaseReadError))
	}
	return ctx.JSON(protocol.NewSuccessResponse(list))
}

// objective detail
//
//	@Summary  objective detail
//	@Tags     okr
//	@Accept   json
//	@Produce  json
//	@Param    sequence  path      int  true  "Sequence"
//	@Success  200       {object}  protocol.Response{data=model.Objective}
//	@Router   /okr/objective/{sequence} [get]
func objectiveDetail(ctx *fiber.Ctx) error {
	uid := route.GetUid(ctx)
	topic := route.GetTopic(ctx)
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
//	@Summary  objective create
//	@Tags     okr
//	@Accept   json
//	@Produce  json
//	@Param    objective  body      model.Objective  true  "objective data"
//	@Success  200        {object}  protocol.Response
//	@Router   /okr/objective [post]
func objectiveCreate(ctx *fiber.Ctx) error {
	uid := route.GetUid(ctx)
	topic := route.GetTopic(ctx)
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
//	@Summary  objective update
//	@Tags     okr
//	@Accept   json
//	@Produce  json
//	@Param    sequence   path      int              true  "Sequence"
//	@Param    objective  body      model.Objective  true  "objective data"
//	@Success  200        {object}  protocol.Response
//	@Router   /okr/objective/{sequence} [put]
func objectiveUpdate(ctx *fiber.Ctx) error {
	uid := route.GetUid(ctx)
	topic := route.GetTopic(ctx)
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
//	@Summary  objective delete
//	@Tags     okr
//	@Accept   json
//	@Produce  json
//	@Param    sequence  path      int  true  "Sequence"
//	@Success  200       {object}  protocol.Response
//	@Router   /okr/objective/{sequence} [delete]
func objectiveDelete(ctx *fiber.Ctx) error {
	uid := route.GetUid(ctx)
	topic := route.GetTopic(ctx)
	s := ctx.Params("sequence")
	sequence, _ := strconv.ParseInt(s, 10, 64)

	err := store.Chatbot.DeleteObjectiveBySequence(uid, topic, sequence)
	if err != nil {
		flog.Error(err)
		return ctx.JSON(protocol.NewFailedResponse(protocol.ErrDatabaseWriteError))
	}
	return ctx.JSON(protocol.NewSuccessResponse(nil))
}
