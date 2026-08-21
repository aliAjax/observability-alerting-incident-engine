package httpapi

import (
	"fmt"
	"net/http"
	"runtime"
	"sync/atomic"
	"time"
)

var (
	startTime    = time.Now()
	requestCount uint64
	errorCount   uint64
	ingestCount  uint64
)

func observeRequest(ok bool, path string) {
	atomic.AddUint64(&requestCount, 1)
	if !ok {
		atomic.AddUint64(&errorCount, 1)
	}
	if path == "/api/v1/ingest" || path == "/api/v1/ingest/batch" {
		atomic.AddUint64(&ingestCount, 1)
	}
}

func metricsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; version=0.0.4")
	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)
	fmt.Fprintf(w, "# HELP observability_uptime_seconds Process uptime in seconds.\n")
	fmt.Fprintf(w, "# TYPE observability_uptime_seconds gauge\n")
	fmt.Fprintf(w, "observability_uptime_seconds %d\n", int64(time.Since(startTime).Seconds()))
	fmt.Fprintf(w, "# HELP observability_http_requests_total Total HTTP requests.\n")
	fmt.Fprintf(w, "# TYPE observability_http_requests_total counter\n")
	fmt.Fprintf(w, "observability_http_requests_total %d\n", atomic.LoadUint64(&requestCount))
	fmt.Fprintf(w, "# HELP observability_http_errors_total Total HTTP errors.\n")
	fmt.Fprintf(w, "# TYPE observability_http_errors_total counter\n")
	fmt.Fprintf(w, "observability_http_errors_total %d\n", atomic.LoadUint64(&errorCount))
	fmt.Fprintf(w, "# HELP observability_ingest_requests_total Total ingest requests.\n")
	fmt.Fprintf(w, "# TYPE observability_ingest_requests_total counter\n")
	fmt.Fprintf(w, "observability_ingest_requests_total %d\n", atomic.LoadUint64(&ingestCount))
	fmt.Fprintf(w, "# HELP go_goroutines Current goroutines.\n")
	fmt.Fprintf(w, "# TYPE go_goroutines gauge\n")
	fmt.Fprintf(w, "go_goroutines %d\n", runtime.NumGoroutine())
	fmt.Fprintf(w, "# HELP go_memstats_alloc_bytes Bytes allocated and still in use.\n")
	fmt.Fprintf(w, "# TYPE go_memstats_alloc_bytes gauge\n")
	fmt.Fprintf(w, "go_memstats_alloc_bytes %d\n", mem.Alloc)
}
