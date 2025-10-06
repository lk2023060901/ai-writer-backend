package sse

import (
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
)

// StreamResponse SSE 流式响应辅助函数
func StreamResponse(c *gin.Context, client *Client, hub *Hub, keepAliveInterval time.Duration) {
	// 设置 SSE 响应头
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")

	// 注册客户端
	hub.Register(client)
	defer hub.Unregister(client)

	// 发送初始连接成功消息
	connectedEvent := Event{
		Type: "connected",
		Data: map[string]string{
			"client_id": client.ID,
			"resource":  client.Resource,
		},
	}
	_, err := fmt.Fprint(c.Writer, connectedEvent.FormatSSE())
	if err != nil {
		return
	}
	c.Writer.Flush()

	// Keep-alive ticker
	ticker := time.NewTicker(keepAliveInterval)
	defer ticker.Stop()

	// 监听客户端断开和消息
	clientGone := c.Request.Context().Done()

	for {
		select {
		case <-clientGone:
			return

		case event := <-client.Channel:
			// 发送事件
			_, err := fmt.Fprint(c.Writer, event.FormatSSE())
			if err != nil {
				return
			}
			c.Writer.Flush()

		case <-ticker.C:
			// 发送心跳
			_, err := fmt.Fprintf(c.Writer, ": heartbeat\n\n")
			if err != nil {
				return
			}
			c.Writer.Flush()
		}
	}
}
