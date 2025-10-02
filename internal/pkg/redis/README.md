# Redis 客户端封装

基于 `github.com/redis/go-redis/v9` 的 Redis 客户端封装，提供完整的 Redis 操作支持，包括单机、哨兵、集群和主从读写分离模式。

## 功能特性

- ✅ **多种部署模式**
  - 单机模式（Single）
  - 哨兵模式（Sentinel）- 自动主从切换
  - 集群模式（Cluster）- 分片存储
  - 主从读写分离模式（Read-Write）- 手动配置主从

- ✅ **完整的 Redis 操作**
  - String 操作（Get/Set/Del/Incr/Decr/SetNX/Expire/TTL）
  - Hash 操作（HGet/HSet/HGetAll/HDel/HExists/HLen/HIncrBy）
  - List 操作（LPush/RPush/LPop/RPop/LLen/LRange/LTrim）
  - Set 操作（SAdd/SRem/SMembers/SIsMember/SCard）
  - ZSet 操作（ZAdd/ZRem/ZRange/ZScore/ZRank/ZIncrBy）

- ✅ **高级功能**
  - Pipeline（批量操作）
  - Transaction（事务）
  - Pub/Sub（发布订阅）
  - Lua Script 执行
  - Scan 迭代器
  - 分布式锁（Lock/Unlock/TryLock/WithLock）
  - Geo 地理位置
  - HyperLogLog 基数统计

- ✅ **读写分离策略**
  - `master` - 只从主节点读
  - `slave` - 只从从节点读
  - `slave-first` - 优先从节点，失败回退主节点
  - `random` - 随机选择节点
  - `round-robin` - 轮询所有节点

- ✅ **企业级特性**
  - 连接池管理
  - 自动重试机制
  - TLS/SSL 支持
  - 健康检查
  - 日志集成（zap）
  - 完整的错误处理

## 快速开始

### 1. 安装依赖

```bash
go get github.com/redis/go-redis/v9
```

### 2. 单机模式

```go
package main

import (
    "context"
    "fmt"
    "time"

    "github.com/lk2023060901/ai-writer-backend/internal/pkg/logger"
    "github.com/lk2023060901/ai-writer-backend/internal/pkg/redis"
)

func main() {
    // 创建 logger
    log, _ := logger.New(&logger.Config{
        Level:  "info",
        Format: "json",
        Output: "console",
    })

    // 创建 Redis 客户端
    client, err := redis.New(&redis.Config{
        Mode:         redis.ModeSingle,
        MasterAddr:   "localhost:6379",
        Password:     "your-password",
        DB:           0,
        PoolSize:     10,
        DialTimeout:  5 * time.Second,
        PoolTimeout:  4 * time.Second,
    }, log)
    if err != nil {
        panic(err)
    }
    defer client.Close()

    ctx := context.Background()

    // String 操作
    client.Set(ctx, "key", "value", time.Hour)
    val, _ := client.Get(ctx, "key")
    fmt.Println(val) // Output: value

    // Hash 操作
    client.HSet(ctx, "user:1", "name", "Alice", "age", "30")
    name, _ := client.HGet(ctx, "user:1", "name")
    fmt.Println(name) // Output: Alice

    // List 操作
    client.RPush(ctx, "queue", "task1", "task2")
    task, _ := client.LPop(ctx, "queue")
    fmt.Println(task) // Output: task1
}
```

### 3. 主从读写分离模式

```go
client, err := redis.New(&redis.Config{
    Mode:         redis.ModeReadWrite,
    MasterAddr:   "localhost:6380",  // 主节点
    SlaveAddrs:   []string{
        "localhost:6381",  // 从节点1
        "localhost:6382",  // 从节点2
    },
    ReadStrategy: redis.ReadFromSlaveFirst, // 优先从从节点读
    PoolSize:     10,
    DialTimeout:  5 * time.Second,
    PoolTimeout:  4 * time.Second,
}, log)
```

### 4. 哨兵模式

```go
client, err := redis.New(&redis.Config{
    Mode: redis.ModeSentinel,
    SentinelAddrs: []string{
        "localhost:26379",
        "localhost:26380",
        "localhost:26381",
    },
    MasterName:     "mymaster",
    RouteByLatency: true,  // 按延迟路由读请求
    PoolSize:       10,
    DialTimeout:    5 * time.Second,
    PoolTimeout:    4 * time.Second,
}, log)
```

### 5. 集群模式

```go
client, err := redis.New(&redis.Config{
    Mode: redis.ModeCluster,
    ClusterAddrs: []string{
        "localhost:7000",
        "localhost:7001",
        "localhost:7002",
    },
    PoolSize:    10,
    DialTimeout: 5 * time.Second,
    PoolTimeout: 4 * time.Second,
}, log)
```

## 使用示例

### Pipeline 批量操作

