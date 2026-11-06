package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"bike-rental/internal/booking"
	"bike-rental/internal/domain"
	"bike-rental/internal/store"
)

// ReservationHandler serves /api/reservations and /api/bikes/available.
type ReservationHandler struct {
	bookings     *booking.Service
	reservations store.ReservationStore
	bikes        store.BikeStore
	persons      store.PersonStore
}

// NewReservationHandler returns a handler using the booking service for all rule checks;
// the stores are only read, to add person and bike names to responses.
func NewReservationHandler(b *booking.Service, r store.ReservationStore, bikes store.BikeStore, persons store.PersonStore) *ReservationHandler {
	return &ReservationHandler{bookings: b, reservations: r, bikes: bikes, persons: persons}
}

// Register adds the reservation routes to mux.
func (h *ReservationHandler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/reservations", h.list)
	mux.HandleFunc("POST /api/reservations", h.create)
	mux.HandleFunc("POST /api/reservations/quote", h.quote)
	mux.HandleFunc("GET /api/reservations/{id}", h.get)
	mux.HandleFunc("PUT /api/reservations/{id}", h.update)
	mux.HandleFunc("POST /api/reservations/{id}/cancel", h.cancel)
	mux.HandleFunc("GET /api/bikes/available", h.availableBikes)
}

// reservationInput is the request body for create, update and quote.
// Times are RFC 3339 strings, parsed explicitly for clear error messages.
type reservationInput struct {
	PersonID      string   `json:"personId"`
	BikeIDs       []string `json:"bikeIds"`
	Start         string   `json:"start"`
	End           string   `json:"end"`
	ReservationID string   `json:"reservationId"` // quote only: the reservation being edited
}

func (in reservationInput) period() (domain.Period, error) {
	return parsePeriod(in.Start, in.End)
}

type reservedBikeView struct {
	BikeID           string          `json:"bikeId"`
	Name             string          `json:"name"`
	Category         domain.Category `json:"category"`
	CategoryLabel    string          `json:"categoryLabel"`
	HourlyPriceCents int64           `json:"hourlyPriceCents"`
}

// reservationView is the JSON shape of a reservation, enriched with names and derived state.
type reservationView struct {
	ID              string                   `json:"id"`
	PersonID        string                   `json:"personId"`
	PersonName      string                   `json:"personName"`
	IDNumber        string                   `json:"idNumber"`
	Bikes           []reservedBikeView       `json:"bikes"`
	Start           time.Time                `json:"start"`
	End             time.Time                `json:"end"`
	BilledHours     int64                    `json:"billedHours"`
	TotalPriceCents int64                    `json:"totalPriceCents"`
	Status          domain.ReservationStatus `json:"status"`
	Phase           domain.ReservationPhase  `json:"phase"`
	CanChange       bool                     `json:"canChange"`
	ChangeDeadline  time.Time                `json:"changeDeadline"`
	CreatedAt       time.Time                `json:"createdAt"`
	UpdatedAt       time.Time                `json:"updatedAt"`
	CancelledAt     time.Time                `json:"cancelledAt,omitzero"`
}

func (h *ReservationHandler) view(r domain.Reservation) reservationView {
	now := h.bookings.Now()
	v := reservationView{
		ID:              r.ID,
		PersonID:        r.PersonID,
		PersonName:      r.PersonID,
		IDNumber:        r.IDNumber,
		Bikes:           make([]reservedBikeView, 0, len(r.Bikes)),
		Start:           r.Start,
		End:             r.End,
		BilledHours:     r.BilledHours,
		TotalPriceCents: r.TotalPriceCents,
		Status:          r.Status,
		Phase:           r.Phase(now),
		CanChange:       r.CanChange(now),
		ChangeDeadline:  r.ChangeDeadline(),
		CreatedAt:       r.CreatedAt,
		UpdatedAt:       r.UpdatedAt,
		CancelledAt:     r.CancelledAt,
	}
	if e, err := h.persons.Get(r.PersonID); err == nil {
		v.PersonName = e.Person.FullName()
	}
	for _, b := range r.Bikes {
		name := b.BikeID
		if bike, err := h.bikes.Get(b.BikeID); err == nil {
			name = bike.Name
		}
		v.Bikes = append(v.Bikes, reservedBikeView{
			BikeID:           b.BikeID,
			Name:             name,
			Category:         b.Category,
			CategoryLabel:    b.Category.Label(),
			HourlyPriceCents: b.HourlyPriceCents,
		})
	}
	return v
}

