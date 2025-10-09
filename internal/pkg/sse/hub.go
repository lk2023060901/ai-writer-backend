package sse

import (
	"encoding/json"
	"sync"
)

// Event SSE 事件
type Event struct {
	Type string      `json:"type"` // 事件类型
	Data interface{} `json:"data"` // 事件数据
}

// Client SSE 客户端连接
type Client struct {
	ID       string
	Channel  chan Event
	Resource string // 订阅的资源 ID (如 doc:xxx, chat:xxx)
	closed   bool   // 标记 channel 是否已关闭
	mu       sync.Mutex
}

// Hub SSE 连接管理器
type Hub struct {
	mu      sync.RWMutex
	clients map[string]map[*Client]bool // resource -> clients
}

// NewHub 创建 Hub
func NewHub() *Hub {
	return &Hub{
		clients: make(map[string]map[*Client]bool),
	}
}

// Register 注册客户端
func (h *Hub) Register(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.clients[client.Resource] == nil {
		h.clients[client.Resource] = make(map[*Client]bool)
	}
	h.clients[client.Resource][client] = true
}

// Unregister 注销客户端
func (h *Hub) Unregister(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if clients, ok := h.clients[client.Resource]; ok {
		if _, exists := clients[client]; exists {
			delete(clients, client)

			// 安全关闭 channel
			client.mu.Lock()
			if !client.closed {
				close(client.Channel)
				client.closed = true
			}
			client.mu.Unlock()

			// 清理空资源
			if len(clients) == 0 {
				delete(h.clients, client.Resource)
			}
		}
	}
}

// Broadcast 向订阅指定资源的所有客户端广播消息
func (h *Hub) Broadcast(resource string, event Event) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if clients, ok := h.clients[resource]; ok {
		for client := range clients {
			// 检查 channel 是否已关闭
			client.mu.Lock()
			closed := client.closed
			client.mu.Unlock()

			if closed {
				continue
			}

			// 使用 defer recover 作为双重保险
			func(c *Client) {
				defer func() {
					if r := recover(); r != nil {
						// channel 已关闭，静默忽略
					}
				}()

				select {
				case c.Channel <- event:
				default:
					// 客户端缓冲区满,跳过
				}
			}(client)
		}
	}
}

// Send 向指定客户端发送消息
func (h *Hub) Send(clientID string, event Event) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	for _, clients := range h.clients {
		for client := range clients {
			if client.ID == clientID {
				// 检查 channel 是否已关闭
				client.mu.Lock()
				closed := client.closed
				client.mu.Unlock()

				if closed {
					return
				}

				// 使用 defer recover 作为双重保险
				func(c *Client) {
					defer func() {
						if r := recover(); r != nil {
							// channel 已关闭，静默忽略
						}
					}()

					select {
					case c.Channel <- event:
					default:
					}
				}(client)
				return
			}
		}
	}
}

// GetClientCount 获取订阅指定资源的客户端数量
func (h *Hub) GetClientCount(resource string) int {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if clients, ok := h.clients[resource]; ok {
		return len(clients)
	}
	return 0
}

// FormatSSE 格式化为 SSE 消息格式
func (e Event) FormatSSE() string {
	// 将事件类型包含在 data 中，方便前端解析
	dataWithType := map[string]interface{}{
		"type": e.Type,
	}

	// 如果 Data 是 map，则合并；否则放在 payload 字段中
	if dataMap, ok := e.Data.(map[string]interface{}); ok {
		for k, v := range dataMap {
			dataWithType[k] = v
		}
	} else {
		dataWithType["payload"] = e.Data
	}

	data, _ := json.Marshal(dataWithType)
	return "event: " + e.Type + "\ndata: " + string(data) + "\n\n"
}
