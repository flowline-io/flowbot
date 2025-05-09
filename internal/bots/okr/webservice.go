package okr

import (
	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/internal/store/model"
	"github.com/flowline-io/flowbot/pkg/route"
	"github.com/flowline-io/flowbot/pkg/types/protocol"
	"github.com/flowline-io/flowbot/pkg/types/ruleset/webservice"
	"github.com/gofiber/fiber/v2"
)

var webserviceRules = []webservice.Rule{
	webservice.Get("/objectives", objectiveList),
	webservice.Get("/objective/:sequence", objectiveDetail),
	webservice.Post("/objective", objectiveCreate),
	webservice.Put("/objective/:sequence", objectiveUpdate),
	webservice.Delete("/objective/:sequence", objectiveDelete),
	webservice.Post("/key_result", keyResultCreate),
	webservice.Put("/key_result/:sequence", keyResultUpdate),
	webservice.Delete("/key_result/:sequence", keyResultDelete),
	webservice.Get("/key_result/:id/values", keyResultValueList),
	webservice.Post("/key_result/:id/value", keyResultValueCreate),
	webservice.Delete("/key_result_value/:id", keyResultValueDelete),
	webservice.Get("/key_result_value/:id", keyResultValue),
}

// objective list
//
//	@Summary	objective list
//	@Tags		okr
//	@Accept		json
//	@Produce	json
//	@Success	200	{object}	protocol.Response{data=[]model.Objective}
//	@Security	ApiKeyAuth
//	@Router		/okr/objectives [get]
func objectiveList(ctx *fiber.Ctx) error {
	uid := route.GetUid(ctx)
	topic := route.GetTopic(ctx)

	list, err := store.Database.ListObjectives(uid, topic)
	if err != nil {
		return protocol.ErrDatabaseReadError.Wrap(err)
	}
	return ctx.JSON(protocol.NewSuccessResponse(list))
}

// objective detail
//
//	@Summary	objective detail
//	@Tags		okr
//	@Accept		json
//	@Produce	json
//	@Param		sequence	path		int	true	"Sequence"
//	@Success	200			{object}	protocol.Response{data=model.Objective}
//	@Security	ApiKeyAuth
//	@Router		/okr/objective/{sequence} [get]
func objectiveDetail(ctx *fiber.Ctx) error {
	uid := route.GetUid(ctx)
	topic := route.GetTopic(ctx)
	sequence := route.GetIntParam(ctx, "sequence")

	obj, err := store.Database.GetObjectiveBySequence(uid, topic, sequence)
	if err != nil {
		return protocol.ErrDatabaseReadError.Wrap(err)
	}
	return ctx.JSON(protocol.NewSuccessResponse(obj))
}

// objective create
//
//	@Summary	objective create
//	@Tags		okr
//	@Accept		json
//	@Produce	json
//	@Param		objective	body		model.Objective	true	"objective data"
//	@Success	200			{object}	protocol.Response
//	@Security	ApiKeyAuth
//	@Router		/okr/objective [post]
func objectiveCreate(ctx *fiber.Ctx) error {
	uid := route.GetUid(ctx)
	topic := route.GetTopic(ctx)

	item := new(model.Objective)
	err := ctx.BodyParser(&item)
	if err != nil {
		return protocol.ErrBadParam.Wrap(err)
	}

	// check
	if item.IsPlan > 0 {
		if item.PlanStart.IsZero() || item.PlanEnd.IsZero() {
			return protocol.ErrParamVerificationFailed.New("is plan emtpy")
		}
	}

	item.UID = uid.String()
	item.Topic = topic
	_, err = store.Database.CreateObjective(item)
	if err != nil {
		return protocol.ErrDatabaseWriteError.Wrap(err)
	}
	return ctx.JSON(protocol.NewSuccessResponse(nil))
}

