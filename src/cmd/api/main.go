package main

import (
	"context"
	"kvs/src/internal/api/handler"
	"kvs/src/internal/api/metrics"
	"kvs/src/internal/kvs"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	databaseURL := os.Getenv("DATABASE_URL")
	var store kvs.KeyValueStore
	if databaseURL != "" {
		var pgStore *kvs.PostgresStore
		var err error
		for i := 0; i < 10; i++ {
			pgStore, err = kvs.NewPostgresStore(ctx, databaseURL)
			if err == nil {
				break
			}
			slog.Warn("postgres not ready, retrying", "attempt", i+1, "max", 10, "error", err)
			time.Sleep(3 * time.Second)
		}
		if err != nil {
			slog.Error("failed to connect to postgres", "error", err)
			os.Exit(1)
		}
		defer pgStore.Close()
		store = pgStore
		slog.Info("using postgres storage")
	} else {
		store = kvs.NewKeyValueStore()
		slog.Info("using in-memory storage")
	}

	cacheTTL := 5 * time.Minute
	if ttlStr := os.Getenv("CACHE_TTL"); ttlStr != "" {
		if sec, err := strconv.Atoi(ttlStr); err == nil {
			cacheTTL = time.Duration(sec) * time.Second
		}
	}

	if redisURL := os.Getenv("REDIS_URL"); redisURL != "" {
		cached, err := kvs.NewCachedStore(store, redisURL, cacheTTL, kvs.CacheHooks{
			OnHit:    metrics.IncCacheHits,
			OnMiss:   metrics.IncCacheMisses,
			OnError:  metrics.IncCacheErrors,
		})
		if err != nil {
			slog.Warn("redis connection failed, running without cache", "error", err)
		} else {
			defer cached.Close()
			store = cached
			slog.Info("using redis cache")
		}
	}

	h := handler.NewKeyValueHandler(store)

	mux := h.Routes()
	mux.Handle("GET /metrics", metrics.Handler())
	wrapped := metrics.Middleware(mux)

	srv := &http.Server{
		Addr:    ":8080",
		Handler: wrapped,
	}

	go func() {
		slog.Info("starting server", "addr", ":8080")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server error", "error", err)
			os.Exit(1)
		}
	}()

	<-ctx.Done()
	slog.Info("shutting down gracefully...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		slog.Error("shutdown error", "error", err)
		os.Exit(1)
	}
	slog.Info("server stopped")
}
