package slack

import (
	"github.com/flowline-io/flowbot/internal/types/protocol"
)

type Adapter struct {
}

func (a *Adapter) MessageConvert(data any) protocol.Message {
	//TODO implement me
	panic("implement me")
}

func (a *Adapter) EventConvert(data any) protocol.Event {
	//TODO implement me
	panic("implement me")
}