// objective update
//
//	@Summary	objective update
//	@Tags		okr
//	@Accept		json
//	@Produce	json
//	@Param		sequence	path		int				true	"Sequence"
//	@Param		objective	body		model.Objective	true	"objective data"
//	@Success	200			{object}	protocol.Response
//	@Security	ApiKeyAuth
//	@Router		/okr/objective/{sequence} [put]
func objectiveUpdate(ctx *fiber.Ctx) error {
	uid := route.GetUid(ctx)
	topic := route.GetTopic(ctx)
	sequence := route.GetIntParam(ctx, "sequence")

	item := new(model.Objective)
	err := ctx.BodyParser(&item)
	if err != nil {
		return protocol.ErrBadParam.Wrap(err)
	}
	item.UID = uid.String()
	item.Topic = topic
	item.Sequence = int32(sequence)
	err = store.Database.UpdateObjective(item)
	if err != nil {
		return protocol.ErrDatabaseWriteError.Wrap(err)
	}
	return ctx.JSON(protocol.NewSuccessResponse(nil))
}

// objective delete
//
//	@Summary	objective delete
//	@Tags		okr
//	@Accept		json
//	@Produce	json
//	@Param		sequence	path		int	true	"Sequence"
//	@Success	200			{object}	protocol.Response
//	@Security	ApiKeyAuth
//	@Router		/okr/objective/{sequence} [delete]
func objectiveDelete(ctx *fiber.Ctx) error {
	uid := route.GetUid(ctx)
	topic := route.GetTopic(ctx)
	sequence := route.GetIntParam(ctx, "sequence")

	err := store.Database.DeleteObjectiveBySequence(uid, topic, sequence)
	if err != nil {
		return protocol.ErrDatabaseWriteError.Wrap(err)
	}
	return ctx.JSON(protocol.NewSuccessResponse(nil))
}

// KeyResult create
//
//	@Summary	KeyResult create
//	@Tags		okr
//	@Accept		json
//	@Produce	json
//	@Param		keyResult	body		model.KeyResult	true	"KeyResult data"
//	@Success	200			{object}	protocol.Response
//	@Security	ApiKeyAuth
//	@Router		/okr/key_result [post]
func keyResultCreate(ctx *fiber.Ctx) error {
	uid := route.GetUid(ctx)
	topic := route.GetTopic(ctx)

	item := new(model.KeyResult)
	err := ctx.BodyParser(&item)
	if err != nil {
		return protocol.ErrBadParam.Wrap(err)
	}
	item.UID = uid.String()
	item.Topic = topic
	_, err = store.Database.CreateKeyResult(item)
	if err != nil {
		return protocol.ErrDatabaseWriteError.Wrap(err)
	}
	return ctx.JSON(protocol.NewSuccessResponse(nil))
}

// KeyResult update
//
//	@Summary	KeyResult update
//	@Tags		okr
//	@Accept		json
//	@Produce	json
//	@Param		sequence	path		int				true	"Sequence"
//	@Param		objective	body		model.KeyResult	true	"KeyResult data"
//	@Success	200			{object}	protocol.Response
//	@Security	ApiKeyAuth
//	@Router		/okr/key_result/{sequence} [put]
func keyResultUpdate(ctx *fiber.Ctx) error {
	uid := route.GetUid(ctx)
	topic := route.GetTopic(ctx)
	sequence := route.GetIntParam(ctx, "sequence")

	item := new(model.KeyResult)
	err := ctx.BodyParser(&item)
	if err != nil {
		return protocol.ErrBadParam.Wrap(err)
	}
	item.UID = uid.String()
	item.Topic = topic
	item.Sequence = int32(sequence)
	err = store.Database.UpdateKeyResult(item)
	if err != nil {
		return protocol.ErrDatabaseWriteError.Wrap(err)
	}
	return ctx.JSON(protocol.NewSuccessResponse(nil))
}

