package booking

import (
	"errors"
	"strings"
	"testing"
	"time"

	"bike-rental/internal/domain"
	"bike-rental/internal/store"
)

var now = time.Date(2026, 11, 6, 9, 0, 0, 0, time.UTC)

func tomorrow(hh, mm int) time.Time {
	return time.Date(2026, 11, 7, hh, mm, 0, 0, time.UTC)
}

func period(sh, sm, eh, em int) domain.Period {
	return domain.Period{Start: tomorrow(sh, sm), End: tomorrow(eh, em)}
}

type fixture struct {
	svc          *Service
	clock        *time.Time
	bikes        *store.MemoryStore
	persons      *store.PersonMemoryStore
	reservations *store.ReservationMemoryStore
	city         domain.Bike // 6 €/h
	kids         domain.Bike // 4 €/h
	broken       domain.Bike // maintenance
	alice        domain.Person
	archived     domain.Person
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	f := &fixture{
		bikes:        store.NewMemoryStore(),
		persons:      store.NewPersonMemoryStore(),
		reservations: store.NewReservationMemoryStore(),
	}
	clock := now
	f.clock = &clock
	f.svc = New(f.bikes, f.persons, f.reservations, func() time.Time { return *f.clock })

	mustBike := func(name string, c domain.Category, s domain.Status) domain.Bike {
		b, err := f.bikes.Create(domain.Bike{Name: name, Category: c, Status: s})
		if err != nil {
			t.Fatal(err)
		}
		return b
	}
	f.city = mustBike("City-01", domain.CategoryCity, domain.StatusAvailable)
	f.kids = mustBike("Kids-01", domain.CategoryChildren, domain.StatusAvailable)
	f.broken = mustBike("Broken-01", domain.CategoryRacing, domain.StatusMaintenance)

	doc, err := domain.NewIDDocument("id.pdf", []byte("%PDF-1.4\n"), now)
	if err != nil {
		t.Fatal(err)
	}
	mustPerson := func(first, idNumber string, archived bool) domain.Person {
		e, err := f.persons.Create(domain.Person{FirstName: first, LastName: "Test", IDNumber: idNumber, Phone: "12345", Archived: archived}, doc)
		if err != nil {
			t.Fatal(err)
		}
		return e.Person
	}
	f.alice = mustPerson("Alice", "L01X00T47", false)
	f.archived = mustPerson("Old", "OLD1", true)
	return f
}

func TestBook(t *testing.T) {
	f := newFixture(t)

	r, err := f.svc.Book(f.alice.ID, []string{f.city.ID, f.kids.ID}, period(10, 0, 11, 1))
	if err != nil {
		t.Fatalf("Book: %v", err)
	}
	// 1 h 1 min → 2 started hours × (6 + 4) € = 20 €
	if r.ID == "" || r.IDNumber != "L01X00T47" || r.BilledHours != 2 || r.TotalPriceCents != 2000 {
		t.Errorf("Book = %+v", r)
	}
	if len(r.Bikes) != 2 || r.Bikes[0].HourlyPriceCents != 600 || r.Bikes[1].Category != domain.CategoryChildren {
		t.Errorf("bike snapshots = %+v", r.Bikes)
	}
}

