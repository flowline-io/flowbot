package okr

import (
	"github.com/emicklei/go-restful/v3"
	"github.com/sysatom/flowbot/internal/store"
	"github.com/sysatom/flowbot/internal/store/model"
	"github.com/sysatom/flowbot/internal/types"
	"net/http"
	"strconv"
)

const serviceVersion = "v1"

func objectiveList(req *restful.Request, resp *restful.Response) {
	uid, _ := req.Attribute("uid").(types.Uid)
	topic, _ := req.Attribute("uid").(string)
	list, err := store.Chatbot.ListObjectives(uid, topic)
	if err != nil {
		_ = resp.WriteAsJson(types.ErrMessage(http.StatusBadRequest, err.Error()))
		return
	}
	_ = resp.WriteAsJson(types.OkMessage(list))
}

func objectiveDetail(req *restful.Request, resp *restful.Response) {
	uid, _ := req.Attribute("uid").(types.Uid)
	topic, _ := req.Attribute("uid").(string)
	s := req.PathParameter("sequence")
	sequence, _ := strconv.ParseInt(s, 10, 64)

	obj, err := store.Chatbot.GetObjectiveBySequence(uid, topic, sequence)
	if err != nil {
		_ = resp.WriteAsJson(types.ErrMessage(http.StatusNotFound, ""))
		return
	}
	_ = resp.WriteAsJson(types.OkMessage(obj))
}

func objectiveCreate(req *restful.Request, resp *restful.Response) {
	uid, _ := req.Attribute("uid").(types.Uid)
	topic, _ := req.Attribute("uid").(string)
	obj := new(model.Objective)
	err := req.ReadEntity(&obj)
	if err != nil {
		_ = resp.WriteAsJson(types.ErrMessage(http.StatusNotFound, err.Error()))
		return
	}
	obj.UID = uid.UserId()
	obj.Topic = topic
	_, err = store.Chatbot.CreateObjective(obj)
	if err != nil {
		_ = resp.WriteAsJson(types.ErrMessage(http.StatusNotFound, err.Error()))
		return
	}
	_ = resp.WriteAsJson(types.OkMessage(nil))
}

func objectiveUpdate(req *restful.Request, resp *restful.Response) {
	uid, _ := req.Attribute("uid").(types.Uid)
	topic, _ := req.Attribute("uid").(string)
	obj := new(model.Objective)
	err := req.ReadEntity(&obj)
	if err != nil {
		_ = resp.WriteAsJson(types.ErrMessage(http.StatusNotFound, err.Error()))
		return
	}
	obj.UID = uid.UserId()
	obj.Topic = topic
	err = store.Chatbot.UpdateObjective(obj)
	if err != nil {
		_ = resp.WriteAsJson(types.ErrMessage(http.StatusNotFound, err.Error()))
		return
	}
	_ = resp.WriteAsJson(types.OkMessage(nil))
}

func objectiveDelete(req *restful.Request, resp *restful.Response) {
	uid, _ := req.Attribute("uid").(types.Uid)
	topic, _ := req.Attribute("uid").(string)
	s := req.PathParameter("sequence")
	sequence, _ := strconv.ParseInt(s, 10, 64)

	err := store.Chatbot.DeleteObjectiveBySequence(uid, topic, sequence)
	if err != nil {
		_ = resp.WriteAsJson(types.ErrMessage(http.StatusNotFound, err.Error()))
		return
	}
	_ = resp.WriteAsJson(types.OkMessage(nil))
}
