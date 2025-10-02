package redis

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

// ==================== String Operations ====================

// Set 设置键值（支持过期时间）
func (c *Client) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	err := c.master.Set(ctx, key, value, expiration).Err()
	if err != nil {
		c.logger.Error("redis set failed",
			zap.String("key", key),
			zap.Error(err),
		)
	}
	return err
}

// Get 获取键值
func (c *Client) Get(ctx context.Context, key string) (string, error) {
	client := c.getReadClient()
	val, err := client.Get(ctx, key).Result()
	if err != nil && !IsNil(err) {
		c.logger.Error("redis get failed",
			zap.String("key", key),
			zap.Error(err),
		)
	}
	return val, err
}

// Del 删除键
func (c *Client) Del(ctx context.Context, keys ...string) (int64, error) {
	n, err := c.master.Del(ctx, keys...).Result()
	if err != nil {
		c.logger.Error("redis del failed",
			zap.Strings("keys", keys),
			zap.Error(err),
		)
	}
	return n, err
}

// Exists 检查键是否存在
func (c *Client) Exists(ctx context.Context, keys ...string) (int64, error) {
	client := c.getReadClient()
	n, err := client.Exists(ctx, keys...).Result()
	if err != nil {
		c.logger.Error("redis exists failed",
			zap.Strings("keys", keys),
			zap.Error(err),
		)
	}
	return n, err
}

// Expire 设置过期时间
func (c *Client) Expire(ctx context.Context, key string, expiration time.Duration) (bool, error) {
	ok, err := c.master.Expire(ctx, key, expiration).Result()
	if err != nil {
		c.logger.Error("redis expire failed",
			zap.String("key", key),
			zap.Duration("expiration", expiration),
			zap.Error(err),
		)
	}
	return ok, err
}

// TTL 获取剩余过期时间
func (c *Client) TTL(ctx context.Context, key string) (time.Duration, error) {
	client := c.getReadClient()
	ttl, err := client.TTL(ctx, key).Result()
	if err != nil {
		c.logger.Error("redis ttl failed",
			zap.String("key", key),
			zap.Error(err),
		)
	}
	return ttl, err
}

// Incr 自增
func (c *Client) Incr(ctx context.Context, key string) (int64, error) {
	val, err := c.master.Incr(ctx, key).Result()
	if err != nil {
		c.logger.Error("redis incr failed",
			zap.String("key", key),
			zap.Error(err),
		)
	}
	return val, err
}

// IncrBy 按指定值自增
func (c *Client) IncrBy(ctx context.Context, key string, value int64) (int64, error) {
	val, err := c.master.IncrBy(ctx, key, value).Result()
	if err != nil {
		c.logger.Error("redis incrby failed",
			zap.String("key", key),
			zap.Int64("value", value),
			zap.Error(err),
		)
	}
	return val, err
}

// Decr 自减
func (c *Client) Decr(ctx context.Context, key string) (int64, error) {
	val, err := c.master.Decr(ctx, key).Result()
	if err != nil {
		c.logger.Error("redis decr failed",
			zap.String("key", key),
			zap.Error(err),
		)
	}
	return val, err
}

// DecrBy 按指定值自减
func (c *Client) DecrBy(ctx context.Context, key string, value int64) (int64, error) {
	val, err := c.master.DecrBy(ctx, key, value).Result()
	if err != nil {
		c.logger.Error("redis decrby failed",
			zap.String("key", key),
			zap.Int64("value", value),
			zap.Error(err),
		)
	}
	return val, err
}

// SetNX 仅当键不存在时设置
func (c *Client) SetNX(ctx context.Context, key string, value interface{}, expiration time.Duration) (bool, error) {
	ok, err := c.master.SetNX(ctx, key, value, expiration).Result()
	if err != nil {
		c.logger.Error("redis setnx failed",
			zap.String("key", key),
			zap.Error(err),
		)
	}
	return ok, err
}

