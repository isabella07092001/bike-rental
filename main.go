// Command bike-rental starts the HTTP server for the bike rental management prototype.
package main

import (
	"errors"
	"log"
	"net/http"
	"os"
	"time"

	"bike-rental/internal/handler"
	"bike-rental/internal/store"
)

func main() {
	addr := ":" + envOr("PORT", "8080")

	srv := &http.Server{
		Addr: addr,
		Handler: handler.NewRouter("web", handler.Deps{
			Bikes:        store.NewMemoryStore(),
			Persons:      store.NewPersonMemoryStore(),
			Reservations: store.NewReservationMemoryStore(),
		}),
		ReadHeaderTimeout: 5 * time.Second,
	}

	log.Printf("bike-rental listening on http://localhost%s", addr)
	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatalf("server stopped: %v", err)
	}
}

// envOr returns the value of the environment variable key, or fallback if it is unset or empty.
func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
