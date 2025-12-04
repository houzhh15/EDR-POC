// Package log 提供 EDR Agent 的日志系统封装
package log

import (
	"os"
	"path/filepath"
	"sync"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

// LogConfig 日志配置
type LogConfig struct {
	Level      string // 日志级别: debug, info, warn, error
	Output     string // 输出方式: console, file, both
	FilePath   string // 日志文件路径
	MaxSizeMB  int    // 单文件最大大小(MB)
	MaxBackups int    // 最大保留文件数
	MaxAgeDays int    // 最大保留天数
}

// Logger 封装 zap.Logger 提供统一日志接口
type Logger struct {
	zap    *zap.Logger
	sugar  *zap.SugaredLogger
	level  zap.AtomicLevel
	config LogConfig
}

var (
	globalLogger *Logger
	globalMu     sync.RWMutex
)

// NewLogger 根据配置创建 Logger
func NewLogger(cfg LogConfig) (*Logger, error) {
	// 解析日志级别
	level := zap.NewAtomicLevel()
	if err := level.UnmarshalText([]byte(cfg.Level)); err != nil {
		// 默认 info 级别
		level.SetLevel(zapcore.InfoLevel)
	}

	// 配置编码器
	encoderCfg := zapcore.EncoderConfig{
		TimeKey:        "time",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		FunctionKey:    zapcore.OmitKey,
		MessageKey:     "msg",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.CapitalLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.StringDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}

	// 创建输出
	var writeSyncer zapcore.WriteSyncer
	switch cfg.Output {
	case "console":
		writeSyncer = zapcore.AddSync(os.Stdout)
	case "file":
		writeSyncer = createFileWriter(cfg)
	case "both":
		writeSyncer = zapcore.NewMultiWriteSyncer(
			zapcore.AddSync(os.Stdout),
			createFileWriter(cfg),
		)
	default:
		writeSyncer = zapcore.AddSync(os.Stdout)
	}

	// 创建核心
	core := zapcore.NewCore(
		zapcore.NewJSONEncoder(encoderCfg),
		writeSyncer,
		level,
	)

	// 构建 Logger
	zapLogger := zap.New(core,
		zap.AddCaller(),
		zap.AddCallerSkip(1),
	)

	logger := &Logger{
		zap:    zapLogger,
		sugar:  zapLogger.Sugar(),
		level:  level,
		config: cfg,
	}

	return logger, nil
}

// createFileWriter 创建文件输出器（支持日志轮转）
func createFileWriter(cfg LogConfig) zapcore.WriteSyncer {
	// 确保日志目录存在
	if dir := filepath.Dir(cfg.FilePath); dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			// 如果创建目录失败，输出错误到 stderr 但不中断程序
			// lumberjack 会在写入时再次尝试
			_, _ = os.Stderr.WriteString("Warning: failed to create log directory: " + err.Error() + "\n")
		}
	}

	return zapcore.AddSync(&lumberjack.Logger{
		Filename:   cfg.FilePath,
		MaxSize:    cfg.MaxSizeMB,
		MaxBackups: cfg.MaxBackups,
		MaxAge:     cfg.MaxAgeDays,
		Compress:   true,
	})
}

// SetLevel 动态调整日志级别
func (l *Logger) SetLevel(level string) error {
	return l.level.UnmarshalText([]byte(level))
}

// GetLevel 获取当前日志级别
func (l *Logger) GetLevel() string {
	return l.level.Level().String()
}

// WithModule 创建带模块名的子 Logger
func (l *Logger) WithModule(module string) *Logger {
	return &Logger{
		zap:    l.zap.With(zap.String("module", module)),
		sugar:  l.sugar.With("module", module),
		level:  l.level,
		config: l.config,
	}
}

// WithField 创建带自定义字段的子 Logger
func (l *Logger) WithField(key string, value interface{}) *Logger {
	return &Logger{
		zap:    l.zap.With(zap.Any(key, value)),
		sugar:  l.sugar.With(key, value),
		level:  l.level,
		config: l.config,
	}
}

