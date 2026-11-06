// Package handler contains the HTTP handlers. Handlers stay thin: they parse the request,
// call into the domain and store, and write a JSON response.
package handler

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"bike-rental/internal/domain"
	"bike-rental/internal/store"
)

// NewRouter wires all API routes and serves the static frontend from webDir.
func NewRouter(webDir string, bikes store.BikeStore, persons store.PersonStore) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /api/health", handleHealth)
	NewBikeHandler(bikes).Register(mux)
	NewPersonHandler(persons).Register(mux)

	mux.Handle("GET /", http.FileServer(http.Dir(webDir)))

	return mux
}

func handleHealth(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// writeJSON writes v as a JSON response with the given status code.
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("write json response: %v", err)
	}
}

// writeError maps domain errors to HTTP status codes and writes {"error": "..."}.
func writeError(w http.ResponseWriter, err error) {
	status := http.StatusInternalServerError
	var maxBytesErr *http.MaxBytesError
	switch {
	case errors.Is(err, domain.ErrInvalid):
		status = http.StatusBadRequest
	case errors.Is(err, domain.ErrNotFound):
		status = http.StatusNotFound
	case errors.Is(err, domain.ErrConflict):
		status = http.StatusConflict
	case errors.Is(err, domain.ErrTooLarge), errors.As(err, &maxBytesErr):
		status = http.StatusRequestEntityTooLarge
	}
	msg := err.Error()
	switch status {
	case http.StatusInternalServerError:
		log.Printf("internal error: %v", err)
		msg = "internal server error"
	case http.StatusRequestEntityTooLarge:
		msg = "request too large: the ID copy must be at most 5 MB"
	}
	writeJSON(w, status, map[string]string{"error": msg})
}
