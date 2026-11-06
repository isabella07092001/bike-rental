package store

import (
	"fmt"
	"slices"
	"sort"
	"strconv"
	"sync"

	"bike-rental/internal/domain"
)

// ReservationStore persists reservations. Implementations must be safe for concurrent use
// and must check bike conflicts atomically with the write, so a bike is never double-booked.
// Reservations are never deleted; they are cancelled via Update.
type ReservationStore interface {
	// List returns all reservations (including cancelled ones), sorted by start.
	List() []domain.Reservation
	// Get returns the reservation with the given ID or an error wrapping domain.ErrNotFound.
	Get(id string) (domain.Reservation, error)
	// Create assigns a new ID and stores r. It fails with domain.ErrConflict if one of its
	// bikes is already booked too close to r's period (see domain.Period.ConflictsWith).
	Create(r domain.Reservation) (domain.Reservation, error)
	// Update replaces the reservation with r.ID, with the same conflict check
	// (ignoring r itself). It fails with domain.ErrNotFound or domain.ErrConflict.
	Update(r domain.Reservation) (domain.Reservation, error)
}

// ReservationMemoryStore is a thread-safe in-memory ReservationStore.
type ReservationMemoryStore struct {
	mu           sync.RWMutex
	reservations map[string]domain.Reservation
	nextID       int
}

// NewReservationMemoryStore returns an empty store.
func NewReservationMemoryStore() *ReservationMemoryStore {
	return &ReservationMemoryStore{reservations: make(map[string]domain.Reservation), nextID: 1}
}

// List implements ReservationStore.
func (s *ReservationMemoryStore) List() []domain.Reservation {
	s.mu.RLock()
	defer s.mu.RUnlock()

	list := make([]domain.Reservation, 0, len(s.reservations))
	for _, r := range s.reservations {
		list = append(list, copyReservation(r))
	}
	sort.Slice(list, func(i, j int) bool {
		if !list[i].Start.Equal(list[j].Start) {
			return list[i].Start.Before(list[j].Start)
		}
		return list[i].ID < list[j].ID
	})
	return list
}

// Get implements ReservationStore.
func (s *ReservationMemoryStore) Get(id string) (domain.Reservation, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	r, ok := s.reservations[id]
	if !ok {
		return domain.Reservation{}, fmt.Errorf("reservation %q: %w", id, domain.ErrNotFound)
	}
	return copyReservation(r), nil
}

// Create implements ReservationStore.
func (s *ReservationMemoryStore) Create(r domain.Reservation) (domain.Reservation, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.checkConflicts(r, ""); err != nil {
		return domain.Reservation{}, err
	}
	r.ID = "r-" + strconv.Itoa(s.nextID)
	s.nextID++
	s.reservations[r.ID] = copyReservation(r)
	return copyReservation(r), nil
}

// Update implements ReservationStore.
func (s *ReservationMemoryStore) Update(r domain.Reservation) (domain.Reservation, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.reservations[r.ID]; !ok {
		return domain.Reservation{}, fmt.Errorf("reservation %q: %w", r.ID, domain.ErrNotFound)
	}
	if err := s.checkConflicts(r, r.ID); err != nil {
		return domain.Reservation{}, err
	}
	s.reservations[r.ID] = copyReservation(r)
	return copyReservation(r), nil
}

// checkConflicts fails if a booked reservation other than exceptID shares a bike with r
// and is too close in time. Cancelled reservations never conflict. The caller must hold the lock.
func (s *ReservationMemoryStore) checkConflicts(r domain.Reservation, exceptID string) error {
	if r.Status != domain.ReservationBooked {
		return nil
	}
	for id, other := range s.reservations {
		if id == exceptID || other.Status != domain.ReservationBooked || !other.Period().ConflictsWith(r.Period()) {
			continue
		}
		for _, b := range r.Bikes {
			if other.HasBike(b.BikeID) {
				return fmt.Errorf("%w: bike %q is already booked from %s to %s (reservation %s; 30 minutes gap required)",
					domain.ErrConflict, b.BikeID, formatTime(other.Start), formatTime(other.End), id)
			}
		}
	}
	return nil
}

func copyReservation(r domain.Reservation) domain.Reservation {
	r.Bikes = slices.Clone(r.Bikes)
	return r
}
