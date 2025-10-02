package redis

import (
	"errors"

	"github.com/redis/go-redis/v9"
)

// 预定义错误
var (
	ErrNil            = redis.Nil // Key 不存在
	ErrClosed         = errors.New("redis: client is closed")
	ErrPoolTimeout    = errors.New("redis: connection pool timeout")
	ErrInvalidConfig  = errors.New("redis: invalid configuration")
	ErrNotInitialized = errors.New("redis: client not initialized")
)

// IsNil 判断是否是 Key 不存在错误
func IsNil(err error) bool {
	return errors.Is(err, redis.Nil)
}

// IsClosed 判断是否是客户端已关闭错误
func IsClosed(err error) bool {
	return errors.Is(err, redis.ErrClosed) || errors.Is(err, ErrClosed)
}

// IsPoolTimeout 判断是否是连接池超时错误
func IsPoolTimeout(err error) bool {
	return errors.Is(err, ErrPoolTimeout)
}
