package api

import (
	"log/slog"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type WebSocketHub struct {
	mu      sync.Mutex
	clients map[string]map[*websocket.Conn]struct{}
}

func NewWebSocketHub() *WebSocketHub {
	return &WebSocketHub{
		clients: make(map[string]map[*websocket.Conn]struct{}),
	}
}

func (h *WebSocketHub) Run() {}

func (h *WebSocketHub) Subscribe(topic string, conn *websocket.Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.clients[topic] == nil {
		h.clients[topic] = make(map[*websocket.Conn]struct{})
	}
	h.clients[topic][conn] = struct{}{}
}

func (h *WebSocketHub) Unsubscribe(topic string, conn *websocket.Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if clients, ok := h.clients[topic]; ok {
		delete(clients, conn)
		if len(clients) == 0 {
			delete(h.clients, topic)
		}
	}
}

func (h *WebSocketHub) Broadcast(topic string, msg any) {
	h.mu.Lock()
	defer h.mu.Unlock()
	clients, ok := h.clients[topic]
	if !ok {
		return
	}
	var toDelete []*websocket.Conn
	for conn := range clients {
		if err := conn.WriteJSON(msg); err != nil {
			slog.Warn("websocket write error", "topic", topic, "error", err)
			conn.Close()
			toDelete = append(toDelete, conn)
		}
	}
	for _, conn := range toDelete {
		delete(clients, conn)
	}
	if len(clients) == 0 {
		delete(h.clients, topic)
	}
}
