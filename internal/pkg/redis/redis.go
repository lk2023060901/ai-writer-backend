package redis

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"math/rand"
	"os"
	"sync/atomic"
	"time"

	"github.com/lk2023060901/ai-writer-backend/internal/pkg/logger"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

// Client Redis 客户端封装
type Client struct {
	config *Config
	logger *logger.Logger

	// 客户端实例
	master  redis.UniversalClient   // 主节点客户端（写操作）
	slaves  []redis.UniversalClient // 从节点客户端（读操作）
	cluster redis.UniversalClient   // 集群客户端

	// 读写分离相关
	slaveIndex   atomic.Int32 // 轮询索引
	readStrategy ReadStrategy
}

// New 创建 Redis 客户端
func New(cfg *Config, log *logger.Logger) (*Client, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	client := &Client{
		config:       cfg,
		logger:       log,
		readStrategy: cfg.ReadStrategy,
	}

	// 根据模式创建客户端
	switch cfg.Mode {
	case ModeSingle:
		if err := client.initSingleMode(); err != nil {
			return nil, err
		}
	case ModeSentinel:
		if err := client.initSentinelMode(); err != nil {
			return nil, err
		}
	case ModeCluster:
		if err := client.initClusterMode(); err != nil {
			return nil, err
		}
	case ModeReadWrite:
		if err := client.initReadWriteMode(); err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("unsupported mode: %s", cfg.Mode)
	}

	// 健康检查
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx); err != nil {
		client.Close()
		return nil, fmt.Errorf("redis ping failed: %w", err)
	}

	client.logger.Info("redis client initialized successfully",
		zap.String("mode", string(cfg.Mode)),
		zap.String("master_addr", cfg.MasterAddr),
	)

	return client, nil
}

// initSingleMode 初始化单机模式
func (c *Client) initSingleMode() error {
	opts := &redis.Options{
		Addr:     c.config.MasterAddr,
		Username: c.config.Username,
		Password: c.config.Password,
		DB:       c.config.DB,

		PoolSize:     c.config.PoolSize,
		MinIdleConns: c.config.MinIdleConns,

		DialTimeout:  c.config.DialTimeout,
		ReadTimeout:  c.config.ReadTimeout,
		WriteTimeout: c.config.WriteTimeout,
		PoolTimeout:  c.config.PoolTimeout,

		MaxRetries:      c.config.MaxRetries,
		MinRetryBackoff: c.config.MinRetryBackoff,
		MaxRetryBackoff: c.config.MaxRetryBackoff,

		PoolFIFO:        c.config.PoolFIFO,
		ConnMaxIdleTime: c.config.ConnMaxIdleTime,
		ConnMaxLifetime: c.config.ConnMaxLifetime,
	}

	// 配置TLS
	if c.config.EnableTLS {
		tlsConfig, err := c.loadTLSConfig()
		if err != nil {
			return err
		}
		opts.TLSConfig = tlsConfig
	}

	c.master = redis.NewClient(opts)
	return nil
}

// initSentinelMode 初始化哨兵模式
func (c *Client) initSentinelMode() error {
	opts := &redis.FailoverOptions{
		MasterName:    c.config.MasterName,
		SentinelAddrs: c.config.SentinelAddrs,
		Username:      c.config.Username,
		Password:      c.config.Password,
		DB:            c.config.DB,

		PoolSize:     c.config.PoolSize,
		MinIdleConns: c.config.MinIdleConns,

		DialTimeout:  c.config.DialTimeout,
		ReadTimeout:  c.config.ReadTimeout,
		WriteTimeout: c.config.WriteTimeout,
		PoolTimeout:  c.config.PoolTimeout,

		MaxRetries:      c.config.MaxRetries,
		MinRetryBackoff: c.config.MinRetryBackoff,
		MaxRetryBackoff: c.config.MaxRetryBackoff,

		PoolFIFO:        c.config.PoolFIFO,
		ConnMaxIdleTime: c.config.ConnMaxIdleTime,
		ConnMaxLifetime: c.config.ConnMaxLifetime,

		// 哨兵模式读写分离配置
		RouteByLatency: c.config.RouteByLatency,
		RouteRandomly:  c.config.RouteRandomly,
		ReplicaOnly:    c.config.SlaveReadOnly,
	}

	// 配置TLS
	if c.config.EnableTLS {
		tlsConfig, err := c.loadTLSConfig()
		if err != nil {
			return err
		}
		opts.TLSConfig = tlsConfig
	}

	c.master = redis.NewFailoverClient(opts)
	return nil
}

