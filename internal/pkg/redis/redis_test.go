package redis

import (
	"context"
	"testing"
	"time"

	"github.com/lk2023060901/ai-writer-backend/internal/pkg/logger"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testRedisAddr = "localhost:6379"
)

func setupTestClient(t *testing.T) *Client {
	log, err := logger.New(&logger.Config{
		Level:  "debug",
		Format: "json",
		Output: "console",
	})
	require.NoError(t, err)

	cfg := &Config{
		Mode:         ModeSingle,
		MasterAddr:   testRedisAddr,
		DB:           0,
		PoolSize:     10,
		MinIdleConns: 5,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
		PoolTimeout:  4 * time.Second,
	}

	client, err := New(cfg, log)
	require.NoError(t, err)
	require.NotNil(t, client)

	return client
}

func TestNew(t *testing.T) {
	log, err := logger.New(&logger.Config{
		Level:  "info",
		Format: "json",
		Output: "console",
	})
	require.NoError(t, err)

	tests := []struct {
		name    string
		config  *Config
		wantErr bool
	}{
		{
			name: "valid config",
			config: &Config{
				Mode:        ModeSingle,
				MasterAddr:  testRedisAddr,
				PoolSize:    10,
				DialTimeout: 5 * time.Second,
				PoolTimeout: 4 * time.Second,
			},
			wantErr: false,
		},
		{
			name: "missing master addr",
			config: &Config{
				Mode:     ModeSingle,
				PoolSize: 10,
			},
			wantErr: true,
		},
		{
			name: "invalid pool size",
			config: &Config{
				Mode:        ModeSingle,
				MasterAddr:  testRedisAddr,
				PoolSize:    0,
				DialTimeout: 5 * time.Second,
			},
			wantErr: true,
		},
		{
			name: "invalid db number",
			config: &Config{
				Mode:        ModeSingle,
				MasterAddr:  testRedisAddr,
				DB:          16,
				PoolSize:    10,
				DialTimeout: 5 * time.Second,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := New(tt.config, log)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, client)
			} else {
				require.NoError(t, err)
				require.NotNil(t, client)
				defer client.Close()

				// 验证连接
				ctx := context.Background()
				err = client.Ping(ctx)
				assert.NoError(t, err)

				t.Logf("=== Redis Client Info ===")
				t.Logf("Mode: %s", tt.config.Mode)
				t.Logf("Address: %s", tt.config.MasterAddr)
				t.Logf("DB: %d", tt.config.DB)
				t.Logf("========================")
			}
		})
	}
}

func TestPing(t *testing.T) {
	client := setupTestClient(t)
	defer client.Close()

	ctx := context.Background()
	err := client.Ping(ctx)
	assert.NoError(t, err)

	t.Logf("✓ Ping successful")
}

func TestClose(t *testing.T) {
	client := setupTestClient(t)

	err := client.Close()
	assert.NoError(t, err)

	t.Logf("✓ Client closed successfully")
}

