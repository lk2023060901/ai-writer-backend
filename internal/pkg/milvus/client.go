package milvus

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/lk2023060901/ai-writer-backend/internal/pkg/logger"
	"github.com/milvus-io/milvus/client/v2/milvusclient"
	"go.uber.org/zap"
)

// Client Milvus 客户端封装
type Client struct {
	cfg    *Config
	client *milvusclient.Client
	logger *logger.Logger
	mu     sync.RWMutex
	closed bool
}

// New 创建新的 Milvus 客户端
func New(ctx context.Context, cfg *Config, log *logger.Logger) (*Client, error) {
	if cfg == nil {
		return nil, ErrInvalidConfig
	}

	if err := cfg.Validate(); err != nil {
		return nil, WrapError("New", err, "", "")
	}

	if log == nil {
		log = logger.L()
	}

	// 设置默认值
	cfg.SetDefaults()

	// 构建客户端配置
	clientCfg := &milvusclient.ClientConfig{
		Address: cfg.Address,
	}

	// 设置认证
	if cfg.Username != "" && cfg.Password != "" {
		clientCfg.Username = cfg.Username
		clientCfg.Password = cfg.Password
	}

	if cfg.APIKey != "" {
		clientCfg.APIKey = cfg.APIKey
	}

	// 设置数据库
	if cfg.Database != "" {
		clientCfg.DBName = cfg.Database
	}

	// 创建带超时的上下文
	dialCtx := ctx
	if cfg.DialTimeout > 0 {
		var cancel context.CancelFunc
		dialCtx, cancel = context.WithTimeout(ctx, cfg.DialTimeout)
		defer cancel()
	}

	// 连接到 Milvus
	client, err := milvusclient.New(dialCtx, clientCfg)
	if err != nil {
		return nil, WrapError("New", err, "", "")
	}

	log.Info("milvus client created successfully",
		zap.String("address", cfg.Address),
		zap.String("database", cfg.Database))

	return &Client{
		cfg:    cfg,
		client: client,
		logger: log,
		closed: false,
	}, nil
}

// Close 关闭客户端连接
func (c *Client) Close(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return ErrClientClosed
	}

	if c.client != nil {
		if err := c.client.Close(ctx); err != nil {
			c.logger.Error("failed to close milvus client", zap.Error(err))
			return WrapError("Close", err, "", "")
		}
	}

	c.closed = true
	c.logger.Info("milvus client closed successfully")
	return nil
}

// IsClosed 检查客户端是否已关闭
func (c *Client) IsClosed() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.closed
}

// GetClient 获取底层的 Milvus 客户端
func (c *Client) GetClient() *milvusclient.Client {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.closed {
		return nil
	}

	return c.client
}

// Ping 检查与 Milvus 服务器的连接
func (c *Client) Ping(ctx context.Context) error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.closed {
		return ErrClientClosed
	}

	// 使用 ListCollections 作为健康检查
	_, err := c.client.ListCollections(ctx, milvusclient.NewListCollectionOption())
	if err != nil {
		return WrapError("Ping", err, "", "")
	}

	return nil
}

// WithTimeout 创建带超时的上下文
func (c *Client) WithTimeout(ctx context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	if timeout <= 0 {
		timeout = c.cfg.RequestTimeout
	}
	return context.WithTimeout(ctx, timeout)
}

// execWithRetry 执行操作并支持重试
func (c *Client) execWithRetry(ctx context.Context, op string, fn func(context.Context) error) error {
	var err error
	maxRetries := c.cfg.MaxRetries
	retryDelay := c.cfg.RetryDelay

	for i := 0; i <= maxRetries; i++ {
		if i > 0 {
			c.logger.Warn("retrying operation",
				zap.String("operation", op),
				zap.Int("attempt", i),
				zap.Int("max_retries", maxRetries),
				zap.Error(err))

			// 等待后重试
			select {
			case <-ctx.Done():
				return WrapError(op, ctx.Err(), "", "")
			case <-time.After(retryDelay):
			}
		}

		err = fn(ctx)
		if err == nil {
			return nil
		}

		// 判断是否应该重试
		if !isRetryable(err) {
			return WrapError(op, err, "", "")
		}
	}

	return WrapError(op, fmt.Errorf("max retries exceeded: %w", err), "", "")
}

// isRetryable 判断错误是否可重试
func isRetryable(err error) bool {
	if err == nil {
		return false
	}

	// 超时错误可重试
	if IsTimeout(err) {
		return true
	}

	// 连接失败可重试
	if IsConnectionFailed(err) {
		return true
	}

	return false
}

// GetConfig 获取客户端配置
func (c *Client) GetConfig() *Config {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.cfg.Clone()
}

// GetLogger 获取日志记录器
func (c *Client) GetLogger() *logger.Logger {
	return c.logger
}

// Stats 客户端统计信息
type Stats struct {
	Address      string        `json:"address"`
	Database     string        `json:"database"`
	Connected    bool          `json:"connected"`
	Uptime       time.Duration `json:"uptime"`
	RequestCount int64         `json:"request_count"`
}

// GetStats 获取客户端统计信息
func (c *Client) GetStats() *Stats {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return &Stats{
		Address:   c.cfg.Address,
		Database:  c.cfg.Database,
		Connected: !c.closed,
	}
}
