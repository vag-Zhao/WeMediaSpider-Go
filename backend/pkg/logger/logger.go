package logger

import (
	"io"
	"os"
	"path/filepath"

	"github.com/sirupsen/logrus"
)

var Log *logrus.Logger

// Init 初始化日志系统
func Init() error {
	Log = logrus.New()

	// 设置日志格式
	Log.SetFormatter(&logrus.TextFormatter{
		FullTimestamp:   true,
		TimestampFormat: "2006-01-02 15:04:05",
		ForceColors:     false,
	})

	// 设置日志级别
	Log.SetLevel(logrus.InfoLevel)

	// 初始化日志缓冲区（保留最近 1000 条）
	InitBuffer(1000)

	// TODO: 将在 Task 3 中替换
	// Log.AddHook(&BufferHook{})

	// 创建日志目录
	homeDir, _ := os.UserHomeDir()
	logDir := filepath.Join(homeDir, ".wemediaspider", "logs")
	os.MkdirAll(logDir, 0755)

	// 创建日志文件
	logFile := filepath.Join(logDir, "app.log")
	file, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return err
	}

	// 同时输出到文件和控制台
	multiWriter := io.MultiWriter(os.Stdout, file)
	Log.SetOutput(multiWriter)

	return nil
}

// TODO: 将在 Task 3 中替换
// BufferHook 日志缓冲区 Hook
// type BufferHook struct{}
//
// func (h *BufferHook) Levels() []logrus.Level {
// 	return logrus.AllLevels
// }
//
// func (h *BufferHook) Fire(entry *logrus.Entry) error {
// 	if buffer != nil {
// 		logLine := fmt.Sprintf("[%s] %s: %s",
// 			entry.Time.Format("2006-01-02 15:04:05"),
// 			entry.Level.String(),
// 			entry.Message,
// 		)
// 		buffer.AddLog(logLine)
// 	}
// 	return nil
// }

// Info 记录信息日志
func Info(args ...interface{}) {
	if Log != nil {
		Log.Info(args...)
	}
}

// Infof 记录格式化信息日志
func Infof(format string, args ...interface{}) {
	if Log != nil {
		Log.Infof(format, args...)
	}
}

// Warn 记录警告日志
func Warn(args ...interface{}) {
	if Log != nil {
		Log.Warn(args...)
	}
}

// Warnf 记录格式化警告日志
func Warnf(format string, args ...interface{}) {
	if Log != nil {
		Log.Warnf(format, args...)
	}
}

// Error 记录错误日志
func Error(args ...interface{}) {
	if Log != nil {
		Log.Error(args...)
	}
}

// Errorf 记录格式化错误日志
func Errorf(format string, args ...interface{}) {
	if Log != nil {
		Log.Errorf(format, args...)
	}
}

// Debug 记录调试日志
func Debug(args ...interface{}) {
	if Log != nil {
		Log.Debug(args...)
	}
}

// Debugf 记录格式化调试日志
func Debugf(format string, args ...interface{}) {
	if Log != nil {
		Log.Debugf(format, args...)
	}
}

// SetLevel 设置日志级别
func SetLevel(level string) {
	if Log == nil {
		return
	}

	switch level {
	case "debug":
		Log.SetLevel(logrus.DebugLevel)
	case "info":
		Log.SetLevel(logrus.InfoLevel)
	case "warn":
		Log.SetLevel(logrus.WarnLevel)
	case "error":
		Log.SetLevel(logrus.ErrorLevel)
	default:
		Log.SetLevel(logrus.InfoLevel)
	}
}
