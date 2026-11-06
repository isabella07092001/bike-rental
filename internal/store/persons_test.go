package store

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"bike-rental/internal/domain"
)

func newPerson(first, last, idNumber string) domain.Person {
	return domain.Person{FirstName: first, LastName: last, IDNumber: idNumber, Phone: "0170 1234567"}
}

func newDoc(t *testing.T, content string) domain.IDDocument {
	t.Helper()
	doc, err := domain.NewIDDocument("id.pdf", []byte("%PDF-1.4\n"+content), time.Now())
	if err != nil {
		t.Fatalf("NewIDDocument: %v", err)
	}
	return doc
}

func TestPersonCreateGetAndDocument(t *testing.T) {
	s := NewPersonMemoryStore()
	doc := newDoc(t, "alice")

	created, err := s.Create(newPerson("Alice", "Meyer", "A1"), doc)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if created.Person.ID == "" {
		t.Fatal("Create did not assign an ID")
	}
	if created.Document != doc.Meta() {
		t.Errorf("Document = %+v, want %+v", created.Document, doc.Meta())
	}

	got, err := s.Get(created.Person.ID)
	if err != nil || got != created {
		t.Fatalf("Get = %+v, %v; want %+v", got, err, created)
	}

	stored, err := s.GetIDDocument(created.Person.ID)
	if err != nil {
		t.Fatalf("GetIDDocument: %v", err)
	}
	if string(stored.Data) != string(doc.Data) {
		t.Errorf("stored data = %q, want %q", stored.Data, doc.Data)
	}
}

func TestPersonNotFound(t *testing.T) {
	s := NewPersonMemoryStore()
	checks := map[string]error{}
	_, checks["Get"] = s.Get("p-99")
	_, checks["Update"] = s.Update(domain.Person{ID: "p-99", IDNumber: "X"})
	_, checks["SetIDDocument"] = s.SetIDDocument("p-99", newDoc(t, ""))
	_, checks["GetIDDocument"] = s.GetIDDocument("p-99")
	for name, err := range checks {
		if !errors.Is(err, domain.ErrNotFound) {
			t.Errorf("%s: err = %v, want ErrNotFound", name, err)
		}
	}
}

func TestPersonDuplicateIDNumber(t *testing.T) {
	s := NewPersonMemoryStore()
	a, _ := s.Create(newPerson("Alice", "Meyer", "AB123"), newDoc(t, ""))
	b, _ := s.Create(newPerson("Bob", "Schmidt", "CD456"), newDoc(t, ""))

	if _, err := s.Create(newPerson("Eve", "X", "ab123"), newDoc(t, "")); !errors.Is(err, domain.ErrConflict) {
		t.Errorf("Create duplicate: err = %v, want ErrConflict", err)
	}

	// An archived person still blocks the ID number.
	a.Person.Archive()
	if _, err := s.Update(a.Person); err != nil {
		t.Fatalf("archive: %v", err)
	}
	if _, err := s.Create(newPerson("Eve", "X", "AB123"), newDoc(t, "")); !errors.Is(err, domain.ErrConflict) {
		t.Errorf("Create duplicate of archived: err = %v, want ErrConflict", err)
	}

	// Taking another person's ID number on update is rejected; keeping your own is fine.
	b.Person.IDNumber = "AB123"
	if _, err := s.Update(b.Person); !errors.Is(err, domain.ErrConflict) {
		t.Errorf("Update to taken ID number: err = %v, want ErrConflict", err)
	}
	b.Person.IDNumber = "CD456"
	b.Person.Phone = "0711 999999"
	if _, err := s.Update(b.Person); err != nil {
		t.Errorf("Update keeping own ID number: %v", err)
	}
}

func TestPersonUpdateKeepsDocument(t *testing.T) {
	s := NewPersonMemoryStore()
	doc := newDoc(t, "original")
	e, _ := s.Create(newPerson("Alice", "Meyer", "A1"), doc)

	e.Person.LastName = "Müller"
	updated, err := s.Update(e.Person)
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if updated.Person.LastName != "Müller" || updated.Document != doc.Meta() {
		t.Errorf("after Update: %+v", updated)
	}
}

func TestPersonReplaceDocument(t *testing.T) {
	s := NewPersonMemoryStore()
	e, _ := s.Create(newPerson("Alice", "Meyer", "A1"), newDoc(t, "old"))

	replacement := newDoc(t, "new and longer")
	updated, err := s.SetIDDocument(e.Person.ID, replacement)
	if err != nil {
		t.Fatalf("SetIDDocument: %v", err)
	}
	if updated.Document != replacement.Meta() {
		t.Errorf("Document = %+v, want %+v", updated.Document, replacement.Meta())
	}
	stored, _ := s.GetIDDocument(e.Person.ID)
	if !strings.HasSuffix(string(stored.Data), "new and longer") {
		t.Errorf("stored data = %q", stored.Data)
	}
}

func TestPersonDocumentIsCopied(t *testing.T) {
	s := NewPersonMemoryStore()
	doc := newDoc(t, "secret")
	e, _ := s.Create(newPerson("Alice", "Meyer", "A1"), doc)

	doc.Data[0] = 'X' // caller modifies its own slice after storing
	got, _ := s.GetIDDocument(e.Person.ID)
	got.Data[1] = 'Y' // caller modifies the returned slice

	again, _ := s.GetIDDocument(e.Person.ID)
	if !strings.HasPrefix(string(again.Data), "%PDF") {
		t.Errorf("stored data was modified from outside: %q", again.Data)
	}
}

func TestPersonListSorted(t *testing.T) {
	s := NewPersonMemoryStore()
	for i, p := range []domain.Person{
		newPerson("Zoe", "meyer", "1"),
		newPerson("Bob", "Albers", "2"),
		newPerson("anna", "Meyer", "3"),
	} {
		if _, err := s.Create(p, newDoc(t, fmt.Sprint(i))); err != nil {
			t.Fatalf("Create: %v", err)
		}
	}
	var names []string
	for _, e := range s.List() {
		names = append(names, e.Person.FullName())
	}
	want := "Bob Albers,anna Meyer,Zoe meyer"
	if got := strings.Join(names, ","); got != want {
		t.Errorf("List = %q, want %q", got, want)
	}
}

func TestPersonConcurrentAccess(t *testing.T) {
	s := NewPersonMemoryStore()
	const n = 50
	doc := newDoc(t, "")

	var wg sync.WaitGroup
	for i := range n {
		wg.Add(1)
		go func() {
			defer wg.Done()
			e, err := s.Create(newPerson("P", fmt.Sprint(i), fmt.Sprintf("ID%d", i)), doc)
			if err != nil {
				t.Errorf("Create: %v", err)
				return
			}
			e.Person.Archive()
			if _, err := s.Update(e.Person); err != nil {
				t.Errorf("Update: %v", err)
			}
			if _, err := s.SetIDDocument(e.Person.ID, doc); err != nil {
				t.Errorf("SetIDDocument: %v", err)
			}
			s.List()
			if _, err := s.GetIDDocument(e.Person.ID); err != nil {
				t.Errorf("GetIDDocument: %v", err)
			}
		}()
	}
	wg.Wait()

	if got := len(s.List()); got != n {
		t.Errorf("len(List()) = %d, want %d", got, n)
	}
}
