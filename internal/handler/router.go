// Package handler contains the HTTP handlers. Handlers stay thin: they parse the request,
// call into the domain and store, and write a JSON response.
package handler

import (
	"encoding/json"
	"log"
	"net/http"
)

// NewRouter wires all API routes and serves the static frontend from webDir.
func NewRouter(webDir string) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /api/health", handleHealth)

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
