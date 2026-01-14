package observability

import (
	"context"
	"io"
	"log/slog"
	"os"
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

// InitLogger 初始化日志系统
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
