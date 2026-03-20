# 日志系统重构实施计划 — logrus → zap

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 将日志库从 logrus 替换为 go.uber.org/zap 严格模式，统一日志语言为中文，修复错误处理问题。

**Architecture:** 重写 `pkg/logger` 包（`logger.go` + `buffer.go`），新增 `BufferCore` 实现 `zapcore.Core` 接口替代原 `BufferHook`；三路输出：控制台 ConsoleEncoder、文件 JSONEncoder（lumberjack 轮转）、内存环形缓冲区。删除所有包级快捷函数，调用方统一使用 `logger.Log.Xxx()` 类型化字段 API。

**Tech Stack:** `go.uber.org/zap v1.27.x`、`gopkg.in/lumberjack.v2 v2.2.x`（移除 `github.com/sirupsen/logrus`）

**Spec:** `docs/superpowers/specs/2026-03-20-zap-logging-redesign.md`

---

## 文件变更清单

| 操作 | 文件 | 说明 |
|------|------|------|
| 修改 | `go.mod` / `go.sum` | 新增 zap、lumberjack；移除 logrus |
| 重写 | `backend/pkg/logger/logger.go` | Init/SetLevel/GetBuffer/Sync + 全局 Log |
| 重写 | `backend/pkg/logger/buffer.go` | BufferCore（zapcore.Core）+ LogBuffer |
| 新增 | `backend/pkg/logger/logger_test.go` | BufferCore 单元测试 |
| 修改 | `backend/pkg/errors/errors.go` | 删除 Details、新增 Cause+Unwrap |
| 新增 | `backend/pkg/errors/errors_test.go` | errors.Is 穿透测试 |
| 改写 | `backend/app/app.go` + `init.go` | logger.Init() 调用 |
| 改写 | `backend/app/scrape_handler.go` | 含 strings.Contains → errors.Is 修复（2处） |
| 改写 | `backend/app/system_handler.go` | logger 调用改写 |
| 改写 | `backend/app/data_handler.go` | logger 调用改写 |
| 改写 | `backend/app/schedule_handler.go` | logger 调用改写 |
| 改写 | `backend/internal/spider/async_scraper.go` | emoji+英文→中文 |
| 改写 | `backend/internal/spider/scraper.go` | logger 调用改写 |
| 改写 | `backend/internal/spider/login.go` | logger 调用改写 |
| 改写 | `backend/internal/spider/image_downloader.go` | logger 调用改写 |
| 改写 | `backend/internal/scheduler/cron_manager.go` | logger 调用改写 |
| 改写 | `backend/internal/scheduler/task_scheduler.go` | logger 调用改写 |
| 改写 | `backend/internal/analytics/analyzer.go` | logger 调用改写 |
| 改写 | `backend/internal/config/manager.go` + `system_config.go` + `datamanager.go` | logger 调用改写 |
| 改写 | `backend/internal/database/db.go` | logger 调用改写 |
| 改写 | `backend/internal/export/excel.go` + `csv.go` + `markdown.go` + `json.go` | emoji 日志清理 |
| 改写 | `backend/internal/autostart/autostart.go` | 英文→中文 |
| 改写 | `backend/internal/tray/tray.go` | logger 调用改写 |
| 改写 | `backend/pkg/windows/manager.go` | logger 调用改写 |
| 改写 | `backend/cmd/migrate/main.go` | logger.Init() + 调用方式 |

---

## Task 1: 添加依赖

**Files:**
- Modify: `go.mod`, `go.sum`

- [ ] **Step 1: 添加 zap 和 lumberjack**

```bash
cd G:/WeMediaSpider3
go get go.uber.org/zap@latest
go get gopkg.in/lumberjack.v2@latest
```

Expected: 无报错，`go.mod` 出现两个新依赖行。

- [ ] **Step 2: 验证**

```bash
grep -E 'uber.org/zap|lumberjack' go.mod
```

Expected: 显示 `go.uber.org/zap` 和 `gopkg.in/lumberjack.v2` 两行。

- [ ] **Step 3: 提交**

```bash
git add go.mod go.sum
git commit -m "chore: 添加 zap 和 lumberjack 依赖"
```

---

## Task 2: 重写 `pkg/logger/buffer.go`（含测试）

**Files:**
- Rewrite: `backend/pkg/logger/buffer.go`
- Create: `backend/pkg/logger/logger_test.go`

- [ ] **Step 1: 写失败测试（新建 `backend/pkg/logger/logger_test.go`）**