// Debug 输出 debug 级别日志
func (l *Logger) Debug(msg string, fields ...zap.Field) {
	l.zap.Debug(msg, fields...)
}

// Info 输出 info 级别日志
func (l *Logger) Info(msg string, fields ...zap.Field) {
	l.zap.Info(msg, fields...)
}

// Warn 输出 warn 级别日志
func (l *Logger) Warn(msg string, fields ...zap.Field) {
	l.zap.Warn(msg, fields...)
}

// Error 输出 error 级别日志
func (l *Logger) Error(msg string, fields ...zap.Field) {
	l.zap.Error(msg, fields...)
}

// Fatal 输出 fatal 级别日志并退出
func (l *Logger) Fatal(msg string, fields ...zap.Field) {
	l.zap.Fatal(msg, fields...)
}

// Debugf 格式化输出 debug 级别日志
func (l *Logger) Debugf(format string, args ...interface{}) {
	l.sugar.Debugf(format, args...)
}

// Infof 格式化输出 info 级别日志
func (l *Logger) Infof(format string, args ...interface{}) {
	l.sugar.Infof(format, args...)
}

// Warnf 格式化输出 warn 级别日志
func (l *Logger) Warnf(format string, args ...interface{}) {
	l.sugar.Warnf(format, args...)
}

// Errorf 格式化输出 error 级别日志
func (l *Logger) Errorf(format string, args ...interface{}) {
	l.sugar.Errorf(format, args...)
}

// Fatalf 格式化输出 fatal 级别日志并退出
func (l *Logger) Fatalf(format string, args ...interface{}) {
	l.sugar.Fatalf(format, args...)
}

// Sync 刷新日志缓冲区
func (l *Logger) Sync() error {
	return l.zap.Sync()
}

// Close 关闭 Logger
func (l *Logger) Close() error {
	return l.Sync()
}

// ============================================================
// 全局 Logger 接口
// ============================================================

// Init 初始化全局 Logger
func Init(cfg LogConfig) error {
	logger, err := NewLogger(cfg)
	if err != nil {
		return err
	}

	globalMu.Lock()
	defer globalMu.Unlock()
	globalLogger = logger
	return nil
}

// Global 获取全局 Logger
func Global() *Logger {
	globalMu.RLock()
	defer globalMu.RUnlock()

	if globalLogger == nil {
		// 返回默认 Logger
		logger, _ := NewLogger(LogConfig{
			Level:  "info",
			Output: "console",
		})
		return logger
	}
	return globalLogger
}

// SetGlobalLevel 设置全局日志级别
func SetGlobalLevel(level string) error {
	globalMu.RLock()
	defer globalMu.RUnlock()

	if globalLogger == nil {
		return nil
	}
	return globalLogger.SetLevel(level)
}

// Debug 全局 debug 日志
func Debug(msg string, fields ...zap.Field) {
	Global().Debug(msg, fields...)
}

// Info 全局 info 日志
func Info(msg string, fields ...zap.Field) {
	Global().Info(msg, fields...)
}

// Warn 全局 warn 日志
func Warn(msg string, fields ...zap.Field) {
	Global().Warn(msg, fields...)
}

// Error 全局 error 日志
func Error(msg string, fields ...zap.Field) {
	Global().Error(msg, fields...)
}

// Debugf 全局格式化 debug 日志
func Debugf(format string, args ...interface{}) {
	Global().Debugf(format, args...)
}

// Infof 全局格式化 info 日志
func Infof(format string, args ...interface{}) {
	Global().Infof(format, args...)
}

// Warnf 全局格式化 warn 日志
func Warnf(format string, args ...interface{}) {
	Global().Warnf(format, args...)
}

// Errorf 全局格式化 error 日志
func Errorf(format string, args ...interface{}) {
	Global().Errorf(format, args...)
}
