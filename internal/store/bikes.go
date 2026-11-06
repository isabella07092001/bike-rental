// Package store provides the thread-safe in-memory persistence behind a Store interface,
// so the implementation can be swapped for a database later.
package store

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"

	"bike-rental/internal/domain"
)

// BikeStore persists bikes. Implementations must be safe for concurrent use.
// Bikes are never deleted; they are retired via Update.
type BikeStore interface {
	// List returns all bikes (including retired ones), sorted by name.
	List() []domain.Bike
	// Get returns the bike with the given ID or an error wrapping domain.ErrNotFound.
	Get(id string) (domain.Bike, error)
	// Create assigns a new ID, stores the bike and returns it.
	// It fails with domain.ErrConflict if the name is already taken.
	Create(b domain.Bike) (domain.Bike, error)
	// Update replaces the bike with b.ID.
	// It fails with domain.ErrNotFound or domain.ErrConflict.
	Update(b domain.Bike) (domain.Bike, error)
}

// MemoryStore is a thread-safe in-memory BikeStore.
type MemoryStore struct {
	mu     sync.RWMutex
	bikes  map[string]domain.Bike
	nextID int
}

// NewMemoryStore returns an empty store.
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{bikes: make(map[string]domain.Bike), nextID: 1}
}

// List implements BikeStore.
func (s *MemoryStore) List() []domain.Bike {
	s.mu.RLock()
	defer s.mu.RUnlock()

	list := make([]domain.Bike, 0, len(s.bikes))
	for _, b := range s.bikes {
		list = append(list, b)
	}
	sort.Slice(list, func(i, j int) bool {
		a, b := strings.ToLower(list[i].Name), strings.ToLower(list[j].Name)
		if a != b {
			return a < b
		}
		return list[i].ID < list[j].ID
	})
	return list
}

// Get implements BikeStore.
func (s *MemoryStore) Get(id string) (domain.Bike, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	b, ok := s.bikes[id]
	if !ok {
		return domain.Bike{}, fmt.Errorf("bike %q: %w", id, domain.ErrNotFound)
	}
	return b, nil
}

// Create implements BikeStore.
func (s *MemoryStore) Create(b domain.Bike) (domain.Bike, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.checkNameFree(b.Name, ""); err != nil {
		return domain.Bike{}, err
	}
	b.ID = "b-" + strconv.Itoa(s.nextID)
	s.nextID++
	s.bikes[b.ID] = b
	return b, nil
}

// Update implements BikeStore.
func (s *MemoryStore) Update(b domain.Bike) (domain.Bike, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.bikes[b.ID]; !ok {
		return domain.Bike{}, fmt.Errorf("bike %q: %w", b.ID, domain.ErrNotFound)
	}
	if err := s.checkNameFree(b.Name, b.ID); err != nil {
		return domain.Bike{}, err
	}
	s.bikes[b.ID] = b
	return b, nil
}

// checkNameFree reports a conflict if another bike (not exceptID) already uses name,
// ignoring case. Retired bikes count too. The caller must hold the lock.
func (s *MemoryStore) checkNameFree(name, exceptID string) error {
	for id, other := range s.bikes {
		if id != exceptID && strings.EqualFold(other.Name, name) {
			return fmt.Errorf("%w: a bike named %q already exists", domain.ErrConflict, other.Name)
		}
	}
	return nil
}
