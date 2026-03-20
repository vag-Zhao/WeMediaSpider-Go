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
