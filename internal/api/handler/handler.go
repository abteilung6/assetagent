package handler

import (
	"encoding/json"
	"net/http"

	"github.com/abteilung6/assetagent/internal/api/gen"
)

type Handler struct{}

func New() *Handler {
	return &Handler{}
}

func (h *Handler) GetHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, gen.HealthResponse{Status: "ok"})
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}
