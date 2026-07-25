package handler_test

import (
	"encoding/json"
	"kvs/src/internal/api/handler"
	"kvs/src/internal/kvs"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func setupTestServer(t *testing.T) *httptest.Server {
	t.Helper()
	h := handler.NewKeyValueHandler(kvs.NewKeyValueStore())
	return httptest.NewServer(h.Routes())
}

func testClient() *http.Client {
	return &http.Client{Timeout: 5 * time.Second}
}

func TestHealthz(t *testing.T) {
	srv := setupTestServer(t)
	defer srv.Close()

	resp, err := testClient().Get(srv.URL + "/healthz")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
}

func TestReadyz(t *testing.T) {
	srv := setupTestServer(t)
	defer srv.Close()

	resp, err := testClient().Get(srv.URL + "/readyz")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
}

func TestPutKey(t *testing.T) {
	srv := setupTestServer(t)
	defer srv.Close()

	req, _ := http.NewRequest(http.MethodPut, srv.URL+"/api/v1/key/mykey", strings.NewReader("myvalue"))
	resp, err := testClient().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Errorf("expected 201, got %d", resp.StatusCode)
	}
}

func TestGetKey(t *testing.T) {
	srv := setupTestServer(t)
	defer srv.Close()

	putReq, _ := http.NewRequest(http.MethodPut, srv.URL+"/api/v1/key/hello", strings.NewReader("world"))
	testClient().Do(putReq)

	resp, err := testClient().Get(srv.URL + "/api/v1/key/hello")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}

	var body map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if body["value"] != "world" {
		t.Errorf("expected 'world', got %v", body["value"])
	}
}

func TestGetKeyNotFound(t *testing.T) {
	srv := setupTestServer(t)
	defer srv.Close()

	resp, err := testClient().Get(srv.URL + "/api/v1/key/nonexistent")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected 404, got %d", resp.StatusCode)
	}
}

func TestDeleteKey(t *testing.T) {
	srv := setupTestServer(t)
	defer srv.Close()

	putReq, _ := http.NewRequest(http.MethodPut, srv.URL+"/api/v1/key/todelete", strings.NewReader("val"))
	testClient().Do(putReq)

	delReq, _ := http.NewRequest(http.MethodDelete, srv.URL+"/api/v1/key/todelete", nil)
	resp, err := testClient().Do(delReq)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		t.Errorf("expected 204, got %d", resp.StatusCode)
	}
}

func TestGetKeyAfterDelete(t *testing.T) {
	srv := setupTestServer(t)
	defer srv.Close()

	putReq, _ := http.NewRequest(http.MethodPut, srv.URL+"/api/v1/key/temp", strings.NewReader("val"))
	testClient().Do(putReq)

	delReq, _ := http.NewRequest(http.MethodDelete, srv.URL+"/api/v1/key/temp", nil)
	testClient().Do(delReq)

	resp, err := testClient().Get(srv.URL + "/api/v1/key/temp")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected 404 after delete, got %d", resp.StatusCode)
	}
}

func TestDeleteNonExistentKey(t *testing.T) {
	srv := setupTestServer(t)
	defer srv.Close()

	req, _ := http.NewRequest(http.MethodDelete, srv.URL+"/api/v1/key/ghost", nil)
	resp, err := testClient().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		t.Errorf("expected 204, got %d", resp.StatusCode)
	}
}
