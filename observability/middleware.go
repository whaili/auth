package observability

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
)

// MetricsMiddleware Prometheus 指标收集中间件
func MetricsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// 在途请求计数
		HTTPRequestsInFlight.Inc()
		defer HTTPRequestsInFlight.Dec()

		// 包装 ResponseWriter 以捕获状态码
		wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		next.ServeHTTP(wrapped, r)

		// 计算耗时
		duration := time.Since(start).Seconds()

		// 提取路由模式（如 /api/v2/tokens/{id} 而非 /api/v2/tokens/tk_xxx）
		// 这样可以避免高基数问题
		route := mux.CurrentRoute(r)
		path := r.URL.Path
		if route != nil {
			if tpl, err := route.GetPathTemplate(); err == nil {
				path = tpl
			}
		}

		// 记录指标
		HTTPRequestsTotal.WithLabelValues(r.Method, path, strconv.Itoa(wrapped.statusCode)).Inc()
		HTTPRequestDuration.WithLabelValues(r.Method, path).Observe(duration)
	})
}

// responseWriter 包装器用于捕获状态码
type responseWriter struct {
	http.ResponseWriter
	statusCode    int
	headerWritten bool
}

func (rw *responseWriter) WriteHeader(code int) {
	if !rw.headerWritten {
		rw.statusCode = code
		rw.headerWritten = true
	}
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	if !rw.headerWritten {
		rw.headerWritten = true
	}
	return rw.ResponseWriter.Write(b)
}

// Flush 实现 http.Flusher 接口
func (rw *responseWriter) Flush() {
	if flusher, ok := rw.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}
