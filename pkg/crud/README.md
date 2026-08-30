# crud

基于 GORM + ProtectedCache 的通用 CRUD 基础库，参考 go-zero model cache 设计。

## 安装

```shell
go get cnbattle.com/ai/pkg/crud
```

## 快速开始

```go
import (
    "cnbattle.com/ai"
    "cnbattle.com/ai/pkg/crud"
)

type User struct {
    crud.CachedModel[int64]
    Name string `json:"name"`
    Age  int    `json:"age"`
}

func (User) TableName() string { return "users" }

// 初始化
q := crud.NewRepo[User, User](ai.DB, ai.Cache, 5*time.Minute)

// read-through 查询
user, err := q.FindByID(ctx, &User{ID: 1})

// 按字段查询（字段缓存只存主键，减少内存占用）
user, err = q.FindByField(ctx, &User{}, "name", "张三")

// 写入（自动失效相关缓存）
q.Create(ctx, &User{ID: 2, Name: "李四", Age: 30})
q.UpdateByID(ctx, &User{ID: 2}, map[string]any{"age": 31})
q.DeleteByID(ctx, &User{ID: 2})

// 批量操作（自动先查后删/更 + 失效）
q.DeleteBatch(ctx, "age < ?", 18)
q.UpdateBatch(ctx, map[string]any{"status": 1}, "status = ?", 0)
```

## 架构

```
┌──────────────────────────────────────────────┐
│                  Repo[T, M]                  │
│  FindByID / Create / Save / UpdateByID       │
│  DeleteByID / CreateBatch / DeleteBatch      │
├──────────────────────────────────────────────┤
│               Query[T, M]                    │
│  FindOneByPK / FindOneByField                │
│  Insert / Update / Delete                    │
│  InsertBatch / DeleteBatch / UpdateBatch     │
│  take() → PK cache                           │
│  takeByField() → field cache → PK cache      │
├──────────────────────────────────────────────┤
│             CachedQuery                      │
│  db *gorm.DB + cache Cache + KeyBuilder      │
├──────────────────────────────────────────────┤
│           ProtectedCache (pkg/cache)         │
│  雪崩(Jitter) + 击穿(Singleflight) + 穿透    │
└──────────────────────────────────────────────┘
```

## Model 接口

业务 model 需要实现 `Model` 接口，嵌入 `CachedModel` 可获得默认实现：

```go
type Model interface {
    TableName() string         // 表名
    PrimaryKey() any           // 主键值
    PrimaryKeyColumn() string  // 主键列名，默认 "id"
    CachePrefix() string       // 缓存 key 前缀
}
```

```go
type User struct {
    crud.CachedModel[int64]
    Name string
    Age  int
}

func (User) TableName() string { return "users" }
```

自定义主键列名：

```go
type Order struct {
    crud.CachedModel[string]
    OrderNo string
}

func (Order) PrimaryKeyColumn() string { return "order_no" }
func (Order) TableName() string        { return "orders" }
```

## Cache 接口

```go
type Cache interface {
    Set(key, value string, exp time.Duration) error
    Get(key string) (string, error)
    Del(key string) error
    DelMulti(keys ...string) error
    GetOrLoad(ctx context.Context, key string, exp time.Duration, loader cache.Loader) (string, error)
    DeleteEmpty(key string) error
}
```

兼容 `*cache.ProtectedCache`，直接传入 `ai.Cache` 即可。

## 缓存 Key 设计

参考 go-zero，统一命名空间：

| 类型 | 格式 | 缓存内容 | 示例 |
|------|------|---------|------|
| 主键 | `{prefix}:{table}:id:{pk}` | 完整记录 JSON | `cache:users:id:1` |
| 字段索引 | `{prefix}:{table}:{field}:{value}` | 主键字符串 | `cache:users:name:张三` → `"1"` |
| 复合索引 | `{prefix}:{table}:idx_{f1}_{f2}:{v1}_{v2}` | 主键字符串 | `cache:users:idx_name_age:张三_25` → `"1"` |

```go
kb := crud.NewKeyBuilder()
kb.Primary("cache", "users", 1)                    // "cache:users:id:1"
kb.Field("cache", "users", "name", "张三")          // "cache:users:name:张三"
kb.Composite("cache", "users", []string{"name","age"}, []any{"张三",25})
// "cache:users:idx_name_age:张三_25"
```

## Read-Through 查询

| 方法 | 说明 |
|------|------|
| `FindByID(ctx, model)` | 按主键查询（read-through） |
| `FindByField(ctx, model, field, value)` | 按字段索引查询（read-through） |
| `FindOneByPK(ctx, model)` | 按主键查询（read-through） |
| `FindOneByField(ctx, model, field, value)` | 按字段索引查询（read-through） |
| `FindOneByComposite(ctx, model, fields, values)` | 按复合索引查询（read-through） |
| `Find(ctx, dest, conds...)` | 查询列表（不缓存） |
| `FindAll(ctx, dest)` | 查询全部（不缓存） |
| `Count(ctx, model, conds...)` | 统计（不缓存） |

### 主键查询流程

```
cache hit → 直接返回
cache miss → singleflight 合并 → 查 DB → marshal → 写缓存 → 返回
```

### 字段查询流程

