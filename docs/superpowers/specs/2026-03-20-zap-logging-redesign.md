# 日志系统重构设计文档 — logrus → zap

**日期**：2026-03-20
**状态**：待实施
**影响范围**：27 个文件，292 处调用

---

## 1. 背景与目标

### 现状问题

| 问题 | 位置 |
|------|------|
| 日志语言混用（中文 + 英文 + emoji） | `async_scraper.go`, `task_scheduler.go` 等 |
| 错误检测用字符串比较（`strings.Contains(err.Error(), "canceled")`） | `scrape_handler.go:281` |
| `AppError` 无 `Unwrap()`，无法配合 `errors.Is()`/`errors.As()` | `pkg/errors/errors.go` |
| `BufferHook.Fire()` 丢弃所有 logrus fields | `logger.go:62` |
| `app.log` 无日志轮转，长期运行会无限增大 | `logger.go:41` |
| logrus 格式化 API（`Infof/Errorf`）无类型字段，无法按字段过滤 | 全局 |

### 重构目标

1. 替换 logrus 为 `go.uber.org/zap`，使用严格模式（`zap.Logger`）
2. 所有日志使用**中文消息 + 类型化字段**，无 emoji
3. 修复 `errors.Is()` 兼容问题（`AppError.Unwrap()`）
4. 修复 `strings.Contains(err.Error(), ...)` 为 `errors.Is()`
5. 实现日志轮转（lumberjack，10MB/5份）
6. `BufferCore` 保留完整字段，供前端日志面板消费

---

## 2. 架构设计

### 数据流

```
业务代码
  └─ logger.Info("消息", zap.String("account", name), zap.Error(err))
       └─ zap.Logger（AtomicLevel，可运行时动态调整）
            ├─ zapcore.ConsoleEncoder → os.Stdout（开发友好格式）
            ├─ zapcore.JSONEncoder   → lumberjack 文件（10MB，5份，30天）
            └─ BufferCore（自定义 zapcore.Core）→ 内存环形缓冲区（1000条）
```

### 依赖变更

```
新增：go.uber.org/zap v1.27.x
新增：gopkg.in/lumberjack.v2 v2.2.x
移除：github.com/sirupsen/logrus v1.9.4
```

---

## 3. `pkg/logger` 包设计

### 3.1 对外 API（`logger.go`）

```go
// 包级全局 logger，Init() 后可用
var Log *zap.Logger

// Init 初始化日志系统，应在 main 或 NewApp 中调用一次
func Init() error

// SetLevel 运行时动态调整日志级别（"debug"|"info"|"warn"|"error"）
func SetLevel(level string)

// GetBuffer 返回内存缓冲区（供前端日志面板使用）
func GetBuffer() *LogBuffer

// Sync 刷新所有缓冲写入（程序退出前调用）
func Sync()
```

**原有包级快捷函数（`logger.Info/Infof/Error/Errorf/Warn/Warnf/Debug/Debugf`）全部删除。** 所有调用方统一改用 `logger.Log.Xxx()` 形式。

调用方直接使用包级 `Log`：

```go
logger.Log.Info("开始爬取", zap.String("account", name))
logger.Log.Error("爬取失败", zap.String("account", name), zap.Error(err))
logger.Log.Warn("任务跳过", zap.Uint("task_id", taskID), zap.String("原因", "已在运行"))
```

### 3.2 `BufferCore`（`buffer.go` 重写）

实现 `zapcore.Core` 接口，替代原 `BufferHook`：

```go
type BufferCore struct {
    enc    zapcore.Encoder  // ConsoleEncoder，格式同控制台
    buf    *LogBuffer
    level  zapcore.LevelEnabler
}

// 实现 zapcore.Core 接口
func (c *BufferCore) Enabled(l zapcore.Level) bool
func (c *BufferCore) With(fields []zapcore.Field) zapcore.Core  // 必须返回持有预编码字段副本的新 BufferCore 实例，不得修改原实例（并发安全要求）
func (c *BufferCore) Check(e zapcore.Entry, ce *zapcore.CheckedEntry) *zapcore.CheckedEntry
func (c *BufferCore) Write(e zapcore.Entry, fields []zapcore.Field) error
func (c *BufferCore) Sync() error
```

