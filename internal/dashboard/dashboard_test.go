package dashboard

import (
	"strings"
	"testing"
	"time"

	"bike-rental/internal/domain"
	"bike-rental/internal/store"
)

var (
	now      = time.Date(2026, 11, 6, 12, 0, 0, 0, time.UTC)
	bookedAt = time.Date(2026, 11, 1, 9, 0, 0, 0, time.UTC) // all test bookings were made earlier
)

type fixture struct {
	bikes        *store.MemoryStore
	persons      *store.PersonMemoryStore
	reservations *store.ReservationMemoryStore
	builder      *Builder
	ids          map[string]string // bike name → ID
	alice        domain.Person
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	f := &fixture{
		bikes:        store.NewMemoryStore(),
		persons:      store.NewPersonMemoryStore(),
		reservations: store.NewReservationMemoryStore(),
		ids:          map[string]string{},
	}
	f.builder = NewBuilder(f.bikes, f.persons, f.reservations, func() time.Time { return now })

	doc, _ := domain.NewIDDocument("id.pdf", []byte("%PDF-1.4\n"), bookedAt)
	e, err := f.persons.Create(domain.Person{FirstName: "Alice", LastName: "Meyer", IDNumber: "A1", Phone: "12345"}, doc)
	if err != nil {
		t.Fatal(err)
	}
	f.alice = e.Person
	return f
}

func (f *fixture) bike(t *testing.T, name string, c domain.Category, s domain.Status) {
	t.Helper()
	b, err := f.bikes.Create(domain.Bike{Name: name, Category: c, Status: s})
	if err != nil {
		t.Fatal(err)
	}
	f.ids[name] = b.ID
}

// book stores a reservation that was made at bookedAt.
func (f *fixture) book(t *testing.T, start, end time.Time, bikeNames ...string) domain.Reservation {
	t.Helper()
	bikes := make([]domain.ReservedBike, len(bikeNames))
	for i, name := range bikeNames {
		b, err := f.bikes.Get(f.ids[name])
		if err != nil {
			t.Fatal(err)
		}
		bikes[i] = domain.ReservedBikeFrom(b)
	}
	r, err := domain.NewReservation(f.alice.ID, f.alice.IDNumber, bikes, domain.Period{Start: start, End: end}, bookedAt)
	if err != nil {
		t.Fatal(err)
	}
	if r, err = f.reservations.Create(r); err != nil {
		t.Fatal(err)
	}
	return r
}

func (f *fixture) cancel(t *testing.T, r domain.Reservation) {
	t.Helper()
	if err := r.Cancel(bookedAt); err != nil {
		t.Fatal(err)
	}
	if _, err := f.reservations.Update(r); err != nil {
		t.Fatal(err)
	}
}

func TestFleet(t *testing.T) {
	f := newFixture(t)
	f.bike(t, "City-01", domain.CategoryCity, domain.StatusAvailable)       // rented now
	f.bike(t, "City-02", domain.CategoryCity, domain.StatusMaintenance)     // rented now, set to maintenance meanwhile
	f.bike(t, "City-03", domain.CategoryCity, domain.StatusAvailable)       // rental ended exactly now
	f.bike(t, "Kids-01", domain.CategoryChildren, domain.StatusMaintenance) // in maintenance
	f.bike(t, "MTB-01", domain.CategoryMountain, domain.StatusRetired)      // not in fleet
	f.bike(t, "Race-01", domain.CategoryRacing, domain.StatusAvailable)     // rental starts exactly now

	h := time.Hour
	f.book(t, now.Add(-h), now.Add(h), "City-01")
	f.book(t, now.Add(-2*h), now.Add(2*h), "City-02")
	f.book(t, now.Add(-4*h), now, "City-03") // end is exclusive → not rented
	cancelled := f.book(t, now.Add(-3*h), now.Add(-2*h), "Race-01")
	f.cancel(t, cancelled)
	f.book(t, now, now.Add(h), "Race-01") // start is inclusive → rented

	got := f.builder.Build().Fleet

	if got.Total != 5 || got.RentedNow != 3 || got.Maintenance != 1 || got.AvailableNow != 1 {
		t.Errorf("fleet = total %d, rented %d, maintenance %d, available %d; want 5, 3, 1, 1",
			got.Total, got.RentedNow, got.Maintenance, got.AvailableNow)
	}
	want := []CategoryFleet{
		{domain.CategoryChildren, "Children's bike", 1, 0, 0, 1},
		{domain.CategoryCity, "City bike", 3, 1, 2, 0},
		{domain.CategoryMountain, "Mountain bike", 0, 0, 0, 0},
		{domain.CategoryRacing, "Racing bike", 1, 0, 1, 0},
	}
	if len(got.ByCategory) != len(want) {
		t.Fatalf("len(ByCategory) = %d", len(got.ByCategory))
	}
	for i, w := range want {
		c := got.ByCategory[i]
		if c != w {
			t.Errorf("ByCategory[%d] = %+v, want %+v", i, c, w)
		}
		if c.Total != c.AvailableNow+c.RentedNow+c.Maintenance {
			t.Errorf("%s: figures do not add up: %+v", c.Category, c)
		}
	}
}

func TestUpcoming(t *testing.T) {
	f := newFixture(t)
	for _, name := range []string{"A", "B", "C", "D", "E", "F"} {
		f.bike(t, name, domain.CategoryCity, domain.StatusAvailable)
	}
	week := 7 * 24 * time.Hour
	h := time.Hour

	inWeek := f.book(t, now.Add(week), now.Add(week+h), "A", "B") // exactly 7 days → included
	f.book(t, now.Add(week+time.Second), now.Add(week+h), "C")    // 7 days + 1 s → excluded
	in3Days := f.book(t, now.Add(72*h), now.Add(74*h), "D")
	f.book(t, now, now.Add(h), "E") // starts now → active, not upcoming
	f.cancel(t, f.book(t, now.Add(48*h), now.Add(49*h), "F"))
	f.book(t, now.Add(-3*h), now.Add(-2*h), "F") // completed

	got := f.builder.Build().Upcoming

	if got.Days != 7 || got.Count != 2 || len(got.Reservations) != 2 {
		t.Fatalf("upcoming = %+v", got)
	}
	first, second := got.Reservations[0], got.Reservations[1]
	if first.ID != in3Days.ID || second.ID != inWeek.ID {
		t.Errorf("order = %s, %s; want %s, %s", first.ID, second.ID, in3Days.ID, inWeek.ID)
	}
	if second.PersonName != "Alice Meyer" || strings.Join(second.Bikes, ",") != "A,B" || second.TotalPriceCents != inWeek.TotalPriceCents {
		t.Errorf("entry = %+v", second)
	}
	if !first.Start.Equal(in3Days.Start) || !first.End.Equal(in3Days.End) {
		t.Errorf("times = %v – %v", first.Start, first.End)
	}
}

func TestEmpty(t *testing.T) {
	f := newFixture(t)
	s := f.builder.Build()

	if !s.GeneratedAt.Equal(now) {
		t.Errorf("GeneratedAt = %v", s.GeneratedAt)
	}
	if s.Fleet.Total != 0 || len(s.Fleet.ByCategory) != 4 {
		t.Errorf("fleet = %+v", s.Fleet)
	}
	if s.Upcoming.Count != 0 || s.Upcoming.Reservations == nil {
		t.Errorf("upcoming = %+v (reservations must be an empty list, not null)", s.Upcoming)
	}
}
