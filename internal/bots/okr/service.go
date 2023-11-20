package okr

import (
	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/internal/store/model"
	"github.com/flowline-io/flowbot/internal/types/protocol"
	"github.com/flowline-io/flowbot/pkg/route"
	"github.com/gofiber/fiber/v2"
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
		return ctx.JSON(protocol.NewFailedResponseWithError(protocol.ErrDatabaseReadError, err))
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
	sequence := route.GetIntParam(ctx, "sequence")

	obj, err := store.Chatbot.GetObjectiveBySequence(uid, topic, sequence)
	if err != nil {
		return ctx.JSON(protocol.NewFailedResponseWithError(protocol.ErrDatabaseReadError, err))
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
		return ctx.JSON(protocol.NewFailedResponseWithError(protocol.ErrBadParam, err))
	}

	// check
	if item.IsPlan > 0 {
		if item.PlanStart.IsZero() || item.PlanEnd.IsZero() {
			return ctx.JSON(protocol.NewFailedResponse(protocol.ErrParamVerificationFailed))
		}
	}

	item.UID = uid.String()
	item.Topic = topic
	_, err = store.Chatbot.CreateObjective(item)
	if err != nil {
		return ctx.JSON(protocol.NewFailedResponseWithError(protocol.ErrDatabaseWriteError, err))
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
	sequence := route.GetIntParam(ctx, "sequence")

	item := new(model.Objective)
	err := ctx.BodyParser(&item)
	if err != nil {
		return ctx.JSON(protocol.NewFailedResponseWithError(protocol.ErrBadParam, err))
	}
	item.UID = uid.String()
	item.Topic = topic
	item.Sequence = int32(sequence)
	err = store.Chatbot.UpdateObjective(item)
	if err != nil {
		return ctx.JSON(protocol.NewFailedResponseWithError(protocol.ErrDatabaseWriteError, err))
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
	sequence := route.GetIntParam(ctx, "sequence")

	err := store.Chatbot.DeleteObjectiveBySequence(uid, topic, sequence)
	if err != nil {
		return ctx.JSON(protocol.NewFailedResponseWithError(protocol.ErrDatabaseWriteError, err))
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
		return ctx.JSON(protocol.NewFailedResponseWithError(protocol.ErrBadParam, err))
	}
	item.UID = uid.String()
	item.Topic = topic
	_, err = store.Chatbot.CreateKeyResult(item)
	if err != nil {
		return ctx.JSON(protocol.NewFailedResponseWithError(protocol.ErrDatabaseWriteError, err))
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
	sequence := route.GetIntParam(ctx, "sequence")

	item := new(model.KeyResult)
	err := ctx.BodyParser(&item)
	if err != nil {
		return ctx.JSON(protocol.NewFailedResponseWithError(protocol.ErrBadParam, err))
	}
	item.UID = uid.String()
	item.Topic = topic
	item.Sequence = int32(sequence)
	err = store.Chatbot.UpdateKeyResult(item)
	if err != nil {
		return ctx.JSON(protocol.NewFailedResponseWithError(protocol.ErrDatabaseWriteError, err))
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
	sequence := route.GetIntParam(ctx, "sequence")

	err := store.Chatbot.DeleteKeyResultBySequence(uid, topic, sequence)
	if err != nil {
		return ctx.JSON(protocol.NewFailedResponseWithError(protocol.ErrDatabaseWriteError, err))
	}
	return ctx.JSON(protocol.NewSuccessResponse(nil))
}

// key result value list
//
//	@Summary  key result value list
//	@Tags     okr
//	@Accept   json
//	@Produce  json
//	@Param    id   path      int  true  "key result id"
//	@Success  200  {object}  protocol.Response{data=[]model.KeyResultValue}
//	@Router   /okr/key_result/{id}/values [get]
func keyResultValueList(ctx *fiber.Ctx) error {
	keyResultId := route.GetIntParam(ctx, "id")
	if keyResultId == 0 {
		return ctx.JSON(protocol.NewFailedResponse(protocol.ErrBadParam))
	}

	list, err := store.Chatbot.GetKeyResultValues(keyResultId)
	if err != nil {
		return ctx.JSON(protocol.NewFailedResponseWithError(protocol.ErrDatabaseReadError, err))
	}
	return ctx.JSON(protocol.NewSuccessResponse(list))
}

// KeyResult value create
//
//	@Summary  KeyResult value create
//	@Tags     okr
//	@Accept   json
//	@Produce  json
//	@Param    id              path      int                   true  "key result id"
//	@Param    KeyResultValue  body      model.KeyResultValue  true  "KeyResultValue data"
//	@Success  200             {object}  protocol.Response
//	@Router   /okr/key_result/{id}/value [post]
func keyResultValueCreate(ctx *fiber.Ctx) error {
	keyResultId := route.GetIntParam(ctx, "id")
	if keyResultId == 0 {
		return ctx.JSON(protocol.NewFailedResponse(protocol.ErrBadParam))
	}

	item := new(model.KeyResultValue)
	err := ctx.BodyParser(&item)
	if err != nil {
		return ctx.JSON(protocol.NewFailedResponseWithError(protocol.ErrBadParam, err))
	}
	item.KeyResultID = keyResultId
	_, err = store.Chatbot.CreateKeyResultValue(item)
	if err != nil {
		return ctx.JSON(protocol.NewFailedResponseWithError(protocol.ErrDatabaseWriteError, err))
	}
	return ctx.JSON(protocol.NewSuccessResponse(nil))
}

// KeyResult value delete
//
//	@Summary  KeyResult value delete
//	@Tags     okr
//	@Accept   json
//	@Produce  json
//	@Param    id              path      int                   true  "key result id"
//	@Success  200             {object}  protocol.Response
//	@Router   /okr/key_result_value/{id} [delete]
func keyResultValueDelete(ctx *fiber.Ctx) error {
	keyResultValueId := route.GetIntParam(ctx, "id")
	if keyResultValueId == 0 {
		return ctx.JSON(protocol.NewFailedResponse(protocol.ErrBadParam))
	}

	err := store.Chatbot.DeleteKeyResultValue(keyResultValueId)
	if err != nil {
		return ctx.JSON(protocol.NewFailedResponseWithError(protocol.ErrDatabaseWriteError, err))
	}
	return ctx.JSON(protocol.NewSuccessResponse(nil))
}

// KeyResult value detail
//
//	@Summary  KeyResult value detail
//	@Tags     okr
//	@Accept   json
//	@Produce  json
//	@Param    id              path      int                   true  "key result id"
//	@Success  200             {object}  protocol.Response{data=model.KeyResultValue}
//	@Router   /okr/key_result_value/{id} [delete]
func keyResultValue(ctx *fiber.Ctx) error {
	keyResultValueId := route.GetIntParam(ctx, "id")
	if keyResultValueId == 0 {
		return ctx.JSON(protocol.NewFailedResponse(protocol.ErrBadParam))
	}

	item, err := store.Chatbot.GetKeyResultValue(keyResultValueId)
	if err != nil {
		return ctx.JSON(protocol.NewFailedResponseWithError(protocol.ErrDatabaseReadError, err))
	}
	return ctx.JSON(protocol.NewSuccessResponse(item))
}
