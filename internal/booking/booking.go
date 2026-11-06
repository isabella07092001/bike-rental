// Package booking combines persons, bikes and reservations to apply the booking rules
// that need more than one store: who may book, which bikes are bookable, and when.
package booking

import (
	"fmt"
	"strings"
	"time"

	"bike-rental/internal/domain"
	"bike-rental/internal/store"
)

// Service books, changes and cancels reservations.
type Service struct {
	bikes        store.BikeStore
	persons      store.PersonStore
	reservations store.ReservationStore
	now          func() time.Time
}

// New returns a booking service. now is the clock (time.Now in production).
func New(bikes store.BikeStore, persons store.PersonStore, reservations store.ReservationStore, now func() time.Time) *Service {
	return &Service{bikes: bikes, persons: persons, reservations: reservations, now: now}
}

// Now returns the service's current time.
func (s *Service) Now() time.Time {
	return s.now()
}

// Quote is a price and availability preview for a planned booking.
type Quote struct {
	BilledHours     int64    `json:"billedHours"`
	TotalPriceCents int64    `json:"totalPriceCents"`
	Conflicts       []string `json:"conflicts"`
}

// Book creates a reservation. The person must exist, be active and have an ID number and
// ID copy (Business Rule 1); all bikes must exist, be available and be free in the period.
func (s *Service) Book(personID string, bikeIDs []string, period domain.Period) (domain.Reservation, error) {
	person, err := s.bookablePerson(personID)
	if err != nil {
		return domain.Reservation{}, err
	}
	bikes, err := s.bookableBikes(bikeIDs)
	if err != nil {
		return domain.Reservation{}, err
	}
	r, err := domain.NewReservation(person.ID, person.IDNumber, bikes, period, s.now())
	if err != nil {
		return domain.Reservation{}, err
	}
	if err := s.checkConflicts(bikeIDs, period, ""); err != nil {
		return domain.Reservation{}, err
	}
	// The store checks conflicts again atomically, in case of a concurrent booking.
	return s.reservations.Create(r)
}

// Change replaces period and bikes of a reservation until 24 hours before its current start.
// The person stays the same; the price is recalculated.
func (s *Service) Change(id string, bikeIDs []string, period domain.Period) (domain.Reservation, error) {
	r, err := s.reservations.Get(id)
	if err != nil {
		return domain.Reservation{}, err
	}
	// Report a passed deadline before complaining about the new input.
	if err := r.CheckEditable(s.now()); err != nil {
		return domain.Reservation{}, err
	}
	bikes, err := s.bookableBikes(bikeIDs)
	if err != nil {
		return domain.Reservation{}, err
	}
	if err := r.Reschedule(bikes, period, s.now()); err != nil {
		return domain.Reservation{}, err
	}
	if err := s.checkConflicts(bikeIDs, period, id); err != nil {
		return domain.Reservation{}, err
	}
	return s.reservations.Update(r)
}

// Cancel cancels a reservation until 24 hours before its start.
func (s *Service) Cancel(id string) (domain.Reservation, error) {
	r, err := s.reservations.Get(id)
	if err != nil {
		return domain.Reservation{}, err
	}
	if err := r.Cancel(s.now()); err != nil {
		return domain.Reservation{}, err
	}
	return s.reservations.Update(r)
}

// Quote calculates the price for the given bikes and period and lists everything that would
// prevent the booking (unavailable or already booked bikes). excludeID ignores one reservation,
// so a reservation being edited does not conflict with itself.
func (s *Service) Quote(bikeIDs []string, period domain.Period, excludeID string) (Quote, error) {
	if err := period.Validate(s.now()); err != nil {
		return Quote{}, err
	}
	if err := checkDuplicates(bikeIDs); err != nil {
		return Quote{}, err
	}
	q := Quote{BilledHours: period.BilledHours(), Conflicts: []string{}}
	var reserved []domain.ReservedBike
	for _, id := range bikeIDs {
		b, err := s.bikes.Get(id)
		if err != nil {
			return Quote{}, err
		}
		if b.Status != domain.StatusAvailable {
			q.Conflicts = append(q.Conflicts, unavailableMessage(b))
		}
		reserved = append(reserved, domain.ReservedBikeFrom(b))
	}
	q.TotalPriceCents = domain.CalculatePrice(q.BilledHours, reserved)
	q.Conflicts = append(q.Conflicts, s.conflicts(bikeIDs, period, excludeID)...)
	return q, nil
}

