package handler

import (
	"encoding/json"

	"bike-rental/internal/store"
)

// newTestDeps returns empty in-memory stores with the real clock.
func newTestDeps() Deps {
	return Deps{
		Bikes:        store.NewMemoryStore(),
		Persons:      store.NewPersonMemoryStore(),
		Reservations: store.NewReservationMemoryStore(),
	}
}

func jsonMarshal(v any) ([]byte, error)   { return json.Marshal(v) }
func jsonUnmarshal(b []byte, v any) error { return json.Unmarshal(b, v) }