```go
package logger

import (
	"strings"
	"testing"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func TestBufferCoreWritePreservesFields(t *testing.T) {
	InitBuffer(10)
	enc := zapcore.NewConsoleEncoder(zap.NewDevelopmentEncoderConfig())
	core := &BufferCore{enc: enc, buf: GetBuffer(), level: zapcore.DebugLevel}

	entry := zapcore.Entry{Level: zapcore.InfoLevel, Message: "测试消息"}
	fields := []zapcore.Field{zap.String("account", "新华社")}

	if err := core.Write(entry, fields); err != nil {
		t.Fatalf("Write() error: %v", err)
	}
	logs := GetBuffer().GetLogs()
	if len(logs) != 1 {
		t.Fatalf("expected 1 log, got %d", len(logs))
	}
	if !strings.Contains(logs[0], "测试消息") {
		t.Errorf("log missing message: %s", logs[0])
	}
	if !strings.Contains(logs[0], "新华社") {
		t.Errorf("log missing field value: %s", logs[0])
	}
}

func TestBufferCoreWithReturnsCopy(t *testing.T) {
	InitBuffer(10)
	enc := zapcore.NewConsoleEncoder(zap.NewDevelopmentEncoderConfig())
	core := &BufferCore{enc: enc, buf: GetBuffer(), level: zapcore.DebugLevel}

	core2 := core.With([]zapcore.Field{zap.String("k", "v")})
	if core2 == core {
		t.Error("With() 必须返回新实例")
	}
}

func TestLogBufferMaxSize(t *testing.T) {
	InitBuffer(3)
	for i := 0; i < 5; i++ {
		GetBuffer().AddLog("line")
	}
	if len(GetBuffer().GetLogs()) != 3 {
		t.Errorf("expected 3 logs, got %d", len(GetBuffer().GetLogs()))
	}
}
```

- [ ] **Step 2: 运行测试，确认失败**

```bash
cd G:/WeMediaSpider3
go test ./backend/pkg/logger/... 2>&1 | head -20
```

Expected: 编译错误，`BufferCore undefined`。

- [ ] **Step 3: 重写 `backend/pkg/logger/buffer.go`**

```go
package logger

import (
	"sync"

	"go.uber.org/zap/zapcore"
)

// LogBuffer 日志环形缓冲区，对外 API 与旧版保持一致
type LogBuffer struct {
	logs    []string
	maxSize int
	mu      sync.RWMutex
}

var buffer *LogBuffer

func InitBuffer(maxSize int) {
	buffer = &LogBuffer{
		logs:    make([]string, 0, maxSize),
		maxSize: maxSize,
	}
}

func (lb *LogBuffer) AddLog(log string) {
	lb.mu.Lock()
	defer lb.mu.Unlock()
	lb.logs = append(lb.logs, log)
	if len(lb.logs) > lb.maxSize {
		lb.logs = lb.logs[len(lb.logs)-lb.maxSize:]
	}
}

func (lb *LogBuffer) GetLogs() []string {
	lb.mu.RLock()
	defer lb.mu.RUnlock()
	result := make([]string, len(lb.logs))
	copy(result, lb.logs)
	return result
}

func (lb *LogBuffer) GetRecentLogs(n int) []string {
	lb.mu.RLock()
	defer lb.mu.RUnlock()
	if n > len(lb.logs) {
		n = len(lb.logs)
	}
	result := make([]string, n)
	copy(result, lb.logs[len(lb.logs)-n:])
	return result
}

func (lb *LogBuffer) Clear() {
	lb.mu.Lock()
	defer lb.mu.Unlock()
	lb.logs = make([]string, 0, lb.maxSize)
}

func GetBuffer() *LogBuffer { return buffer }

// BufferCore 实现 zapcore.Core，将日志写入内存缓冲区（含完整字段）
type BufferCore struct {
	enc   zapcore.Encoder
	buf   *LogBuffer
	level zapcore.LevelEnabler
}

func (c *BufferCore) Enabled(l zapcore.Level) bool { return c.level.Enabled(l) }

// With 返回持有预编码字段副本的新实例（并发安全）
func (c *BufferCore) With(fields []zapcore.Field) zapcore.Core {
	clone := c.enc.Clone()
	for _, f := range fields {
		f.AddTo(clone)
	}
	return &BufferCore{enc: clone, buf: c.buf, level: c.level}
}

func (c *BufferCore) Check(e zapcore.Entry, ce *zapcore.CheckedEntry) *zapcore.CheckedEntry {
	if c.Enabled(e.Level) {
		return ce.AddCore(e, c)
	}
	return ce
}

func (c *BufferCore) Write(e zapcore.Entry, fields []zapcore.Field) error {
	buf, err := c.enc.EncodeEntry(e, fields)
	if err != nil {
		return err
	}
	c.buf.AddLog(buf.String())
	buf.Free()
	return nil
}

func (c *BufferCore) Sync() error { return nil }
```

