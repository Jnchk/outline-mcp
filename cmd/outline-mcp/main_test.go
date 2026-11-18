package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Jnchk/outline-mcp/internal/staging"
)

func TestUploadHandler(t *testing.T) {
	store, err := staging.New(t.TempDir(), time.Minute, 16)
	if err != nil {
		t.Fatal(err)
	}
	h := uploadHandler(store)

	req := httptest.NewRequest(http.MethodGet, "/upload?filename=x.md", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("GET = %d", rec.Code)
	}

	req = httptest.NewRequest(http.MethodPost, "/upload", strings.NewReader("x"))
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("missing filename = %d", rec.Code)
	}

	req = httptest.NewRequest(http.MethodPost, "/upload?filename=doc.md", strings.NewReader("# 标题\n"))
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
	var info staging.Info
	if err := json.Unmarshal(rec.Body.Bytes(), &info); err != nil {
		t.Fatal(err)
	}
	if info.ID == "" || info.Filename != "doc.md" {
		t.Fatalf("info = %#v", info)
	}

	req = httptest.NewRequest(http.MethodPost, "/upload?filename=x.md", strings.NewReader("\xff\xfe"))
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("binary = %d", rec.Code)
	}

	req = httptest.NewRequest(http.MethodPost, "/upload?filename=x.md", strings.NewReader("way too long for the cap"))
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("oversize = %d", rec.Code)
	}
}

func TestWithToken(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusNoContent) })
	h := withToken("secret", next)

	req := httptest.NewRequest(http.MethodPost, "/mcp", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("no token = %d", rec.Code)
	}

	req = httptest.NewRequest(http.MethodPost, "/mcp", nil)
	req.Header.Set("Authorization", "Bearer secret")
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("with token = %d", rec.Code)
	}
}
