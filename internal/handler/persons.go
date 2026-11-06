package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"mime"
	"net/http"
	"strconv"
	"strings"
	"time"

	"bike-rental/internal/domain"
	"bike-rental/internal/store"
)

// maxUploadBody is the largest accepted multipart request: the ID copy plus room for the
// text fields and multipart framing. ParseMultipartForm keeps everything up to this size
// in memory, so ID copies are never written to temporary files.
const maxUploadBody = domain.MaxIDDocumentSize + 64<<10

// idDocumentField is the multipart field name of the ID copy.
const idDocumentField = "idDocument"

// PersonHandler serves the /api/persons endpoints.
type PersonHandler struct {
	store store.PersonStore
	now   func() time.Time
	// hasOpenReservations reports whether a person has upcoming or active bookings;
	// such persons cannot be archived.
	hasOpenReservations func(personID string) bool
}

// NewPersonHandler returns a handler backed by s. now and hasOpenReservations may be nil.
func NewPersonHandler(s store.PersonStore, now func() time.Time, hasOpenReservations func(personID string) bool) *PersonHandler {
	if now == nil {
		now = time.Now
	}
	if hasOpenReservations == nil {
		hasOpenReservations = func(string) bool { return false }
	}
	return &PersonHandler{store: s, now: now, hasOpenReservations: hasOpenReservations}
}

// Register adds the person routes to mux.
func (h *PersonHandler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/persons", h.list)
	mux.HandleFunc("POST /api/persons", h.create)
	mux.HandleFunc("GET /api/persons/{id}", h.get)
	mux.HandleFunc("PUT /api/persons/{id}", h.update)
	mux.HandleFunc("GET /api/persons/{id}/id-document", h.getDocument)
	mux.HandleFunc("PUT /api/persons/{id}/id-document", h.replaceDocument)
	mux.HandleFunc("POST /api/persons/{id}/archive", h.archive)
	mux.HandleFunc("POST /api/persons/{id}/reactivate", h.reactivate)
}

// personResponse is the JSON shape of a person: its fields plus ID copy metadata (never the bytes).
type personResponse struct {
	domain.Person
	IDDocument domain.IDDocumentMeta `json:"idDocument"`
}

func toResponse(e store.PersonEntry) personResponse {
	return personResponse{Person: e.Person, IDDocument: e.Document}
}