- [ ] **Step 4: 运行测试，确认通过**

```bash
go test ./backend/pkg/logger/... -v
```

Expected: 3 个测试全部 PASS。

- [ ] **Step 5: 提交**

```bash
git add backend/pkg/logger/buffer.go backend/pkg/logger/logger_test.go
git commit -m "feat(logger): 新增 BufferCore 实现 zapcore.Core 接口"
```

---

## Task 3: 重写 `pkg/logger/logger.go`

**Files:**
- Rewrite: `backend/pkg/logger/logger.go`

- [ ] **Step 1: 重写文件**

```go
package logger

import (
	"os"
	"path/filepath"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/lumberjack.v2"
)

// Log 全局 logger，Init() 后可直接使用
var (
	Log         *zap.Logger
	atomicLevel zap.AtomicLevel
)

// Init 初始化日志系统。目录创建失败时降级为仅 Stdout 输出。
func Init() error {
	atomicLevel = zap.NewAtomicLevelAt(zapcore.InfoLevel)
	InitBuffer(1000)

	consoleEnc := zapcore.NewConsoleEncoder(zapcore.EncoderConfig{
		TimeKey:        "T",
		LevelKey:       "L",
		MessageKey:     "M",
		StacktraceKey:  "S",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.CapitalLevelEncoder,
		EncodeTime:     zapcore.TimeEncoderOfLayout("2006-01-02 15:04:05"),
		EncodeDuration: zapcore.StringDurationEncoder,
	})

	jsonEnc := zapcore.NewJSONEncoder(zapcore.EncoderConfig{
		TimeKey:        "ts",
		LevelKey:       "level",
		MessageKey:     "msg",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeTime:     zapcore.TimeEncoderOfLayout("2006-01-02 15:04:05"),
		EncodeDuration: zapcore.StringDurationEncoder,
	})

	// 控制台输出
	consoleCore := zapcore.NewCore(consoleEnc, zapcore.AddSync(os.Stdout), atomicLevel)

	// 内存缓冲区输出
	bufCore := &BufferCore{
		enc:   consoleEnc.Clone(),
		buf:   buffer,
		level: atomicLevel,
	}

	cores := []zapcore.Core{consoleCore, bufCore}

	// 文件输出（失败时降级）
	homeDir, err := os.UserHomeDir()
	if err != nil {
		// 降级：仅 Stdout + 缓冲区
		Log = zap.New(zapcore.NewTee(cores...))
		Log.Warn("无法获取用户目录，日志文件输出已禁用")
		return nil
	}

	logDir := filepath.Join(homeDir, ".wemediaspider", "logs")
	if err := os.MkdirAll(logDir, 0755); err != nil {
		// 降级：仅 Stdout + 缓冲区
		Log = zap.New(zapcore.NewTee(cores...))
		Log.Warn("无法创建日志目录，日志文件输出已禁用", zap.Error(err))
		return nil
	}

	fileWriter := &lumberjack.Logger{
		Filename:   filepath.Join(logDir, "app.log"),
		MaxSize:    10,
		MaxBackups: 5,
		MaxAge:     30,
		Compress:   true,
		LocalTime:  true,
	}
	fileCore := zapcore.NewCore(jsonEnc, zapcore.AddSync(fileWriter), atomicLevel)
	cores = append(cores, fileCore)

	Log = zap.New(zapcore.NewTee(cores...))
	return nil
}

// SetLevel 运行时动态调整日志级别（"debug"|"info"|"warn"|"error"）
func SetLevel(level string) {
	switch level {
	case "debug":
		atomicLevel.SetLevel(zapcore.DebugLevel)
	case "info":
		atomicLevel.SetLevel(zapcore.InfoLevel)
	case "warn":
		atomicLevel.SetLevel(zapcore.WarnLevel)
	case "error":
		atomicLevel.SetLevel(zapcore.ErrorLevel)
	default:
		atomicLevel.SetLevel(zapcore.InfoLevel)
	}
}

// Sync 刷新所有缓冲写入，程序退出前调用
func Sync() {
	if Log != nil {
		_ = Log.Sync()
	}
}
```

