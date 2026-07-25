package main

import (
	"context"
	"kvs/src/internal/api/handler"
	"kvs/src/internal/kvs"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	databaseURL := os.Getenv("DATABASE_URL")
	var store kvs.KeyValueStore
	if databaseURL != "" {
		pgStore, err := kvs.NewPostgresStore(ctx, databaseURL)
		if err != nil {
			log.Fatalf("failed to connect to postgres: %v", err)
		}
		defer pgStore.Close()
		store = pgStore
		log.Println("using postgres storage")
	} else {
		store = kvs.NewKeyValueStore()
		log.Println("using in-memory storage")
	}

	h := handler.NewKeyValueHandler(store)

	srv := &http.Server{
		Addr:    ":8080",
		Handler: h.Routes(),
	}

	go func() {
		log.Println("starting server on :8080")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	<-ctx.Done()
	log.Println("shutting down gracefully...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("shutdown error: %v", err)
	}
	log.Println("server stopped")
}
