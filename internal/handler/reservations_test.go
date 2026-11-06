package handler

import (
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"bike-rental/internal/booking"
	"bike-rental/internal/domain"
)

var resTestNow = time.Date(2026, 11, 6, 9, 0, 0, 0, time.UTC)

// tm returns an RFC 3339 time on 2026-11-07 (the day after resTestNow).
func tm(hh, mm int) string {
	return time.Date(2026, 11, 7, hh, mm, 0, 0, time.UTC).Format(time.RFC3339)
}

type resFixture struct {
	router   http.Handler
	clock    *time.Time
	city     domain.Bike
	kids     domain.Bike
	alice    domain.Person
	bob      domain.Person
	archived domain.Person
}

func newResFixture(t *testing.T) *resFixture {
	t.Helper()
	clock := resTestNow
	d := newTestDeps()
	d.Now = func() time.Time { return clock }
	f := &resFixture{router: NewRouter("", d), clock: &clock}

	var err error
	if f.city, err = d.Bikes.Create(domain.Bike{Name: "City-01", Category: domain.CategoryCity, Status: domain.StatusAvailable}); err != nil {
		t.Fatal(err)
	}
	if f.kids, err = d.Bikes.Create(domain.Bike{Name: "Kids-01", Category: domain.CategoryChildren, Status: domain.StatusAvailable}); err != nil {
		t.Fatal(err)
	}
	doc, _ := domain.NewIDDocument("id.pdf", testPDF, resTestNow)
	person := func(first string, archived bool) domain.Person {
		e, err := d.Persons.Create(domain.Person{FirstName: first, LastName: "Test", IDNumber: "ID-" + first, Phone: "12345", Archived: archived}, doc)
		if err != nil {
			t.Fatal(err)
		}
		return e.Person
	}
	f.alice, f.bob, f.archived = person("Alice", false), person("Bob", false), person("Old", true)
	return f
}

func (f *resFixture) book(t *testing.T, personID string, bikeIDs []string, start, end string) (reservationView, int, string) {
	t.Helper()
	body := fmt.Sprintf(`{"personId":%q,"bikeIds":[%s],"start":%q,"end":%q}`, personID, quoteAll(bikeIDs), start, end)
	var v reservationView
	var resp map[string]any
	code := doRaw(t, f.router, http.MethodPost, "/api/reservations", body, &v, &resp)
	msg, _ := resp["error"].(string)
	return v, code, msg
}

func quoteAll(ids []string) string {
	q := make([]string, len(ids))
	for i, id := range ids {
		q[i] = fmt.Sprintf("%q", id)
	}
	return strings.Join(q, ",")
}

// doRaw decodes a success response into ok and an error response into fail.
func doRaw(t *testing.T, router http.Handler, method, path, body string, ok any, fail *map[string]any) int {
	t.Helper()
	var raw map[string]any
	code := do(t, router, method, path, body, &raw)
	if code >= 400 {
		*fail = raw
		return code
	}
	b, _ := jsonMarshal(raw)
	if err := jsonUnmarshal(b, ok); err != nil {
		t.Fatalf("decode %s %s: %v", method, path, err)
	}
	return code
}

func TestReservationCreateAndView(t *testing.T) {
	f := newResFixture(t)

	v, code, msg := f.book(t, f.alice.ID, []string{f.city.ID, f.kids.ID}, tm(10, 0), tm(12, 30))
	if code != http.StatusCreated {
		t.Fatalf("create: %d %s", code, msg)
	}
	// 2 h 30 min → 3 h × (6 + 4) € = 30 €
	if v.BilledHours != 3 || v.TotalPriceCents != 3000 || v.PersonName != "Alice Test" || v.IDNumber != "ID-Alice" {
		t.Errorf("view = %+v", v)
	}
	if len(v.Bikes) != 2 || v.Bikes[0].Name != "City-01" || v.Bikes[1].CategoryLabel != "Children's bike" {
		t.Errorf("bikes = %+v", v.Bikes)
	}
	if v.Phase != domain.PhaseUpcoming || !v.CanChange || v.Status != domain.ReservationBooked {
		t.Errorf("phase/canChange/status = %q/%v/%q", v.Phase, v.CanChange, v.Status)
	}

	var got reservationView
	if code := do(t, f.router, http.MethodGet, "/api/reservations/"+v.ID, "", &got); code != http.StatusOK || got.ID != v.ID {
		t.Fatalf("get: %d %+v", code, got)
	}
}

