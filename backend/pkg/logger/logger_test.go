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