```go
pipe := client.Pipeline()

pipe.Set(ctx, "key1", "value1", 0)
pipe.Set(ctx, "key2", "value2", 0)
pipe.Incr(ctx, "counter")

cmds, err := pipe.Exec(ctx)
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Executed %d commands\n", len(cmds))
```

### 事务（Transaction）

```go
err := client.Watch(ctx, func(tx *redis.Tx) error {
    // 读取当前值
    val, err := tx.Get(ctx, "balance").Result()
    if err != nil {
        return err
    }

    // 事务操作
    _, err = tx.TxPipelined(ctx, func(pipe redis.Pipeliner) error {
        pipe.Set(ctx, "balance", "updated_value", 0)
        return nil
    })
    return err
}, "balance")
```

### 发布订阅（Pub/Sub）

```go
// 订阅
pubsub := client.Subscribe(ctx, "channel1", "channel2")
defer pubsub.Close()

// 接收消息
ch := pubsub.Channel()
for msg := range ch {
    fmt.Printf("Channel: %s, Message: %s\n", msg.Channel, msg.Payload)
}

// 发布
client.Publish(ctx, "channel1", "Hello!")
```

### Lua 脚本

```go
script := `
    local current = redis.call('GET', KEYS[1])
    if current then
        return redis.call('INCR', KEYS[1])
    else
        redis.call('SET', KEYS[1], ARGV[1])
        return ARGV[1]
    end
`

result, err := client.Eval(ctx, script, []string{"counter"}, 100)
```

### 分布式锁

```go
// 方式1：手动加锁解锁
token, err := client.Lock(ctx, "resource:lock", 10*time.Second)
if err != nil {
    log.Fatal("Failed to acquire lock")
}
defer client.Unlock(ctx, "resource:lock", token)

// 执行业务逻辑
// ...

// 方式2：使用 WithLock 辅助函数
err = client.WithLock(ctx, "resource:lock", 10*time.Second, func() error {
    // 在锁保护下执行业务逻辑
    return nil
})

// 方式3：带重试的锁
token, err = client.TryLock(ctx, "resource:lock", 10*time.Second, 5, 100*time.Millisecond)
```

### Geo 地理位置

```go
// 添加地理位置
client.GeoAdd(ctx, "cities",
    &redis.GeoLocation{Longitude: 121.47, Latitude: 31.23, Name: "Shanghai"},
    &redis.GeoLocation{Longitude: 116.40, Latitude: 39.90, Name: "Beijing"},
)

// 查询半径内的城市
locations, err := client.GeoRadius(ctx, "cities", 121.47, 31.23, &redis.GeoRadiusQuery{
    Radius:   1000,
    Unit:     "km",
    WithDist: true,
    Count:    10,
})

// 计算距离
dist, err := client.GeoDist(ctx, "cities", "Shanghai", "Beijing", "km")
fmt.Printf("Distance: %.2f km\n", dist)
```

### HyperLogLog 基数统计

```go
// 添加元素
for i := 1; i <= 10000; i++ {
    client.PFAdd(ctx, "unique_visitors", fmt.Sprintf("user:%d", i))
}

// 统计唯一访客数
count, err := client.PFCount(ctx, "unique_visitors")
fmt.Printf("Unique visitors: %d\n", count)
```

## 配置说明

### 完整配置示例

```go
config := &redis.Config{
    // 部署模式
    Mode: redis.ModeSingle, // single|sentinel|cluster|read-write

    // 单机/主从模式
    MasterAddr: "localhost:6379",
    SlaveAddrs: []string{"localhost:6380", "localhost:6381"},

    // 哨兵模式
    SentinelAddrs:  []string{"localhost:26379"},
    MasterName:     "mymaster",
    RouteByLatency: true,
    RouteRandomly:  false,

    // 集群模式
    ClusterAddrs: []string{"localhost:7000", "localhost:7001"},

    // 读写分离
    ReadStrategy:  redis.ReadFromSlaveFirst,
    SlaveReadOnly: true,

    // 认证
    Username: "default",
    Password: "your-password",
    DB:       0,

    // 连接池
    PoolSize:     10,
    MinIdleConns: 5,

    // 超时
    DialTimeout:  5 * time.Second,
    ReadTimeout:  3 * time.Second,
    WriteTimeout: 3 * time.Second,
    PoolTimeout:  4 * time.Second,

    // 重试
    MaxRetries:      3,
    MinRetryBackoff: 8 * time.Millisecond,
    MaxRetryBackoff: 512 * time.Millisecond,

    // 连接
    PoolFIFO:        false,
    ConnMaxIdleTime: 5 * time.Minute,
    ConnMaxLifetime: 0,

    // TLS
    EnableTLS:     false,
    TLSCertFile:   "/path/to/cert.pem",
    TLSKeyFile:    "/path/to/key.pem",
    TLSCAFile:     "/path/to/ca.pem",
    TLSSkipVerify: false,
}
```

## 测试

