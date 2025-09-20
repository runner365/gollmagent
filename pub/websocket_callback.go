package pub

type WebSocketCallback interface {
	OnMessage(id string, message []byte, ws WsStreamI) error
	OnClose() error
}

type WsStreamI interface {
	Send(msg []byte)
}

type ChatMessageBaseInfo struct {
	MsgType string `json:"type"`
	UserId  string `json:"userId"`
}

type ChatMessageInfo struct {
	MsgType string `json:"type"`
	UserId  string `json:"userId"`
	Role    string `json:"role"` // "user" or "assistant"
	ItemId  string `json:"itemId"`
	Content string `json:"content"`
	Ts      int64  `json:"timestamp"`
	Done    bool   `json:"done,omitempty"` // Indicates if the message is complete
}

type ChatVoiceInfo struct {
	MsgType    string `json:"type"`
	UserId     string `json:"userId"`
	ItemId     string `json:"itemId"`
	Seq        int64  `json:"seq"`
	AudioType  string `json:"audioType"` // "delta" or "done"
	Base64Data string `json:"base64Data"`
	Ts         int64  `json:"timestamp"`
}