func TestReservationCreateErrors(t *testing.T) {
	f := newResFixture(t)
	if _, code, msg := f.book(t, f.alice.ID, []string{f.city.ID}, tm(10, 0), tm(12, 0)); code != http.StatusCreated {
		t.Fatalf("setup booking: %d %s", code, msg)
	}

	tests := []struct {
		name       string
		person     string
		bikes      []string
		start, end string
		wantStatus int
		wantMsg    string
	}{
		{"29 minutes", f.bob.ID, []string{f.kids.ID}, tm(10, 0), tm(10, 29), 400, "at least 30 minutes"},
		{"48 h 1 min", f.bob.ID, []string{f.kids.ID}, tm(10, 0), time.Date(2026, 11, 9, 10, 1, 0, 0, time.UTC).Format(time.RFC3339), 400, "at most 48 hours"},
		{"start in the past", f.bob.ID, []string{f.kids.ID}, "2026-11-06T08:00:00Z", "2026-11-06T10:00:00Z", 400, "in the future"},
		{"invalid time", f.bob.ID, []string{f.kids.ID}, "tomorrow", tm(11, 0), 400, "RFC 3339"},
		{"missing end", f.bob.ID, []string{f.kids.ID}, tm(10, 0), "", 400, "required"},
		{"no bikes", f.bob.ID, nil, tm(10, 0), tm(11, 0), 400, "at least one bike"},
		{"archived person", f.archived.ID, []string{f.kids.ID}, tm(10, 0), tm(11, 0), 400, "archived"},
		{"unknown person", "p-99", []string{f.kids.ID}, tm(10, 0), tm(11, 0), 404, "not found"},
		{"unknown bike", f.bob.ID, []string{"b-99"}, tm(10, 0), tm(11, 0), 404, "not found"},
		{"double booking", f.bob.ID, []string{f.city.ID}, tm(11, 0), tm(12, 0), 409, "City-01 is already booked"},
		{"29 min gap", f.bob.ID, []string{f.city.ID}, tm(12, 29), tm(13, 0), 409, "30 minutes gap"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, code, msg := f.book(t, tt.person, tt.bikes, tt.start, tt.end)
			if code != tt.wantStatus || !strings.Contains(msg, tt.wantMsg) {
				t.Errorf("got %d %q, want %d containing %q", code, msg, tt.wantStatus, tt.wantMsg)
			}
		})
	}

	if _, code, msg := f.book(t, f.bob.ID, []string{f.city.ID}, tm(12, 30), tm(13, 0)); code != http.StatusCreated {
		t.Errorf("exactly 30 min gap: %d %s", code, msg)
	}
}

