package tailchat

const ID = "tailchat"

type PayloadData struct {
	GroupID             string `json:"groupId"`
	ConverseID          string `json:"converseId"`
	MessageID           string `json:"messageId"`
	MessageAuthor       string `json:"messageAuthor"`
	MessageSnippet      string `json:"messageSnippet"`
	MessagePlainContent string `json:"messagePlainContent"`
}

type Payload struct {
	ID      string      `json:"_id"`
	UserID  string      `json:"userId"`
	Type    string      `json:"type"`
	Payload PayloadData `json:"payload"`
}

type SendMessageData struct {
	ConverseId string          `json:"converseId"`
	GroupId    string          `json:"groupId"`
	Content    string          `json:"content"`
	Plain      string          `json:"plain"`
	Meta       SendMessageMeta `json:"meta"`
}

type SendMessageMeta struct {
	Mentions []string         `json:"mentions"`
	Reply    SendMessageReply `json:"reply"`
}

type SendMessageReply struct {
	Id      string `json:"_id"`
	Author  string `json:"author"`
	Content string `json:"content"`
}

type TokenData struct {
	Jwt string `json:"jwt"`
}

type TokenResponse struct {
	Data TokenData `json:"data"`
}