```bash
# 启动 Redis（单机模式）
redis-server --port 6379

# 运行测试
go test -v ./internal/pkg/redis/...

# 运行指定测试
go test -v ./internal/pkg/redis/... -run TestStringOperations
```

## API 文档

### 基础操作

#### String
- `Set(ctx, key, value, expiration) error`
- `Get(ctx, key) (string, error)`
- `Del(ctx, keys...) (int64, error)`
- `Exists(ctx, keys...) (int64, error)`
- `Expire(ctx, key, expiration) (bool, error)`
- `TTL(ctx, key) (time.Duration, error)`
- `Incr(ctx, key) (int64, error)`
- `IncrBy(ctx, key, value) (int64, error)`
- `Decr(ctx, key) (int64, error)`
- `DecrBy(ctx, key, value) (int64, error)`
- `SetNX(ctx, key, value, expiration) (bool, error)`

#### Hash
- `HSet(ctx, key, values...) (int64, error)`
- `HGet(ctx, key, field) (string, error)`
- `HGetAll(ctx, key) (map[string]string, error)`
- `HDel(ctx, key, fields...) (int64, error)`
- `HExists(ctx, key, field) (bool, error)`
- `HLen(ctx, key) (int64, error)`
- `HIncrBy(ctx, key, field, incr) (int64, error)`

#### List
- `LPush(ctx, key, values...) (int64, error)`
- `RPush(ctx, key, values...) (int64, error)`
- `LPop(ctx, key) (string, error)`
- `RPop(ctx, key) (string, error)`
- `LLen(ctx, key) (int64, error)`
- `LRange(ctx, key, start, stop) ([]string, error)`
- `LTrim(ctx, key, start, stop) error`

#### Set
- `SAdd(ctx, key, members...) (int64, error)`
- `SRem(ctx, key, members...) (int64, error)`
- `SMembers(ctx, key) ([]string, error)`
- `SIsMember(ctx, key, member) (bool, error)`
- `SCard(ctx, key) (int64, error)`

#### ZSet
- `ZAdd(ctx, key, members...) (int64, error)`
- `ZRem(ctx, key, members...) (int64, error)`
- `ZRange(ctx, key, start, stop) ([]string, error)`
- `ZRevRange(ctx, key, start, stop) ([]string, error)`
- `ZRangeWithScores(ctx, key, start, stop) ([]redis.Z, error)`
- `ZScore(ctx, key, member) (float64, error)`
- `ZRank(ctx, key, member) (int64, error)`
- `ZRevRank(ctx, key, member) (int64, error)`
- `ZCard(ctx, key) (int64, error)`
- `ZIncrBy(ctx, key, increment, member) (float64, error)`

### 高级功能

#### Pipeline & Transaction
- `Pipeline() redis.Pipeliner`
- `TxPipeline() redis.Pipeliner`
- `Watch(ctx, fn, keys...) error`

#### Pub/Sub
- `Publish(ctx, channel, message) (int64, error)`
- `Subscribe(ctx, channels...) *redis.PubSub`
- `PSubscribe(ctx, patterns...) *redis.PubSub`

#### Lua Script
- `Eval(ctx, script, keys, args...) (interface{}, error)`
- `EvalSha(ctx, sha1, keys, args...) (interface{}, error)`
- `ScriptLoad(ctx, script) (string, error)`

#### Scan
- `Scan(ctx, cursor, match, count) ([]string, uint64, error)`
- `HScan(ctx, key, cursor, match, count) ([]string, uint64, error)`
- `SScan(ctx, key, cursor, match, count) ([]string, uint64, error)`
- `ZScan(ctx, key, cursor, match, count) ([]string, uint64, error)`

#### 分布式锁
- `Lock(ctx, key, expiration) (string, error)`
- `Unlock(ctx, key, token) error`
- `TryLock(ctx, key, expiration, maxRetries, retryDelay) (string, error)`
- `WithLock(ctx, key, expiration, fn) error`

#### Geo
- `GeoAdd(ctx, key, geoLocation...) (int64, error)`
- `GeoRadius(ctx, key, longitude, latitude, query) ([]redis.GeoLocation, error)`
- `GeoRadiusByMember(ctx, key, member, query) ([]redis.GeoLocation, error)`
- `GeoDist(ctx, key, member1, member2, unit) (float64, error)`

#### HyperLogLog
- `PFAdd(ctx, key, els...) (int64, error)`
- `PFCount(ctx, keys...) (int64, error)`
- `PFMerge(ctx, dest, keys...) error`

## 最佳实践

1. **连接池配置**：根据实际并发量调整 `PoolSize` 和 `MinIdleConns`
2. **超时设置**：合理设置各类超时时间，避免长时间阻塞
3. **读写分离**：高读场景下使用主从读写分离降低主节点压力
4. **Pipeline**：批量操作使用 Pipeline 提高性能
5. **分布式锁**：设置合理的过期时间，防止死锁
6. **错误处理**：使用 `IsNil(err)` 判断 Key 不存在错误

## 许可证

MIT License
