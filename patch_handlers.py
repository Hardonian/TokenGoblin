
# Update billing_handlers.go
with open('internal/api/billing_handlers.go', 'r') as f:
    content = f.read()

# Add imports
if '"time"' not in content:
    content = content.replace('"net/http"', '"net/http"\n\t"time"')
if '"github.com/Hardonian/TokenGoblin/internal/moat"' not in content:
    content = content.replace('"github.com/Hardonian/TokenGoblin/internal/billing"', '"github.com/Hardonian/TokenGoblin/internal/billing"\n\t"github.com/Hardonian/TokenGoblin/internal/moat"')

new_methods = """
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
"""

if "HandleTenantLogin" not in content:
    content += new_methods

with open('internal/api/billing_handlers.go', 'w') as f:
    f.write(content)

# Update router.go
with open('internal/api/router.go', 'r') as f:
    router_content = f.read()

new_routes = """
	// API Key Routes (requires auth)
	mux.Handle("/api/tenant/login", AuthMiddleware(repo, http.HandlerFunc(billingHandler.HandleTenantLogin)))
	mux.Handle("/api/tenant/keys", AuthMiddleware(repo, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			billingHandler.HandleListAPIKeys(w, r)
		case http.MethodPost:
			billingHandler.HandleGenerateAPIKey(w, r)
		case http.MethodDelete:
			billingHandler.HandleRevokeAPIKey(w, r)
		default:
			writeMethodError(w)
		}
	})))
"""
if "/api/tenant/login" not in router_content:
    router_content = router_content.replace('// Stripe webhooks', new_routes + '\n\t// Stripe webhooks')
    
with open('internal/api/router.go', 'w') as f:
    f.write(router_content)

print("success")
