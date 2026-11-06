package handler

import (
	"net/http"
	"testing"
	"time"

	"bike-rental/internal/dashboard"
)

func TestDashboard(t *testing.T) {
	f := newResFixture(t) // City-01, Kids-01; clock at resTestNow
	if _, code, msg := f.book(t, f.alice.ID, []string{f.city.ID, f.kids.ID}, tm(10, 0), tm(12, 0)); code != http.StatusCreated {
		t.Fatalf("book: %d %s", code, msg)
	}
	do(t, f.router, http.MethodPut, "/api/bikes/"+f.kids.ID, `{"name":"Kids-01","category":"children","status":"maintenance"}`, nil)

	var s dashboard.Summary
	if code := do(t, f.router, http.MethodGet, "/api/dashboard", "", &s); code != http.StatusOK {
		t.Fatalf("status = %d", code)
	}
	if !s.GeneratedAt.Equal(resTestNow) {
		t.Errorf("generatedAt = %v, want the injected clock", s.GeneratedAt)
	}
	// Before the rental starts: one available city bike, one children's bike in maintenance.
	if s.Fleet.Total != 2 || s.Fleet.AvailableNow != 1 || s.Fleet.Maintenance != 1 || s.Fleet.RentedNow != 0 {
		t.Errorf("fleet = %+v", s.Fleet)
	}
	if s.Upcoming.Count != 1 || s.Upcoming.Reservations[0].PersonName != "Alice Test" || len(s.Upcoming.Reservations[0].Bikes) != 2 {
		t.Errorf("upcoming = %+v", s.Upcoming)
	}

	// During the rental both bikes count as rented, even the one set to maintenance.
	*f.clock = resTestNow.Add(25 * time.Hour) // tomorrow 10:00 = rental start (inclusive)
	do(t, f.router, http.MethodGet, "/api/dashboard", "", &s)
	if s.Fleet.RentedNow != 2 || s.Fleet.Maintenance != 0 || s.Upcoming.Count != 0 {
		t.Errorf("during rental: fleet = %+v, upcoming = %d", s.Fleet, s.Upcoming.Count)
	}
}
