# cache

统一的缓存接口，支持 Redis、FreeCache、BigCache 三种后端，内置防雪崩/击穿/穿透三重防护。

## 安装

```shell
go get cnbattle.com/ai/pkg/cache
```

## 快速开始

```go
import "cnbattle.com/ai/pkg/cache"

// 通过 provider 创建
c, _ := cache.NewClient("Redis", "127.0.0.1:6379", "", 0, 0, context.Background())

// 直接创建
c := cache.Init("127.0.0.1:6379", "", 0, context.Background())

// 读写
c.Set("key", "value", time.Minute)
val, _ := c.Get("key")
c.Del("key")
```

## Provider 对比

| Provider | 后端 | 过期支持 | 适用场景 |
|----------|------|---------|---------|
| `Redis` | go-redis/v9 | 精确 | 生产环境，分布式 |
| `FreeCache` | coocood/freecache | 秒级 | 单机，无 GC 压力 |
| `BigCache` | allegro/bigcache/v3 | 粗粒度 | 单机，大量小对象 |

### Redis

```go
c := cache.NewClient("Redis", "127.0.0.1:6379", "password", 0, 0, ctx)
```

| 参数 | 说明 |
|------|------|
| addr | `host:port` |
| password | 密码，无则 `""` |
| db | 数据库编号 |
| ext | 未使用 |

### FreeCache

```go
c := cache.NewClient("FreeCache", "", "", 0, 104857600, ctx)
```

| 参数 | 说明 |
|------|------|
| ext | 缓存大小（字节），默认 100MB |

> **注意**：FreeCache 的 TTL 以秒为单位，小于1秒的 TTL 等同于不过期。

### BigCache

```go
c := cache.NewClient("BigCache", "", "", 0, 300, ctx)
```

| 参数 | 说明 |
|------|------|
| ext | 清理间隔（秒） |

> **注意**：BigCache 的 `Set` 方法忽略传入的 TTL，过期时间由初始化时的 `eviction` 配置决定。

## Cache 接口

```go
type Cache interface {
    Set(key, value string, exp time.Duration) error
    Get(key string) (string, error)
    Del(key string) error
}
```

所有 provider 实现此接口，可互换使用。

## ProtectedCache — 三重防护

在任意 `Cache` 实现上包装一层，自动获得防雪崩/击穿/穿透能力：

```go
pc := cache.NewProtectedCache(c)

// 可选：自定义配置
pc := cache.NewProtectedCache(c,
    cache.WithJitterRatio(0.2),           // TTL 抖动 ±20%（默认 0.1 = ±10%）
    cache.WithEmptyTTL(5 * time.Second),  // 空值缓存过期时间（默认 1s）
)
```

### 防雪崩 — TTL 随机抖动

大量 key 同时写入时，TTL 加 ±10% 随机偏移，避免同时过期打垮 DB：

```go
// SetWithJitter：TTL 在 [exp * 0.9, exp * 1.1] 范围内随机
pc.SetWithJitter(ctx, "key", "value", time.Hour) // 54m ~ 66m
```

对比普通 `Set`（所有 key 同时过期）：

```go
pc.Set("k1", "v1", time.Hour) // 所有 key 60 分钟后同时过期
pc.Set("k2", "v2", time.Hour)
```

### 防击穿 — singleflight 合并并发请求

热 key 过期时，多个并发请求只有一个真正查 DB，其余等待复用结果：

```go
val, err := pc.GetOrLoad(ctx, "user:123", 5*time.Minute,
    func(ctx context.Context) (string, error) {
        // 此函数在并发请求下只执行一次
        return db.GetUser(ctx, "123")
    },
)
```

### 防穿透 — 空值缓存

查询不存在的数据时，缓存空值占位，防止请求直达 DB：

```go
val, err := pc.GetOrLoad(ctx, "user:999", 5*time.Minute,
    func(ctx context.Context) (string, error) {
        user, err := db.GetUser(ctx, "999")
        if err != nil {
            return "", err   // 返回 error → 不缓存，传播错误
        }
        return user, nil    // user 为 "" 时 → 自动缓存空值
    },
)
if err == cache.ErrCacheEmpty {
    // 数据不存在（已缓存空值，emptyTTL 内不重复查 DB）
}

// 数据变更后手动清除空值
_ = pc.DeleteEmpty("user:999")
```

### Loader 返回值约定

| loader 返回 | 行为 | GetOrLoad 返回 |
|------------|------|---------------|
| `(数据, nil)` | 缓存数据，正常返回 | `(数据, nil)` |
| `("", nil)` | 缓存空值占位（`emptyTTL`） | `("", ErrCacheEmpty)` |
| `("", err)` | 不缓存，原样传播 | `("", err)` |

### 配置项

| Option | 默认值 | 说明 |
|--------|-------|------|
| `WithJitterRatio(r)` | `0.1` | TTL 抖动比例，`r=0.2` 表示 ±20% |
| `WithEmptyTTL(d)` | `1s` | 空值缓存过期时间 |

### 方法一览

| 方法 | 说明 |
|------|------|
| `Set(key, value, exp)` | 直接写入（继承自 `Cache`） |
| `Get(key)` | 读取（继承自 `Cache`） |
| `Del(key)` | 删除（继承自 `Cache`） |
| `SetWithJitter(ctx, key, value, exp)` | 写入，TTL 随机抖动 |
| `GetOrLoad(ctx, key, exp, loader)` | 读取或加载，自动 singleflight + 空值缓存 |
| `DeleteEmpty(key)` | 清除空值缓存，下次 `GetOrLoad` 重新加载 |

## 环境变量配置

通过主库 `ai` 自动初始化时，支持以下环境变量：

| 环境变量 | 说明 | 默认值 |
|---------|------|-------|
| `CACHE` | 是否启用 | `false` |
| `CACHE_PROVIDER` | 提供商 | `Redis` |
| `CACHE_HOST` | 地址 | |
| `CACHE_PASS` | 密码 | |
| `CACHE_DB` | 数据库编号 | `1` |
| `CACHE_EXT` | 扩展参数 | `10` |
| `CACHE_JITTER_RATIO` | TTL 抖动比例 | `0.1` |
| `CACHE_EMPTY_TTL` | 空值缓存过期时间 | `1s` |

Tag 模式下加 `_TAG_` 后缀：

```env
CACHE_USER_HOST=127.0.0.1:6379
CACHE_USER_JITTER_RATIO=0.2
CACHE_USER_EMPTY_TTL=5s
```

## 并发安全

- `Cache` 接口实现：Redis（连接池）、FreeCache（分片 Mutex）、BigCache（分片 RWMutex）— 均线程安全
- `ProtectedCache`：`singleflight.Group` 线程安全，`rand/v2` per-P 无锁随机
- 已通过 `go test -race` 全量竞态检测