func TestReservationUpdateAndCancel(t *testing.T) {
	f := newResFixture(t)
	v, _, _ := f.book(t, f.alice.ID, []string{f.city.ID}, tm(10, 0), tm(12, 0))

	var updated reservationView
	code := do(t, f.router, http.MethodPut, "/api/reservations/"+v.ID,
		fmt.Sprintf(`{"bikeIds":[%q,%q],"start":%q,"end":%q}`, f.city.ID, f.kids.ID, tm(10, 0), tm(10, 30)), &updated)
	if code != http.StatusOK || updated.TotalPriceCents != 1000 || len(updated.Bikes) != 2 || updated.PersonID != f.alice.ID {
		t.Fatalf("update: %d %+v", code, updated)
	}

	// 1 second after the deadline: no more changes.
	*f.clock = time.Date(2026, 11, 6, 10, 0, 1, 0, time.UTC)
	var resp map[string]string
	code = do(t, f.router, http.MethodPut, "/api/reservations/"+v.ID,
		fmt.Sprintf(`{"bikeIds":[%q],"start":%q,"end":%q}`, f.city.ID, tm(14, 0), tm(15, 0)), &resp)
	if code != http.StatusConflict || !strings.Contains(resp["error"], "24 hours") {
		t.Errorf("update after deadline: %d %q", code, resp["error"])
	}
	code = do(t, f.router, http.MethodPost, "/api/reservations/"+v.ID+"/cancel", "", &resp)
	if code != http.StatusConflict || !strings.Contains(resp["error"], "24 hours") {
		t.Errorf("cancel after deadline: %d %q", code, resp["error"])
	}
	var view reservationView
	do(t, f.router, http.MethodGet, "/api/reservations/"+v.ID, "", &view)
	if view.CanChange {
		t.Error("canChange = true after deadline")
	}

	// Exactly at the deadline: cancel works.
	*f.clock = time.Date(2026, 11, 6, 10, 0, 0, 0, time.UTC)
	var cancelled reservationView
	code = do(t, f.router, http.MethodPost, "/api/reservations/"+v.ID+"/cancel", "", &cancelled)
	if code != http.StatusOK || cancelled.Status != domain.ReservationCancelled || cancelled.Phase != domain.PhaseCancelled || cancelled.CancelledAt.IsZero() {
		t.Fatalf("cancel at deadline: %d %+v", code, cancelled)
	}

	for _, req := range []struct{ method, path, body string }{
		{http.MethodGet, "/api/reservations/r-99", ""},
		{http.MethodPut, "/api/reservations/r-99", `{}`},
		{http.MethodPost, "/api/reservations/r-99/cancel", ""},
	} {
		if code := do(t, f.router, req.method, req.path, req.body, nil); code != http.StatusNotFound {
			t.Errorf("%s %s: %d, want 404", req.method, req.path, code)
		}
	}
}

func TestReservationListFilters(t *testing.T) {
	f := newResFixture(t)
	a, _, _ := f.book(t, f.alice.ID, []string{f.city.ID}, tm(10, 0), tm(11, 0))
	f.book(t, f.bob.ID, []string{f.kids.ID}, tm(8, 0), tm(9, 0))
	c, _, _ := f.book(t, f.bob.ID, []string{f.city.ID}, tm(14, 0), tm(15, 0))
	do(t, f.router, http.MethodPost, "/api/reservations/"+c.ID+"/cancel", "", nil)

	tests := []struct {
		query string
		want  int
	}{
		{"", 3},
		{"?phase=upcoming", 2},
		{"?phase=cancelled", 1},
		{"?personId=" + f.alice.ID, 1},
		{"?bikeId=" + f.city.ID, 2},
		{"?bikeId=" + f.city.ID + "&phase=upcoming", 1},
	}
	for _, tt := range tests {
		var list []reservationView
		if code := do(t, f.router, http.MethodGet, "/api/reservations"+tt.query, "", &list); code != http.StatusOK || len(list) != tt.want {
			t.Errorf("%q: %d, %d results; want %d", tt.query, code, len(list), tt.want)
		}
	}

	var list []reservationView
	do(t, f.router, http.MethodGet, "/api/reservations", "", &list)
	if list[0].Start.After(list[1].Start) {
		t.Error("list not sorted by start")
	}
	if list[1].ID != a.ID {
		t.Errorf("second entry = %s, want %s", list[1].ID, a.ID)
	}

	if code := do(t, f.router, http.MethodGet, "/api/reservations?phase=soon", "", nil); code != http.StatusBadRequest {
		t.Errorf("invalid phase: %d", code)
	}
}