func TestBookRules(t *testing.T) {
	f := newFixture(t)
	if _, err := f.svc.Book(f.alice.ID, []string{f.city.ID}, period(10, 0, 12, 0)); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name     string
		personID string
		bikeIDs  []string
		period   domain.Period
		wantErr  error
		wantMsg  string
	}{
		{"unknown person", "p-99", []string{f.kids.ID}, period(10, 0, 11, 0), domain.ErrNotFound, "not found"},
		{"no person", "", []string{f.kids.ID}, period(10, 0, 11, 0), domain.ErrInvalid, "select a person"},
		{"archived person", f.archived.ID, []string{f.kids.ID}, period(10, 0, 11, 0), domain.ErrInvalid, "archived"},
		{"no bikes", f.alice.ID, nil, period(10, 0, 11, 0), domain.ErrInvalid, "at least one bike"},
		{"duplicate bike", f.alice.ID, []string{f.kids.ID, f.kids.ID}, period(10, 0, 11, 0), domain.ErrInvalid, "twice"},
		{"unknown bike", f.alice.ID, []string{"b-99"}, period(10, 0, 11, 0), domain.ErrNotFound, "not found"},
		{"bike in maintenance", f.alice.ID, []string{f.broken.ID}, period(10, 0, 11, 0), domain.ErrInvalid, "Broken-01 cannot be booked"},
		{"too short", f.alice.ID, []string{f.kids.ID}, period(10, 0, 10, 29), domain.ErrInvalid, "at least 30 minutes"},
		{"double booking", f.alice.ID, []string{f.kids.ID, f.city.ID}, period(11, 0, 13, 0), domain.ErrConflict, "City-01 is already booked"},
		{"gap too small", f.alice.ID, []string{f.city.ID}, period(12, 29, 13, 0), domain.ErrConflict, "30 minutes gap"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := f.svc.Book(tt.personID, tt.bikeIDs, tt.period)
			if !errors.Is(err, tt.wantErr) || !strings.Contains(err.Error(), tt.wantMsg) {
				t.Fatalf("err = %v, want %v containing %q", err, tt.wantErr, tt.wantMsg)
			}
		})
	}

	if _, err := f.svc.Book(f.alice.ID, []string{f.city.ID}, period(12, 30, 13, 0)); err != nil {
		t.Errorf("booking with exactly 30 min gap: %v", err)
	}
}

func TestChange(t *testing.T) {
	f := newFixture(t)
	r, _ := f.svc.Book(f.alice.ID, []string{f.city.ID}, period(10, 0, 12, 0))
	other, _ := f.svc.Book(f.alice.ID, []string{f.kids.ID}, period(14, 0, 16, 0))

	// Move and add a bike: 30 min → 1 h × 10 €. Overlapping its own old period is fine.
	changed, err := f.svc.Change(r.ID, []string{f.city.ID, f.kids.ID}, period(11, 0, 11, 30))
	if err != nil {
		t.Fatalf("Change: %v", err)
	}
	if changed.TotalPriceCents != 1000 || len(changed.Bikes) != 2 || changed.PersonID != f.alice.ID {
		t.Errorf("Change = %+v", changed)
	}

	// Conflicts with the other reservation's bike.
	if _, err := f.svc.Change(r.ID, []string{f.kids.ID}, period(15, 0, 16, 0)); !errors.Is(err, domain.ErrConflict) {
		t.Errorf("Change into other booking: err = %v", err)
	}

	// After the deadline of the current start (other starts at 14:00 tomorrow).
	*f.clock = other.Start.Add(-24*time.Hour + time.Second)
	_, err = f.svc.Change(other.ID, []string{f.kids.ID}, period(18, 0, 19, 0))
	if !errors.Is(err, domain.ErrConflict) || !strings.Contains(err.Error(), "24 hours") {
		t.Errorf("Change after deadline: err = %v", err)
	}
}

func TestCancel(t *testing.T) {
	f := newFixture(t)
	r, _ := f.svc.Book(f.alice.ID, []string{f.city.ID}, period(10, 0, 12, 0))

	*f.clock = r.Start.Add(-24 * time.Hour) // exactly at the deadline: still allowed
	cancelled, err := f.svc.Cancel(r.ID)
	if err != nil || cancelled.Status != domain.ReservationCancelled {
		t.Fatalf("Cancel = %+v, %v", cancelled, err)
	}
	if _, err := f.svc.Cancel(r.ID); !errors.Is(err, domain.ErrConflict) {
		t.Errorf("second Cancel: err = %v", err)
	}
	if _, err := f.svc.Cancel("r-99"); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("Cancel unknown: err = %v", err)
	}

	// The bike is free again.
	*f.clock = now
	if _, err := f.svc.Book(f.alice.ID, []string{f.city.ID}, period(10, 0, 12, 0)); err != nil {
		t.Errorf("rebooking after cancel: %v", err)
	}
}