// initClusterMode 初始化集群模式
func (c *Client) initClusterMode() error {
	opts := &redis.ClusterOptions{
		Addrs:    c.config.ClusterAddrs,
		Username: c.config.Username,
		Password: c.config.Password,

		PoolSize:     c.config.PoolSize,
		MinIdleConns: c.config.MinIdleConns,

		DialTimeout:  c.config.DialTimeout,
		ReadTimeout:  c.config.ReadTimeout,
		WriteTimeout: c.config.WriteTimeout,
		PoolTimeout:  c.config.PoolTimeout,

		MaxRetries:      c.config.MaxRetries,
		MinRetryBackoff: c.config.MinRetryBackoff,
		MaxRetryBackoff: c.config.MaxRetryBackoff,

		PoolFIFO:        c.config.PoolFIFO,
		ConnMaxIdleTime: c.config.ConnMaxIdleTime,
		ConnMaxLifetime: c.config.ConnMaxLifetime,

		RouteByLatency: c.config.RouteByLatency,
		RouteRandomly:  c.config.RouteRandomly,
	}

	// 配置TLS
	if c.config.EnableTLS {
		tlsConfig, err := c.loadTLSConfig()
		if err != nil {
			return err
		}
		opts.TLSConfig = tlsConfig
	}

	c.cluster = redis.NewClusterClient(opts)
	c.master = c.cluster
	return nil
}

// initReadWriteMode 初始化主从读写分离模式
func (c *Client) initReadWriteMode() error {
	// 创建主节点客户端
	masterOpts := &redis.Options{
		Addr:     c.config.MasterAddr,
		Username: c.config.Username,
		Password: c.config.Password,
		DB:       c.config.DB,

		PoolSize:     c.config.PoolSize,
		MinIdleConns: c.config.MinIdleConns,

		DialTimeout:  c.config.DialTimeout,
		ReadTimeout:  c.config.ReadTimeout,
		WriteTimeout: c.config.WriteTimeout,
		PoolTimeout:  c.config.PoolTimeout,

		MaxRetries:      c.config.MaxRetries,
		MinRetryBackoff: c.config.MinRetryBackoff,
		MaxRetryBackoff: c.config.MaxRetryBackoff,

		PoolFIFO:        c.config.PoolFIFO,
		ConnMaxIdleTime: c.config.ConnMaxIdleTime,
		ConnMaxLifetime: c.config.ConnMaxLifetime,
	}

	// 配置TLS
	var tlsConfig *tls.Config
	if c.config.EnableTLS {
		var err error
		tlsConfig, err = c.loadTLSConfig()
		if err != nil {
			return err
		}
		masterOpts.TLSConfig = tlsConfig
	}

	c.master = redis.NewClient(masterOpts)

	// 创建从节点客户端
	c.slaves = make([]redis.UniversalClient, len(c.config.SlaveAddrs))
	for i, addr := range c.config.SlaveAddrs {
		slaveOpts := &redis.Options{
			Addr:     addr,
			Username: c.config.Username,
			Password: c.config.Password,
			DB:       c.config.DB,

			PoolSize:     c.config.PoolSize,
			MinIdleConns: c.config.MinIdleConns,

			DialTimeout:  c.config.DialTimeout,
			ReadTimeout:  c.config.ReadTimeout,
			WriteTimeout: c.config.WriteTimeout,
			PoolTimeout:  c.config.PoolTimeout,

			MaxRetries:      c.config.MaxRetries,
			MinRetryBackoff: c.config.MinRetryBackoff,
			MaxRetryBackoff: c.config.MaxRetryBackoff,

			PoolFIFO:        c.config.PoolFIFO,
			ConnMaxIdleTime: c.config.ConnMaxIdleTime,
			ConnMaxLifetime: c.config.ConnMaxLifetime,
		}

		if c.config.EnableTLS {
			slaveOpts.TLSConfig = tlsConfig
		}

		c.slaves[i] = redis.NewClient(slaveOpts)

		c.logger.Info("slave client initialized",
			zap.String("addr", addr),
			zap.Int("index", i),
		)
	}

	return nil
}

