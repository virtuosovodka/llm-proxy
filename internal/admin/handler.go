package admin

import (
	_ "embed"
	"context"
	"encoding/json"
	"net/http"

	"github.com/Instawork/llm-proxy/internal/apikeys"
	"github.com/Instawork/llm-proxy/internal/providers"
	"github.com/gorilla/mux"
)

//go:embed admin.html
var adminHTML []byte

// AdminKeyStore is the interface for accessing API keys.
// This is the same as providers.APIKeyStore plus additional methods from apikeys.Store.
type AdminKeyStore interface {
	providers.APIKeyStore
	CreateKey(ctx context.Context, provider, actualKey, description string, dailyCostLimit int64, tags map[string]string) (*apikeys.APIKey, error)
	GetKey(ctx context.Context, key string) (*apikeys.APIKey, error)
	UpdateKey(ctx context.Context, key string, updates map[string]interface{}) error
	DeleteKey(ctx context.Context, key string) error
	ListKeys(ctx context.Context, provider string) ([]*apikeys.APIKey, error)
}

// Handler owns the HTTP handlers for the admin API endpoints.
type Handler struct {
	store AdminKeyStore
	token string
}

// NewHandler constructs the admin handler.
// If token is non-empty, authentication is required.
func NewHandler(s AdminKeyStore, token string) *Handler {
	return &Handler{
		store: s,
		token: token,
	}
}

// RegisterRoutes wires the admin routes into the given mux router.
// Call this before provider catch-all PathPrefix routes are registered.
func (h *Handler) RegisterRoutes(router *mux.Router) {
	// Serve HTML UI (no auth required - JS will fetch API with token)
	router.HandleFunc("/admin", h.ServeUI).Methods("GET", "HEAD")
	router.HandleFunc("/admin/", h.ServeUI).Methods("GET", "HEAD")

	// API endpoints (all require auth via middleware)
	router.HandleFunc("/admin/api/keys", h.AuthMiddleware(h.CreateKey)).Methods("POST")
	router.HandleFunc("/admin/api/keys", h.AuthMiddleware(h.ListKeys)).Methods("GET")
	router.HandleFunc("/admin/api/keys/{id}", h.AuthMiddleware(h.GetKey)).Methods("GET")
	router.HandleFunc("/admin/api/keys/{id}", h.AuthMiddleware(h.UpdateKey)).Methods("PUT")
	router.HandleFunc("/admin/api/keys/{id}", h.AuthMiddleware(h.DeleteKey)).Methods("DELETE")
}

// ServeUI serves the embedded admin.html page (no auth required).
func (h *Handler) ServeUI(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	w.Write(adminHTML)
}

// CreateKey handles POST /admin/api/keys
func (h *Handler) CreateKey(w http.ResponseWriter, r *http.Request) {
	var req CreateKeyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request: "+err.Error())
		return
	}

	// Validate required fields
	if req.Provider == "" {
		respondError(w, http.StatusBadRequest, "provider is required")
		return
	}
	if req.ActualKey == "" {
		respondError(w, http.StatusBadRequest, "actual_key is required")
		return
	}

	// Validate provider
	validProviders := map[string]bool{"openai": true, "anthropic": true, "gemini": true}
	if !validProviders[req.Provider] {
		respondError(w, http.StatusBadRequest, "invalid provider (must be openai, anthropic, or gemini)")
		return
	}

	// Default daily cost limit to $100 if not specified
	if req.DailyCostLimit == 0 {
		req.DailyCostLimit = 10000 // cents
	}

	// Create key in store
	apiKey, err := h.store.CreateKey(
		context.Background(),
		req.Provider,
		req.ActualKey,
		req.Description,
		req.DailyCostLimit,
		req.Tags,
	)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to create key: "+err.Error())
		return
	}

	// Set expiration if provided
	if req.ExpiresAt != nil {
		if err := h.store.UpdateKey(context.Background(), apiKey.PK, map[string]interface{}{
			"expires_at": req.ExpiresAt,
		}); err != nil {
			respondError(w, http.StatusInternalServerError, "failed to set expiration: "+err.Error())
			return
		}
	}

	// Return created key
	keyResp := apiKeyToResponse(apiKey)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(keyResp)
}