func TestQuoteAndAvailableBikes(t *testing.T) {
	f := newResFixture(t)
	v, _, _ := f.book(t, f.alice.ID, []string{f.city.ID}, tm(10, 0), tm(12, 0))

	var q booking.Quote
	code := do(t, f.router, http.MethodPost, "/api/reservations/quote",
		fmt.Sprintf(`{"bikeIds":[%q,%q],"start":%q,"end":%q}`, f.city.ID, f.kids.ID, tm(11, 0), tm(12, 1)), &q)
	if code != http.StatusOK || q.BilledHours != 2 || q.TotalPriceCents != 2000 || len(q.Conflicts) != 1 {
		t.Fatalf("quote: %d %+v", code, q)
	}

	code = do(t, f.router, http.MethodPost, "/api/reservations/quote",
		fmt.Sprintf(`{"bikeIds":[%q],"start":%q,"end":%q,"reservationId":%q}`, f.city.ID, tm(11, 0), tm(12, 1), v.ID), &q)
	if code != http.StatusOK || len(q.Conflicts) != 0 {
		t.Errorf("quote excluding own reservation: %d %+v", code, q)
	}

	var bikes []domain.Bike
	do(t, f.router, http.MethodGet, "/api/bikes/available?start="+tm(11, 0)+"&end="+tm(12, 0), "", &bikes)
	if len(bikes) != 1 || bikes[0].ID != f.kids.ID {
		t.Errorf("available = %+v", bikes)
	}
	do(t, f.router, http.MethodGet, "/api/bikes/available?start="+tm(11, 0)+"&end="+tm(12, 0)+"&excludeReservation="+v.ID, "", &bikes)
	if len(bikes) != 2 {
		t.Errorf("available excluding own reservation = %d bikes", len(bikes))
	}
	if code := do(t, f.router, http.MethodGet, "/api/bikes/available?start="+tm(11, 0), "", nil); code != http.StatusBadRequest {
		t.Errorf("available without end: %d", code)
	}
}

func TestRetireAndArchiveHooks(t *testing.T) {
	f := newResFixture(t)
	v, _, _ := f.book(t, f.alice.ID, []string{f.city.ID}, tm(10, 0), tm(12, 0))

	var resp map[string]string
	if code := do(t, f.router, http.MethodPost, "/api/bikes/"+f.city.ID+"/retire", "", &resp); code != http.StatusConflict ||
		!strings.Contains(resp["error"], "City-01 has upcoming or active reservations") {
		t.Errorf("retire booked bike: %d %q", code, resp["error"])
	}
	if code := do(t, f.router, http.MethodPut, "/api/bikes/"+f.city.ID,
		`{"name":"City-01","category":"city","status":"retired"}`, &resp); code != http.StatusConflict {
		t.Errorf("retire booked bike via PUT: %d", code)
	}
	if code := do(t, f.router, http.MethodPut, "/api/bikes/"+f.city.ID,
		`{"name":"City-01","category":"city","status":"maintenance"}`, nil); code != http.StatusOK {
		t.Errorf("maintenance for booked bike should stay allowed: %d", code)
	}
	if code := do(t, f.router, http.MethodPost, "/api/persons/"+f.alice.ID+"/archive", "", &resp); code != http.StatusConflict ||
		!strings.Contains(resp["error"], "Alice Test has upcoming or active reservations") {
		t.Errorf("archive person with booking: %d %q", code, resp["error"])
	}

	// Unbooked bike and person can still be retired/archived.
	if code := do(t, f.router, http.MethodPost, "/api/bikes/"+f.kids.ID+"/retire", "", nil); code != http.StatusOK {
		t.Errorf("retire free bike: %d", code)
	}
	if code := do(t, f.router, http.MethodPost, "/api/persons/"+f.bob.ID+"/archive", "", nil); code != http.StatusOK {
		t.Errorf("archive free person: %d", code)
	}

	// After cancelling, the hooks no longer block.
	do(t, f.router, http.MethodPost, "/api/reservations/"+v.ID+"/cancel", "", nil)
	if code := do(t, f.router, http.MethodPost, "/api/bikes/"+f.city.ID+"/retire", "", nil); code != http.StatusOK {
		t.Errorf("retire after cancel: %d", code)
	}
	if code := do(t, f.router, http.MethodPost, "/api/persons/"+f.alice.ID+"/archive", "", nil); code != http.StatusOK {
		t.Errorf("archive after cancel: %d", code)
	}
}
