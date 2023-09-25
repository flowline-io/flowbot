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
	item := new(model.Objective)
	err := ctx.BodyParser(&item)
	if err != nil {
		return ctx.JSON(protocol.NewFailedResponse(protocol.ErrBadParam))
	}
	item.UID = uid.String()
	item.Topic = topic
	_, err = store.Chatbot.CreateObjective(item)
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
	item := new(model.Objective)
	err := ctx.BodyParser(&item)
	if err != nil {
		return ctx.JSON(protocol.NewFailedResponse(protocol.ErrBadParam))
	}
	item.UID = uid.String()
	item.Topic = topic
	err = store.Chatbot.UpdateObjective(item)
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

// KeyResult create
//
//	@Summary  KeyResult create
//	@Tags     okr
//	@Accept   json
//	@Produce  json
//	@Param    keyResult  body      model.KeyResult  true  "KeyResult data"
//	@Success  200        {object}  protocol.Response
//	@Router   /okr/key_result [post]
func keyResultCreate(ctx *fiber.Ctx) error {
	uid := route.GetUid(ctx)
	topic := route.GetTopic(ctx)
	item := new(model.KeyResult)
	err := ctx.BodyParser(&item)
	if err != nil {
		return ctx.JSON(protocol.NewFailedResponse(protocol.ErrBadParam))
	}
	item.UID = uid.String()
	item.Topic = topic
	_, err = store.Chatbot.CreateKeyResult(item)
	if err != nil {
		flog.Error(err)
		return ctx.JSON(protocol.NewFailedResponse(protocol.ErrDatabaseWriteError))
	}
	return ctx.JSON(protocol.NewSuccessResponse(nil))
}

// KeyResult update
//
//	@Summary  KeyResult update
//	@Tags     okr
//	@Accept   json
//	@Produce  json
//	@Param    sequence   path      int              true  "Sequence"
//	@Param    objective  body      model.KeyResult  true  "KeyResult data"
//	@Success  200        {object}  protocol.Response
//	@Router   /okr/key_result/{sequence} [put]
func keyResultUpdate(ctx *fiber.Ctx) error {
	uid := route.GetUid(ctx)
	topic := route.GetTopic(ctx)
	item := new(model.KeyResult)
	err := ctx.BodyParser(&item)
	if err != nil {
		return ctx.JSON(protocol.NewFailedResponse(protocol.ErrBadParam))
	}
	item.UID = uid.String()
	item.Topic = topic
	err = store.Chatbot.UpdateKeyResult(item)
	if err != nil {
		flog.Error(err)
		return ctx.JSON(protocol.NewFailedResponse(protocol.ErrDatabaseWriteError))
	}
	return ctx.JSON(protocol.NewSuccessResponse(nil))
}

// KeyResult delete
//
//	@Summary  KeyResult delete
//	@Tags     okr
//	@Accept   json
//	@Produce  json
//	@Param    sequence  path      int  true  "Sequence"
//	@Success  200       {object}  protocol.Response
//	@Router   /okr/key_result/{sequence} [delete]
func keyResultDelete(ctx *fiber.Ctx) error {
	uid := route.GetUid(ctx)
	topic := route.GetTopic(ctx)
	s := ctx.Params("sequence")
	sequence, _ := strconv.ParseInt(s, 10, 64)

	err := store.Chatbot.DeleteKeyResultBySequence(uid, topic, sequence)
	if err != nil {
		flog.Error(err)
		return ctx.JSON(protocol.NewFailedResponse(protocol.ErrDatabaseWriteError))
	}
	return ctx.JSON(protocol.NewSuccessResponse(nil))
}