func TestStringOperations(t *testing.T) {
	client := setupTestClient(t)
	defer client.Close()

	ctx := context.Background()

	t.Run("Set and Get", func(t *testing.T) {
		key := "test:string:set_get"
		value := "hello redis"
		defer client.Del(ctx, key)

		err := client.Set(ctx, key, value, 0)
		require.NoError(t, err)

		result, err := client.Get(ctx, key)
		require.NoError(t, err)
		assert.Equal(t, value, result)

		t.Logf("✓ Set and Get: %s = %s", key, result)
	})

	t.Run("Get non-existent key", func(t *testing.T) {
		result, err := client.Get(ctx, "test:nonexistent")
		assert.Error(t, err)
		assert.True(t, IsNil(err))
		assert.Empty(t, result)

		t.Logf("✓ Get non-existent key returns ErrNil")
	})

	t.Run("Set with expiration", func(t *testing.T) {
		key := "test:string:expire"
		defer client.Del(ctx, key)

		err := client.Set(ctx, key, "expiring value", 2*time.Second)
		require.NoError(t, err)

		ttl, err := client.TTL(ctx, key)
		require.NoError(t, err)
		assert.Greater(t, ttl, 0*time.Second)
		assert.LessOrEqual(t, ttl, 2*time.Second)

		t.Logf("✓ Key will expire in: %v", ttl)
	})

	t.Run("SetNX", func(t *testing.T) {
		key := "test:string:setnx"
		defer client.Del(ctx, key)

		// 第一次设置应该成功
		ok, err := client.SetNX(ctx, key, "first", 0)
		require.NoError(t, err)
		assert.True(t, ok)

		// 第二次设置应该失败（key已存在）
		ok, err = client.SetNX(ctx, key, "second", 0)
		require.NoError(t, err)
		assert.False(t, ok)

		// 验证值没有被覆盖
		result, err := client.Get(ctx, key)
		require.NoError(t, err)
		assert.Equal(t, "first", result)

		t.Logf("✓ SetNX prevents overwriting existing key")
	})

	t.Run("Incr and Decr", func(t *testing.T) {
		key := "test:string:counter"
		defer client.Del(ctx, key)

		// Incr
		val, err := client.Incr(ctx, key)
		require.NoError(t, err)
		assert.Equal(t, int64(1), val)

		// IncrBy
		val, err = client.IncrBy(ctx, key, 10)
		require.NoError(t, err)
		assert.Equal(t, int64(11), val)

		// Decr
		val, err = client.Decr(ctx, key)
		require.NoError(t, err)
		assert.Equal(t, int64(10), val)

		// DecrBy
		val, err = client.DecrBy(ctx, key, 5)
		require.NoError(t, err)
		assert.Equal(t, int64(5), val)

		t.Logf("✓ Counter operations: final value = %d", val)
	})

	t.Run("Del and Exists", func(t *testing.T) {
		key := "test:string:del"

		err := client.Set(ctx, key, "to be deleted", 0)
		require.NoError(t, err)

		// 检查存在
		n, err := client.Exists(ctx, key)
		require.NoError(t, err)
		assert.Equal(t, int64(1), n)

		// 删除
		n, err = client.Del(ctx, key)
		require.NoError(t, err)
		assert.Equal(t, int64(1), n)

		// 检查不存在
		n, err = client.Exists(ctx, key)
		require.NoError(t, err)
		assert.Equal(t, int64(0), n)

		t.Logf("✓ Del and Exists work correctly")
	})

	t.Run("Expire and TTL", func(t *testing.T) {
		key := "test:string:ttl"
		defer client.Del(ctx, key)

		err := client.Set(ctx, key, "temporary", 0)
		require.NoError(t, err)

		// 设置过期时间
		ok, err := client.Expire(ctx, key, 10*time.Second)
		require.NoError(t, err)
		assert.True(t, ok)

		// 获取TTL
		ttl, err := client.TTL(ctx, key)
		require.NoError(t, err)
		assert.Greater(t, ttl, 0*time.Second)
		assert.LessOrEqual(t, ttl, 10*time.Second)

		t.Logf("✓ TTL: %v", ttl)
	})
}

