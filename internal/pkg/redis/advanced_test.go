package redis

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPipeline(t *testing.T) {
	client := setupTestClient(t)
	defer client.Close()

	ctx := context.Background()
	keys := []string{"test:pipe:1", "test:pipe:2", "test:pipe:3"}
	defer func() {
		client.Del(ctx, keys...)
	}()

	t.Run("Pipeline batch operations", func(t *testing.T) {
		pipe := client.Pipeline()

		// 批量设置
		for i, key := range keys {
			pipe.Set(ctx, key, fmt.Sprintf("value%d", i+1), 0)
		}

		// 执行Pipeline
		cmds, err := pipe.Exec(ctx)
		require.NoError(t, err)
		assert.Len(t, cmds, 3)

		// 验证结果
		for _, key := range keys {
			val, err := client.Get(ctx, key)
			require.NoError(t, err)
			assert.NotEmpty(t, val)
		}

		t.Logf("✓ Pipeline executed %d commands successfully", len(cmds))
	})

	t.Run("Pipeline with mixed operations", func(t *testing.T) {
		pipe := client.Pipeline()

		setCmd := pipe.Set(ctx, "test:pipe:mixed", "value", 0)
		incrCmd := pipe.Incr(ctx, "test:pipe:counter")
		hsetCmd := pipe.HSet(ctx, "test:pipe:hash", "field", "value")

		_, err := pipe.Exec(ctx)
		require.NoError(t, err)

		// 检查结果
		assert.Equal(t, "OK", setCmd.Val())
		assert.Greater(t, incrCmd.Val(), int64(0))
		assert.Greater(t, hsetCmd.Val(), int64(0))

		// 清理
		client.Del(ctx, "test:pipe:mixed", "test:pipe:counter", "test:pipe:hash")

		t.Logf("✓ Pipeline with mixed operations succeeded")
	})
}

func TestTransaction(t *testing.T) {
	client := setupTestClient(t)
	defer client.Close()

	ctx := context.Background()

	t.Run("Watch and transaction", func(t *testing.T) {
		key := "test:tx:balance"
		defer client.Del(ctx, key)

		// 初始化余额
		err := client.Set(ctx, key, "100", 0)
		require.NoError(t, err)

		// 使用Watch实现乐观锁
		err = client.Watch(ctx, func(tx *redis.Tx) error {
			// 读取当前值
			val, err := tx.Get(ctx, key).Result()
			if err != nil {
				return err
			}

			// 模拟业务逻辑
			t.Logf("Current balance: %s", val)

			// 事务操作
			_, err = tx.TxPipelined(ctx, func(pipe redis.Pipeliner) error {
				pipe.Set(ctx, key, "90", 0) // 扣减10
				return nil
			})
			return err
		}, key)

		require.NoError(t, err)

		// 验证结果
		result, err := client.Get(ctx, key)
		require.NoError(t, err)
		assert.Equal(t, "90", result)

		t.Logf("✓ Transaction completed: balance = %s", result)
	})
}

func TestPubSub(t *testing.T) {
	client := setupTestClient(t)
	defer client.Close()

	ctx := context.Background()
	channel := "test:pubsub:channel"

	t.Run("Publish and Subscribe", func(t *testing.T) {
		// 订阅
		pubsub := client.Subscribe(ctx, channel)
		defer pubsub.Close()

		// 等待订阅确认
		_, err := pubsub.Receive(ctx)
		require.NoError(t, err)

		// 发布消息
		var wg sync.WaitGroup
		wg.Add(1)

		go func() {
			defer wg.Done()
			time.Sleep(100 * time.Millisecond)

			n, err := client.Publish(ctx, channel, "Hello PubSub!")
			require.NoError(t, err)
			t.Logf("✓ Published to %d receivers", n)
		}()

		// 接收消息
		ch := pubsub.Channel()
		select {
		case msg := <-ch:
			assert.Equal(t, channel, msg.Channel)
			assert.Equal(t, "Hello PubSub!", msg.Payload)
			t.Logf("✓ Received message: %s", msg.Payload)
		case <-time.After(2 * time.Second):
			t.Fatal("Timeout waiting for message")
		}

		wg.Wait()
	})

	t.Run("Pattern Subscribe", func(t *testing.T) {
		pattern := "test:pubsub:*"
		testChannel := "test:pubsub:test123"

		// 模式订阅
		pubsub := client.PSubscribe(ctx, pattern)
		defer pubsub.Close()

		// 等待订阅确认
		_, err := pubsub.Receive(ctx)
		require.NoError(t, err)

		// 发布消息
		go func() {
			time.Sleep(100 * time.Millisecond)
			client.Publish(ctx, testChannel, "Pattern message")
		}()

		// 接收消息
		ch := pubsub.Channel()
		select {
		case msg := <-ch:
			assert.Equal(t, testChannel, msg.Channel)
			assert.Equal(t, "Pattern message", msg.Payload)
			assert.Equal(t, pattern, msg.Pattern)
			t.Logf("✓ Received pattern message on channel: %s", msg.Channel)
		case <-time.After(2 * time.Second):
			t.Fatal("Timeout waiting for pattern message")
		}
	})
}

