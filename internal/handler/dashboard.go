package handler

import (
	"net/http"

	"bike-rental/internal/dashboard"
)

// DashboardHandler serves GET /api/dashboard.
type DashboardHandler struct {
	builder *dashboard.Builder
}

// NewDashboardHandler returns a handler using b to calculate the figures.
func NewDashboardHandler(b *dashboard.Builder) *DashboardHandler {
	return &DashboardHandler{builder: b}
}

// Register adds the dashboard route to mux.
func (h *DashboardHandler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/dashboard", h.get)
}

func (h *DashboardHandler) get(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, h.builder.Build())
}
