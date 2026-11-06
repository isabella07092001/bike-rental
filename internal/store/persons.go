package store

import (
	"bytes"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"

	"bike-rental/internal/domain"
)

// PersonEntry is a person together with the metadata of their ID copy.
type PersonEntry struct {
	Person   domain.Person
	Document domain.IDDocumentMeta
}

// PersonStore persists persons and their ID copies. Implementations must be safe for
// concurrent use. Every person has exactly one ID copy; persons are never deleted.
type PersonStore interface {
	// List returns all persons (including archived ones), sorted by last and first name.
	List() []PersonEntry
	// Get returns the person with the given ID or an error wrapping domain.ErrNotFound.
	Get(id string) (PersonEntry, error)
	// Create assigns a new ID and stores the person together with the ID copy.
	// It fails with domain.ErrConflict if the ID number is already taken.
	Create(p domain.Person, doc domain.IDDocument) (PersonEntry, error)
	// Update replaces the person with p.ID; the ID copy is kept.
	// It fails with domain.ErrNotFound or domain.ErrConflict.
	Update(p domain.Person) (PersonEntry, error)
	// SetIDDocument replaces the ID copy of the person with the given ID.
	SetIDDocument(id string, doc domain.IDDocument) (PersonEntry, error)
	// GetIDDocument returns the ID copy, including its content.
	GetIDDocument(id string) (domain.IDDocument, error)
}

type personRecord struct {
	person domain.Person
	doc    domain.IDDocument
}

func (r personRecord) entry() PersonEntry {
	return PersonEntry{Person: r.person, Document: r.doc.Meta()}
}

// PersonMemoryStore is a thread-safe in-memory PersonStore.
type PersonMemoryStore struct {
	mu      sync.RWMutex
	persons map[string]personRecord
	nextID  int
}

// NewPersonMemoryStore returns an empty store.
func NewPersonMemoryStore() *PersonMemoryStore {
	return &PersonMemoryStore{persons: make(map[string]personRecord), nextID: 1}
}

// List implements PersonStore.
func (s *PersonMemoryStore) List() []PersonEntry {
	s.mu.RLock()
	defer s.mu.RUnlock()

	list := make([]PersonEntry, 0, len(s.persons))
	for _, r := range s.persons {
		list = append(list, r.entry())
	}
	sort.Slice(list, func(i, j int) bool {
		a, b := list[i].Person, list[j].Person
		if x, y := strings.ToLower(a.LastName), strings.ToLower(b.LastName); x != y {
			return x < y
		}
		if x, y := strings.ToLower(a.FirstName), strings.ToLower(b.FirstName); x != y {
			return x < y
		}
		return a.ID < b.ID
	})
	return list
}

// Get implements PersonStore.
func (s *PersonMemoryStore) Get(id string) (PersonEntry, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	r, ok := s.persons[id]
	if !ok {
		return PersonEntry{}, notFound(id)
	}
	return r.entry(), nil
}

// Create implements PersonStore.
func (s *PersonMemoryStore) Create(p domain.Person, doc domain.IDDocument) (PersonEntry, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.checkIDNumberFree(p.IDNumber, ""); err != nil {
		return PersonEntry{}, err
	}
	p.ID = "p-" + strconv.Itoa(s.nextID)
	s.nextID++
	r := personRecord{person: p, doc: copyDoc(doc)}
	s.persons[p.ID] = r
	return r.entry(), nil
}

// Update implements PersonStore.
func (s *PersonMemoryStore) Update(p domain.Person) (PersonEntry, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	r, ok := s.persons[p.ID]
	if !ok {
		return PersonEntry{}, notFound(p.ID)
	}
	if err := s.checkIDNumberFree(p.IDNumber, p.ID); err != nil {
		return PersonEntry{}, err
	}
	r.person = p
	s.persons[p.ID] = r
	return r.entry(), nil
}

// SetIDDocument implements PersonStore.
func (s *PersonMemoryStore) SetIDDocument(id string, doc domain.IDDocument) (PersonEntry, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	r, ok := s.persons[id]
	if !ok {
		return PersonEntry{}, notFound(id)
	}
	r.doc = copyDoc(doc)
	s.persons[id] = r
	return r.entry(), nil
}

// GetIDDocument implements PersonStore.
func (s *PersonMemoryStore) GetIDDocument(id string) (domain.IDDocument, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	r, ok := s.persons[id]
	if !ok {
		return domain.IDDocument{}, notFound(id)
	}
	return copyDoc(r.doc), nil
}

// checkIDNumberFree reports a conflict if another person (not exceptID) already has idNumber,
// ignoring case. Archived persons count too. The caller must hold the lock.
func (s *PersonMemoryStore) checkIDNumberFree(idNumber, exceptID string) error {
	for id, r := range s.persons {
		if id != exceptID && strings.EqualFold(r.person.IDNumber, idNumber) {
			return fmt.Errorf("%w: a person with ID number %q already exists (%s)",
				domain.ErrConflict, r.person.IDNumber, r.person.FullName())
		}
	}
	return nil
}

func notFound(id string) error {
	return fmt.Errorf("%w: person %q not found", domain.ErrNotFound, id)
}

// copyDoc copies the byte slice so callers can never modify stored data.
func copyDoc(d domain.IDDocument) domain.IDDocument {
	d.Data = bytes.Clone(d.Data)
	return d
}