func TestLuaScript(t *testing.T) {
	client := setupTestClient(t)
	defer client.Close()

	ctx := context.Background()

	t.Run("Eval simple script", func(t *testing.T) {
		script := `return redis.call('SET', KEYS[1], ARGV[1])`
		key := "test:lua:key"
		defer client.Del(ctx, key)

		result, err := client.Eval(ctx, script, []string{key}, "hello lua")
		require.NoError(t, err)
		assert.Equal(t, "OK", result)

		// 验证
		val, err := client.Get(ctx, key)
		require.NoError(t, err)
		assert.Equal(t, "hello lua", val)

		t.Logf("✓ Lua script executed: %s = %s", key, val)
	})

	t.Run("ScriptLoad and EvalSha", func(t *testing.T) {
		script := `return redis.call('GET', KEYS[1])`
		key := "test:lua:evalsha"
		defer client.Del(ctx, key)

		// 预先设置值
		err := client.Set(ctx, key, "cached value", 0)
		require.NoError(t, err)

		// 加载脚本
		sha1, err := client.ScriptLoad(ctx, script)
		require.NoError(t, err)
		assert.NotEmpty(t, sha1)

		// 通过SHA1执行
		result, err := client.EvalSha(ctx, sha1, []string{key})
		require.NoError(t, err)
		assert.Equal(t, "cached value", result)

		t.Logf("✓ Script loaded (SHA1: %s) and executed via EvalSha", sha1[:8])
	})

	t.Run("Atomic increment with Lua", func(t *testing.T) {
		key := "test:lua:atomic_incr"
		defer client.Del(ctx, key)

		// 原子性递增并获取值
		script := `
			local current = redis.call('INCR', KEYS[1])
			return current
		`

		result, err := client.Eval(ctx, script, []string{key})
		require.NoError(t, err)
		assert.Equal(t, int64(1), result)

		result, err = client.Eval(ctx, script, []string{key})
		require.NoError(t, err)
		assert.Equal(t, int64(2), result)

		t.Logf("✓ Atomic increment via Lua: %d", result)
	})
}

func TestDistributedLock(t *testing.T) {
	client := setupTestClient(t)
	defer client.Close()

	ctx := context.Background()
	lockKey := "test:lock:resource"

	t.Run("Lock and Unlock", func(t *testing.T) {
		// 获取锁
		token, err := client.Lock(ctx, lockKey, 10*time.Second)
		require.NoError(t, err)
		assert.NotEmpty(t, token)

		t.Logf("✓ Lock acquired: %s", token)

		// 验证锁存在
		exists, err := client.Exists(ctx, lockKey)
		require.NoError(t, err)
		assert.Equal(t, int64(1), exists)

		// 释放锁
		err = client.Unlock(ctx, lockKey, token)
		require.NoError(t, err)

		// 验证锁已释放
		exists, err = client.Exists(ctx, lockKey)
		require.NoError(t, err)
		assert.Equal(t, int64(0), exists)

		t.Logf("✓ Lock released successfully")
	})

	t.Run("Lock collision", func(t *testing.T) {
		// 第一个客户端获取锁
		token1, err := client.Lock(ctx, lockKey, 10*time.Second)
		require.NoError(t, err)
		defer client.Unlock(ctx, lockKey, token1)

		// 第二个客户端尝试获取同一个锁（应该失败）
		_, err = client.Lock(ctx, lockKey, 10*time.Second)
		assert.Error(t, err)

		t.Logf("✓ Lock collision prevented")
	})

	t.Run("TryLock with retry", func(t *testing.T) {
		lockKey2 := "test:lock:trylock"

		// 第一个goroutine获取锁并持有1秒
		go func() {
			token, err := client.Lock(ctx, lockKey2, 5*time.Second)
			if err == nil {
				time.Sleep(1 * time.Second)
				client.Unlock(ctx, lockKey2, token)
			}
		}()

		// 等待第一个goroutine获取锁
		time.Sleep(100 * time.Millisecond)

		// 第二个goroutine尝试获取锁（带重试）
		start := time.Now()
		token, err := client.TryLock(ctx, lockKey2, 5*time.Second, 10, 100*time.Millisecond)
		duration := time.Since(start)

		require.NoError(t, err)
		assert.NotEmpty(t, token)
		defer client.Unlock(ctx, lockKey2, token)

		t.Logf("✓ TryLock succeeded after %.2fs", duration.Seconds())
	})

	t.Run("WithLock helper", func(t *testing.T) {
		lockKey3 := "test:lock:withlock"
		counter := 0

		err := client.WithLock(ctx, lockKey3, 5*time.Second, func() error {
			counter++
			time.Sleep(100 * time.Millisecond)
			return nil
		})

		require.NoError(t, err)
		assert.Equal(t, 1, counter)

		t.Logf("✓ WithLock executed function successfully")
	})

	t.Run("Unlock with wrong token", func(t *testing.T) {
		lockKey4 := "test:lock:wrong_token"

		token, err := client.Lock(ctx, lockKey4, 10*time.Second)
		require.NoError(t, err)
		defer client.Unlock(ctx, lockKey4, token)

		// 尝试用错误的token释放锁
		err = client.Unlock(ctx, lockKey4, "wrong-token")
		assert.Error(t, err)

		t.Logf("✓ Unlock with wrong token rejected")
	})
}