// AvailableBikes returns all bikes with status available that are free in the period.
func (s *Service) AvailableBikes(period domain.Period, excludeID string) ([]domain.Bike, error) {
	if err := period.Validate(s.now()); err != nil {
		return nil, err
	}
	reservations := s.reservations.List()
	result := []domain.Bike{}
	for _, b := range s.bikes.List() {
		if b.Status == domain.StatusAvailable && !bookedIn(reservations, b.ID, period, excludeID) {
			result = append(result, b)
		}
	}
	return result, nil
}

// BikeHasOpenReservations reports whether the bike is part of an upcoming or active booking.
func (s *Service) BikeHasOpenReservations(bikeID string) bool {
	now := s.now()
	for _, r := range s.reservations.List() {
		if r.IsOpen(now) && r.HasBike(bikeID) {
			return true
		}
	}
	return false
}

// PersonHasOpenReservations reports whether the person has an upcoming or active booking.
func (s *Service) PersonHasOpenReservations(personID string) bool {
	now := s.now()
	for _, r := range s.reservations.List() {
		if r.IsOpen(now) && r.PersonID == personID {
			return true
		}
	}
	return false
}

func (s *Service) bookablePerson(id string) (domain.Person, error) {
	if id == "" {
		return domain.Person{}, fmt.Errorf("%w: select a person", domain.ErrInvalid)
	}
	e, err := s.persons.Get(id)
	if err != nil {
		return domain.Person{}, err
	}
	p := e.Person
	switch {
	case p.Archived:
		return domain.Person{}, fmt.Errorf("%w: %s is archived and cannot book", domain.ErrInvalid, p.FullName())
	case p.IDNumber == "":
		return domain.Person{}, fmt.Errorf("%w: %s has no identification number", domain.ErrInvalid, p.FullName())
	case e.Document.Size == 0:
		return domain.Person{}, fmt.Errorf("%w: %s has no uploaded ID copy", domain.ErrInvalid, p.FullName())
	}
	return p, nil
}

func (s *Service) bookableBikes(ids []string) ([]domain.ReservedBike, error) {
	if len(ids) == 0 {
		return nil, fmt.Errorf("%w: select at least one bike", domain.ErrInvalid)
	}
	if err := checkDuplicates(ids); err != nil {
		return nil, err
	}
	bikes := make([]domain.ReservedBike, 0, len(ids))
	for _, id := range ids {
		b, err := s.bikes.Get(id)
		if err != nil {
			return nil, err
		}
		if b.Status != domain.StatusAvailable {
			return nil, fmt.Errorf("%w: %s", domain.ErrInvalid, unavailableMessage(b))
		}
		bikes = append(bikes, domain.ReservedBikeFrom(b))
	}
	return bikes, nil
}

// checkConflicts returns an ErrConflict listing all bikes that are booked too close to period.
func (s *Service) checkConflicts(bikeIDs []string, period domain.Period, excludeID string) error {
	if msgs := s.conflicts(bikeIDs, period, excludeID); len(msgs) > 0 {
		return fmt.Errorf("%w: %s", domain.ErrConflict, strings.Join(msgs, "; "))
	}
	return nil
}

// conflicts describes, with bike names, every booked reservation that blocks one of the bikes.
func (s *Service) conflicts(bikeIDs []string, period domain.Period, excludeID string) []string {
	var msgs []string
	for _, r := range s.reservations.List() {
		if r.ID == excludeID || r.Status != domain.ReservationBooked || !r.Period().ConflictsWith(period) {
			continue
		}
		for _, id := range bikeIDs {
			if r.HasBike(id) {
				msgs = append(msgs, fmt.Sprintf("%s is already booked from %s to %s (30 minutes gap required)",
					s.bikeName(id), formatTime(r.Start), formatTime(r.End)))
			}
		}
	}
	return msgs
}

func (s *Service) bikeName(id string) string {
	if b, err := s.bikes.Get(id); err == nil {
		return b.Name
	}
	return id
}

func bookedIn(reservations []domain.Reservation, bikeID string, period domain.Period, excludeID string) bool {
	for _, r := range reservations {
		if r.ID != excludeID && r.Status == domain.ReservationBooked && r.HasBike(bikeID) && r.Period().ConflictsWith(period) {
			return true
		}
	}
	return false
}

func checkDuplicates(ids []string) error {
	seen := make(map[string]bool, len(ids))
	for _, id := range ids {
		if seen[id] {
			return fmt.Errorf("%w: bike %q is selected twice", domain.ErrInvalid, id)
		}
		seen[id] = true
	}
	return nil
}

func unavailableMessage(b domain.Bike) string {
	return fmt.Sprintf("%s cannot be booked (status: %s)", b.Name, b.Status)
}

// formatTime formats a time for messages in the shop's local time.
func formatTime(t time.Time) string {
	return t.Local().Format("02.01.2006 15:04")
}
