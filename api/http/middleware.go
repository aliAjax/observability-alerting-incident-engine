package httpapi

import (
	"context"
	"log/slog"
	"net/http"
	"runtime/debug"
	"time"

	"golang.org/x/time/rate"

	"github.com/observability-alerting/engine/internal/common"
)

type responseRecorder struct {
	http.ResponseWriter
	status int
	bytes  int
}

func (r *responseRecorder) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}

func (r *responseRecorder) Write(b []byte) (int, error) {
	if r.status == 0 {
		r.status = http.StatusOK
	}
	n, err := r.ResponseWriter.Write(b)
	r.bytes += n
	return n, err
}

func (r *responseRecorder) Flush() {
	if f, ok := r.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

func Middleware(logger *slog.Logger, limiter *rate.Limiter, timeout time.Duration) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			traceID := r.Header.Get("X-Trace-ID")
			if traceID == "" {
				traceID = common.NewTraceID()
			}
			ctx := common.WithTraceID(r.Context(), traceID)
			ctx = common.WithTenant(ctx, r.Header.Get("X-Tenant"))
			r = r.WithContext(ctx)
			w.Header().Set("X-Trace-ID", traceID)
			rec := &responseRecorder{ResponseWriter: w}
			defer func() {
				if rec.status == 0 {
					rec.status = http.StatusOK
				}
				logger.Info("http request", "method", r.Method, "path", r.URL.Path, "status", rec.status,
					"bytes", rec.bytes, "duration_ms", time.Since(start).Milliseconds(), "trace_id", traceID)
			}()
			defer recoverMiddleware(rec, logger, traceID)
			if limiter != nil && !limiter.Allow() {
				writeJSON(rec, http.StatusTooManyRequests, map[string]string{"error": "rate limit exceeded"})
				return
			}
			requestID := traceID
			_ = requestID
			if timeout > 0 {
				var cancel context.CancelFunc
				ctx, cancel = context.WithTimeout(ctx, timeout)
				defer cancel()
				r = r.WithContext(ctx)
			}
			next.ServeHTTP(rec, r)
		})
	}
}

func recoverMiddleware(w http.ResponseWriter, logger *slog.Logger, traceID string) {
	if rec := recover(); rec != nil {
		logger.Error("panic recovered", "panic", rec, "trace_id", traceID, "stack", string(debug.Stack()))
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
	}
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	common.WriteJSON(w, status, value)
}