// loadTLSConfig 加载TLS配置
func (c *Client) loadTLSConfig() (*tls.Config, error) {
	tlsConfig := &tls.Config{
		InsecureSkipVerify: c.config.TLSSkipVerify,
		ServerName:         c.config.TLSServerName,
	}

	// 加载客户端证书
	if c.config.TLSCertFile != "" && c.config.TLSKeyFile != "" {
		cert, err := tls.LoadX509KeyPair(c.config.TLSCertFile, c.config.TLSKeyFile)
		if err != nil {
			return nil, fmt.Errorf("load client cert failed: %w", err)
		}
		tlsConfig.Certificates = []tls.Certificate{cert}
	}

	// 加载CA证书
	if c.config.TLSCAFile != "" {
		caCert, err := os.ReadFile(c.config.TLSCAFile)
		if err != nil {
			return nil, fmt.Errorf("read CA file failed: %w", err)
		}

		caCertPool := x509.NewCertPool()
		if !caCertPool.AppendCertsFromPEM(caCert) {
			return nil, fmt.Errorf("append CA cert failed")
		}
		tlsConfig.RootCAs = caCertPool
	}

	return tlsConfig, nil
}

// getReadClient 根据读取策略获取客户端
func (c *Client) getReadClient() redis.UniversalClient {
	// 单机/哨兵/集群模式直接返回主客户端
	if c.config.Mode != ModeReadWrite {
		return c.master
	}

	// 没有从节点，返回主节点
	if len(c.slaves) == 0 {
		return c.master
	}

	switch c.readStrategy {
	case ReadFromMaster:
		return c.master

	case ReadFromSlave:
		return c.selectSlave()

	case ReadFromSlaveFirst:
		// 随机选择一个从节点，失败时会自动重试其他从节点
		return c.selectSlave()

	case ReadRandom:
		// 随机选择主节点或从节点
		all := append([]redis.UniversalClient{c.master}, c.slaves...)
		return all[rand.Intn(len(all))]

	case ReadRoundRobin:
		// 轮询所有节点（主+从）
		all := append([]redis.UniversalClient{c.master}, c.slaves...)
		idx := int(c.slaveIndex.Add(1)) % len(all)
		return all[idx]

	default:
		return c.master
	}
}

// selectSlave 选择从节点（轮询）
func (c *Client) selectSlave() redis.UniversalClient {
	if len(c.slaves) == 0 {
		return c.master
	}

	idx := int(c.slaveIndex.Add(1)) % len(c.slaves)
	return c.slaves[idx]
}

// Ping 健康检查
func (c *Client) Ping(ctx context.Context) error {
	if c.master == nil {
		return fmt.Errorf("redis client not initialized")
	}

	if err := c.master.Ping(ctx).Err(); err != nil {
		c.logger.Error("redis master ping failed", zap.Error(err))
		return err
	}

	// 检查从节点
	for i, slave := range c.slaves {
		if err := slave.Ping(ctx).Err(); err != nil {
			c.logger.Warn("redis slave ping failed",
				zap.Int("index", i),
				zap.Error(err),
			)
			// 从节点失败不影响整体健康检查
		}
	}

	return nil
}

// Close 关闭客户端
func (c *Client) Close() error {
	if c.master != nil {
		if err := c.master.Close(); err != nil {
			c.logger.Error("close master client failed", zap.Error(err))
			return err
		}
	}

	for i, slave := range c.slaves {
		if err := slave.Close(); err != nil {
			c.logger.Error("close slave client failed",
				zap.Int("index", i),
				zap.Error(err),
			)
		}
	}

	c.logger.Info("redis client closed")
	return nil
}

// GetMasterClient 获取主节点客户端（用于高级操作）
func (c *Client) GetMasterClient() redis.UniversalClient {
	return c.master
}

// GetSlaveClients 获取所有从节点客户端
func (c *Client) GetSlaveClients() []redis.UniversalClient {
	return c.slaves
}
