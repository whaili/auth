package oauth

import (
	"context"
	"net/http"
	"strings"
	"sync"
	"time"
)

type ctxHeaderKey struct{}

// CtxHeaderMarker header key in http context
var CtxHeaderMarker = &ctxHeaderKey{}

// SafeHeader wraps http header with concurrency-safe
type SafeHeader struct {
	sync.RWMutex
	http.Header
}

// SetRequestHeader sets header for request using data in context
func SetRequestHeader(ctx context.Context, req *http.Request) {
	if ctx == nil || req == nil || ctx.Value(CtxHeaderMarker) == nil {
		return
	}

	if m, ok := ctx.Value(CtxHeaderMarker).(*SafeHeader); ok {
		m.RLock()
		defer m.RUnlock()

		for k, v := range m.Header {
			req.Header.Set(k, strings.Join(v, ","))
		}
	}
}

func Seconds() int64 {
	return time.Now().Unix()
}
