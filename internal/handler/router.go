// Package handler contains the HTTP handlers. Handlers stay thin: they parse the request,
// call into the domain, booking service and stores, and write a JSON response.
package handler

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"time"

	"bike-rental/internal/booking"
	"bike-rental/internal/domain"
	"bike-rental/internal/store"
)

// Deps are the stores and the clock the handlers work with.
type Deps struct {
	Bikes        store.BikeStore
	Persons      store.PersonStore
	Reservations store.ReservationStore
	// Now is the clock; nil means time.Now. Tests inject a fixed time.
	Now func() time.Time
}

// NewRouter wires all API routes and serves the static frontend from webDir.
func NewRouter(webDir string, d Deps) http.Handler {
	if d.Now == nil {
		d.Now = time.Now
	}
	bookings := booking.New(d.Bikes, d.Persons, d.Reservations, d.Now)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/health", handleHealth)
	NewBikeHandler(d.Bikes, bookings.BikeHasOpenReservations).Register(mux)
	NewPersonHandler(d.Persons, d.Now, bookings.PersonHasOpenReservations).Register(mux)
	NewReservationHandler(bookings, d.Reservations, d.Bikes, d.Persons).Register(mux)

	mux.Handle("GET /", noCache(http.FileServer(http.Dir(webDir))))

	return mux
}

// noCache makes browsers revalidate static files on every load (cheap 304 via Last-Modified),
// so an updated frontend is never mixed with stale cached scripts.
func noCache(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-cache")
		next.ServeHTTP(w, r)
	})
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