```
字段缓存 hit → 检查格式（兼容旧数据）→ 是 PK → 主键缓存 hit → 返回完整记录
字段缓存 miss → 查 DB → 写主键缓存（完整记录）+ 写字段缓存（PK）→ 返回
主键缓存 miss → 查 DB by PK → 写主键缓存 → 返回
PK 不存在 → 清理字段缓存 → 返回 ErrNotFound
```

**字段缓存只存主键**，完整数据从主键缓存获取，减少内存占用。

## 写入操作

### 单条

| 方法 | 自动失效 | 说明 |
|------|---------|------|
| `Create(ctx, v)` | — | 插入 |
| `CreateWithFields(ctx, v, model, fieldValues)` | 字段缓存 | 插入 + 失效指定字段缓存 |
| `Save(ctx, v)` | 主键缓存 | upsert（`*T` 需实现 `Model`） |
| `UpdateByID(ctx, model, updates)` | 主键缓存 | 按 ID 更新 |
| `DeleteByID(ctx, model)` | 主键缓存 | 按 ID 删除 |

### 批量

| 方法 | 自动失效 | 说明 |
|------|---------|------|
| `CreateBatch(ctx, models)` | 主键缓存 | 批量插入 |
| `CreateBatchWithFields(ctx, models, fieldValues)` | 主键+字段 | 批量插入 + 失效字段缓存 |
| `DeleteBatch(ctx, conds...)` | 主键缓存 | 先查后删，自动失效 |
| `UpdateBatch(ctx, updates, conds...)` | 主键缓存 | 先查后更，自动失效 |

### 字段缓存失效示例

```go
// 插入前曾查询 name=新用户 不存在，缓存了空值
// 插入后需要失效该空值缓存
user := &User{ID: 1, Name: "新用户", Age: 25}
m := &User{ID: 1}
q.CreateWithFields(ctx, user, m, map[string]any{
    "name": "新用户",  // 失效 cache:users:name:新用户
})
```

## 缓存失效

### 自动失效

| 操作 | 主键缓存 | 字段缓存 |
|------|---------|---------|
| Update / UpdateByID | ✅ | — |
| Delete / DeleteByID | ✅ | — |
| InsertBatch | ✅ | — |
| InsertBatchWithFields | ✅ | ✅ |
| DeleteBatch | ✅ | — |
| UpdateBatch | ✅ | — |
| Save | ✅ | — |

### 手动失效

| 方法 | 说明 |
|------|------|
| `Invalidate(keys...)` | 批量失效指定 key |
| `InvalidateByPK(model)` | 失效主键缓存 |
| `InvalidateByPKs(models...)` | 批量失效主键缓存 |
| `InvalidateByField(model, field, value)` | 失效字段缓存 |
| `InvalidateByFields(model, fieldValues)` | 批量失效字段缓存 |
| `InvalidateByComposite(model, fields, values)` | 失效复合索引缓存 |
| `InvalidateKeys(keys...)` | 批量失效（≤8 串行，>8 并发，上限 50） |
| `PrimaryKey(model)` | 返回主键缓存 key |
| `FieldKey(model, field, value)` | 返回字段缓存 key |

## 并发安全

- `ProtectedCache` 的 `GetOrLoad` 内置 singleflight，同一 key 并发请求只查一次 DB
- `InvalidateKeys` 大批次（>8）自动并发 `Del`，上限 50 并发
- 底层 `go-redis`/`freecache`/`bigcache` 均线程安全
- 已通过 `go test -race` 全量竞态检测

## 环境变量

通过主库 `ai` 自动初始化时，缓存配置见 `pkg/cache` 文档。crud 包本身无需额外环境变量。

## 完整示例

```go
package main

import (
    "context"
    "fmt"
    "time"

    "cnbattle.com/ai"
    "cnbattle.com/ai/pkg/crud"
)

type User struct {
    crud.CachedModel[int64]
    Name string `json:"name"`
    Age  int    `json:"age"`
}

func (User) TableName() string { return "users" }

func main() {
    ctx := context.Background()
    q := crud.NewRepo[User, User](ai.DB, ai.Cache, 5*time.Minute)

    // 创建
    user := &User{ID: 1, Name: "张三", Age: 25}
    q.Create(ctx, user)

    // read-through 查询
    found, err := q.FindByID(ctx, &User{ID: 1})
    if err != nil {
        panic(err)
    }
    fmt.Printf("user: %+v\n", found)

    // 按字段查询（字段缓存只存 PK，内存友好）
    found, err = q.FindByField(ctx, &User{}, "name", "张三")

    // 更新（自动失效主键缓存）
    q.UpdateByID(ctx, &User{ID: 1}, map[string]any{"age": 26})

    // 插入并失效字段缓存
    q.CreateWithFields(ctx, &User{ID: 2, Name: "李四", Age: 30}, User{ID: 2}, map[string]any{
        "name": "李四",
    })

    // 批量删除（自动先查后删+失效）
    q.DeleteBatch(ctx, "age < ?", 40)

    // 手动批量失效
    q.InvalidatePKs(User{ID: 1}, User{ID: 2})

    // 手动失效字段缓存
    q.InvalidateFields(User{ID: 1}, map[string]any{"name": "张三"})
}
```