// ==================== Hash Operations ====================

// HSet 设置哈希字段
func (c *Client) HSet(ctx context.Context, key string, values ...interface{}) (int64, error) {
	n, err := c.master.HSet(ctx, key, values...).Result()
	if err != nil {
		c.logger.Error("redis hset failed",
			zap.String("key", key),
			zap.Error(err),
		)
	}
	return n, err
}

// HGet 获取哈希字段值
func (c *Client) HGet(ctx context.Context, key, field string) (string, error) {
	client := c.getReadClient()
	val, err := client.HGet(ctx, key, field).Result()
	if err != nil && !IsNil(err) {
		c.logger.Error("redis hget failed",
			zap.String("key", key),
			zap.String("field", field),
			zap.Error(err),
		)
	}
	return val, err
}

// HGetAll 获取哈希所有字段
func (c *Client) HGetAll(ctx context.Context, key string) (map[string]string, error) {
	client := c.getReadClient()
	vals, err := client.HGetAll(ctx, key).Result()
	if err != nil {
		c.logger.Error("redis hgetall failed",
			zap.String("key", key),
			zap.Error(err),
		)
	}
	return vals, err
}

// HDel 删除哈希字段
func (c *Client) HDel(ctx context.Context, key string, fields ...string) (int64, error) {
	n, err := c.master.HDel(ctx, key, fields...).Result()
	if err != nil {
		c.logger.Error("redis hdel failed",
			zap.String("key", key),
			zap.Strings("fields", fields),
			zap.Error(err),
		)
	}
	return n, err
}

// HExists 检查哈希字段是否存在
func (c *Client) HExists(ctx context.Context, key, field string) (bool, error) {
	client := c.getReadClient()
	ok, err := client.HExists(ctx, key, field).Result()
	if err != nil {
		c.logger.Error("redis hexists failed",
			zap.String("key", key),
			zap.String("field", field),
			zap.Error(err),
		)
	}
	return ok, err
}

// HLen 获取哈希字段数量
func (c *Client) HLen(ctx context.Context, key string) (int64, error) {
	client := c.getReadClient()
	n, err := client.HLen(ctx, key).Result()
	if err != nil {
		c.logger.Error("redis hlen failed",
			zap.String("key", key),
			zap.Error(err),
		)
	}
	return n, err
}

// HIncrBy 哈希字段自增
func (c *Client) HIncrBy(ctx context.Context, key, field string, incr int64) (int64, error) {
	val, err := c.master.HIncrBy(ctx, key, field, incr).Result()
	if err != nil {
		c.logger.Error("redis hincrby failed",
			zap.String("key", key),
			zap.String("field", field),
			zap.Int64("incr", incr),
			zap.Error(err),
		)
	}
	return val, err
}

// ==================== List Operations ====================

// LPush 从列表左侧插入元素
func (c *Client) LPush(ctx context.Context, key string, values ...interface{}) (int64, error) {
	n, err := c.master.LPush(ctx, key, values...).Result()
	if err != nil {
		c.logger.Error("redis lpush failed",
			zap.String("key", key),
			zap.Error(err),
		)
	}
	return n, err
}

// RPush 从列表右侧插入元素
func (c *Client) RPush(ctx context.Context, key string, values ...interface{}) (int64, error) {
	n, err := c.master.RPush(ctx, key, values...).Result()
	if err != nil {
		c.logger.Error("redis rpush failed",
			zap.String("key", key),
			zap.Error(err),
		)
	}
	return n, err
}

// LPop 从列表左侧弹出元素
func (c *Client) LPop(ctx context.Context, key string) (string, error) {
	val, err := c.master.LPop(ctx, key).Result()
	if err != nil && !IsNil(err) {
		c.logger.Error("redis lpop failed",
			zap.String("key", key),
			zap.Error(err),
		)
	}
	return val, err
}

