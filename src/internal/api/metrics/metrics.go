package metrics

import (
	"fmt"
	"net/http"
	"sync/atomic"
	"time"
)

var (
	httpRequestsTotal int64
	httpRequests2xx   int64
	httpRequests4xx   int64
	httpRequests5xx   int64
	cacheHitsTotal    int64
	cacheMissesTotal  int64
	cacheErrorsTotal  int64
	startTime         = time.Now()
)

var latencyBuckets = []float64{0.001, 0.005, 0.01, 0.05, 0.1, 0.5, 1, 5}
var latencyCounts [8]int64

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
		start := time.Now()
		next.ServeHTTP(rw, r)
		dur := time.Since(start).Seconds()

		atomic.AddInt64(&httpRequestsTotal, 1)
		switch {
		case rw.statusCode >= 500:
			atomic.AddInt64(&httpRequests5xx, 1)
		case rw.statusCode >= 400:
			atomic.AddInt64(&httpRequests4xx, 1)
		default:
			atomic.AddInt64(&httpRequests2xx, 1)
		}

		for i, b := range latencyBuckets {
			if dur <= b {
				atomic.AddInt64(&latencyCounts[i], 1)
				break
			}
		}
	})
}

func Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; version=0.0.4")

		fmt.Fprintf(w, "# HELP http_requests_total Total number of HTTP requests\n")
		fmt.Fprintf(w, "# TYPE http_requests_total counter\n")
		fmt.Fprintf(w, "http_requests_total{code=\"2xx\"} %d\n", atomic.LoadInt64(&httpRequests2xx))
		fmt.Fprintf(w, "http_requests_total{code=\"4xx\"} %d\n", atomic.LoadInt64(&httpRequests4xx))
		fmt.Fprintf(w, "http_requests_total{code=\"5xx\"} %d\n\n", atomic.LoadInt64(&httpRequests5xx))

		fmt.Fprintf(w, "# HELP http_request_duration_seconds Request latency distribution\n")
		fmt.Fprintf(w, "# TYPE http_request_duration_seconds histogram\n")
		for i, b := range latencyBuckets {
			le := "+Inf"
			if i < len(latencyBuckets)-1 {
				le = fmt.Sprintf("%g", b)
			}
			fmt.Fprintf(w, "http_request_duration_seconds_bucket{le=%q} %d\n", le, atomic.LoadInt64(&latencyCounts[i]))
		}
		fmt.Fprintf(w, "http_request_duration_seconds_count %d\n\n", atomic.LoadInt64(&httpRequestsTotal))

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
