package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/Hardonian/TokenGoblin/internal/billing"
	"github.com/Hardonian/TokenGoblin/internal/moat"
	"github.com/Hardonian/TokenGoblin/internal/storage"
	"github.com/google/uuid"
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
		TenantID string `json:"tenant_id"` // Kept for backwards compatibility but ignored
		Name     string `json:"name"`
	}
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, Envelope{
			OK:     false,
			Status: "error",
			Error:  issue("invalid_json", "Request body must include name."),
		})
		return
	}

	if req.Name == "" {
		writeJSON(w, http.StatusBadRequest, Envelope{
			OK:     false,
			Status: "error",
			Error:  issue("invalid_request", "name is required."),
		})
		return
	}

	// ALWAYS auto-generate the Tenant ID for security, ignoring any requested one.
	tenantID := uuid.NewString()

	tenant, apiKey, err := billing.RegisterTenant(r.Context(), h.Repo, tenantID, req.Name)
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

// HandleTenantLogin verifies an API key and returns tenant info.
func (h *BillingHandler) HandleTenantLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeMethodError(w)
		return
	}
	tenantID, ok := tenantFromRequest(w, r)
	if !ok {
		return
	}
	writeJSON(w, http.StatusOK, Envelope{
		OK:     true,
		Status: "success",
		Data: map[string]interface{}{
			"tenant_id": tenantID,
		},
	})
}

// HandleListAPIKeys lists all API keys for the tenant.
func (h *BillingHandler) HandleListAPIKeys(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeMethodError(w)
		return
	}
	tenantID, ok := tenantFromRequest(w, r)
	if !ok {
		return
	}
	keys, err := h.Repo.ListAPIKeys(r.Context(), tenantID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, Envelope{OK: false, Status: "error", Error: issue("db_error", err.Error())})
		return
	}
	type keyResponse struct {
		KeyID      string     `json:"key_id"`
		Name       string     `json:"name"`
		Role       string     `json:"role"`
		CreatedAt  time.Time  `json:"created_at"`
		LastUsedAt *time.Time `json:"last_used_at,omitempty"`
	}
	var res []keyResponse
	for _, k := range keys {
		res = append(res, keyResponse{
			KeyID:      k.KeyID,
			Name:       k.Name,
			Role:       k.Role,
			CreatedAt:  k.CreatedAt,
			LastUsedAt: k.LastUsedAt,
		})
	}
	writeJSON(w, http.StatusOK, Envelope{OK: true, Status: "success", Data: res})
}

// HandleGenerateAPIKey generates a new API key.
func (h *BillingHandler) HandleGenerateAPIKey(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeMethodError(w)
		return
	}
	tenantID, ok := tenantFromRequest(w, r)
	if !ok {
		return
	}
	var req struct {
		Name string `json:"name"`
	}
	decoder := json.NewDecoder(r.Body)
	_ = decoder.Decode(&req)
	if req.Name == "" {
		req.Name = "generated-key"
	}

	apiKey, compoundToken, err := moat.GenerateAPIKey(tenantID, req.Name)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, Envelope{OK: false, Status: "error", Error: issue("generation_failed", err.Error())})
		return
	}
	if err := h.Repo.SaveAPIKey(r.Context(), apiKey); err != nil {
		writeJSON(w, http.StatusInternalServerError, Envelope{OK: false, Status: "error", Error: issue("save_failed", err.Error())})
		return
	}

	writeJSON(w, http.StatusCreated, Envelope{
		OK:     true,
		Status: "success",
		Data: map[string]interface{}{
			"key_id":  apiKey.KeyID,
			"name":    apiKey.Name,
			"api_key": compoundToken,
		},
	})
}

// HandleRevokeAPIKey revokes an API key.
func (h *BillingHandler) HandleRevokeAPIKey(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		writeMethodError(w)
		return
	}
	tenantID, ok := tenantFromRequest(w, r)
	if !ok {
		return
	}
	keyID := r.URL.Query().Get("key_id")
	if keyID == "" {
		writeJSON(w, http.StatusBadRequest, Envelope{OK: false, Status: "error", Error: issue("invalid_request", "key_id required")})
		return
	}
	if err := h.Repo.RevokeAPIKey(r.Context(), tenantID, keyID); err != nil {
		writeJSON(w, http.StatusInternalServerError, Envelope{OK: false, Status: "error", Error: issue("revoke_failed", err.Error())})
		return
	}
	writeJSON(w, http.StatusOK, Envelope{OK: true, Status: "success"})
}
