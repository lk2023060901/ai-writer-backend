package redis

import (
	"errors"
	"time"
)

// DeployMode Redis 部署模式
type DeployMode string

const (
	ModeSingle    DeployMode = "single"     // 单机模式
	ModeSentinel  DeployMode = "sentinel"   // 哨兵模式
	ModeCluster   DeployMode = "cluster"    // 集群模式
	ModeReadWrite DeployMode = "read-write" // 主从读写分离模式
)

// ReadStrategy 读取策略
type ReadStrategy string

const (
	ReadFromMaster     ReadStrategy = "master"      // 只从主节点读
	ReadFromSlave      ReadStrategy = "slave"       // 只从从节点读
	ReadFromSlaveFirst ReadStrategy = "slave-first" // 优先从节点，失败回退主节点
	ReadRandom         ReadStrategy = "random"      // 随机读（主+从）
	ReadRoundRobin     ReadStrategy = "round-robin" // 轮询读（主+从）
)

// Config Redis 配置
type Config struct {
	// 部署模式
	Mode DeployMode `mapstructure:"mode" yaml:"mode"`

	// 单机/主从模式配置
	MasterAddr string   `mapstructure:"master_addr" yaml:"master_addr"` // 主节点地址 (host:port)
	SlaveAddrs []string `mapstructure:"slave_addrs" yaml:"slave_addrs"` // 从节点地址列表

	// 哨兵模式配置
	SentinelAddrs  []string `mapstructure:"sentinel_addrs" yaml:"sentinel_addrs"`   // 哨兵地址列表
	MasterName     string   `mapstructure:"master_name" yaml:"master_name"`         // 主节点名称
	RouteByLatency bool     `mapstructure:"route_by_latency" yaml:"route_by_latency"` // 按延迟路由读请求
	RouteRandomly  bool     `mapstructure:"route_randomly" yaml:"route_randomly"`   // 随机路由读请求

	// 集群模式配置
	ClusterAddrs []string `mapstructure:"cluster_addrs" yaml:"cluster_addrs"` // 集群节点地址列表

	// 读写分离配置
	ReadStrategy  ReadStrategy `mapstructure:"read_strategy" yaml:"read_strategy"`   // 读取策略
	SlaveReadOnly bool         `mapstructure:"slave_read_only" yaml:"slave_read_only"` // 从节点只读

	// 认证配置
	Password string `mapstructure:"password" yaml:"password"` // 密码
	Username string `mapstructure:"username" yaml:"username"` // 用户名（Redis 6.0+）
	DB       int    `mapstructure:"db" yaml:"db"`             // 数据库编号

	// 连接池配置
	PoolSize     int `mapstructure:"pool_size" yaml:"pool_size"`         // 连接池大小
	MinIdleConns int `mapstructure:"min_idle_conns" yaml:"min_idle_conns"` // 最小空闲连接数

	// 超时配置
	DialTimeout  time.Duration `mapstructure:"dial_timeout" yaml:"dial_timeout"`   // 连接超时
	ReadTimeout  time.Duration `mapstructure:"read_timeout" yaml:"read_timeout"`   // 读超时
	WriteTimeout time.Duration `mapstructure:"write_timeout" yaml:"write_timeout"` // 写超时
	PoolTimeout  time.Duration `mapstructure:"pool_timeout" yaml:"pool_timeout"`   // 连接池超时

	// 重试配置
	MaxRetries      int           `mapstructure:"max_retries" yaml:"max_retries"`           // 最大重试次数
	MinRetryBackoff time.Duration `mapstructure:"min_retry_backoff" yaml:"min_retry_backoff"` // 最小重试间隔
	MaxRetryBackoff time.Duration `mapstructure:"max_retry_backoff" yaml:"max_retry_backoff"` // 最大重试间隔

	// 连接配置
	MaxConnAge         time.Duration `mapstructure:"max_conn_age" yaml:"max_conn_age"`             // 连接最大存活时间
	PoolFIFO           bool          `mapstructure:"pool_fifo" yaml:"pool_fifo"`                   // 连接池FIFO模式
	ConnMaxIdleTime    time.Duration `mapstructure:"conn_max_idle_time" yaml:"conn_max_idle_time"` // 连接最大空闲时间
	ConnMaxLifetime    time.Duration `mapstructure:"conn_max_lifetime" yaml:"conn_max_lifetime"`   // 连接最大生命周期

	// TLS配置
	EnableTLS        bool   `mapstructure:"enable_tls" yaml:"enable_tls"`             // 启用TLS
	TLSCertFile      string `mapstructure:"tls_cert_file" yaml:"tls_cert_file"`       // TLS证书文件
	TLSKeyFile       string `mapstructure:"tls_key_file" yaml:"tls_key_file"`         // TLS密钥文件
	TLSCAFile        string `mapstructure:"tls_ca_file" yaml:"tls_ca_file"`           // TLS CA文件
	TLSSkipVerify    bool   `mapstructure:"tls_skip_verify" yaml:"tls_skip_verify"`   // 跳过TLS验证
	TLSServerName    string `mapstructure:"tls_server_name" yaml:"tls_server_name"`   // TLS服务器名称

	// 其他配置
	EnableTracing bool `mapstructure:"enable_tracing" yaml:"enable_tracing"` // 启用链路追踪
	EnableMetrics bool `mapstructure:"enable_metrics" yaml:"enable_metrics"` // 启用指标监控
}

