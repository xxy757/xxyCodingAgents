// Package api 的 websocket 文件实现 WebSocket 连接的发布/订阅管理。
// WebSocketHub 维护主题到连接的映射，支持订阅、取消订阅和广播消息。
package api

import (
	"log/slog"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
)

// upgrader 是 WebSocket 升级器，允许所有来源的连接（开发模式）。
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// WebSocketHub 管理 WebSocket 客户端连接的发布/订阅。
// 支持按主题（topic）分组，向同一主题的所有连接广播消息。
type WebSocketHub struct {
	mu      sync.Mutex                          // 保护 clients 的并发访问
	clients map[string]map[*websocket.Conn]struct{} // 主题 -> 连接集合
}

// NewWebSocketHub 创建一个新的 WebSocket Hub。
func NewWebSocketHub() *WebSocketHub {
	return &WebSocketHub{
		clients: make(map[string]map[*websocket.Conn]struct{}),
	}
}

// Run 启动 Hub 的事件循环（当前为空实现，预留扩展）。
func (h *WebSocketHub) Run() {}

// Subscribe 将一个 WebSocket 连接订阅到指定主题。
func (h *WebSocketHub) Subscribe(topic string, conn *websocket.Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.clients[topic] == nil {
		h.clients[topic] = make(map[*websocket.Conn]struct{})
	}
	h.clients[topic][conn] = struct{}{}
}

// Unsubscribe 将一个 WebSocket 连接从指定主题取消订阅。
// 如果该主题不再有任何连接，则删除该主题。
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

// Broadcast 向指定主题的所有连接广播消息。
// 写入失败的连接会被自动关闭并移除。
func (h *WebSocketHub) Broadcast(topic string, msg any) {
	h.mu.Lock()
	defer h.mu.Unlock()
	clients, ok := h.clients[topic]
	if !ok {
		return
	}
	// 收集写入失败的连接，延迟删除
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
