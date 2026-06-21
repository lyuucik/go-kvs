package handler

import (
	"encoding/json"
	"net/http"
)

type envelope map[string]any

func writeJSON(w http.ResponseWriter, data envelope, status int, headers http.Header) error {
	js, err := json.Marshal(data)
	if err != nil {
		return err
	}

	for k, v := range headers {
		w.Header()[k] = v
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	w.Write(js)

	return nil
}
