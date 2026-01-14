package observability

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"net/http"
	"time"
)

const (
	// RequestIDHeader HTTP 请求头名称
	RequestIDHeader = "X-Request-ID"
)

// RequestTrackingMiddleware 请求追踪中间件
// 生成唯一的 Request ID，记录请求开始和结束日志
func RequestTrackingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// 获取或生成 Request ID
		requestID := r.Header.Get(RequestIDHeader)
		if requestID == "" {
			requestID = generateRequestID()
		}

		// 设置响应头
		w.Header().Set(RequestIDHeader, requestID)

		// 注入到 context
		ctx := context.WithValue(r.Context(), RequestIDKey, requestID)
		r = r.WithContext(ctx)

		// 包装 ResponseWriter 以捕获状态码
		wrapped := &responseWriterTracker{ResponseWriter: w, statusCode: http.StatusOK}

		// 请求开始日志
		WithContext(ctx).Debug("HTTP request started",
			slog.String("method", r.Method),
			slog.String("path", r.URL.Path),
			slog.String("remote_addr", r.RemoteAddr),
			slog.String("user_agent", r.UserAgent()),
		)

		next.ServeHTTP(wrapped, r)

		// 请求结束日志
		duration := time.Since(start)
		level := slog.LevelInfo
		if wrapped.statusCode >= 400 {
			level = slog.LevelWarn
		}
		if wrapped.statusCode >= 500 {
			level = slog.LevelError
		}

		WithContext(ctx).Log(ctx, level, "HTTP request completed",
			slog.String("method", r.Method),
			slog.String("path", r.URL.Path),
			slog.Int("status", wrapped.statusCode),
			slog.Float64("duration_ms", float64(duration.Microseconds())/1000),
		)
	})
}

// generateRequestID 生成请求 ID
// 格式: req_ + 24位随机十六进制字符
func generateRequestID() string {
	b := make([]byte, 12)
	rand.Read(b)
	return "req_" + hex.EncodeToString(b)
}

// responseWriterTracker 用于追踪响应状态码
type responseWriterTracker struct {
	http.ResponseWriter
	statusCode    int
	headerWritten bool
}

func (rw *responseWriterTracker) WriteHeader(code int) {
	if !rw.headerWritten {
		rw.statusCode = code
		rw.headerWritten = true
	}
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *responseWriterTracker) Write(b []byte) (int, error) {
	if !rw.headerWritten {
		rw.headerWritten = true
	}
	return rw.ResponseWriter.Write(b)
}

// Flush 实现 http.Flusher 接口
func (rw *responseWriterTracker) Flush() {
	if flusher, ok := rw.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}

// SetRequestIDToContext 设置 Request ID 到上下文
func SetRequestIDToContext(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, RequestIDKey, requestID)
}
