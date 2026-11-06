// Command bike-rental starts the HTTP server for the Power Bike rental management prototype.
//
// Usage:
//
//	go run .         start with empty stores
//	go run . -demo   start with demo bikes, customers and reservations
//
// The port is taken from the PORT environment variable (default 8080).
package main

import (
	"errors"
	"flag"
	"log"
	"net/http"
	"os"
	"time"

	"bike-rental/internal/handler"
	"bike-rental/internal/seed"
	"bike-rental/internal/store"
)

func main() {
	demo := flag.Bool("demo", false, "load demo bikes, customers and reservations at startup")
	flag.Parse()

	addr := ":" + envOr("PORT", "8080")
	deps := handler.Deps{
		Bikes:        store.NewMemoryStore(),
		Persons:      store.NewPersonMemoryStore(),
		Reservations: store.NewReservationMemoryStore(),
	}
	if *demo {
		if err := seed.Load(deps.Bikes, deps.Persons, deps.Reservations, time.Now()); err != nil {
			log.Fatalf("load demo data: %v", err)
		}
		log.Print("demo data loaded")
	}

	srv := &http.Server{
		Addr:              addr,
		Handler:           handler.NewRouter("web", deps),
		ReadHeaderTimeout: 5 * time.Second,
	}

	log.Printf("Power Bike listening on http://localhost%s", addr)
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
