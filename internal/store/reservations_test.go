package store

import (
	"errors"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"bike-rental/internal/domain"
)

var resNow = time.Date(2026, 11, 6, 9, 0, 0, 0, time.UTC)

func tomorrow(hh, mm int) time.Time {
	return time.Date(2026, 11, 7, hh, mm, 0, 0, time.UTC)
}

func newReservation(t *testing.T, start, end time.Time, bikeIDs ...string) domain.Reservation {
	t.Helper()
	bikes := make([]domain.ReservedBike, len(bikeIDs))
	for i, id := range bikeIDs {
		bikes[i] = domain.ReservedBike{BikeID: id, Category: domain.CategoryCity, HourlyPriceCents: 600}
	}
	r, err := domain.NewReservation("p-1", "X1", bikes, domain.Period{Start: start, End: end}, resNow)
	if err != nil {
		t.Fatalf("NewReservation: %v", err)
	}
	return r
}

func TestReservationCreateGetList(t *testing.T) {
	s := NewReservationMemoryStore()
	late, err := s.Create(newReservation(t, tomorrow(15, 0), tomorrow(16, 0), "b-1"))
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	early, err := s.Create(newReservation(t, tomorrow(9, 0), tomorrow(10, 0), "b-2"))
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if late.ID == "" || late.ID == early.ID {
		t.Fatalf("IDs: %q, %q", late.ID, early.ID)
	}

	got, err := s.Get(late.ID)
	if err != nil || got.ID != late.ID || len(got.Bikes) != 1 {
		t.Fatalf("Get = %+v, %v", got, err)
	}

	list := s.List()
	if len(list) != 2 || list[0].ID != early.ID || list[1].ID != late.ID {
		t.Errorf("List not sorted by start: %v, %v", list[0].ID, list[1].ID)
	}

	if _, err := s.Get("r-99"); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("Get unknown: err = %v", err)
	}
	if _, err := s.Update(domain.Reservation{ID: "r-99"}); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("Update unknown: err = %v", err)
	}
}

func TestReservationConflicts(t *testing.T) {
	s := NewReservationMemoryStore()
	if _, err := s.Create(newReservation(t, tomorrow(10, 0), tomorrow(12, 0), "b-1", "b-2")); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name     string
		start    time.Time
		end      time.Time
		bikes    []string
		conflict bool
	}{
		{"same bike, overlapping", tomorrow(11, 0), tomorrow(13, 0), []string{"b-1"}, true},
		{"second bike of reservation", tomorrow(11, 0), tomorrow(13, 0), []string{"b-3", "b-2"}, true},
		{"same bike, 29 min gap", tomorrow(12, 29), tomorrow(14, 0), []string{"b-1"}, true},
		{"same bike, exactly 30 min gap", tomorrow(12, 30), tomorrow(14, 0), []string{"b-1"}, false},
		{"other bike, overlapping", tomorrow(10, 0), tomorrow(12, 0), []string{"b-3"}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := s.Create(newReservation(t, tt.start, tt.end, tt.bikes...))
			if tt.conflict {
				if !errors.Is(err, domain.ErrConflict) || !strings.Contains(err.Error(), "already booked") {
					t.Fatalf("err = %v, want ErrConflict", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("err = %v, want nil", err)
			}
		})
	}
}

func TestCancelledReservationDoesNotBlock(t *testing.T) {
	s := NewReservationMemoryStore()
	r, _ := s.Create(newReservation(t, tomorrow(10, 0), tomorrow(12, 0), "b-1"))
	if err := r.Cancel(resNow); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Update(r); err != nil {
		t.Fatalf("Update (cancel): %v", err)
	}
	if _, err := s.Create(newReservation(t, tomorrow(10, 0), tomorrow(12, 0), "b-1")); err != nil {
		t.Fatalf("booking after cancellation: %v", err)
	}
}

func TestUpdateIgnoresItselfButNotOthers(t *testing.T) {
	s := NewReservationMemoryStore()
	a, _ := s.Create(newReservation(t, tomorrow(10, 0), tomorrow(12, 0), "b-1"))
	if _, err := s.Create(newReservation(t, tomorrow(14, 0), tomorrow(16, 0), "b-1")); err != nil {
		t.Fatal(err)
	}

	// Moving a by 30 minutes overlaps only with itself → fine.
	if err := a.Reschedule(a.Bikes, domain.Period{Start: tomorrow(10, 30), End: tomorrow(12, 30)}, resNow); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Update(a); err != nil {
		t.Fatalf("Update overlapping itself: %v", err)
	}

	// Moving a too close to the other reservation → conflict.
	if err := a.Reschedule(a.Bikes, domain.Period{Start: tomorrow(12, 0), End: tomorrow(13, 45)}, resNow); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Update(a); !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("Update into other reservation: err = %v, want ErrConflict", err)
	}
}

func TestReservationReturnsCopies(t *testing.T) {
	s := NewReservationMemoryStore()
	r, _ := s.Create(newReservation(t, tomorrow(10, 0), tomorrow(12, 0), "b-1"))
	r.Bikes[0].BikeID = "changed"
	s.List()[0].Bikes[0].BikeID = "changed"

	got, _ := s.Get(r.ID)
	if got.Bikes[0].BikeID != "b-1" {
		t.Errorf("stored bikes modified from outside: %+v", got.Bikes)
	}
}

func TestConcurrentBookingOfSameBike(t *testing.T) {
	s := NewReservationMemoryStore()
	const attempts = 50

	var succeeded atomic.Int32
	var wg sync.WaitGroup
	for range attempts {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := s.Create(newReservation(t, tomorrow(10, 0), tomorrow(12, 0), "b-1"))
			switch {
			case err == nil:
				succeeded.Add(1)
			case !errors.Is(err, domain.ErrConflict):
				t.Errorf("unexpected error: %v", err)
			}
		}()
	}
	wg.Wait()

	if got := succeeded.Load(); got != 1 {
		t.Errorf("%d concurrent bookings succeeded, want exactly 1", got)
	}
}