func TestHashOperations(t *testing.T) {
	client := setupTestClient(t)
	defer client.Close()

	ctx := context.Background()
	key := "test:hash:user"
	defer client.Del(ctx, key)

	t.Run("HSet and HGet", func(t *testing.T) {
		n, err := client.HSet(ctx, key, "name", "Alice", "age", "30", "city", "Shanghai")
		require.NoError(t, err)
		assert.Equal(t, int64(3), n)

		name, err := client.HGet(ctx, key, "name")
		require.NoError(t, err)
		assert.Equal(t, "Alice", name)

		t.Logf("✓ HSet 3 fields, HGet name = %s", name)
	})

	t.Run("HGetAll", func(t *testing.T) {
		all, err := client.HGetAll(ctx, key)
		require.NoError(t, err)
		assert.Len(t, all, 3)
		assert.Equal(t, "Alice", all["name"])
		assert.Equal(t, "30", all["age"])
		assert.Equal(t, "Shanghai", all["city"])

		t.Logf("✓ HGetAll returned %d fields: %v", len(all), all)
	})

	t.Run("HExists", func(t *testing.T) {
		ok, err := client.HExists(ctx, key, "name")
		require.NoError(t, err)
		assert.True(t, ok)

		ok, err = client.HExists(ctx, key, "nonexistent")
		require.NoError(t, err)
		assert.False(t, ok)

		t.Logf("✓ HExists checks field existence correctly")
	})

	t.Run("HLen", func(t *testing.T) {
		n, err := client.HLen(ctx, key)
		require.NoError(t, err)
		assert.Equal(t, int64(3), n)

		t.Logf("✓ Hash has %d fields", n)
	})

	t.Run("HIncrBy", func(t *testing.T) {
		val, err := client.HIncrBy(ctx, key, "score", 100)
		require.NoError(t, err)
		assert.Equal(t, int64(100), val)

		val, err = client.HIncrBy(ctx, key, "score", 50)
		require.NoError(t, err)
		assert.Equal(t, int64(150), val)

		t.Logf("✓ HIncrBy score = %d", val)
	})

	t.Run("HDel", func(t *testing.T) {
		n, err := client.HDel(ctx, key, "age", "city")
		require.NoError(t, err)
		assert.Equal(t, int64(2), n)

		n, err = client.HLen(ctx, key)
		require.NoError(t, err)
		assert.Equal(t, int64(2), n) // name, score 剩余

		t.Logf("✓ HDel removed 2 fields, %d remaining", n)
	})
}

func TestListOperations(t *testing.T) {
	client := setupTestClient(t)
	defer client.Close()

	ctx := context.Background()
	key := "test:list:queue"
	defer client.Del(ctx, key)

	t.Run("LPush and RPush", func(t *testing.T) {
		n, err := client.LPush(ctx, key, "left1", "left2")
		require.NoError(t, err)
		assert.Equal(t, int64(2), n)

		n, err = client.RPush(ctx, key, "right1", "right2")
		require.NoError(t, err)
		assert.Equal(t, int64(4), n)

		t.Logf("✓ List length after pushes: %d", n)
	})

	t.Run("LRange", func(t *testing.T) {
		vals, err := client.LRange(ctx, key, 0, -1)
		require.NoError(t, err)
		assert.Len(t, vals, 4)

		t.Logf("✓ List contents: %v", vals)
	})

	t.Run("LLen", func(t *testing.T) {
		n, err := client.LLen(ctx, key)
		require.NoError(t, err)
		assert.Equal(t, int64(4), n)

		t.Logf("✓ List length: %d", n)
	})

	t.Run("LPop and RPop", func(t *testing.T) {
		leftVal, err := client.LPop(ctx, key)
		require.NoError(t, err)
		assert.NotEmpty(t, leftVal)

		rightVal, err := client.RPop(ctx, key)
		require.NoError(t, err)
		assert.NotEmpty(t, rightVal)

		n, err := client.LLen(ctx, key)
		require.NoError(t, err)
		assert.Equal(t, int64(2), n)

		t.Logf("✓ Popped left='%s', right='%s', remaining: %d", leftVal, rightVal, n)
	})

	t.Run("LTrim", func(t *testing.T) {
		err := client.LTrim(ctx, key, 0, 0)
		require.NoError(t, err)

		n, err := client.LLen(ctx, key)
		require.NoError(t, err)
		assert.Equal(t, int64(1), n)

		t.Logf("✓ List trimmed to 1 element")
	})
}

func TestSetOperations(t *testing.T) {
	client := setupTestClient(t)
	defer client.Close()

	ctx := context.Background()
	key := "test:set:tags"
	defer client.Del(ctx, key)

	t.Run("SAdd", func(t *testing.T) {
		n, err := client.SAdd(ctx, key, "go", "python", "java", "rust")
		require.NoError(t, err)
		assert.Equal(t, int64(4), n)

		t.Logf("✓ Added %d members to set", n)
	})

	t.Run("SMembers", func(t *testing.T) {
		members, err := client.SMembers(ctx, key)
		require.NoError(t, err)
		assert.Len(t, members, 4)

		t.Logf("✓ Set members: %v", members)
	})

	t.Run("SIsMember", func(t *testing.T) {
		ok, err := client.SIsMember(ctx, key, "go")
		require.NoError(t, err)
		assert.True(t, ok)

		ok, err = client.SIsMember(ctx, key, "javascript")
		require.NoError(t, err)
		assert.False(t, ok)

		t.Logf("✓ SIsMember checks membership correctly")
	})

	t.Run("SCard", func(t *testing.T) {
		n, err := client.SCard(ctx, key)
		require.NoError(t, err)
		assert.Equal(t, int64(4), n)

		t.Logf("✓ Set cardinality: %d", n)
	})

	t.Run("SRem", func(t *testing.T) {
		n, err := client.SRem(ctx, key, "java", "rust")
		require.NoError(t, err)
		assert.Equal(t, int64(2), n)

		n, err = client.SCard(ctx, key)
		require.NoError(t, err)
		assert.Equal(t, int64(2), n)

		t.Logf("✓ Removed 2 members, %d remaining", n)
	})
}

