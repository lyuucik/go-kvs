package handler

import (
	"errors"
	"io"
	"kvs/src/internal/kvs"
	"net/http"

	"github.com/gorilla/mux"
)

type keyValueHandler struct {
	kvs kvs.KeyValueStore
}

type Handler interface {
	Routes() http.Handler
}

var _ Handler = &keyValueHandler{}

func NewKeyValueHandler(kvs kvs.KeyValueStore) Handler {
	return &keyValueHandler{kvs}
}

func (h *keyValueHandler) Routes() http.Handler {
	router := mux.NewRouter()

	router.HandleFunc("/api/v1/key/{key}", h.getKeyHandler()).Methods(http.MethodGet)
	router.HandleFunc("/api/v1/key/{key}", h.putKeyHandler()).Methods(http.MethodPut)
	router.HandleFunc("/api/v1/key/{key}", h.deleteKeyHandler()).Methods(http.MethodDelete)

	return router
}

func (h *keyValueHandler) putKeyHandler() func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		key := vars["key"]
		value, err := io.ReadAll(r.Body)
		defer r.Body.Close()

		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		err = h.kvs.Put(key, string(value))
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		w.WriteHeader(http.StatusCreated)
	}
}

func (h *keyValueHandler) getKeyHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)

		v, err := h.kvs.Get(vars["key"])

		switch {
		case errors.Is(err, kvs.ErrorNoSuchKey):
			http.Error(w, err.Error(), http.StatusNotFound)
		case err != nil:
			http.Error(w, err.Error(), http.StatusInternalServerError)
		default:
		}

		writeJSON(w, envelope{"value": v}, http.StatusCreated, nil)
	}
}

func (h *keyValueHandler) deleteKeyHandler() func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)

		h.kvs.Delete(vars["key"])

		w.WriteHeader(http.StatusCreated)
	}
}
