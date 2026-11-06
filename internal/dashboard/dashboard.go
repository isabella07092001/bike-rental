// Package dashboard aggregates bikes and reservations into the figures shown on the
// dashboard: the fleet status right now and the upcoming reservations.
package dashboard

import (
	"time"

	"bike-rental/internal/domain"
	"bike-rental/internal/store"
)

// UpcomingDays is how far ahead upcoming reservations are listed.
const UpcomingDays = 7

// CategoryFleet holds the fleet figures of one bike category.
type CategoryFleet struct {
	Category     domain.Category `json:"category"`
	Label        string          `json:"label"`
	Total        int             `json:"total"`
	AvailableNow int             `json:"availableNow"`
	RentedNow    int             `json:"rentedNow"`
	Maintenance  int             `json:"maintenance"`
}

// Fleet holds the fleet figures overall and per category. Retired bikes are not part of
// the fleet; every other bike counts in exactly one of AvailableNow, RentedNow, Maintenance.
type Fleet struct {
	Total        int             `json:"total"`
	AvailableNow int             `json:"availableNow"`
	RentedNow    int             `json:"rentedNow"`
	Maintenance  int             `json:"maintenance"`
	ByCategory   []CategoryFleet `json:"byCategory"`
}

// UpcomingReservation is a reservation as listed on the dashboard.
type UpcomingReservation struct {
	ID              string    `json:"id"`
	Start           time.Time `json:"start"`
	End             time.Time `json:"end"`
	PersonName      string    `json:"personName"`
	Bikes           []string  `json:"bikes"`
	TotalPriceCents int64     `json:"totalPriceCents"`
}

// Upcoming lists the booked reservations starting within the next UpcomingDays days.
type Upcoming struct {
	Days         int                   `json:"days"`
	Count        int                   `json:"count"`
	Reservations []UpcomingReservation `json:"reservations"`
}

// Summary is everything the dashboard shows.
type Summary struct {
	GeneratedAt time.Time `json:"generatedAt"`
	Fleet       Fleet     `json:"fleet"`
	Upcoming    Upcoming  `json:"upcoming"`
}

// Builder calculates dashboard summaries from the stores.
type Builder struct {
	bikes        store.BikeStore
	persons      store.PersonStore
	reservations store.ReservationStore
	now          func() time.Time
}

// NewBuilder returns a builder. now is the clock (time.Now in production).
func NewBuilder(bikes store.BikeStore, persons store.PersonStore, reservations store.ReservationStore, now func() time.Time) *Builder {
	return &Builder{bikes: bikes, persons: persons, reservations: reservations, now: now}
}

// Build calculates the summary for the current time.
func (b *Builder) Build() Summary {
	now := b.now()
	reservations := b.reservations.List()
	return Summary{
		GeneratedAt: now,
		Fleet:       b.fleet(rentedNow(reservations, now)),
		Upcoming:    b.upcoming(reservations, now),
	}
}

// rentedNow returns the IDs of all bikes in a booked reservation that is active at now.
func rentedNow(reservations []domain.Reservation, now time.Time) map[string]bool {
	rented := map[string]bool{}
	for _, r := range reservations {
		if r.Phase(now) == domain.PhaseActive {
			for _, id := range r.BikeIDs() {
				rented[id] = true
			}
		}
	}
	return rented
}

func (b *Builder) fleet(rented map[string]bool) Fleet {
	categories := domain.AllCategories()
	byCategory := make([]CategoryFleet, len(categories))
	index := make(map[domain.Category]int, len(categories))
	for i, c := range categories {
		byCategory[i] = CategoryFleet{Category: c, Label: c.Label()}
		index[c] = i
	}

	var f Fleet
	for _, bike := range b.bikes.List() {
		if bike.IsRetired() {
			continue
		}
		i, ok := index[bike.Category]
		if !ok {
			continue // unknown category cannot be stored; skip defensively
		}
		cat := &byCategory[i]
		f.Total++
		cat.Total++
		// A rented bike is physically out, even if it was set to maintenance meanwhile.
		switch {
		case rented[bike.ID]:
			f.RentedNow++
			cat.RentedNow++
		case bike.Status == domain.StatusMaintenance:
			f.Maintenance++
			cat.Maintenance++
		default:
			f.AvailableNow++
			cat.AvailableNow++
		}
	}
	f.ByCategory = byCategory
	return f
}

// upcoming lists booked reservations with now < start ≤ now + UpcomingDays, sorted by start
// (the store already returns them sorted).
func (b *Builder) upcoming(reservations []domain.Reservation, now time.Time) Upcoming {
	until := now.AddDate(0, 0, UpcomingDays)
	u := Upcoming{Days: UpcomingDays, Reservations: []UpcomingReservation{}}
	for _, r := range reservations {
		if r.Status != domain.ReservationBooked || !r.Start.After(now) || r.Start.After(until) {
			continue
		}
		u.Reservations = append(u.Reservations, UpcomingReservation{
			ID:              r.ID,
			Start:           r.Start,
			End:             r.End,
			PersonName:      b.personName(r.PersonID),
			Bikes:           b.bikeNames(r.BikeIDs()),
			TotalPriceCents: r.TotalPriceCents,
		})
	}
	u.Count = len(u.Reservations)
	return u
}

func (b *Builder) personName(id string) string {
	if e, err := b.persons.Get(id); err == nil {
		return e.Person.FullName()
	}
	return id
}

func (b *Builder) bikeNames(ids []string) []string {
	names := make([]string, len(ids))
	for i, id := range ids {
		names[i] = id
		if bike, err := b.bikes.Get(id); err == nil {
			names[i] = bike.Name
		}
	}
	return names
}