func TestZSetOperations(t *testing.T) {
	client := setupTestClient(t)
	defer client.Close()

	ctx := context.Background()
	key := "test:zset:leaderboard"
	defer client.Del(ctx, key)

	t.Run("ZAdd", func(t *testing.T) {
		n, err := client.ZAdd(ctx, key,
			redis.Z{Score: 100, Member: "Alice"},
			redis.Z{Score: 200, Member: "Bob"},
			redis.Z{Score: 150, Member: "Charlie"},
		)
		require.NoError(t, err)
		assert.Equal(t, int64(3), n)

		t.Logf("✓ Added %d members to sorted set", n)
	})

	t.Run("ZRange", func(t *testing.T) {
		members, err := client.ZRange(ctx, key, 0, -1)
		require.NoError(t, err)
		assert.Equal(t, []string{"Alice", "Charlie", "Bob"}, members)

		t.Logf("✓ ZRange (ascending): %v", members)
	})

	t.Run("ZRevRange", func(t *testing.T) {
		members, err := client.ZRevRange(ctx, key, 0, -1)
		require.NoError(t, err)
		assert.Equal(t, []string{"Bob", "Charlie", "Alice"}, members)

		t.Logf("✓ ZRevRange (descending): %v", members)
	})

	t.Run("ZRangeWithScores", func(t *testing.T) {
		members, err := client.ZRangeWithScores(ctx, key, 0, -1)
		require.NoError(t, err)
		assert.Len(t, members, 3)

		t.Logf("✓ ZRangeWithScores:")
		for _, m := range members {
			t.Logf("  - %s: %.0f", m.Member, m.Score)
		}
	})

	t.Run("ZScore", func(t *testing.T) {
		score, err := client.ZScore(ctx, key, "Bob")
		require.NoError(t, err)
		assert.Equal(t, 200.0, score)

		t.Logf("✓ Bob's score: %.0f", score)
	})

	t.Run("ZRank and ZRevRank", func(t *testing.T) {
		rank, err := client.ZRank(ctx, key, "Charlie")
		require.NoError(t, err)
		assert.Equal(t, int64(1), rank)

		revRank, err := client.ZRevRank(ctx, key, "Charlie")
		require.NoError(t, err)
		assert.Equal(t, int64(1), revRank)

		t.Logf("✓ Charlie's rank: %d, reverse rank: %d", rank, revRank)
	})

	t.Run("ZIncrBy", func(t *testing.T) {
		score, err := client.ZIncrBy(ctx, key, 50, "Alice")
		require.NoError(t, err)
		assert.Equal(t, 150.0, score)

		t.Logf("✓ Alice's score after increment: %.0f", score)
	})

	t.Run("ZCard", func(t *testing.T) {
		n, err := client.ZCard(ctx, key)
		require.NoError(t, err)
		assert.Equal(t, int64(3), n)

		t.Logf("✓ Sorted set cardinality: %d", n)
	})

	t.Run("ZRem", func(t *testing.T) {
		n, err := client.ZRem(ctx, key, "Charlie")
		require.NoError(t, err)
		assert.Equal(t, int64(1), n)

		n, err = client.ZCard(ctx, key)
		require.NoError(t, err)
		assert.Equal(t, int64(2), n)

		t.Logf("✓ Removed 1 member, %d remaining", n)
	})
}
