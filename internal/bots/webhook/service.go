package webhook

import (
	"fmt"
	"github.com/emicklei/go-restful/v3"
	"github.com/sysatom/flowbot/internal/store"
	"github.com/sysatom/flowbot/internal/types"
	"github.com/sysatom/flowbot/pkg/event"
	"github.com/sysatom/flowbot/pkg/logs"
	"github.com/sysatom/flowbot/pkg/route"
	"io"
)

const serviceVersion = "v1"

func webhook(req *restful.Request, resp *restful.Response) {
	flag := req.PathParameter("flag")

	p, err := store.Chatbot.ParameterGet(flag)
	if err != nil {
		route.ErrorResponse(resp, "flag error")
		return
	}
	if p.IsExpired() {
		route.ErrorResponse(resp, "page expired")
		return
	}

	//uid, _ := types.KV(p.Params).String("uid")
	//userUid := types.ParseUserId(uid)
	botUid := types.Uid(0) // fixme
	topic := ""            // fixme

	d, _ := io.ReadAll(req.Request.Body)

	txt := ""
	if len(d) > 1000 {
		txt = fmt.Sprintf("[webhook:%s] body too long", flag)
	} else {
		txt = fmt.Sprintf("[webhook:%s] %s", flag, string(d))
	}
	// send
	err = event.Emit(event.SendEvent, types.KV{
		"topic":     topic,
		"topic_uid": int64(botUid),
		"message":   txt,
	})
	if err != nil {
		logs.Err.Println(err)
		_, _ = resp.Write([]byte("send error"))
		return
	}

	_, _ = resp.Write([]byte("ok"))
}
