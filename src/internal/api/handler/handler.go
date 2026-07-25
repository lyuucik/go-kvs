package handler

import (
	"context"
	"errors"
	"io"
	"kvs/src/internal/kvs"
	"log"
	"net/http"
	"time"
)

type keyValueHandler struct {
	kvs    kvs.KeyValueStore
	readyz func(context.Context) error
}

func NewKeyValueHandler(kvs kvs.KeyValueStore) *keyValueHandler {
	return &keyValueHandler{kvs: kvs, readyz: func(ctx context.Context) error { return kvs.Ping() }}
}

func (h *keyValueHandler) Routes() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /healthz", h.healthz)
	mux.HandleFunc("GET /readyz", h.readyzHandler)
	mux.HandleFunc("GET /api/v1/key/{key}", h.getKey)
	mux.HandleFunc("PUT /api/v1/key/{key}", h.putKey)
	mux.HandleFunc("DELETE /api/v1/key/{key}", h.deleteKey)

	return mux
}

func (h *keyValueHandler) healthz(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}

func (h *keyValueHandler) readyzHandler(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	if err := h.readyz(ctx); err != nil {
		log.Printf("readyz check failed: %v", err)
		http.Error(w, "not ready", http.StatusServiceUnavailable)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}

func (h *keyValueHandler) putKey(w http.ResponseWriter, r *http.Request) {
	key := r.PathValue("key")
	value, err := io.ReadAll(r.Body)
	defer r.Body.Close()

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err := h.kvs.Put(key, string(value)); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusCreated)
}

func (h *keyValueHandler) getKey(w http.ResponseWriter, r *http.Request) {
	key := r.PathValue("key")

	v, err := h.kvs.Get(key)

	switch {
	case errors.Is(err, kvs.ErrorNoSuchKey):
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	case err != nil:
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, envelope{"value": v}, http.StatusOK, nil)
}

func (h *keyValueHandler) deleteKey(w http.ResponseWriter, r *http.Request) {
	key := r.PathValue("key")

	if err := h.kvs.Delete(key); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