- [ ] **Step 2: 构建验证**

```bash
cd G:/WeMediaSpider3
go build ./backend/pkg/logger/...
```

Expected: 无编译错误。

- [ ] **Step 3: 运行测试**

```bash
go test ./backend/pkg/logger/... -v
```

Expected: 全部 PASS。

- [ ] **Step 4: 提交**

```bash
git add backend/pkg/logger/logger.go
git commit -m "feat(logger): 重写 logger.go，替换 logrus 为 zap"
```

---

## Task 4: 修改 `pkg/errors/errors.go`（含测试）

**Files:**
- Modify: `backend/pkg/errors/errors.go`
- Create: `backend/pkg/errors/errors_test.go`

- [ ] **Step 1: 写失败测试（新建 `backend/pkg/errors/errors_test.go`）**

```go
package errors

import (
	"errors"
	"testing"
)

func TestAppErrorUnwrap(t *testing.T) {
	cause := ErrNotLoggedIn
	appErr := NewAppError("LOGIN_001", "操作失败", cause)

	if !errors.Is(appErr, ErrNotLoggedIn) {
		t.Error("errors.Is 应能穿透 AppError 找到 cause")
	}
}

func TestAppErrorNilCause(t *testing.T) {
	appErr := NewAppError("EXPORT_001", "导出失败", nil)
	if appErr.Unwrap() != nil {
		t.Error("Unwrap() 应返回 nil")
	}
}

func TestAppErrorMessage(t *testing.T) {
	appErr := NewAppError("CFG_001", "配置无效", nil)
	if appErr.Error() != "配置无效" {
		t.Errorf("Error() = %q, want %q", appErr.Error(), "配置无效")
	}
}
```

- [ ] **Step 2: 运行测试，确认失败**

```bash
go test ./backend/pkg/errors/... 2>&1 | head -20
```

Expected: 编译错误，`NewAppError` 参数个数不匹配。

- [ ] **Step 3: 修改 `backend/pkg/errors/errors.go`**

```go
package errors

import "errors"

var (
	// 登录相关错误
	ErrNotLoggedIn  = errors.New("未登录")
	ErrLoginFailed  = errors.New("登录失败")
	ErrTokenExpired = errors.New("Token 已过期")
	ErrInvalidToken = errors.New("无效的 Token")

	// 爬取相关错误
	ErrAccountNotFound = errors.New("未找到公众号")
	ErrNoArticles      = errors.New("没有找到文章")
	ErrScrapeFailed    = errors.New("爬取失败")
	ErrScrapeCancelled = errors.New("爬取已取消")

	// 导出相关错误
	ErrExportFailed  = errors.New("导出失败")
	ErrInvalidFormat = errors.New("不支持的导出格式")

	// 配置相关错误
	ErrConfigNotFound = errors.New("配置文件不存在")
	ErrInvalidConfig  = errors.New("无效的配置")
)

// AppError 应用错误（支持 errors.Is/As 穿透）
type AppError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Cause   error  `json:"-"` // 包装的原始错误，不序列化
}

func (e *AppError) Error() string  { return e.Message }
func (e *AppError) Unwrap() error  { return e.Cause }

// NewAppError 创建应用错误。cause 为 nil 表示无底层错误。
func NewAppError(code, message string, cause error) *AppError {
	return &AppError{Code: code, Message: message, Cause: cause}
}
```

> 注意：`Details string` 字段已删除。若现有代码中有 `NewAppError(code, msg, detailsString)` 调用，需将 `detailsString` 融入 `message` 或改为 `errors.New(detailsString)`。

- [ ] **Step 3b: 扫描并修复所有 `NewAppError` 调用方**

```bash
grep -rn 'NewAppError(' backend/ --include='*.go'
```

逐一检查每处调用，将第三参数从 `string` 改为 `error`（若无底层错误传 `nil`）。例如：

```go
// 改前
NewAppError("EXPORT_001", "导出失败", "文件已存在")

// 改后
NewAppError("EXPORT_001", "导出失败：文件已存在", nil)
// 或
NewAppError("EXPORT_001", "导出失败", errors.New("文件已存在"))
```

修改后执行：

```bash
go build ./...
```

Expected: 无编译错误（`NewAppError` 类型不匹配会产生编译错误，确认全部修复后方可继续）。

- [ ] **Step 4: 运行测试，确认通过**

```bash
go test ./backend/pkg/errors/... -v
```

