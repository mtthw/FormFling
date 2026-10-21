package handlers

import (
	"encoding/json"
	"net/http"

	"formfling/internal/models"
)

type HealthHandler struct {
	version string
}

func NewHealthHandler(version string) *HealthHandler {
	return &HealthHandler{version: version}
}

func (h *HealthHandler) Handle(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	response := models.Response{Status: "ok", Version: h.version}
	json.NewEncoder(w).Encode(response)
}
