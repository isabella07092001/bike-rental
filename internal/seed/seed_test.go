package seed

import (
	"testing"
	"time"

	"bike-rental/internal/dashboard"
	"bike-rental/internal/domain"
	"bike-rental/internal/store"
)

func TestLoad(t *testing.T) {
	// Run at different times of day to catch boundary problems (e.g. shortly after midnight).
	for _, now := range []time.Time{
		time.Date(2026, 11, 6, 12, 30, 0, 0, time.Local),
		time.Date(2026, 11, 6, 0, 5, 0, 0, time.Local),
		time.Date(2026, 11, 6, 23, 55, 0, 0, time.Local),
	} {
		t.Run(now.Format("15:04"), func(t *testing.T) {
			bikes := store.NewMemoryStore()
			persons := store.NewPersonMemoryStore()
			reservations := store.NewReservationMemoryStore()

			if err := Load(bikes, persons, reservations, now); err != nil {
				t.Fatalf("Load: %v", err)
			}
			if got := len(bikes.List()); got != len(demoBikes) {
				t.Errorf("bikes = %d, want %d", got, len(demoBikes))
			}
			for _, e := range persons.List() {
				doc, err := persons.GetIDDocument(e.Person.ID)
				if err != nil || doc.ContentType != domain.ContentTypePNG {
					t.Errorf("%s: ID copy %q, %v", e.Person.FullName(), doc.ContentType, err)
				}
			}

			phases := map[domain.ReservationPhase]int{}
			for _, r := range reservations.List() {
				phases[r.Phase(now)]++
			}
			want := map[domain.ReservationPhase]int{
				domain.PhaseActive: 1, domain.PhaseCompleted: 1, domain.PhaseUpcoming: 4, domain.PhaseCancelled: 1,
			}
			for p, n := range want {
				if phases[p] != n {
					t.Errorf("phase %s: %d reservations, want %d (all: %v)", p, phases[p], n, phases)
				}
			}

			s := dashboard.NewBuilder(bikes, persons, reservations, func() time.Time { return now }).Build()
			if s.Fleet.Total != 9 || s.Fleet.RentedNow != 2 || s.Fleet.Maintenance != 1 || s.Upcoming.Count != 3 {
				t.Errorf("dashboard: fleet %+v, upcoming %d", s.Fleet, s.Upcoming.Count)
			}
		})
	}
}

func TestLoadTwiceFails(t *testing.T) {
	bikes, persons, reservations := store.NewMemoryStore(), store.NewPersonMemoryStore(), store.NewReservationMemoryStore()
	now := time.Now()
	if err := Load(bikes, persons, reservations, now); err != nil {
		t.Fatal(err)
	}
	if err := Load(bikes, persons, reservations, now); err == nil {
		t.Error("second Load into filled stores succeeded, want a conflict error")
	}
}
