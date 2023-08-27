package event

import (
	"github.com/flowline-io/flowbot/internal/types"
	"github.com/flowline-io/flowbot/pkg/logs"
	"github.com/gookit/event"
)

type ListenerFunc func(data types.KV) error

func eventName(name string) string {
	return name
}

func On(name string, listener ListenerFunc) {
	event.Std().On(eventName(name), event.ListenerFunc(func(e event.Event) error {
		return listener(e.Data())
	}))
}

func Emit(name string, params types.KV) error {
	err, _ := event.Std().Fire(eventName(name), params)
	return err
}

func AsyncEmit(name string, params types.KV) {
	event.Std().FireC(eventName(name), params)
}

func Shutdown() {
	err := event.Std().CloseWait()
	if err != nil {
		logs.Err.Println(err)
		return
	}
	logs.Info.Println("event stopped")
}
