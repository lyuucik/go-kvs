package metrics

import (
	"fmt"
	"net/http"
	"sync/atomic"
	"time"
)

var (
	httpRequestsTotal int64
	cacheHitsTotal    int64
	cacheMissesTotal  int64
	cacheErrorsTotal  int64
	startTime         = time.Now()
)

type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}
		next.ServeHTTP(rw, r)
		atomic.AddInt64(&httpRequestsTotal, 1)
	})
}

func Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; version=0.0.4")

		fmt.Fprintf(w, "# HELP http_requests_total Total number of HTTP requests\n")
		fmt.Fprintf(w, "# TYPE http_requests_total counter\n")
		fmt.Fprintf(w, "http_requests_total %d\n\n", atomic.LoadInt64(&httpRequestsTotal))

		fmt.Fprintf(w, "# HELP kv_cache_hits_total Total number of cache hits\n")
		fmt.Fprintf(w, "# TYPE kv_cache_hits_total counter\n")
		fmt.Fprintf(w, "kv_cache_hits_total %d\n\n", atomic.LoadInt64(&cacheHitsTotal))

		fmt.Fprintf(w, "# HELP kv_cache_misses_total Total number of cache misses\n")
		fmt.Fprintf(w, "# TYPE kv_cache_misses_total counter\n")
		fmt.Fprintf(w, "kv_cache_misses_total %d\n\n", atomic.LoadInt64(&cacheMissesTotal))

		fmt.Fprintf(w, "# HELP kv_cache_errors_total Total number of cache backend errors\n")
		fmt.Fprintf(w, "# TYPE kv_cache_errors_total counter\n")
		fmt.Fprintf(w, "kv_cache_errors_total %d\n\n", atomic.LoadInt64(&cacheErrorsTotal))

		fmt.Fprintf(w, "# HELP kv_uptime_seconds Application uptime in seconds\n")
		fmt.Fprintf(w, "# TYPE kv_uptime_seconds gauge\n")
		fmt.Fprintf(w, "kv_uptime_seconds %d\n", int64(time.Since(startTime).Seconds()))
	})
}

func IncCacheHits()   { atomic.AddInt64(&cacheHitsTotal, 1) }
func IncCacheMisses() { atomic.AddInt64(&cacheMissesTotal, 1) }
func IncCacheErrors() { atomic.AddInt64(&cacheErrorsTotal, 1) }