`Write()` 将 entry 编码为字符串后存入 `LogBuffer`，**包含所有字段**。`With()` 实现时需克隆编码器（`enc.Clone()`）并将字段预编码到克隆体，再返回新 `BufferCore`，确保多 goroutine 并发调用安全。

`LogBuffer` 的对外 API 保持不变：
- `GetLogs() []string`
- `GetRecentLogs(n int) []string`
- `Clear()`

### 3.3 输出格式

| 输出目标 | 编码器 | 示例 |
|----------|--------|------|
| 控制台（Stdout） | ConsoleEncoder | `2026-03-20 10:05:01	INFO	开始爬取	{"account": "新华社"}` |
| 文件（JSON） | JSONEncoder | `{"ts":"...","level":"info","msg":"开始爬取","account":"新华社"}` |
| 内存缓冲区 | ConsoleEncoder | 同控制台格式 |

---

## 4. 错误处理规范

### 4.1 `AppError` 添加 `Unwrap()`（破坏性变更）

当前 `AppError` 只有三个 `string` 字段，无法包装底层错误。需新增 `Cause error` 字段并修改构造函数签名。这是破坏性变更，所有 `NewAppError(code, message, details)` 调用方需同步修改。

```go
// 修改后的结构体
type AppError struct {
    Code    string `json:"code"`
    Message string `json:"message"`
    Details string `json:"details,omitempty"`
    Cause   error  `json:"-"`  // 新增：包装的原始错误，不序列化
}

func (e *AppError) Error() string { return e.Message }
func (e *AppError) Unwrap() error { return e.Cause } // 新增：支持 errors.Is/As 穿透

// 新签名：cause 替换原 details 参数
func NewAppError(code, message string, cause error) *AppError {
    return &AppError{Code: code, Message: message, Cause: cause}
}
```

> **`Details` 字段处理**：`Details` 字段从结构体中删除。现有调用方中 `details` 参数若携带有意义的额外上下文，应将其合并到 `message` 中（如 `"导出失败: 磁盘空间不足"`），或通过 `cause` 错误链传递。
>
> **调用方迁移**：搜索所有 `NewAppError(` 调用，评估原第三个 `details string` 参数的内容：若是空字符串或冗余信息，直接传 `nil`；若有实质内容，将其融入 `message` 或构造为 `errors.New(details)` 传入 `cause`。

### 4.2 取消检测修正

将 `context.Canceled` 和 `context.DeadlineExceeded` 的字符串比较统一改为 `errors.Is()`：

```go
// 修改前（scrape_handler.go:281）
errMsg := err.Error()
if errMsg != "context canceled" && !strings.Contains(errMsg, "canceled") {
    runtime.EventsEmit(a.ctx, "image:error", ...)
}

// 修改后
if !errors.Is(err, context.Canceled) && !errors.Is(err, context.DeadlineExceeded) {
    runtime.EventsEmit(a.ctx, "image:error", ...)
}
```

同理，`async_scraper.go` 中通过 `ctx.Err()` 返回的取消错误也应使用 `errors.Is()` 判断，而非字符串比较。

---

## 5. 调用方改写规范

### 5.1 改写映射

| 原调用 | 新调用 |
|--------|--------|
| `logger.Info("msg")` | `logger.Log.Info("msg")` |
| `logger.Infof("msg %s", v)` | `logger.Log.Info("msg", zap.String("key", v))` |
| `logger.Errorf("failed: %v", err)` | `logger.Log.Error("操作失败", zap.Error(err))` |
| `logger.Warnf("task %d skip", id)` | `logger.Log.Warn("任务跳过", zap.Uint("task_id", id))` |

### 5.2 关键字段规范

