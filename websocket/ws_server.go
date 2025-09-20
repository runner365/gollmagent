package websocket

import (
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
	log "github.com/gollmagent/logging"
	"github.com/gollmagent/pub"
)

type WsServer struct {
	cb       pub.WebSocketCallback // 回调接口
	upgrader websocket.Upgrader    // WebSocket 升级器
	clients  map[string]*WsStream
	mu       sync.Mutex // 保护 clients 的并发访问
}

func NewWsServer(cb pub.WebSocketCallback) *WsServer {
	return &WsServer{
		cb: cb,
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin: func(r *http.Request) bool {
				return true // 允许所有跨域请求（生产环境应限制）
			},
		},
		clients: make(map[string]*WsStream),
	}
}

func (s *WsServer) authId(id string) bool {
	return true
}

func (s *WsServer) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	id := query.Get("userId")

	if id == "" {
		http.Error(w, "Missing 'id' parameter", http.StatusBadRequest)
		return
	}

	if !s.authId(id) {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	log.Infof("WebSocket connection request from userId: %s", id)
	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Errorf("WebSocket upgrade failed: %v", err)
		return
	}

	stream := NewWsStream(id, conn, s.cb)

	s.mu.Lock()
	s.clients[id] = stream
	s.mu.Unlock()

	log.Infof("WebSocket connection established for userId: %s, clients total: %d", id, len(s.clients))

	go stream.Run()
}