// RPop 从列表右侧弹出元素
func (c *Client) RPop(ctx context.Context, key string) (string, error) {
	val, err := c.master.RPop(ctx, key).Result()
	if err != nil && !IsNil(err) {
		c.logger.Error("redis rpop failed",
			zap.String("key", key),
			zap.Error(err),
		)
	}
	return val, err
}

// LLen 获取列表长度
func (c *Client) LLen(ctx context.Context, key string) (int64, error) {
	client := c.getReadClient()
	n, err := client.LLen(ctx, key).Result()
	if err != nil {
		c.logger.Error("redis llen failed",
			zap.String("key", key),
			zap.Error(err),
		)
	}
	return n, err
}

// LRange 获取列表范围内的元素
func (c *Client) LRange(ctx context.Context, key string, start, stop int64) ([]string, error) {
	client := c.getReadClient()
	vals, err := client.LRange(ctx, key, start, stop).Result()
	if err != nil {
		c.logger.Error("redis lrange failed",
			zap.String("key", key),
			zap.Int64("start", start),
			zap.Int64("stop", stop),
			zap.Error(err),
		)
	}
	return vals, err
}

// LTrim 修剪列表
func (c *Client) LTrim(ctx context.Context, key string, start, stop int64) error {
	err := c.master.LTrim(ctx, key, start, stop).Err()
	if err != nil {
		c.logger.Error("redis ltrim failed",
			zap.String("key", key),
			zap.Int64("start", start),
			zap.Int64("stop", stop),
			zap.Error(err),
		)
	}
	return err
}

// ==================== Set Operations ====================

// SAdd 添加集合成员
func (c *Client) SAdd(ctx context.Context, key string, members ...interface{}) (int64, error) {
	n, err := c.master.SAdd(ctx, key, members...).Result()
	if err != nil {
		c.logger.Error("redis sadd failed",
			zap.String("key", key),
			zap.Error(err),
		)
	}
	return n, err
}

// SRem 删除集合成员
func (c *Client) SRem(ctx context.Context, key string, members ...interface{}) (int64, error) {
	n, err := c.master.SRem(ctx, key, members...).Result()
	if err != nil {
		c.logger.Error("redis srem failed",
			zap.String("key", key),
			zap.Error(err),
		)
	}
	return n, err
}

// SMembers 获取集合所有成员
func (c *Client) SMembers(ctx context.Context, key string) ([]string, error) {
	client := c.getReadClient()
	members, err := client.SMembers(ctx, key).Result()
	if err != nil {
		c.logger.Error("redis smembers failed",
			zap.String("key", key),
			zap.Error(err),
		)
	}
	return members, err
}

// SIsMember 检查是否是集合成员
func (c *Client) SIsMember(ctx context.Context, key string, member interface{}) (bool, error) {
	client := c.getReadClient()
	ok, err := client.SIsMember(ctx, key, member).Result()
	if err != nil {
		c.logger.Error("redis sismember failed",
			zap.String("key", key),
			zap.Error(err),
		)
	}
	return ok, err
}

// SCard 获取集合成员数量
func (c *Client) SCard(ctx context.Context, key string) (int64, error) {
	client := c.getReadClient()
	n, err := client.SCard(ctx, key).Result()
	if err != nil {
		c.logger.Error("redis scard failed",
			zap.String("key", key),
			zap.Error(err),
		)
	}
	return n, err
}

// ==================== Sorted Set Operations ====================

// ZAdd 添加有序集合成员
func (c *Client) ZAdd(ctx context.Context, key string, members ...redis.Z) (int64, error) {
	n, err := c.master.ZAdd(ctx, key, members...).Result()
	if err != nil {
		c.logger.Error("redis zadd failed",
			zap.String("key", key),
			zap.Error(err),
		)
	}
	return n, err
}

