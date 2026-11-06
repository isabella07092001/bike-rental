package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"bike-rental/internal/domain"
)

func TestHealth(t *testing.T) {
	router := NewRouter(t.TempDir(), newTestDeps())

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/health", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", ct)
	}
	var body map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body["status"] != "ok" {
		t.Errorf(`body["status"] = %q, want "ok"`, body["status"])
	}
}

func TestHealthRejectsPost(t *testing.T) {
	router := NewRouter(t.TempDir(), newTestDeps())

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/health", nil))

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusMethodNotAllowed)
	}
}

func TestServesStaticFrontend(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "index.html"), []byte("<h1>Bike Rental</h1>"), 0o644); err != nil {
		t.Fatalf("write index.html: %v", err)
	}
	router := NewRouter(dir, newTestDeps())

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if cc := rec.Header().Get("Cache-Control"); cc != "no-cache" {
		t.Errorf("Cache-Control = %q, want no-cache", cc)
	}
	if !strings.Contains(rec.Body.String(), "Bike Rental") {
		t.Errorf("body = %q, want it to contain index.html content", rec.Body.String())
	}
}

func TestUserMessage(t *testing.T) {
	tests := []struct {
		err  error
		want string
	}{
		{fmt.Errorf("%w: name is required", domain.ErrInvalid), "Name is required"},
		{fmt.Errorf("%w: bike %q not found", domain.ErrNotFound, "b-9"), `Bike "b-9" not found`},
		{fmt.Errorf("%w: a bike named %q already exists", domain.ErrConflict, "X"), `A bike named "X" already exists`},
		{fmt.Errorf("%w: ünknown", domain.ErrInvalid), "Ünknown"},
		{fmt.Errorf("wrapped: %w", domain.ErrInvalid), "Wrapped: invalid input"},
	}
	for _, tt := range tests {
		if got := userMessage(tt.err); got != tt.want {
			t.Errorf("userMessage(%q) = %q, want %q", tt.err, got, tt.want)
		}
	}
}
