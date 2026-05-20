package admin

import "time"

// CreateKeyRequest is the request body for creating a new API key.
type CreateKeyRequest struct {
	Provider       string            `json:"provider"` // "openai", "anthropic", or "gemini"
	ActualKey      string            `json:"actual_key"`
	Description    string            `json:"description,omitempty"`
	DailyCostLimit int64             `json:"daily_cost_limit,omitempty"` // in cents
	ExpiresAt      *time.Time        `json:"expires_at,omitempty"`
	Tags           map[string]string `json:"tags,omitempty"`
}

// UpdateKeyRequest is the request body for updating an existing API key.
type UpdateKeyRequest struct {
	Description    *string           `json:"description,omitempty"`
	DailyCostLimit *int64            `json:"daily_cost_limit,omitempty"`
	ExpiresAt      *time.Time        `json:"expires_at,omitempty"`
	Enabled        *bool             `json:"enabled,omitempty"`
	Tags           map[string]string `json:"tags,omitempty"`
}

// KeyResponse is the JSON response for a single API key.
type KeyResponse struct {
	PK             string            `json:"pk"`
	Provider       string            `json:"provider"`
	ActualKey      string            `json:"actual_key"` // always returned in admin context
	DailyCostLimit int64             `json:"daily_cost_limit"`
	Description    string            `json:"description"`
	CreatedAt      time.Time         `json:"created_at"`
	UpdatedAt      time.Time         `json:"updated_at"`
	ExpiresAt      *time.Time        `json:"expires_at,omitempty"`
	Enabled        bool              `json:"enabled"`
	Tags           map[string]string `json:"tags,omitempty"`
}

// ListKeysResponse is the JSON response for listing keys.
type ListKeysResponse struct {
	Keys  []KeyResponse `json:"keys"`
	Total int           `json:"total"`
}

// ErrorResponse is a generic error response.
type ErrorResponse struct {
	Error  string `json:"error"`
	Status int    `json:"status"`
}

// SuccessResponse is a generic success response for create/update/delete operations.
type SuccessResponse struct {
	Message string      `json:"message"`
	Key     *KeyResponse `json:"key,omitempty"` // included for create/update
	Data    interface{} `json:"data,omitempty"` // included for bulk operations
}

// BulkUpdateRequest is the request body for bulk updating keys by tag.
type BulkUpdateRequest struct {
	TagKey       string `json:"tag_key"`
	TagValue     string `json:"tag_value"`
	NewActualKey string `json:"new_actual_key"`
}

// BulkDeleteRequest is the request body for bulk deleting keys by tag.
type BulkDeleteRequest struct {
	TagKey   string `json:"tag_key"`
	TagValue string `json:"tag_value"`
}