Expected: 3 个测试全部 PASS。

- [ ] **Step 5: 提交**

```bash
git add backend/pkg/errors/errors.go backend/pkg/errors/errors_test.go
git commit -m "feat(errors): AppError 添加 Unwrap()，支持 errors.Is 穿透"
```

---

## Task 5: 改写 `backend/app/` 下的 handler 文件

**Files:**
- Modify: `backend/app/app.go`, `backend/app/init.go`, `backend/app/scrape_handler.go`, `backend/app/system_handler.go`, `backend/app/data_handler.go`, `backend/app/schedule_handler.go`, `backend/app/export_handler.go`, `backend/app/analytics_handler.go`, `backend/app/config_handler.go`

**改写规则：**
- 将所有 `logger.Info/Infof/Warn/Warnf/Error/Errorf/Debug/Debugf(...)` 替换为 `logger.Log.Info/Warn/Error/Debug(...)`
- 格式化参数改为类型化字段：`"msg %v", val` → `"msg", zap.Xxx("key", val)`
- 消息统一中文，去掉 emoji
- import 中移除 `logrus`，按需添加 `"go.uber.org/zap"`

**`scrape_handler.go` 额外修复（两处字符串比较）：**

```go
// 修复前（约 :216 和 :281）
errMsg := err.Error()
if errMsg != "context canceled" && !strings.Contains(errMsg, "canceled") {

// 修复后
if !errors.Is(err, context.Canceled) && !errors.Is(err, context.DeadlineExceeded) {
```

同时在 `scrape_handler.go` 的 import 中添加 `"context"` 和 `"errors"`（如未已存在），移除 `"strings"` import（如不再使用）。

- [ ] **Step 1: 改写 `backend/app/app.go`**

将 `logger.Init()` 调用处保持不变（签名未变），移除 logrus import。

- [ ] **Step 2: 改写 `backend/app/init.go`**

将所有 `logger.Errorf/Infof` 改为 `logger.Log.Error/Info`，格式化参数改为 zap 字段。示例：

```go
// 改前
logger.Errorf("Failed to create cache manager: %v", err)

// 改后
logger.Log.Error("创建缓存管理器失败", zap.Error(err))
```

- [ ] **Step 3: 改写 `backend/app/scrape_handler.go`（含 errors.Is 修复）**

重点：找到两处 `strings.Contains(errMsg, "canceled")` 替换为 `errors.Is(err, context.Canceled)`。

- [ ] **Step 4: 改写剩余 handler 文件**

逐个处理 `system_handler.go`、`data_handler.go`、`schedule_handler.go`、`export_handler.go`、`analytics_handler.go`、`config_handler.go`。

- [ ] **Step 5: 构建验证**

```bash
cd G:/WeMediaSpider3
go build ./backend/app/...
```

Expected: 无编译错误。

- [ ] **Step 6: 提交**

```bash
git add backend/app/
git commit -m "refactor(app): 迁移至 zap，修复 context.Canceled 检测"
```

---

## Task 6: 改写 `backend/internal/spider/` 文件

**Files:**
- Modify: `backend/internal/spider/async_scraper.go`, `scraper.go`, `login.go`, `image_downloader.go`

**改写规则：**
- 同 Task 5，额外去掉所有 emoji（`❌`、`✅`、`🔎` 等）
- `async_scraper.go` 中的英文消息改为中文
- 移除 `"strings"` import（若仅用于 `strings.Contains` 错误检测）

示例（`async_scraper.go`）：

```go
// 改前
logger.Errorf("❌ 爬取过程发生 panic [%s]: %v", accountName, r)
logger.Warnf("⚠️  爬取被取消 [%s]", accountName)
logger.Infof("🔎 开始处理公众号 [%s]", accountName)

// 改后
logger.Log.Error("爬取过程发生 panic", zap.String("account", accountName), zap.Any("panic", r))
logger.Log.Warn("爬取已取消", zap.String("account", accountName))
logger.Log.Info("开始处理公众号", zap.String("account", accountName))
```

- [ ] **Step 1: 改写 4 个文件**

- [ ] **Step 2: 构建验证**

```bash
go build ./backend/internal/spider/...
```

Expected: 无编译错误。

- [ ] **Step 3: 提交**

```bash
git add backend/internal/spider/
git commit -m "refactor(spider): 迁移至 zap，统一中文日志，去掉 emoji"
```

---

## Task 7: 改写 `backend/internal/scheduler/` 和 `analytics/`