// DefaultConfig 返回默认配置
func DefaultConfig() *Config {
	return &Config{
		Mode:       ModeSingle,
		MasterAddr: "localhost:6379",
		DB:         0,

		PoolSize:     10,
		MinIdleConns: 5,

		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
		PoolTimeout:  4 * time.Second,

		MaxRetries:      3,
		MinRetryBackoff: 8 * time.Millisecond,
		MaxRetryBackoff: 512 * time.Millisecond,

		MaxConnAge:      0, // 0表示不限制
		PoolFIFO:        false,
		ConnMaxIdleTime: 5 * time.Minute,
		ConnMaxLifetime: 0, // 0表示不限制

		ReadStrategy:  ReadFromMaster,
		SlaveReadOnly: true,

		RouteByLatency: false,
		RouteRandomly:  false,

		EnableTLS:     false,
		TLSSkipVerify: false,

		EnableTracing: false,
		EnableMetrics: false,
	}
}

// Validate 验证配置
func (c *Config) Validate() error {
	// 验证部署模式
	switch c.Mode {
	case ModeSingle:
		if c.MasterAddr == "" {
			return errors.New("redis: master_addr is required in single mode")
		}
	case ModeSentinel:
		if len(c.SentinelAddrs) == 0 {
			return errors.New("redis: sentinel_addrs is required in sentinel mode")
		}
		if c.MasterName == "" {
			return errors.New("redis: master_name is required in sentinel mode")
		}
	case ModeCluster:
		if len(c.ClusterAddrs) == 0 {
			return errors.New("redis: cluster_addrs is required in cluster mode")
		}
	case ModeReadWrite:
		if c.MasterAddr == "" {
			return errors.New("redis: master_addr is required in read-write mode")
		}
		if len(c.SlaveAddrs) == 0 {
			return errors.New("redis: slave_addrs is required in read-write mode")
		}
	default:
		return errors.New("redis: invalid mode, must be one of: single, sentinel, cluster, read-write")
	}

	// 验证数据库编号
	if c.DB < 0 || c.DB > 15 {
		return errors.New("redis: db must be between 0 and 15")
	}

	// 验证连接池配置
	if c.PoolSize <= 0 {
		return errors.New("redis: pool_size must be > 0")
	}
	if c.MinIdleConns < 0 {
		return errors.New("redis: min_idle_conns must be >= 0")
	}
	if c.MinIdleConns > c.PoolSize {
		return errors.New("redis: min_idle_conns cannot exceed pool_size")
	}

	// 验证超时配置
	if c.DialTimeout <= 0 {
		return errors.New("redis: dial_timeout must be > 0")
	}
	if c.ReadTimeout < 0 {
		return errors.New("redis: read_timeout must be >= 0")
	}
	if c.WriteTimeout < 0 {
		return errors.New("redis: write_timeout must be >= 0")
	}
	if c.PoolTimeout <= 0 {
		return errors.New("redis: pool_timeout must be > 0")
	}

	// 验证重试配置
	if c.MaxRetries < 0 {
		return errors.New("redis: max_retries must be >= 0")
	}
	if c.MinRetryBackoff < 0 {
		return errors.New("redis: min_retry_backoff must be >= 0")
	}
	if c.MaxRetryBackoff < 0 {
		return errors.New("redis: max_retry_backoff must be >= 0")
	}
	if c.MinRetryBackoff > c.MaxRetryBackoff {
		return errors.New("redis: min_retry_backoff cannot exceed max_retry_backoff")
	}

	// 验证读取策略
	if c.Mode == ModeReadWrite {
		switch c.ReadStrategy {
		case ReadFromMaster, ReadFromSlave, ReadFromSlaveFirst, ReadRandom, ReadRoundRobin:
			// 有效策略
		default:
			return errors.New("redis: invalid read_strategy, must be one of: master, slave, slave-first, random, round-robin")
		}
	}

	// 验证TLS配置
	if c.EnableTLS {
		if c.TLSCertFile == "" && c.TLSKeyFile == "" && c.TLSCAFile == "" && !c.TLSSkipVerify {
			return errors.New("redis: TLS enabled but no certificate files provided and TLS verification not skipped")
		}
	}

	return nil
}