// KeyResult delete
//
//	@Summary	KeyResult delete
//	@Tags		okr
//	@Accept		json
//	@Produce	json
//	@Param		sequence	path		int	true	"Sequence"
//	@Success	200			{object}	protocol.Response
//	@Security	ApiKeyAuth
//	@Router		/okr/key_result/{sequence} [delete]
func keyResultDelete(ctx *fiber.Ctx) error {
	uid := route.GetUid(ctx)
	topic := route.GetTopic(ctx)
	sequence := route.GetIntParam(ctx, "sequence")

	err := store.Database.DeleteKeyResultBySequence(uid, topic, sequence)
	if err != nil {
		return protocol.ErrDatabaseWriteError.Wrap(err)
	}
	return ctx.JSON(protocol.NewSuccessResponse(nil))
}

// key result value list
//
//	@Summary	key result value list
//	@Tags		okr
//	@Accept		json
//	@Produce	json
//	@Param		id	path		int	true	"key result id"
//	@Success	200	{object}	protocol.Response{data=[]model.KeyResultValue}
//	@Security	ApiKeyAuth
//	@Router		/okr/key_result/{id}/values [get]
func keyResultValueList(ctx *fiber.Ctx) error {
	keyResultId := route.GetIntParam(ctx, "id")
	if keyResultId == 0 {
		return protocol.ErrBadParam.New("id empty")
	}

	list, err := store.Database.GetKeyResultValues(keyResultId)
	if err != nil {
		return protocol.ErrDatabaseReadError.Wrap(err)
	}
	return ctx.JSON(protocol.NewSuccessResponse(list))
}

// KeyResult value create
//
//	@Summary	KeyResult value create
//	@Tags		okr
//	@Accept		json
//	@Produce	json
//	@Param		id				path		int						true	"key result id"
//	@Param		KeyResultValue	body		model.KeyResultValue	true	"KeyResultValue data"
//	@Success	200				{object}	protocol.Response
//	@Security	ApiKeyAuth
//	@Router		/okr/key_result/{id}/value [post]
func keyResultValueCreate(ctx *fiber.Ctx) error {
	keyResultId := route.GetIntParam(ctx, "id")
	if keyResultId == 0 {
		return protocol.ErrBadParam.New("id empty")
	}

	item := new(model.KeyResultValue)
	err := ctx.BodyParser(&item)
	if err != nil {
		return protocol.ErrBadParam.Wrap(err)
	}
	item.KeyResultID = keyResultId
	_, err = store.Database.CreateKeyResultValue(item)
	if err != nil {
		return protocol.ErrDatabaseWriteError.Wrap(err)
	}
	return ctx.JSON(protocol.NewSuccessResponse(nil))
}

// KeyResult value delete
//
//	@Summary	KeyResult value delete
//	@Tags		okr
//	@Accept		json
//	@Produce	json
//	@Param		id	path		int	true	"key result id"
//	@Success	200	{object}	protocol.Response
//	@Security	ApiKeyAuth
//	@Router		/okr/key_result_value/{id} [delete]
func keyResultValueDelete(ctx *fiber.Ctx) error {
	keyResultValueId := route.GetIntParam(ctx, "id")
	if keyResultValueId == 0 {
		return protocol.ErrBadParam.New("id emtpy")
	}

	err := store.Database.DeleteKeyResultValue(keyResultValueId)
	if err != nil {
		return protocol.ErrDatabaseWriteError.Wrap(err)
	}
	return ctx.JSON(protocol.NewSuccessResponse(nil))
}

// KeyResult value detail
//
//	@Summary	KeyResult value detail
//	@Tags		okr
//	@Accept		json
//	@Produce	json
//	@Param		id	path		int	true	"key result id"
//	@Success	200	{object}	protocol.Response{data=model.KeyResultValue}
//	@Security	ApiKeyAuth
//	@Router		/okr/key_result_value/{id} [get]
func keyResultValue(ctx *fiber.Ctx) error {
	keyResultValueId := route.GetIntParam(ctx, "id")
	if keyResultValueId == 0 {
		return protocol.ErrBadParam.New("id emtpy")
	}

	item, err := store.Database.GetKeyResultValue(keyResultValueId)
	if err != nil {
		return protocol.ErrDatabaseReadError.Wrap(err)
	}
	return ctx.JSON(protocol.NewSuccessResponse(item))
}