func TestScan(t *testing.T) {
	client := setupTestClient(t)
	defer client.Close()

	ctx := context.Background()

	// 准备测试数据
	keys := []string{
		"test:scan:user:1",
		"test:scan:user:2",
		"test:scan:user:3",
		"test:scan:order:1",
		"test:scan:order:2",
	}

	for _, key := range keys {
		err := client.Set(ctx, key, "value", 0)
		require.NoError(t, err)
	}

	defer func() {
		client.Del(ctx, keys...)
	}()

	t.Run("Scan with pattern", func(t *testing.T) {
		var allKeys []string
		var cursor uint64

		// 扫描所有匹配的键
		for {
			var foundKeys []string
			var err error

			foundKeys, cursor, err = client.Scan(ctx, cursor, "test:scan:user:*", 10)
			require.NoError(t, err)

			allKeys = append(allKeys, foundKeys...)

			if cursor == 0 {
				break
			}
		}

		assert.GreaterOrEqual(t, len(allKeys), 3)

		t.Logf("✓ Scan found %d keys matching pattern 'test:scan:user:*'", len(allKeys))
	})
}

func TestGeoOperations(t *testing.T) {
	client := setupTestClient(t)
	defer client.Close()

	ctx := context.Background()
	key := "test:geo:cities"
	defer client.Del(ctx, key)

	t.Run("GeoAdd and GeoRadius", func(t *testing.T) {
		// 添加地理位置
		n, err := client.GeoAdd(ctx, key,
			&redis.GeoLocation{Longitude: 121.47, Latitude: 31.23, Name: "Shanghai"},
			&redis.GeoLocation{Longitude: 116.40, Latitude: 39.90, Name: "Beijing"},
			&redis.GeoLocation{Longitude: 113.26, Latitude: 23.13, Name: "Guangzhou"},
		)
		require.NoError(t, err)
		assert.Equal(t, int64(3), n)

		// 查询半径内的城市
		locations, err := client.GeoRadius(ctx, key, 121.47, 31.23, &redis.GeoRadiusQuery{
			Radius:    1000,
			Unit:      "km",
			WithDist:  true,
			WithCoord: true,
			Count:     10,
		})
		require.NoError(t, err)
		assert.Greater(t, len(locations), 0)

		t.Logf("✓ GeoRadius found %d cities within 1000km of Shanghai:", len(locations))
		for _, loc := range locations {
			t.Logf("  - %s (%.2fkm)", loc.Name, loc.Dist)
		}
	})

	t.Run("GeoDist", func(t *testing.T) {
		dist, err := client.GeoDist(ctx, key, "Shanghai", "Beijing", "km")
		require.NoError(t, err)
		assert.Greater(t, dist, 0.0)

		t.Logf("✓ Distance between Shanghai and Beijing: %.2f km", dist)
	})

	t.Run("GeoRadiusByMember", func(t *testing.T) {
		locations, err := client.GeoRadiusByMember(ctx, key, "Shanghai", &redis.GeoRadiusQuery{
			Radius:   1500,
			Unit:     "km",
			WithDist: true,
			Count:    10,
		})
		require.NoError(t, err)
		assert.Greater(t, len(locations), 0)

		t.Logf("✓ Cities within 1500km of Shanghai: %d", len(locations))
	})
}

func TestHyperLogLog(t *testing.T) {
	client := setupTestClient(t)
	defer client.Close()

	ctx := context.Background()
	key := "test:hll:visitors"
	defer client.Del(ctx, key)

	t.Run("PFAdd and PFCount", func(t *testing.T) {
		// 添加元素
		for i := 1; i <= 1000; i++ {
			_, err := client.PFAdd(ctx, key, fmt.Sprintf("user:%d", i))
			require.NoError(t, err)
		}

		// 统计基数
		count, err := client.PFCount(ctx, key)
		require.NoError(t, err)

		// HyperLogLog 有一定误差，检查是否在合理范围内
		assert.InDelta(t, 1000, count, 50) // 允许5%误差

		t.Logf("✓ HyperLogLog estimated cardinality: %d (actual: 1000)", count)
	})

	t.Run("PFMerge", func(t *testing.T) {
		key1 := "test:hll:set1"
		key2 := "test:hll:set2"
		merged := "test:hll:merged"

		defer func() {
			client.Del(ctx, key1, key2, merged)
		}()

		// 创建两个HyperLogLog
		client.PFAdd(ctx, key1, "a", "b", "c")
		client.PFAdd(ctx, key2, "c", "d", "e")

		// 合并
		err := client.PFMerge(ctx, merged, key1, key2)
		require.NoError(t, err)

		// 统计合并后的基数
		count, err := client.PFCount(ctx, merged)
		require.NoError(t, err)
		assert.Equal(t, int64(5), count) // a,b,c,d,e

		t.Logf("✓ PFMerge result cardinality: %d", count)
	})
}
