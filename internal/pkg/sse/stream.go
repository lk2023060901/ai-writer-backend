package sse

import (
	"context"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// Stream SSE 流(封装 Client 和 Context)
type Stream struct {
	client     *Client
	ctx        *gin.Context
	hub        *Hub
	resource   string
	bufferSize int
	heartbeat  time.Duration

	// 生命周期钩子
	onConnect    func()
	onDisconnect func()
	onError      func(error)

	// 内部状态
	closed       atomic.Bool
	cancelFunc   context.CancelFunc
	connectTime  time.Time
}

// StreamBuilder 构建器
type StreamBuilder struct {
	ginCtx       *gin.Context
	hub          *Hub
	resource     string
	bufferSize   int
	heartbeat    time.Duration
	onConnect    func()
	onDisconnect func()
	onError      func(error)
}

// NewStream 创建 Stream 构建器
func NewStream(c *gin.Context, hub *Hub) *StreamBuilder {
	return &StreamBuilder{
		ginCtx:     c,
		hub:        hub,
		bufferSize: 10,              // 默认缓冲区
		heartbeat:  30 * time.Second, // 默认 30s 心跳
	}
}

// WithResource 设置资源 ID
func (b *StreamBuilder) WithResource(resource string) *StreamBuilder {
	b.resource = resource
	return b
}

// WithBufferSize 设置 Channel 缓冲区大小
func (b *StreamBuilder) WithBufferSize(size int) *StreamBuilder {
	b.bufferSize = size
	return b
}

// WithHeartbeat 设置心跳间隔(0 表示禁用心跳)
func (b *StreamBuilder) WithHeartbeat(interval time.Duration) *StreamBuilder {
	b.heartbeat = interval
	return b
}

// OnConnect 设置连接建立钩子
func (b *StreamBuilder) OnConnect(fn func()) *StreamBuilder {
	b.onConnect = fn
	return b
}

// OnDisconnect 设置连接断开钩子
func (b *StreamBuilder) OnDisconnect(fn func()) *StreamBuilder {
	b.onDisconnect = fn
	return b
}

// OnError 设置错误处理钩子
func (b *StreamBuilder) OnError(fn func(error)) *StreamBuilder {
	b.onError = fn
	return b
}

// Build 构建 Stream
func (b *StreamBuilder) Build() *Stream {
	client := &Client{
		ID:       uuid.New().String(),
		Channel:  make(chan Event, b.bufferSize),
		Resource: b.resource,
	}

	stream := &Stream{
		client:       client,
		ctx:          b.ginCtx,
		hub:          b.hub,
		resource:     b.resource,
		bufferSize:   b.bufferSize,
		heartbeat:    b.heartbeat,
		onConnect:    b.onConnect,
		onDisconnect: b.onDisconnect,
		onError:      b.onError,
		connectTime:  time.Now(),
	}

	return stream
}

// Send 发送事件(并发安全)
func (s *Stream) Send(eventType string, data interface{}) error {
	if s.closed.Load() {
		return fmt.Errorf("stream closed")
	}

	event := Event{
		Type: eventType,
		Data: data,
	}

	select {
	case s.client.Channel <- event:
		return nil
	default:
		// Channel 已满,丢弃消息并记录错误
		err := fmt.Errorf("stream buffer full, event dropped: %s", eventType)
		if s.onError != nil {
			s.onError(err)
		}
		return err
	}
}

// Close 关闭流(幂等)
func (s *Stream) Close() error {
	if !s.closed.CompareAndSwap(false, true) {
		return nil // 已关闭
	}

	s.hub.Unregister(s.client)

	if s.cancelFunc != nil {
		s.cancelFunc()
	}

	if s.onDisconnect != nil {
		s.onDisconnect()
	}

	return nil
}

// StartStreaming 开始流式传输(阻塞直到连接关闭)
func (s *Stream) StartStreaming() {
	// 设置 SSE 响应头
	s.ctx.Header("Content-Type", "text/event-stream")
	s.ctx.Header("Cache-Control", "no-cache")
	s.ctx.Header("Connection", "keep-alive")
	s.ctx.Header("X-Accel-Buffering", "no")

	// 注册客户端
	s.hub.Register(s.client)
	defer s.Close()

	// 触发连接钩子
	if s.onConnect != nil {
		s.onConnect()
	}

	// 发送初始连接成功消息
	connectedEvent := Event{
		Type: "connected",
		Data: map[string]string{
			"client_id": s.client.ID,
			"resource":  s.client.Resource,
		},
	}
	_, err := fmt.Fprint(s.ctx.Writer, connectedEvent.FormatSSE())
	if err != nil {
		if s.onError != nil {
			s.onError(err)
		}
		return
	}
	s.ctx.Writer.Flush()

	// 启动心跳(如果启用)
	var heartbeatCtx context.Context
	if s.heartbeat > 0 {
		heartbeatCtx, s.cancelFunc = context.WithCancel(context.Background())
		go s.startHeartbeat(heartbeatCtx)
	}

	// 监听客户端断开和消息
	clientGone := s.ctx.Request.Context().Done()

	for {
		select {
		case <-clientGone:
			return

		case event, ok := <-s.client.Channel:
			if !ok {
				// Channel 已关闭
				return
			}

			// 发送事件
			_, err := fmt.Fprint(s.ctx.Writer, event.FormatSSE())
			if err != nil {
				if s.onError != nil {
					s.onError(err)
				}
				return
			}
			s.ctx.Writer.Flush()
		}
	}
}

// startHeartbeat 启动心跳
func (s *Stream) startHeartbeat(ctx context.Context) {
	ticker := time.NewTicker(s.heartbeat)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			_, err := fmt.Fprintf(s.ctx.Writer, ": heartbeat\n\n")
			if err != nil {
				if s.onError != nil {
					s.onError(err)
				}
				return
			}
			s.ctx.Writer.Flush()
		}
	}
}

// GetClientID 获取客户端 ID
func (s *Stream) GetClientID() string {
	return s.client.ID
}

// GetResource 获取资源 ID
func (s *Stream) GetResource() string {
	return s.resource
}

// GetDuration 获取连接时长
func (s *Stream) GetDuration() time.Duration {
	return time.Since(s.connectTime)
}

// IsClosed 检查是否已关闭
func (s *Stream) IsClosed() bool {
	return s.closed.Load()
}

// SendAndFlush 发送事件并立即刷新(用于紧急消息)
func (s *Stream) SendAndFlush(eventType string, data interface{}) error {
	if err := s.Send(eventType, data); err != nil {
		return err
	}

	// 直接写入并刷新
	select {
	case event := <-s.client.Channel:
		_, err := fmt.Fprint(s.ctx.Writer, event.FormatSSE())
		if err != nil {
			return err
		}
		s.ctx.Writer.Flush()
		return nil
	default:
		return nil
	}
}