func bikeNames(v reservationView) []string {
	names := make([]string, len(v.Bikes))
	for i, b := range v.Bikes {
		names[i] = b.Name
	}
	return names
}

// list returns reservations sorted by start. Filters: ?phase=, ?personId=, ?bikeId=,
// and ?q= searching person name, ID number and bike names.
func (h *ReservationHandler) list(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	phase := domain.ReservationPhase(q.Get("phase"))
	if phase != "" && !phase.Valid() {
		writeError(w, fmt.Errorf("%w: unknown phase %q", domain.ErrInvalid, phase))
		return
	}
	personID, bikeID := q.Get("personId"), q.Get("bikeId")
	search := searchQuery(r)
	now := h.bookings.Now()

	result := []reservationView{}
	for _, res := range h.reservations.List() {
		if phase != "" && res.Phase(now) != phase {
			continue
		}
		if personID != "" && res.PersonID != personID {
			continue
		}
		if bikeID != "" && !res.HasBike(bikeID) {
			continue
		}
		v := h.view(res)
		if !matchesAny(search, append([]string{v.PersonName, v.IDNumber}, bikeNames(v)...)...) {
			continue
		}
		result = append(result, v)
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *ReservationHandler) get(w http.ResponseWriter, r *http.Request) {
	res, err := h.reservations.Get(r.PathValue("id"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, h.view(res))
}

func (h *ReservationHandler) create(w http.ResponseWriter, r *http.Request) {
	in, p, err := decodeReservationInput(w, r)
	if err != nil {
		writeError(w, err)
		return
	}
	res, err := h.bookings.Book(in.PersonID, in.BikeIDs, p)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, h.view(res))
}

// update changes period and bikes; the person cannot be changed.
func (h *ReservationHandler) update(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if _, err := h.reservations.Get(id); err != nil {
		writeError(w, err)
		return
	}
	in, p, err := decodeReservationInput(w, r)
	if err != nil {
		writeError(w, err)
		return
	}
	res, err := h.bookings.Change(id, in.BikeIDs, p)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, h.view(res))
}

func (h *ReservationHandler) cancel(w http.ResponseWriter, r *http.Request) {
	res, err := h.bookings.Cancel(r.PathValue("id"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, h.view(res))
}

func (h *ReservationHandler) quote(w http.ResponseWriter, r *http.Request) {
	in, p, err := decodeReservationInput(w, r)
	if err != nil {
		writeError(w, err)
		return
	}
	q, err := h.bookings.Quote(in.BikeIDs, p, in.ReservationID)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, q)
}

// availableBikes lists bookable bikes free in ?start=&end=, ignoring ?excludeReservation=.
func (h *ReservationHandler) availableBikes(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	p, err := parsePeriod(q.Get("start"), q.Get("end"))
	if err != nil {
		writeError(w, err)
		return
	}
	bikes, err := h.bookings.AvailableBikes(p, q.Get("excludeReservation"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, bikes)
}

func decodeReservationInput(w http.ResponseWriter, r *http.Request) (reservationInput, domain.Period, error) {
	var in reservationInput
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		return in, domain.Period{}, fmt.Errorf("%w: request body is not valid JSON", domain.ErrInvalid)
	}
	p, err := in.period()
	return in, p, err
}

func parsePeriod(start, end string) (domain.Period, error) {
	if start == "" || end == "" {
		return domain.Period{}, fmt.Errorf("%w: start and end are required", domain.ErrInvalid)
	}
	s, err := time.Parse(time.RFC3339, start)
	if err != nil {
		return domain.Period{}, fmt.Errorf("%w: start must be an RFC 3339 time", domain.ErrInvalid)
	}
	e, err := time.Parse(time.RFC3339, end)
	if err != nil {
		return domain.Period{}, fmt.Errorf("%w: end must be an RFC 3339 time", domain.ErrInvalid)
	}
	return domain.Period{Start: s, End: e}, nil
}
