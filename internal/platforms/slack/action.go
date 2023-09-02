package slack

import (
	"github.com/flowline-io/flowbot/internal/types/protocol"
)

type Action struct {
}

func (a *Action) GetLatestEvents(req protocol.ActionRequest) protocol.ActionResponse {
	return protocol.NewFailedActionResponse(protocol.ErrUnsupportedAction)
}

func (a *Action) GetSupportedActions(req protocol.ActionRequest) protocol.ActionResponse {
	return protocol.NewFailedActionResponse(protocol.ErrUnsupportedAction)
}

func (a *Action) GetStatus(req protocol.ActionRequest) protocol.ActionResponse {
	return protocol.NewFailedActionResponse(protocol.ErrUnsupportedAction)
}

func (a *Action) GetVersion(req protocol.ActionRequest) protocol.ActionResponse {
	return protocol.NewFailedActionResponse(protocol.ErrUnsupportedAction)
}

func (a *Action) SendMessage(req protocol.ActionRequest) protocol.ActionResponse {
	return protocol.NewFailedActionResponse(protocol.ErrUnsupportedAction)
}

func (a *Action) GetUserInfo(req protocol.ActionRequest) protocol.ActionResponse {
	return protocol.NewFailedActionResponse(protocol.ErrUnsupportedAction)
}

func (a *Action) CreateChannel(req protocol.ActionRequest) protocol.ActionResponse {
	return protocol.NewFailedActionResponse(protocol.ErrUnsupportedAction)
}

func (a *Action) GetChannelInfo(req protocol.ActionRequest) protocol.ActionResponse {
	return protocol.NewFailedActionResponse(protocol.ErrUnsupportedAction)
}

func (a *Action) GetChannelList(req protocol.ActionRequest) protocol.ActionResponse {
	return protocol.NewFailedActionResponse(protocol.ErrUnsupportedAction)
}

func (a *Action) RegisterChannels(req protocol.ActionRequest) protocol.ActionResponse {
	return protocol.NewFailedActionResponse(protocol.ErrUnsupportedAction)
}

func (a *Action) RegisterSlashCommands(req protocol.ActionRequest) protocol.ActionResponse {
	return protocol.NewFailedActionResponse(protocol.ErrUnsupportedAction)
}