// personInput holds the editable text fields of a person.
type personInput struct {
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`
	IDNumber  string `json:"idNumber"`
	Phone     string `json:"phone"`
}

func (in personInput) apply(p *domain.Person) {
	p.FirstName = in.FirstName
	p.LastName = in.LastName
	p.IDNumber = in.IDNumber
	p.Phone = in.Phone
}

// list returns active persons, or archived ones with ?archived=true.
// ?q= filters by name, ID number or phone (case-insensitive substring).
func (h *PersonHandler) list(w http.ResponseWriter, r *http.Request) {
	archived := false
	if v := r.URL.Query().Get("archived"); v != "" {
		var err error
		if archived, err = strconv.ParseBool(v); err != nil {
			writeError(w, fmt.Errorf("%w: archived must be true or false", domain.ErrInvalid))
			return
		}
	}
	q := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("q")))

	result := []personResponse{}
	for _, e := range h.store.List() {
		if e.Person.Archived != archived || !matches(e.Person, q) {
			continue
		}
		result = append(result, toResponse(e))
	}
	writeJSON(w, http.StatusOK, result)
}

func matches(p domain.Person, q string) bool {
	if q == "" {
		return true
	}
	for _, field := range []string{p.FullName(), p.LastName + " " + p.FirstName, p.IDNumber, p.Phone} {
		if strings.Contains(strings.ToLower(field), q) {
			return true
		}
	}
	return false
}

func (h *PersonHandler) get(w http.ResponseWriter, r *http.Request) {
	e, err := h.store.Get(r.PathValue("id"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, toResponse(e))
}

// create expects multipart/form-data with the text fields and the ID copy in one request,
// so a person can never exist without an ID copy.
func (h *PersonHandler) create(w http.ResponseWriter, r *http.Request) {
	if err := parseUpload(w, r); err != nil {
		writeError(w, err)
		return
	}
	defer r.MultipartForm.RemoveAll()

	var p domain.Person
	personInput{
		FirstName: r.FormValue("firstName"),
		LastName:  r.FormValue("lastName"),
		IDNumber:  r.FormValue("idNumber"),
		Phone:     r.FormValue("phone"),
	}.apply(&p)
	p.Normalize()
	if err := p.Validate(); err != nil {
		writeError(w, err)
		return
	}

	doc, err := h.readDocument(r)
	if err != nil {
		writeError(w, err)
		return
	}

	created, err := h.store.Create(p, doc)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, toResponse(created))
}

// update changes the text fields only (JSON). The archived flag and the ID copy are kept.
func (h *PersonHandler) update(w http.ResponseWriter, r *http.Request) {
	e, err := h.store.Get(r.PathValue("id"))
	if err != nil {
		writeError(w, err)
		return
	}
	var in personInput
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, fmt.Errorf("%w: request body is not valid JSON", domain.ErrInvalid))
		return
	}
	p := e.Person
	in.apply(&p)
	p.Normalize()
	if err := p.Validate(); err != nil {
		writeError(w, err)
		return
	}
	updated, err := h.store.Update(p)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, toResponse(updated))
}

func (h *PersonHandler) replaceDocument(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if _, err := h.store.Get(id); err != nil {
		writeError(w, err)
		return
	}
	if err := parseUpload(w, r); err != nil {
		writeError(w, err)
		return
	}
	defer r.MultipartForm.RemoveAll()

	doc, err := h.readDocument(r)
	if err != nil {
		writeError(w, err)
		return
	}
	updated, err := h.store.SetIDDocument(id, doc)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, toResponse(updated))
}

// getDocument sends the ID copy so the browser can display it.
func (h *PersonHandler) getDocument(w http.ResponseWriter, r *http.Request) {
	doc, err := h.store.GetIDDocument(r.PathValue("id"))
	if err != nil {
		writeError(w, err)
		return
	}
	header := w.Header()
	header.Set("Content-Type", doc.ContentType)
	header.Set("Content-Length", strconv.Itoa(len(doc.Data)))
	header.Set("Content-Disposition", mime.FormatMediaType("inline", map[string]string{"filename": doc.FileName}))
	header.Set("X-Content-Type-Options", "nosniff")
	header.Set("Cache-Control", "no-store") // sensitive: do not keep in browser caches
	if _, err := w.Write(doc.Data); err != nil {
		log.Printf("write ID copy response: %v", err) // never log document content
	}
}

func (h *PersonHandler) archive(w http.ResponseWriter, r *http.Request) {
	h.setArchived(w, r, true)
}

func (h *PersonHandler) reactivate(w http.ResponseWriter, r *http.Request) {
	h.setArchived(w, r, false)
}

func (h *PersonHandler) setArchived(w http.ResponseWriter, r *http.Request, archived bool) {
	e, err := h.store.Get(r.PathValue("id"))
	if err != nil {
		writeError(w, err)
		return
	}
	if archived && !e.Person.Archived && h.hasOpenReservations(e.Person.ID) {
		writeError(w, fmt.Errorf("%w: %s has upcoming or active reservations and cannot be archived",
			domain.ErrConflict, e.Person.FullName()))
		return
	}
	if archived {
		e.Person.Archive()
	} else {
		e.Person.Reactivate()
	}
	updated, err := h.store.Update(e.Person)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, toResponse(updated))
}

// parseUpload limits the body size and parses a multipart form entirely in memory.
func parseUpload(w http.ResponseWriter, r *http.Request) error {
	r.Body = http.MaxBytesReader(w, r.Body, maxUploadBody)
	if err := r.ParseMultipartForm(maxUploadBody); err != nil {
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			return fmt.Errorf("%w: upload exceeds %d bytes", domain.ErrTooLarge, maxUploadBody)
		}
		return fmt.Errorf("%w: expected a multipart/form-data upload", domain.ErrInvalid)
	}
	return nil
}

// readDocument reads the ID copy from the parsed multipart form and validates it.
func (h *PersonHandler) readDocument(r *http.Request) (domain.IDDocument, error) {
	file, header, err := r.FormFile(idDocumentField)
	if err != nil {
		return domain.IDDocument{}, fmt.Errorf("%w: ID copy is required (field %q)", domain.ErrInvalid, idDocumentField)
	}
	defer file.Close()

	data, err := io.ReadAll(io.LimitReader(file, domain.MaxIDDocumentSize+1))
	if err != nil {
		return domain.IDDocument{}, fmt.Errorf("read ID copy: %w", err)
	}
	return domain.NewIDDocument(header.Filename, data, h.now())
}