// ListKeys handles GET /admin/api/keys
func (h *Handler) ListKeys(w http.ResponseWriter, r *http.Request) {
	provider := r.URL.Query().Get("provider")

	// List keys (filtered by provider if specified)
	keys, err := h.store.ListKeys(context.Background(), provider)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to list keys: "+err.Error())
		return
	}

	if keys == nil {
		keys = make([]*apikeys.APIKey, 0)
	}

	// Convert to response format
	responses := make([]KeyResponse, len(keys))
	for i, key := range keys {
		responses[i] = apiKeyToResponse(key)
	}

	respondJSON(w, http.StatusOK, ListKeysResponse{
		Keys:  responses,
		Total: len(responses),
	})
}

// GetKey handles GET /admin/api/keys/:id
func (h *Handler) GetKey(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	if id == "" {
		respondError(w, http.StatusBadRequest, "key id is required")
		return
	}

	key, err := h.store.GetKey(context.Background(), id)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to get key: "+err.Error())
		return
	}

	if key == nil {
		respondError(w, http.StatusNotFound, "key not found")
		return
	}

	keyResp := apiKeyToResponse(key)
	respondJSON(w, http.StatusOK, keyResp)
}

// UpdateKey handles PUT /admin/api/keys/:id
func (h *Handler) UpdateKey(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	if id == "" {
		respondError(w, http.StatusBadRequest, "key id is required")
		return
	}

	var req UpdateKeyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request: "+err.Error())
		return
	}

	// Build updates map
	updates := make(map[string]interface{})
	if req.Description != nil {
		updates["description"] = *req.Description
	}
	if req.DailyCostLimit != nil {
		updates["daily_cost_limit"] = *req.DailyCostLimit
	}
	if req.ExpiresAt != nil {
		updates["expires_at"] = *req.ExpiresAt
	}
	if req.Enabled != nil {
		updates["enabled"] = *req.Enabled
	}
	if req.Tags != nil {
		updates["tags"] = req.Tags
	}

	if len(updates) == 0 {
		respondError(w, http.StatusBadRequest, "no fields to update")
		return
	}

	// Update key
	if err := h.store.UpdateKey(context.Background(), id, updates); err != nil {
		respondError(w, http.StatusInternalServerError, "failed to update key: "+err.Error())
		return
	}

	// Fetch updated key
	key, err := h.store.GetKey(context.Background(), id)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to fetch updated key: "+err.Error())
		return
	}

	keyResp := apiKeyToResponse(key)
	respondSuccess(w, "key updated", &keyResp)
}

// DeleteKey handles DELETE /admin/api/keys/:id
func (h *Handler) DeleteKey(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	if id == "" {
		respondError(w, http.StatusBadRequest, "key id is required")
		return
	}

	if err := h.store.DeleteKey(context.Background(), id); err != nil {
		respondError(w, http.StatusInternalServerError, "failed to delete key: "+err.Error())
		return
	}

	respondSuccess(w, "key deleted", nil)
}

// apiKeyToResponse converts an apikeys.APIKey to a KeyResponse.
func apiKeyToResponse(key *apikeys.APIKey) KeyResponse {
	resp := KeyResponse{
		PK:             key.PK,
		Provider:       key.Provider,
		ActualKey:      key.ActualKey,
		DailyCostLimit: key.DailyCostLimit,
		Description:    key.Description,
		CreatedAt:      key.CreatedAt,
		UpdatedAt:      key.UpdatedAt,
		ExpiresAt:      key.ExpiresAt,
		Enabled:        key.Enabled,
		Tags:           key.Tags,
	}
	return resp
}