| 场景 | 必须携带字段 |
|------|--------------|
| 爬取操作 | `zap.String("account", name)` |
| 定时任务 | `zap.Uint("task_id", id)`, `zap.String("task_name", name)` |
| 数据库操作失败 | `zap.Error(err)` |
| HTTP 请求失败 | `zap.String("url", url)`, `zap.Error(err)` |
| 文件操作 | `zap.String("path", path)`, `zap.Error(err)` |

### 5.3 日志语言规范

- 所有消息使用中文，无 emoji
- 字段 key 使用英文小写下划线（`task_id`、`account`、`url`）
- 错误原因通过 `zap.Error(err)` 字段传递，消息本身不拼接错误内容

**示例**：
```go
// 错误写法
logger.Log.Error(fmt.Sprintf("爬取失败: %v", err))

// 正确写法
logger.Log.Error("爬取失败", zap.String("account", name), zap.Error(err))
```

---

## 6. 文件轮转配置

使用 `gopkg.in/lumberjack.v2`（import path：`gopkg.in/lumberjack.v2`）：

```go
&lumberjack.Logger{
    Filename:   filepath.Join(homeDir, ".wemediaspider", "logs", "app.log"),
    MaxSize:    10,   // MB
    MaxBackups: 5,    // 保留最近 5 个备份
    MaxAge:     30,   // 天
    Compress:   true, // gzip 压缩旧文件
    LocalTime:  true,
}
```

**目录创建**：在初始化 lumberjack 之前，显式调用 `os.MkdirAll(logDir, 0755)` 创建 `logs/` 子目录，不依赖 lumberjack 的自动创建行为。若 `MkdirAll` 失败，同样触发降级策略。

**降级策略**：若 `os.UserHomeDir()` 返回错误或 `os.MkdirAll` 失败，文件输出降级为仅 Stdout（不创建 lumberjack Writer），`Init()` 仍返回 `nil`，程序正常启动，但不写日志文件。降级时通过 `zap.Logger` 输出一条 Warn 日志说明原因。

---

## 7. 受影响文件清单

### 核心重写（2 个）
- `backend/pkg/logger/logger.go`
- `backend/pkg/logger/buffer.go`

### 错误处理修改（1 个）
- `backend/pkg/errors/errors.go`

### 调用方改写（24 个文件，292 处）

| 目录 | 文件 |
|------|------|
| `backend/app/` | `app.go`, `init.go`, `scrape_handler.go`, `system_handler.go`, `data_handler.go`, `analytics_handler.go`, `config_handler.go`, `export_handler.go`, `schedule_handler.go` |
| `backend/internal/spider/` | `async_scraper.go`, `scraper.go`, `login.go`, `image_downloader.go` |
| `backend/internal/scheduler/` | `cron_manager.go`, `task_scheduler.go` |
| `backend/internal/analytics/` | `analyzer.go` |
| `backend/internal/config/` | `manager.go`, `system_config.go`, `datamanager.go` |
| `backend/internal/database/` | `db.go` |
| `backend/internal/export/` | `csv.go`, `excel.go`, `json.go`, `markdown.go` |
| `backend/internal/autostart/` | `autostart.go` |
| `backend/internal/tray/` | `tray.go` |
| `backend/pkg/windows/` | `manager.go` |
| `backend/cmd/migrate/` | `main.go` |

---

## 8. 测试策略

- `pkg/logger` 包：单元测试验证 `BufferCore` 的 `Write()` 保留字段
- `pkg/errors` 包：单元测试验证 `errors.Is(wrapped, ErrXxx)` 穿透
- 集成验证：`go build ./...` 无编译错误，`go vet ./...` 无告警
- 运行时验证：启动应用，确认控制台输出格式正确，`~/.wemediaspider/logs/app.log` 包含 JSON 行

---

## 9. 不在范围内

- 不替换 zap 为其他日志库
- 不修改前端日志面板的 UI 或 API
- 不修改 `LogBuffer` 的对外接口
- 不修改非日志相关的业务逻辑
