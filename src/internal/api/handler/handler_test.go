package handler_test

import (
	"encoding/json"
	"kvs/src/internal/api/handler"
	"kvs/src/internal/kvs"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func setupTestServer(t *testing.T) *httptest.Server {
	t.Helper()
	h := handler.NewKeyValueHandler(kvs.NewKeyValueStore())
	return httptest.NewServer(h.Routes())
}

func TestPutKey(t *testing.T) {
	srv := setupTestServer(t)
	defer srv.Close()

	req, err := http.NewRequest(http.MethodPut, srv.URL+"/api/v1/key/mykey", strings.NewReader("myvalue"))
	if err != nil {
		t.Fatal(err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Errorf("expected status 201, got %d", resp.StatusCode)
	}
}

func TestGetKey(t *testing.T) {
	srv := setupTestServer(t)
	defer srv.Close()

	putReq, _ := http.NewRequest(http.MethodPut, srv.URL+"/api/v1/key/hello", strings.NewReader("world"))
	http.DefaultClient.Do(putReq)

	getReq, _ := http.NewRequest(http.MethodGet, srv.URL+"/api/v1/key/hello", nil)
	resp, err := http.DefaultClient.Do(getReq)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Errorf("expected status 201, got %d", resp.StatusCode)
	}

	var body map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if body["value"] != "world" {
		t.Errorf("expected value 'world', got %v", body["value"])
	}
}

func TestGetKeyNotFound(t *testing.T) {
	srv := setupTestServer(t)
	defer srv.Close()

	req, _ := http.NewRequest(http.MethodGet, srv.URL+"/api/v1/key/nonexistent", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", resp.StatusCode)
	}
}

func TestDeleteKey(t *testing.T) {
	srv := setupTestServer(t)
	defer srv.Close()

	putReq, _ := http.NewRequest(http.MethodPut, srv.URL+"/api/v1/key/todelete", strings.NewReader("val"))
	http.DefaultClient.Do(putReq)

	delReq, _ := http.NewRequest(http.MethodDelete, srv.URL+"/api/v1/key/todelete", nil)
	resp, err := http.DefaultClient.Do(delReq)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Errorf("expected status 201, got %d", resp.StatusCode)
	}
}

func TestGetKeyAfterDelete(t *testing.T) {
	srv := setupTestServer(t)
	defer srv.Close()

	putReq, _ := http.NewRequest(http.MethodPut, srv.URL+"/api/v1/key/temp", strings.NewReader("val"))
	http.DefaultClient.Do(putReq)

	delReq, _ := http.NewRequest(http.MethodDelete, srv.URL+"/api/v1/key/temp", nil)
	http.DefaultClient.Do(delReq)

	getReq, _ := http.NewRequest(http.MethodGet, srv.URL+"/api/v1/key/temp", nil)
	resp, err := http.DefaultClient.Do(getReq)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected status 404 after delete, got %d", resp.StatusCode)
	}
}

func TestDeleteNonExistentKey(t *testing.T) {
	srv := setupTestServer(t)
	defer srv.Close()

	req, _ := http.NewRequest(http.MethodDelete, srv.URL+"/api/v1/key/ghost", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Errorf("expected status 201, got %d", resp.StatusCode)
	}
}