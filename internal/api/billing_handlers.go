package api

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/Hardonian/TokenGoblin/internal/billing"
	"github.com/Hardonian/TokenGoblin/internal/storage"
)

// BillingHandler groups billing-related HTTP handlers.
type BillingHandler struct {
	Repo storage.Repository
}

// NewBillingHandler creates a new BillingHandler.
func NewBillingHandler(repo storage.Repository) *BillingHandler {
	return &BillingHandler{Repo: repo}
}

// HandleCreateCheckout handles POST /api/billing/checkout
func (h *BillingHandler) HandleCreateCheckout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeMethodError(w)
		return
	}

	tenantID, ok := tenantFromRequest(w, r)
	if !ok {
		return
	}

	var req struct {
		SuccessURL string `json:"success_url"`
		CancelURL  string `json:"cancel_url"`
		PriceID    string `json:"price_id"`
	}
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, Envelope{
			OK:     false,
			Status: "error",
			Error:  issue("invalid_json", "Request body must include success_url, cancel_url, and price_id."),
		})
		return
	}

	if req.SuccessURL == "" || req.CancelURL == "" || req.PriceID == "" {
		writeJSON(w, http.StatusBadRequest, Envelope{
			OK:     false,
			Status: "error",
			Error:  issue("invalid_request", "success_url, cancel_url, and price_id are required."),
		})
		return
	}

	checkoutURL, sessionID, err := billing.CreateCheckoutSession(r.Context(), h.Repo, tenantID, req.SuccessURL, req.CancelURL, req.PriceID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, Envelope{
			OK:     false,
			Status: "error",
			Error:  issue("checkout_failed", err.Error()),
		})
		return
	}

	writeJSON(w, http.StatusOK, Envelope{
		OK:     true,
		Status: "success",
		Data: map[string]string{
			"checkout_url": checkoutURL,
			"session_id":   sessionID,
		},
	})
}

// HandleCreatePortal handles POST /api/billing/portal
func (h *BillingHandler) HandleCreatePortal(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeMethodError(w)
		return
	}

	tenantID, ok := tenantFromRequest(w, r)
	if !ok {
		return
	}

	var req struct {
		ReturnURL string `json:"return_url"`
	}
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, Envelope{
			OK:     false,
			Status: "error",
			Error:  issue("invalid_json", "Request body must include return_url."),
		})
		return
	}

	if req.ReturnURL == "" {
		writeJSON(w, http.StatusBadRequest, Envelope{
			OK:     false,
			Status: "error",
			Error:  issue("invalid_request", "return_url is required."),
		})
		return
	}

	portalURL, err := billing.CreatePortalSession(r.Context(), h.Repo, tenantID, req.ReturnURL)
	if err != nil {
		code := "portal_failed"
		status := http.StatusInternalServerError
		if err.Error() == "tenant has no Stripe customer ID; cannot create portal session" {
			status = http.StatusBadRequest
		}
		writeJSON(w, status, Envelope{
			OK:     false,
			Status: "error",
			Error:  issue(code, err.Error()),
		})
		return
	}

	writeJSON(w, http.StatusOK, Envelope{
		OK:     true,
		Status: "success",
		Data: map[string]string{
			"portal_url": portalURL,
		},
	})
}

// HandleBillingStatus handles GET /api/billing/status
func (h *BillingHandler) HandleBillingStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeMethodError(w)
		return
	}

	tenantID, ok := tenantFromRequest(w, r)
	if !ok {
		return
	}

	status, err := billing.GetBillingStatus(r.Context(), h.Repo, tenantID)
	if err != nil {
		if errors.Is(err, storage.ErrUnavailable) {
			writeJSON(w, http.StatusOK, Envelope{
				OK:     true,
				Status: "degraded",
				Error:  issue("database_unavailable", "Storage is unavailable; returning empty billing status."),
			})
			return
		}
		writeServiceError(w, err, false)
		return
	}

	writeJSON(w, http.StatusOK, Envelope{
		OK:     true,
		Status: "success",
		Data:   status,
	})
}

// HandleRegisterTenant handles POST /api/tenant/register (public, no auth)
func (h *BillingHandler) HandleRegisterTenant(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeMethodError(w)
		return
	}

	var req struct {
		TenantID string `json:"tenant_id"`
		Name     string `json:"name"`
	}
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, Envelope{
			OK:     false,
			Status: "error",
			Error:  issue("invalid_json", "Request body must include tenant_id and name."),
		})
		return
	}

	if req.TenantID == "" || req.Name == "" {
		writeJSON(w, http.StatusBadRequest, Envelope{
			OK:     false,
			Status: "error",
			Error:  issue("invalid_request", "tenant_id and name are required."),
		})
		return
	}

	tenant, apiKey, err := billing.RegisterTenant(r.Context(), h.Repo, req.TenantID, req.Name)
	if err != nil {
		if err.Error() == "tenant already exists: "+req.TenantID {
			writeJSON(w, http.StatusConflict, Envelope{
				OK:     false,
				Status: "error",
				Error:  issue("tenant_exists", "A tenant with this ID already exists."),
			})
			return
		}
		writeJSON(w, http.StatusInternalServerError, Envelope{
			OK:     false,
			Status: "error",
			Error:  issue("registration_failed", err.Error()),
		})
		return
	}

	writeJSON(w, http.StatusCreated, Envelope{
		OK:     true,
		Status: "success",
		Data: map[string]interface{}{
			"tenant_id":  tenant.TenantID,
			"name":       tenant.Name,
			"tier":       tenant.Tier,
			"api_key":    apiKey,
			"created_at": tenant.CreatedAt,
		},
	})
}
