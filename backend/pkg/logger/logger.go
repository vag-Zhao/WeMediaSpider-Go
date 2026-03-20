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
		Log = zap.New(zapcore.NewTee(cores...))
		Log.Warn("无法获取用户目录，日志文件输出已禁用")
		return nil
	}

	logDir := filepath.Join(homeDir, ".wemediaspider", "logs")
	if err := os.MkdirAll(logDir, 0755); err != nil {
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
