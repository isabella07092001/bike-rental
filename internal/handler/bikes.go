package handler

import (
	"encoding/json"
	"fmt"
	"net/http"

	"bike-rental/internal/domain"
	"bike-rental/internal/store"
)

// maxBodyBytes limits the size of JSON request bodies.
const maxBodyBytes = 1 << 20

// BikeHandler serves the /api/bikes and /api/categories endpoints.
type BikeHandler struct {
	store store.BikeStore
	// hasOpenReservations reports whether a bike has upcoming or active bookings;
	// such bikes cannot be retired.
	hasOpenReservations func(bikeID string) bool
}

// NewBikeHandler returns a handler backed by s. hasOpenReservations may be nil.
func NewBikeHandler(s store.BikeStore, hasOpenReservations func(bikeID string) bool) *BikeHandler {
	if hasOpenReservations == nil {
		hasOpenReservations = func(string) bool { return false }
	}
	return &BikeHandler{store: s, hasOpenReservations: hasOpenReservations}
}

// checkRetire rejects retiring a bike that still has upcoming or active reservations.
func (h *BikeHandler) checkRetire(old, updated domain.Bike) error {
	if updated.IsRetired() && !old.IsRetired() && h.hasOpenReservations(old.ID) {
		return fmt.Errorf("%w: %s has upcoming or active reservations and cannot be retired", domain.ErrConflict, old.Name)
	}
	return nil
}

// Register adds the bike routes to mux.
func (h *BikeHandler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/categories", h.listCategories)
	mux.HandleFunc("GET /api/bikes", h.list)
	mux.HandleFunc("POST /api/bikes", h.create)
	mux.HandleFunc("GET /api/bikes/{id}", h.get)
	mux.HandleFunc("PUT /api/bikes/{id}", h.update)
	mux.HandleFunc("POST /api/bikes/{id}/retire", h.retire)
}

// bikeInput is the request body for creating and updating a bike.
// The ID always comes from the URL or the store, never from the body.
type bikeInput struct {
	Name        string          `json:"name"`
	Category    domain.Category `json:"category"`
	Status      domain.Status   `json:"status"`
	Description string          `json:"description"`
}

func (in bikeInput) toBike(id string) domain.Bike {
	return domain.Bike{
		ID:          id,
		Name:        in.Name,
		Category:    in.Category,
		Status:      in.Status,
		Description: in.Description,
	}
}

type categoryResponse struct {
	Value            domain.Category `json:"value"`
	Label            string          `json:"label"`
	HourlyPriceCents int64           `json:"hourlyPriceCents"`
}

func (h *BikeHandler) listCategories(w http.ResponseWriter, _ *http.Request) {
	all := domain.AllCategories()
	resp := make([]categoryResponse, 0, len(all))
	for _, c := range all {
		resp = append(resp, categoryResponse{Value: c, Label: c.Label(), HourlyPriceCents: c.HourlyPrice()})
	}
	writeJSON(w, http.StatusOK, resp)
}

// list returns non-retired bikes by default. ?status= and ?category= filter the result;
// ?status=retired is the only way to see retired bikes in the list. ?q= searches name
// and description.
func (h *BikeHandler) list(w http.ResponseWriter, r *http.Request) {
	q := searchQuery(r)
	category := domain.Category(r.URL.Query().Get("category"))
	if category != "" && !category.Valid() {
		writeError(w, fmt.Errorf("%w: unknown category %q", domain.ErrInvalid, category))
		return
	}
	status := domain.Status(r.URL.Query().Get("status"))
	if status != "" && !status.Valid() {
		writeError(w, fmt.Errorf("%w: unknown status %q", domain.ErrInvalid, status))
		return
	}

	result := []domain.Bike{}
	for _, b := range h.store.List() {
		if category != "" && b.Category != category {
			continue
		}
		if status != "" && b.Status != status {
			continue
		}
		if status == "" && b.IsRetired() {
			continue
		}
		if !matchesAny(q, b.Name, b.Description) {
			continue
		}
		result = append(result, b)
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *BikeHandler) get(w http.ResponseWriter, r *http.Request) {
	b, err := h.store.Get(r.PathValue("id"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, b)
}

func (h *BikeHandler) create(w http.ResponseWriter, r *http.Request) {
	in, err := decodeBikeInput(w, r)
	if err != nil {
		writeError(w, err)
		return
	}
	b := in.toBike("")
	b.Normalize()
	if err := b.Validate(); err != nil {
		writeError(w, err)
		return
	}
	created, err := h.store.Create(b)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, created)
}

func (h *BikeHandler) update(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	old, err := h.store.Get(id)
	if err != nil {
		writeError(w, err)
		return
	}
	in, err := decodeBikeInput(w, r)
	if err != nil {
		writeError(w, err)
		return
	}
	b := in.toBike(id)
	b.Normalize()
	if err := h.checkRetire(old, b); err != nil {
		writeError(w, err)
		return
	}
	if err := b.Validate(); err != nil {
		writeError(w, err)
		return
	}
	updated, err := h.store.Update(b)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, updated)
}

func (h *BikeHandler) retire(w http.ResponseWriter, r *http.Request) {
	b, err := h.store.Get(r.PathValue("id"))
	if err != nil {
		writeError(w, err)
		return
	}
	old := b
	b.Retire()
	if err := h.checkRetire(old, b); err != nil {
		writeError(w, err)
		return
	}
	updated, err := h.store.Update(b)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, updated)
}

func decodeBikeInput(w http.ResponseWriter, r *http.Request) (bikeInput, error) {
	var in bikeInput
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		return bikeInput{}, fmt.Errorf("%w: request body is not valid JSON", domain.ErrInvalid)
	}
	return in, nil
}