// ZRem 删除有序集合成员
func (c *Client) ZRem(ctx context.Context, key string, members ...interface{}) (int64, error) {
	n, err := c.master.ZRem(ctx, key, members...).Result()
	if err != nil {
		c.logger.Error("redis zrem failed",
			zap.String("key", key),
			zap.Error(err),
		)
	}
	return n, err
}

// ZRange 获取有序集合范围内的成员（按分数从小到大）
func (c *Client) ZRange(ctx context.Context, key string, start, stop int64) ([]string, error) {
	client := c.getReadClient()
	members, err := client.ZRange(ctx, key, start, stop).Result()
	if err != nil {
		c.logger.Error("redis zrange failed",
			zap.String("key", key),
			zap.Int64("start", start),
			zap.Int64("stop", stop),
			zap.Error(err),
		)
	}
	return members, err
}

// ZRevRange 获取有序集合范围内的成员（按分数从大到小）
func (c *Client) ZRevRange(ctx context.Context, key string, start, stop int64) ([]string, error) {
	client := c.getReadClient()
	members, err := client.ZRevRange(ctx, key, start, stop).Result()
	if err != nil {
		c.logger.Error("redis zrevrange failed",
			zap.String("key", key),
			zap.Int64("start", start),
			zap.Int64("stop", stop),
			zap.Error(err),
		)
	}
	return members, err
}

// ZRangeWithScores 获取有序集合范围内的成员及分数
func (c *Client) ZRangeWithScores(ctx context.Context, key string, start, stop int64) ([]redis.Z, error) {
	client := c.getReadClient()
	members, err := client.ZRangeWithScores(ctx, key, start, stop).Result()
	if err != nil {
		c.logger.Error("redis zrange with scores failed",
			zap.String("key", key),
			zap.Int64("start", start),
			zap.Int64("stop", stop),
			zap.Error(err),
		)
	}
	return members, err
}

// ZScore 获取有序集合成员分数
func (c *Client) ZScore(ctx context.Context, key, member string) (float64, error) {
	client := c.getReadClient()
	score, err := client.ZScore(ctx, key, member).Result()
	if err != nil && !IsNil(err) {
		c.logger.Error("redis zscore failed",
			zap.String("key", key),
			zap.String("member", member),
			zap.Error(err),
		)
	}
	return score, err
}

// ZRank 获取有序集合成员排名（从小到大）
func (c *Client) ZRank(ctx context.Context, key, member string) (int64, error) {
	client := c.getReadClient()
	rank, err := client.ZRank(ctx, key, member).Result()
	if err != nil && !IsNil(err) {
		c.logger.Error("redis zrank failed",
			zap.String("key", key),
			zap.String("member", member),
			zap.Error(err),
		)
	}
	return rank, err
}

// ZRevRank 获取有序集合成员排名（从大到小）
func (c *Client) ZRevRank(ctx context.Context, key, member string) (int64, error) {
	client := c.getReadClient()
	rank, err := client.ZRevRank(ctx, key, member).Result()
	if err != nil && !IsNil(err) {
		c.logger.Error("redis zrevrank failed",
			zap.String("key", key),
			zap.String("member", member),
			zap.Error(err),
		)
	}
	return rank, err
}

// ZCard 获取有序集合成员数量
func (c *Client) ZCard(ctx context.Context, key string) (int64, error) {
	client := c.getReadClient()
	n, err := client.ZCard(ctx, key).Result()
	if err != nil {
		c.logger.Error("redis zcard failed",
			zap.String("key", key),
			zap.Error(err),
		)
	}
	return n, err
}

// ZIncrBy 有序集合成员分数自增
func (c *Client) ZIncrBy(ctx context.Context, key string, increment float64, member string) (float64, error) {
	score, err := c.master.ZIncrBy(ctx, key, increment, member).Result()
	if err != nil {
		c.logger.Error("redis zincrby failed",
			zap.String("key", key),
			zap.String("member", member),
			zap.Float64("increment", increment),
			zap.Error(err),
		)
	}
	return score, err
}