func TestQuote(t *testing.T) {
	f := newFixture(t)
	r, _ := f.svc.Book(f.alice.ID, []string{f.city.ID}, period(10, 0, 12, 0))

	q, err := f.svc.Quote([]string{f.city.ID, f.kids.ID, f.broken.ID}, period(11, 0, 13, 30), "")
	if err != nil {
		t.Fatalf("Quote: %v", err)
	}
	// 2 h 30 min → 3 h × (6 + 4 + 10) € = 60 €
	if q.BilledHours != 3 || q.TotalPriceCents != 6000 {
		t.Errorf("Quote price = %d h / %d cents", q.BilledHours, q.TotalPriceCents)
	}
	joined := strings.Join(q.Conflicts, " | ")
	if len(q.Conflicts) != 2 || !strings.Contains(joined, "Broken-01 cannot be booked") || !strings.Contains(joined, "City-01 is already booked") {
		t.Errorf("Conflicts = %q", q.Conflicts)
	}

	// Excluding the reservation itself (edit mode) removes the conflict.
	q, _ = f.svc.Quote([]string{f.city.ID}, period(11, 0, 13, 30), r.ID)
	if len(q.Conflicts) != 0 {
		t.Errorf("Conflicts with exclude = %q", q.Conflicts)
	}

	if _, err := f.svc.Quote([]string{f.city.ID}, period(11, 0, 11, 10), ""); !errors.Is(err, domain.ErrInvalid) {
		t.Errorf("Quote too short: err = %v", err)
	}
	if q, err := f.svc.Quote(nil, period(11, 0, 12, 0), ""); err != nil || q.TotalPriceCents != 0 {
		t.Errorf("Quote without bikes = %+v, %v", q, err)
	}
}

func TestAvailableBikes(t *testing.T) {
	f := newFixture(t)
	r, _ := f.svc.Book(f.alice.ID, []string{f.city.ID}, period(10, 0, 12, 0))

	names := func(p domain.Period, exclude string) string {
		bikes, err := f.svc.AvailableBikes(p, exclude)
		if err != nil {
			t.Fatal(err)
		}
		var n []string
		for _, b := range bikes {
			n = append(n, b.Name)
		}
		return strings.Join(n, ",")
	}

	if got := names(period(11, 0, 12, 0), ""); got != "Kids-01" {
		t.Errorf("during booking = %q, want Kids-01 (maintenance bike excluded)", got)
	}
	if got := names(period(12, 30, 13, 0), ""); got != "City-01,Kids-01" {
		t.Errorf("after gap = %q", got)
	}
	if got := names(period(11, 0, 12, 0), r.ID); got != "City-01,Kids-01" {
		t.Errorf("excluding own reservation = %q", got)
	}
}

func TestOpenReservationHooks(t *testing.T) {
	f := newFixture(t)
	r, _ := f.svc.Book(f.alice.ID, []string{f.city.ID}, period(10, 0, 12, 0))

	if !f.svc.BikeHasOpenReservations(f.city.ID) || f.svc.BikeHasOpenReservations(f.kids.ID) {
		t.Error("BikeHasOpenReservations wrong for upcoming booking")
	}
	if !f.svc.PersonHasOpenReservations(f.alice.ID) {
		t.Error("PersonHasOpenReservations = false for upcoming booking")
	}

	*f.clock = r.Start.Add(time.Hour) // active
	if !f.svc.BikeHasOpenReservations(f.city.ID) {
		t.Error("active booking not counted as open")
	}

	*f.clock = r.End // completed
	if f.svc.BikeHasOpenReservations(f.city.ID) || f.svc.PersonHasOpenReservations(f.alice.ID) {
		t.Error("completed booking counted as open")
	}

	*f.clock = now
	if _, err := f.svc.Cancel(r.ID); err != nil {
		t.Fatal(err)
	}
	if f.svc.BikeHasOpenReservations(f.city.ID) {
		t.Error("cancelled booking counted as open")
	}
}
