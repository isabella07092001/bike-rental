package store

import (
	"errors"
	"fmt"
	"sync"
	"testing"

	"bike-rental/internal/domain"
)

func newBike(name string) domain.Bike {
	return domain.Bike{Name: name, Category: domain.CategoryCity, Status: domain.StatusAvailable}
}

func TestCreateAssignsIDsAndGet(t *testing.T) {
	s := NewMemoryStore()

	a, err := s.Create(newBike("City-01"))
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	b, err := s.Create(newBike("City-02"))
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if a.ID == "" || a.ID == b.ID {
		t.Fatalf("IDs not unique: %q, %q", a.ID, b.ID)
	}

	got, err := s.Get(a.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got != a {
		t.Errorf("Get = %+v, want %+v", got, a)
	}
}

func TestGetUnknown(t *testing.T) {
	_, err := NewMemoryStore().Get("b-99")
	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("err = %v, want ErrNotFound", err)
	}
}

func TestListSortedByNameIncludingRetired(t *testing.T) {
	s := NewMemoryStore()
	for _, name := range []string{"racer", "Alpha", "city"} {
		if _, err := s.Create(newBike(name)); err != nil {
			t.Fatalf("Create %q: %v", name, err)
		}
	}
	retired := newBike("Beta")
	retired.Retire()
	if _, err := s.Create(retired); err != nil {
		t.Fatalf("Create retired: %v", err)
	}

	var names []string
	for _, b := range s.List() {
		names = append(names, b.Name)
	}
	want := []string{"Alpha", "Beta", "city", "racer"}
	if fmt.Sprint(names) != fmt.Sprint(want) {
		t.Errorf("List names = %v, want %v", names, want)
	}
}

func TestCreateDuplicateNameIgnoresCase(t *testing.T) {
	s := NewMemoryStore()
	if _, err := s.Create(newBike("City-01")); err != nil {
		t.Fatalf("Create: %v", err)
	}
	_, err := s.Create(newBike("city-01"))
	if !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("err = %v, want ErrConflict", err)
	}
}

func TestDuplicateNameOfRetiredBikeConflicts(t *testing.T) {
	s := NewMemoryStore()
	b, _ := s.Create(newBike("Old"))
	b.Retire()
	if _, err := s.Update(b); err != nil {
		t.Fatalf("Update: %v", err)
	}
	if _, err := s.Create(newBike("Old")); !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("err = %v, want ErrConflict", err)
	}
}

func TestUpdate(t *testing.T) {
	s := NewMemoryStore()
	a, _ := s.Create(newBike("A"))
	b, _ := s.Create(newBike("B"))

	// Keeping its own name (different case) is allowed.
	a.Name = "a"
	a.Status = domain.StatusMaintenance
	if _, err := s.Update(a); err != nil {
		t.Fatalf("Update own name: %v", err)
	}
	if got, _ := s.Get(a.ID); got.Name != "a" || got.Status != domain.StatusMaintenance {
		t.Errorf("after update: %+v", got)
	}

	// Taking another bike's name is not.
	b.Name = "A"
	if _, err := s.Update(b); !errors.Is(err, domain.ErrConflict) {
		t.Errorf("Update to taken name: err = %v, want ErrConflict", err)
	}

	if _, err := s.Update(domain.Bike{ID: "b-99", Name: "X"}); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("Update unknown: err = %v, want ErrNotFound", err)
	}
}

func TestListReturnsCopies(t *testing.T) {
	s := NewMemoryStore()
	created, _ := s.Create(newBike("City-01"))

	list := s.List()
	list[0].Name = "changed"

	if got, _ := s.Get(created.ID); got.Name != "City-01" {
		t.Errorf("store was modified through List result: %q", got.Name)
	}
}

func TestConcurrentAccess(t *testing.T) {
	s := NewMemoryStore()
	const n = 50

	var wg sync.WaitGroup
	for i := range n {
		wg.Add(1)
		go func() {
			defer wg.Done()
			b, err := s.Create(newBike(fmt.Sprintf("bike-%d", i)))
			if err != nil {
				t.Errorf("Create: %v", err)
				return
			}
			b.Status = domain.StatusMaintenance
			if _, err := s.Update(b); err != nil {
				t.Errorf("Update: %v", err)
			}
			s.List()
			if _, err := s.Get(b.ID); err != nil {
				t.Errorf("Get: %v", err)
			}
		}()
	}
	wg.Wait()

	if got := len(s.List()); got != n {
		t.Errorf("len(List()) = %d, want %d", got, n)
	}
}
