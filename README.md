# ai

Go 后端服务基础工具库，提供配置、日志、缓存、数据库、短信、JWT、ID 生成等常用功能的封装。

## 安装

```shell
go get -u cnbattle.com/ai
```

## 快速开始

```go
import "cnbattle.com/ai"

// 直接使用工具函数
hash := ai.MD5("hello")
num := ai.ToInt("123")

// 通过环境变量自动初始化服务（在 .env 或系统环境变量中配置）
// DB=true, DB_DSN=user:pass@tcp(127.0.0.1:3306)/dbname?charset=utf8mb4
// ai.DB 即可使用 *gorm.DB
```

---

## 目录

- [环境变量配置](#环境变量配置)
- [自动初始化](#自动初始化)
- [多实例（Tag 模式）](#多实例tag-模式)
- [工具函数](#工具函数)
  - [配置读取](#配置读取)
  - [日志](#日志)
  - [HTTP 客户端](#http-客户端)
  - [并发池](#并发池)
  - [类型转换](#类型转换)
  - [JSON 解析](#json-解析)
  - [哈希/加密](#哈希加密)
  - [PKCS 填充](#pkcs-填充)
  - [数据库类型](#数据库类型)
- [子包](#子包)
  - [cache — 缓存](#cache--缓存)
  - [guid — ID 生成器](#guid--id-生成器)
  - [sms — 短信发送](#sms--短信发送)
  - [token — JWT](#token--jwt)
  - [aihttp — HTTP 客户端](#aihttp--http-客户端)
  - [uarand — 随机 User-Agent](#uarand--随机-user-agent)

---

## 环境变量配置

库通过 `github.com/joho/godotenv/autoload` 自动加载项目根目录下的 `.env` 文件，无需手动调用 `godotenv.Load()`。

---

## 自动初始化

以下功能通过 `init()` 函数自动初始化，只需设置对应的环境变量即可：

### 数据库（MySQL/GORM）

| 环境变量 | 说明 | 示例 |
|---------|------|------|
| `DB` | 是否启用 | `true` |
| `DB_DSN` | 数据库连接串 | `user:pass@tcp(127.0.0.1:3306)/dbname?charset=utf8mb4` |
| `DB_PREFIX` | 表名前缀（可选） | `t_` |

```go
// .env
DB=true
DB_DSN=root:123456@tcp(127.0.0.1:3306)/mydb?charset=utf8mb4&parseTime=True

// 代码中直接使用
result := ai.DB.Create(&user)
```

### 缓存

| 环境变量 | 说明 | 示例 |
|---------|------|------|
| `CACHE` | 是否启用 | `true` |
| `CACHE_PROVIDER` | 提供商 | `Redis` / `FreeCache` / `BigCache` |
| `CACHE_HOST` | 地址 | `127.0.0.1:6379` |
| `CACHE_PASS` | 密码（可选） | |
| `CACHE_DB` | 数据库编号（可选） | `0` |
| `CACHE_EXT` | 扩展参数（可选） | FreeCache: 缓存大小(字节)，BigCache: 清理间隔(秒) |
| `CACHE_JITTER_RATIO` | TTL 抖动比例（可选） | `0.1`（默认 ±10%） |
| `CACHE_EMPTY_TTL` | 空值缓存过期时间（可选） | `1s`（默认） |

```go
// .env — 使用 Redis
CACHE=true
CACHE_PROVIDER=Redis
CACHE_HOST=127.0.0.1:6379

// .env — 使用 FreeCache（内存缓存，100MB）
CACHE=true
CACHE_PROVIDER=FreeCache
CACHE_EXT=104857600

// 基础用法（ai.Cache 是 *cache.ProtectedCache，兼容 cache.Cache 接口）
val, _ := ai.Cache.Get("key")
ai.Cache.Del("key")
```

`ai.Cache` 是 `*cache.ProtectedCache`，提供两组写入方法：
- `Set` — 直接写入，不做 TTL 抖动
- `SetWithJitter` — 写入时 TTL ±10% 随机偏移（可通过 `WithJitterRatio` 调整）

#### 防雪崩 — 随机抖动 TTL

同一时间写入大量 key 时，给 TTL 加 ±10% 随机偏移（默认，可通过 `WithJitterRatio` 自定义），避免同时过期：

```go
// 传统方式：所有 key 同时过期 → 雪崩
ai.Cache.Set("k1", "v1", time.Hour)
ai.Cache.Set("k2", "v2", time.Hour)

// SetWithJitter：TTL 随机抖动 → 过期时间分散
ai.Cache.SetWithJitter(ctx, "k1", "v1", time.Hour) // 54m ~ 66m
ai.Cache.SetWithJitter(ctx, "k2", "v2", time.Hour) // 54m ~ 66m
```

#### 防击穿 — singleflight 合并并发请求

热 key 过期时，多个并发请求只有一个真正查 DB，其余等待复用结果：

```go
val, err := ai.Cache.GetOrLoad(ctx, "user:123", 5*time.Minute,
    func(ctx context.Context) (string, error) {
        // 此函数在并发请求下只执行一次（singleflight）
        return db.GetUser(ctx, "123")
    },
)
if err != nil {
    // loader 返回的业务错误（非空值缓存场景）
}
```

#### 防穿透 — 空值缓存

查询不存在的数据时，缓存空值占位（`emptyTTL` = 1 秒），防止请求直达 DB：

- loader 返回 `""` → 自动缓存空值，返回 `cache.ErrCacheEmpty`
- loader 返回 error → 原样传播，不缓存
- 空值 1 秒后自动过期，或调用 `DeleteEmpty` 手动清除

```go
val, err := ai.Cache.GetOrLoad(ctx, "user:999", 5*time.Minute,
    func(ctx context.Context) (string, error) {
        user, err := db.GetUser(ctx, "999")
        if err != nil {
            return "", err // 返回 error → 不缓存，传播错误
        }
        return user, nil  // user 为 "" 时 → 自动缓存空值
    },
)
if err == cache.ErrCacheEmpty {
    // 数据不存在（已缓存空值，1秒内不重复查 DB）
}

// 数据变更后清除空值缓存
_ = ai.Cache.DeleteEmpty("user:999")
```

#### Tag 多实例

```go
// 同样返回 *cache.ProtectedCache，自动带三重防护
userCache := ai.InitCacheForTag("user")
sessionCache := ai.InitCacheForTag("session")

userCache.SetWithJitter(ctx, "u:1", "data", time.Hour)
val, err := userCache.GetOrLoad(ctx, "u:1", time.Minute, loader)
```

### 短信

| 环境变量 | 说明 | 示例 |
|---------|------|------|
| `SMS` | 是否启用 | `true` |
| `SMS_PROVIDER` | 提供商 | `AliyunSMS` / `TencentCloudSMS` / `VolcEngineSMS` / `HuyiSMS` |
| `SMS_ACCESS_ID` | Access ID | |
| `SMS_ACCESS_KEY` | Access Key | |
| `SMS_APP_ID` | 应用 ID（腾讯云） | |
| `SMS_SIGN` | 签名 | |
| `SMS_TEMPLATE` | 模板 ID | |

```go
// .env
SMS=true
SMS_PROVIDER=TencentCloudSMS
SMS_ACCESS_ID=xxx
SMS_ACCESS_KEY=xxx
SMS_SIGN=我的应用
SMS_TEMPLATE=123456

// 发送短信
err := ai.SMS.SendMessage(map[string]string{"code": "1234"}, "13800138000")
```

### JWT Token

| 环境变量 | 说明 | 示例 |
|---------|------|------|
| `TOKEN` | 是否启用 | `true` |
| `TOKEN_SECRET` | 签名密钥 | |
| `TOKEN_EXP` | 过期时间（秒，可选） | `7200` |

```go
// .env
TOKEN=true
TOKEN_SECRET=my-secret-key
TOKEN_EXP=7200

// 生成 Token
tokenStr, err := ai.Token.GenerateToken("user123", "admin", 7200)

// 验证 Token
claims, err := ai.Token.VerifyToken(tokenStr)

// 在 Gin 中使用
// ai.Token.VerifyTokenForRole(c, "admin")  // 从 Cookie/Header 中提取并验证
```

### ID 生成器

| 环境变量 | 说明 | 示例 |
|---------|------|------|
| `GUID` | 是否启用 | `true` |
| `GUID_ENGINE` | 引擎 | `idgen` / `sonyflake` / `uuid` / `typeid` |
| `GUID_START_TIME` | 起始时间（idgen 用） | `2024-01-01` |
| `GUID_WORKER_ID` | 工作节点 ID（可选） | `1` |

```go
// .env
GUID=true
GUID_ENGINE=uuid

// 生成 ID
id := ai.GUID.NextID()
idWithPrefix := ai.GUID.WithPrefix("order")
```

---

## 多实例（Tag 模式）

当需要多个同类服务实例时（如多个数据库、多个缓存），使用 `ForTag` 函数：

```go
// 环境变量加 _<TAG> 后缀
DB_ORDER_DSN=root:pass@tcp(host1:3306)/order_db
DB_USER_DSN=root:pass@tcp(host2:3306)/user_db

orderDB := ai.InitGormForTag("order")   // 读取 DB_ORDER_DSN
userDB := ai.InitGormForTag("user")     // 读取 DB_USER_DSN

CACHE_USER_HOST=127.0.0.1:6379
CACHE_SESSION_HOST=127.0.0.1:6380

userCache := ai.InitCacheForTag("user")       // 读取 CACHE_USER_HOST
sessionCache := ai.InitCacheForTag("session") // 读取 CACHE_SESSION_HOST
```

支持的 Tag 函数：

| 函数 | 环境变量模式 |
|------|-------------|
| `InitGormForTag(tag)` | `DB_<TAG>_DSN`, `DB_<TAG>_PREFIX` |
| `InitCacheForTag(tag)` | `CACHE_<TAG>_HOST`, `CACHE_<TAG>_PASS`, ... |
| `InitSmsForTag(tag)` | `SMS_<TAG>_PROVIDER`, `SMS_<TAG>_ACCESS_ID`, ... |
| `InitTokenForTag(tag)` | `TOKEN_<TAG>_SECRET`, `TOKEN_<TAG>_EXP` |
| `InitGuidForTag(tag)` | `GUID_<TAG>_ENGINE`, `GUID_<TAG>_START_TIME`, ... |

---

## 工具函数

### 配置读取

```go
// 读取环境变量
val := ai.GetEnv("MY_KEY")
val := ai.GetDefaultEnv("MY_KEY", "默认值")

// 类型转换读取
num := ai.GetEnvToInt("PORT")
num := ai.GetDefaultEnvToInt("PORT", 8080)
flag := ai.GetEnvToBool("DEBUG")
```

### 日志

```go
// 全局日志实例，基于 logrus，TraceLevel，自动包含调用位置
ai.LOG.Info("服务启动")
ai.LOG.Errorf("数据库连接失败: %v", err)
ai.LOG.WithField("user_id", "123").Info("用户登录")
```

### HTTP 客户端

```go
// Resty v2
client := ai.RestyV2New()
resp, err := client.R().
    SetQueryParam("page", "1").
    Get("https://api.example.com/list")

// Resty v3
client := ai.RestyV3New()
resp, err := client.R().
    SetQueryParam("page", "1").
    Get("https://api.example.com/list")
```

### 并发池

```go
// 创建并发池，最大 10 个 goroutine 并发
pool := ai.NewPool(10)

for _, item := range items {
    pool.Add()
    go func(item string) {
        defer pool.Done()
        // 处理任务
    }(item)
}

pool.Wait() // 等待所有任务完成

// 无限制并发（size <= 0）
pool := ai.NewPool(0)
```

### 类型转换

基于 `spf13/cast` 的封装：

```go
ai.ToBool("true")        // true
ai.ToInt("123")          // 123
ai.ToString(123)         // "123"
ai.ToFloat64("3.14")     // 3.14
ai.ToTime("2024-01-01")  // time.Time
ai.ToStringSlice([]any{"a", "b"})  // []string{"a", "b"}
ai.ToStringMap(map[any]any{"a": 1}) // map[string]any{"a": 1}
```

### JSON 解析

基于 `tidwall/gjson` 的封装：

```go
json := `{"name":"张三","age":25,"scores":[90,85,95]}`

// 解析 JSON
result := ai.JsonParse(json)

// 获取字段值
name := ai.JsonGet(json, "name")       // "张三"
age := ai.JsonGet(json, "age")         // "25"
first := ai.JsonGet(json, "scores.0")  // "90"

// 解析为 Go 对象
m := ai.ParseJson(json)  // map[string]any
```

### 哈希/加密

```go
ai.MD5("hello")              // "5d41402abc4b2a76b9719d911017c592"
ai.SHA1("hello")             // "aaf4c61ddcc5e8a2dabede0f3b482cd9aea9434d"
ai.SHA256("hello")           // "2cf24dba5fb0a30e26e83b2ac5b9e29e..."
ai.HMacSHA256("secret", "msg")  // HMAC-SHA256 签名
ai.Hash(ai.MD5, "hello")     // 指定哈希算法
```

### PKCS 填充

用于 AES 等分组密码的填充：

```go
// PKCS5
padded := ai.PKCS5Padding(data, blockSize)
unpadded := ai.PKCS5UnPadding(padded)

// PKCS7
padded := ai.PKCS7Padding(data, blockSize)
unpadded := ai.PKCS7UnPadding(padded)
```

### 数据库类型

自定义的数据库友好类型，支持 JSON 序列化和 `database/sql` 接口：

```go
// DbDate — 日期（2006-01-02）
type Order struct {
    Date ai.DbDate `json:"date" gorm:"column:date"`
}
order := Order{Date: ai.DbDate{Time: time.Now()}}

// DbTime — 日期时间（2006-01-02 15:04:05）
type Log struct {
    CreatedAt ai.DbTime `json:"created_at" gorm:"column:created_at"`
}

// DbTotal — 金额/数量
type Report struct {
    Total ai.DbTotal `json:"total" gorm:"column:total"`
}
```

---

## 子包

### cache — 缓存

统一的缓存接口，支持 Redis、FreeCache、BigCache 三种后端。

```go
import "cnbattle.com/ai/pkg/cache"

// 直接创建 Redis
c := cache.Init("127.0.0.1:6379", "", 0, context.Background())

// 通过 provider 创建
c, _ := cache.NewClient("Redis", "127.0.0.1:6379", "", 0, 0, context.Background())
c, _ := cache.NewClient("FreeCache", "", "", 0, 104857600, context.Background())
c, _ := cache.NewClient("BigCache", "", "", 0, 300, context.Background())

// 使用
c.Set("key", "value", time.Minute)
val, _ := c.Get("key")
c.Del("key")
```

#### ProtectedCache — 防雪崩/击穿/穿透

```go
import "cnbattle.com/ai/pkg/cache"

// 包装任意 Cache 实现，自动获得三重防护（使用默认配置）
pc := cache.NewProtectedCache(c)

// 自定义配置
pc := cache.NewProtectedCache(c,
    cache.WithJitterRatio(0.2),            // TTL 抖动 ±20%（默认 0.1 = ±10%）
    cache.WithEmptyTTL(5 * time.Second),   // 空值缓存过期时间（默认 1s）
)

// 防雪崩：TTL 随机抖动
pc.SetWithJitter(ctx, "key", "value", time.Hour)

// 防击穿 + 防穿透：singleflight 合并并发请求 + 空值缓存
//
// loader 返回值约定：
//   - 返回 (数据, nil) → 缓存数据，正常返回
//   - 返回 ("", nil)   → 缓存空值占位（emptyTTL），返回 cache.ErrCacheEmpty
//   - 返回 ("", err)   → 不缓存，原样传播 err
val, err := pc.GetOrLoad(ctx, "user:123", 5*time.Minute,
    func(ctx context.Context) (string, error) {
        return db.GetUser(ctx, "123")
    },
)
if err == cache.ErrCacheEmpty {
    // 数据不存在（已缓存空值，emptyTTL 内不重复查 DB）
}

// 数据变更后清除空值缓存
_ = pc.DeleteEmpty("user:123")
```

### crud — GORM 缓存 CRUD

基于 GORM + ProtectedCache 的通用 CRUD 基础库，参考 go-zero model cache 设计。

```go
import "cnbattle.com/ai/pkg/crud"

type User struct {
    crud.CachedModel[int64]
    Name string `json:"name"`
    Age  int    `json:"age"`
}

func (User) TableName() string { return "users" }

// 初始化
q := crud.NewRepo[User, User](ai.DB, ai.Cache, 5*time.Minute)

// read-through 查询（缓存 miss → singleflight → 查 DB → 写缓存）
user, _ := q.FindByID(ctx, &User{ID: 1})

// 按字段查询（字段缓存只存主键，完整数据走主键缓存）
user, _ = q.FindByField(ctx, &User{}, "name", "张三")

// 写入（自动失效相关缓存）
q.Create(ctx, &User{ID: 2, Name: "李四", Age: 30})
q.UpdateByID(ctx, &User{ID: 2}, map[string]any{"age": 31})
q.DeleteByID(ctx, &User{ID: 2})

// 批量操作（自动先查后删/更+失效）
q.DeleteBatch(ctx, "age < ?", 18)
q.UpdateBatch(ctx, map[string]any{"status": 1}, "status = ?", 0)
```

详细文档见 [pkg/crud/README.md](pkg/crud/README.md)。

### guid — ID 生成器

统一的 ID 生成接口，支持多种算法：

```go
import "cnbattle.com/ai/pkg/guid"

// IdGen（雪花算法，高性能）
g := guid.New("idgen", "2024-01-01", 1)
id := g.NextID()

// Sonyflake（分布式 ID）
g := guid.New("sonyflake", "", 0)
id := g.NextID()

// UUID v7（时间有序）
g := guid.New("uuid", "", 0)
id := g.NextID()

// TypeID（带类型前缀）
g := guid.New("typeid", "", 0)
id := g.WithPrefix("order")  // "order_01hxyz..."
```

### sms — 短信发送

统一的短信发送接口：

```go
import "cnbattle.com/ai/pkg/sms"

// 腾讯云短信
c := sms.NewClient("TencentCloudSMS", "secretId", "secretKey", "签名", "模板ID", "sdkAppId")
err := c.SendMessage(map[string]string{"code": "1234"}, "13800138000")

// 阿里云短信
c := sms.NewClient("AliyunSMS", "accessKeyId", "accessKeySecret", "签名", "模板Code", "")
err := c.SendMessage(map[string]string{"code": "1234"}, "13800138000")

// 火山引擎短信
c := sms.NewClient("VolcEngineSMS", "accessKey", "secretKey", "签名", "模板ID", "appID")

// 互易短信
c := sms.NewClient("HuyiSMS", "accountID", "accountKey", "签名", "模板ID", "")
```

### token — JWT

基于 HMAC-SHA256 的 JWT 实现，支持 Gin 框架集成：

```go
import "cnbattle.com/ai/pkg/token"

c := token.NewClient("your-secret-key", 7200)

// 生成 Token
tokenStr, err := c.GenerateToken("user123", "admin", 7200)

// 验证 Token
claims, err := c.VerifyToken(tokenStr)
fmt.Println(claims.UID, claims.Role)

// Gin 中使用（从 Cookie 或 Header 中提取）
// c.VerifyTokenForRole(c gin.Context, "admin")
```

### aihttp — HTTP 客户端

Resty v2 和 v3 的简单封装：

```go
import "cnbattle.com/ai/pkg/aihttp"

client := aihttp.RestyV2New()  // Resty v2
client := aihttp.RestyV3New()  // Resty v3
```

### uarand — 随机 User-Agent

线程安全的随机 User-Agent 生成器：

```go
import "cnbattle.com/ai/pkg/uarand"

ua := uarand.GetRandom()  // 随机获取一个浏览器 UA 字符串

// 自定义列表
myList := []string{"Mozilla/5.0 ...", "Mozilla/5.0 ..."}
gen := uarand.NewWithCustomList(myList)
ua := gen.GetRandom()
```

---

## License

MIT