**Files:**
- Modify: `backend/internal/scheduler/cron_manager.go`, `task_scheduler.go`, `backend/internal/analytics/analyzer.go`

**改写规则：**
- `task_scheduler.go` 中英文消息改为中文
- 格式化参数改为 zap 字段，`task_id` 用 `zap.Uint`

示例（`task_scheduler.go`）：

```go
// 改前
logger.Warnf("Task %d is already running, skipping", taskID)
logger.Errorf("Failed to find task %d: %v", taskID, err)

// 改后
logger.Log.Warn("任务已在运行，跳过", zap.Uint("task_id", uint(taskID)))
logger.Log.Error("查询任务失败", zap.Uint("task_id", uint(taskID)), zap.Error(err))
```

- [ ] **Step 1: 改写 3 个文件**

- [ ] **Step 2: 构建验证**

```bash
go build ./backend/internal/scheduler/... ./backend/internal/analytics/...
```

- [ ] **Step 3: 提交**

```bash
git add backend/internal/scheduler/ backend/internal/analytics/
git commit -m "refactor(scheduler,analytics): 迁移至 zap，统一中文日志"
```

---

## Task 8: 改写 `config/`、`database/`、`export/`、`autostart/`、`tray/`、`windows/`、`migrate`

**Files:**
- Modify: `backend/internal/config/manager.go`, `system_config.go`, `datamanager.go`
- Modify: `backend/internal/database/db.go`
- Modify: `backend/internal/export/excel.go`, `csv.go`, `markdown.go`, `json.go`
- Modify: `backend/internal/autostart/autostart.go`
- Modify: `backend/internal/tray/tray.go`
- Modify: `backend/pkg/windows/manager.go`
- Modify: `backend/cmd/migrate/main.go`（若目录不存在则跳过）

**改写规则：**
- 同前，所有 emoji 去掉，英文消息改中文，格式化改 zap 字段
- `backend/cmd/migrate/main.go` 中 `logger.Init()` 调用签名不变，但需更新 import

示例（`excel.go`）：

```go
// 改前
logger.Infof("📊 开始导出 Excel 文件: %s (文章数: %d)", filename, len(articles))

// 改后
logger.Log.Info("开始导出 Excel 文件", zap.String("file", filename), zap.Int("count", len(articles)))
```

- [ ] **Step 1: 改写所有文件**

- [ ] **Step 2: 构建验证**

```bash
cd G:/WeMediaSpider3
go build ./...
```

Expected: 无编译错误。

- [ ] **Step 3: 提交**

```bash
git add backend/internal/config/ backend/internal/database/ backend/internal/export/
git add backend/internal/autostart/ backend/internal/tray/ backend/pkg/windows/ backend/cmd/
git commit -m "refactor: 迁移剩余模块至 zap，统一中文日志"
```

---

## Task 9: 移除 logrus，全量验证

**Files:**
- Modify: `go.mod`, `go.sum`

- [ ] **Step 1: 清理 logrus**

```bash
cd G:/WeMediaSpider3
go mod tidy
```

Expected: `go.mod` 中 `github.com/sirupsen/logrus` 行被移除。

- [ ] **Step 2: 确认 logrus 不再被引用**

```bash
grep -r 'sirupsen/logrus' backend/ --include='*.go'
```

Expected: 无输出。

- [ ] **Step 3: 全量构建**

```bash
go build ./...
```

Expected: 无编译错误。

- [ ] **Step 4: 全量 vet**

```bash
go vet ./...
```

Expected: 无告警。

- [ ] **Step 5: 运行所有测试**

```bash
go test ./backend/pkg/logger/... ./backend/pkg/errors/... ./backend/internal/... -v 2>&1 | tail -30
```

Expected: 全部 PASS，无 FAIL。

- [ ] **Step 6: 提交**

```bash
git add go.mod go.sum
git commit -m "chore: 移除 logrus，go mod tidy"
```

---

## Task 10: 运行时验证

- [ ] **Step 1: 启动应用，验证控制台输出格式**

启动后控制台应输出类似：
```
2026-03-20 10:05:01	INFO	应用启动	{"close_to_tray": true}
```

- [ ] **Step 2: 验证日志文件存在且包含 JSON 行**

```bash
cat ~/.wemediaspider/logs/app.log | head -5
```

Expected: 每行为合法 JSON，含 `ts`、`level`、`msg` 字段。

- [ ] **Step 3: 最终提交 tag**

```bash
git tag v2.1.0-logging
git push origin main --tags
```
