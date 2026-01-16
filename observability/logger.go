package observability

import (
	"context"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
)

// ContextKey 上下文键类型
type ContextKey string

const (
	RequestIDKey ContextKey = "request_id"
	AccountIDKey ContextKey = "account_id"
	TokenIDKey   ContextKey = "token_id"
)

// Logger 全局日志实例
var Logger *slog.Logger

// logFile 日志文件句柄（用于关闭）
var logFile *os.File

// InitLogger 初始化日志系统
// level: 日志级别 (debug, info, warn, error)
// format: 日志格式 (json, text)
// output: 自定义输出（nil 则输出到 stdout）
func InitLogger(level string, format string, output io.Writer) {
	if output == nil {
		output = os.Stdout
	}

	// 解析日志级别
	var logLevel slog.Level
	switch strings.ToLower(level) {
	case "debug":
		logLevel = slog.LevelDebug
	case "info":
		logLevel = slog.LevelInfo
	case "warn", "warning":
		logLevel = slog.LevelWarn
	case "error":
		logLevel = slog.LevelError
	default:
		logLevel = slog.LevelInfo
	}

	opts := &slog.HandlerOptions{
		Level:     logLevel,
		AddSource: logLevel == slog.LevelDebug, // Debug 模式显示源码位置
	}

	var handler slog.Handler
	if strings.ToLower(format) == "json" {
		handler = slog.NewJSONHandler(output, opts)
	} else {
		handler = slog.NewTextHandler(output, opts)
	}

	Logger = slog.New(handler)
	slog.SetDefault(Logger)
}

// InitLoggerWithFile 初始化日志系统，同时输出到 stdout 和文件
// level: 日志级别 (debug, info, warn, error)
// format: 日志格式 (json, text)
// filePath: 日志文件路径（空字符串则不写文件）
func InitLoggerWithFile(level string, format string, filePath string) error {
	var output io.Writer = os.Stdout

	if filePath != "" {
		// 确保日志目录存在
		dir := filepath.Dir(filePath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}

		// 打开日志文件（追加模式）
		file, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			return err
		}
		logFile = file

		// 同时输出到 stdout 和文件
		output = io.MultiWriter(os.Stdout, file)
	}

	InitLogger(level, format, output)
	return nil
}

// CloseLogger 关闭日志文件（程序退出时调用）
func CloseLogger() {
	if logFile != nil {
		logFile.Close()
		logFile = nil
	}
}

// WithContext 从 context 提取公共字段创建新的 logger
func WithContext(ctx context.Context) *slog.Logger {
	if Logger == nil {
		InitLogger("info", "text", nil)
	}
	logger := Logger

	if reqID, ok := ctx.Value(RequestIDKey).(string); ok && reqID != "" {
		logger = logger.With(slog.String("request_id", reqID))
	}

	if accountID, ok := ctx.Value(AccountIDKey).(string); ok && accountID != "" {
		logger = logger.With(slog.String("account_id", accountID))
	}

	if tokenID, ok := ctx.Value(TokenIDKey).(string); ok && tokenID != "" {
		logger = logger.With(slog.String("token_id", tokenID))
	}

	return logger
}

// LogInfo 便捷方法：记录 Info 日志
func LogInfo(ctx context.Context, msg string, args ...any) {
	WithContext(ctx).Info(msg, args...)
}

// LogError 便捷方法：记录 Error 日志
func LogError(ctx context.Context, msg string, err error, args ...any) {
	allArgs := append([]any{slog.String("error", err.Error())}, args...)
	WithContext(ctx).Error(msg, allArgs...)
}

// LogWarn 便捷方法：记录 Warn 日志
func LogWarn(ctx context.Context, msg string, args ...any) {
	WithContext(ctx).Warn(msg, args...)
}

// LogDebug 便捷方法：记录 Debug 日志
func LogDebug(ctx context.Context, msg string, args ...any) {
	WithContext(ctx).Debug(msg, args...)
}

// SetAccountIDToContext 设置 AccountID 到上下文
func SetAccountIDToContext(ctx context.Context, accountID string) context.Context {
	return context.WithValue(ctx, AccountIDKey, accountID)
}

// SetTokenIDToContext 设置 TokenID 到上下文
func SetTokenIDToContext(ctx context.Context, tokenID string) context.Context {
	return context.WithValue(ctx, TokenIDKey, tokenID)
}

// GetRequestID 从上下文获取 Request ID
func GetRequestID(ctx context.Context) string {
	if reqID, ok := ctx.Value(RequestIDKey).(string); ok {
		return reqID
	}
	return ""
}
