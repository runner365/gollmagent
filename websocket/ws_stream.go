package websocket

import (
	"github.com/gorilla/websocket"

	log "github.com/gollmagent/logging"
	"github.com/gollmagent/pub"
)

const kSendBufferSize = 1000

type WsStream struct {
	id   string
	conn *websocket.Conn
	cb   pub.WebSocketCallback
	send chan []byte
}

func NewWsStream(id string, conn *websocket.Conn, cb pub.WebSocketCallback) *WsStream {
	log.Infof("Creating new WebSocket stream for userId: %s", id)
	return &WsStream{
		id:   id,
		conn: conn,
		cb:   cb,
		send: make(chan []byte, kSendBufferSize),
	}
}

func (stream *WsStream) Run() error {
	log.Infof("Starting WebSocket stream for userId: %s", stream.id)
	go stream.onSend()
	go stream.onReceive()
	return nil
}

func (stream *WsStream) onSend() {
	defer func() {
		log.Infof("WebSocket stream for userId %s closed", stream.id)
		if r := recover(); r != nil {
			log.Errorf("Recovered from panic in onSend: %v", r)
		}
	}()
	log.Infof("WebSocket stream for userId %s is ready to send messages", stream.id)
	for {
		msg, ok := <-stream.send
		if !ok {
			break
		}
		stream.handleSend(msg)
	}
}

func (stream *WsStream) handleSend(msg []byte) error {
	err := stream.conn.WriteMessage(websocket.TextMessage, msg)
	if err != nil {
		log.Errorf("Failed to send WebSocket message: %v", err)
		return err
	}
	log.Debugf("WebSocket message sent: %s", msg)
	return nil
}

func (stream *WsStream) Send(msg []byte) {
	defer func() {
		if r := recover(); r != nil {
			log.Errorf("Recovered from panic in Send: %v", r)
		}
	}()

	stream.send <- msg
}

func (stream *WsStream) onReceive() {
	defer func() {
		log.Infof("WebSocket stream for userId %s closed", stream.id)
		if r := recover(); r != nil {
			log.Errorf("Recovered from panic in onReceive: %v", r)
		}
	}()
	log.Infof("WebSocket stream for userId %s is ready to receive messages", stream.id)
	for {
		_, msg, err := stream.conn.ReadMessage()
		if err != nil {
			return
		}
		stream.handleReceive(msg)
	}
}

func (stream *WsStream) handleReceive(msg []byte) {
	defer func() {
		if r := recover(); r != nil {
			log.Errorf("Recovered from panic in handleReceive: %v", r)
		}
	}()
	// 处理接收到的消息
	log.Debugf("WebSocket message received: %s", msg)
	stream.cb.OnMessage(stream.id, msg, stream)
}
