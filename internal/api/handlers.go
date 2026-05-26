package api

import (
	"encoding/json"
	"net/http"

	"github.com/Hardonian/TokenGoblin/internal/domain"
	"github.com/Hardonian/TokenGoblin/internal/ingestion"
)

type IngestionHandler struct {
	Service ingestion.Service
}

func NewIngestionHandler(service ingestion.Service) *IngestionHandler {
	return &IngestionHandler{Service: service}
}

func (h *IngestionHandler) HandleTokenEvent(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var event domain.TokenEvent
	if err := json.NewDecoder(r.Body).Decode(&event); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	if err := h.Service.ProcessTokenEvent(r.Context(), event); err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusAccepted)
}

func (h *IngestionHandler) HandleTaskCompletion(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var completion domain.TaskCompletion
	if err := json.NewDecoder(r.Body).Decode(&completion); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	if err := h.Service.ProcessTaskCompletion(r.Context(), completion); err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusAccepted)
}
