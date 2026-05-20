package admin

import (
	"encoding/json"
	"net/http"
	"strings"
)

// AuthMiddleware checks the admin token if one is configured.
// Returns a wrapped http.HandlerFunc that checks auth before calling the inner handler.
func (h *Handler) AuthMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// If no token configured, skip auth check
		if h.token == "" {
			next(w, r)
			return
		}

		// Check token from query parameter or Authorization header
		token := r.URL.Query().Get("token")
		if token == "" {
			// Try Authorization: Bearer <token> header
			authHeader := r.Header.Get("Authorization")
			if authHeader != "" && len(authHeader) > 7 {
				if authHeader[:7] == "Bearer " {
					token = authHeader[7:]
				}
			}
		}

		// Reject if token is missing or incorrect
		if token != h.token {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(ErrorResponse{
				Error:  "unauthorized",
				Status: http.StatusUnauthorized,
			})
			return
		}

		// Token is valid, proceed
		next(w, r)
	}
}

// parseToken extracts token from query param or Authorization header.
func parseToken(r *http.Request) string {
	token := r.URL.Query().Get("token")
	if token == "" {
		authHeader := r.Header.Get("Authorization")
		if authHeader != "" && len(authHeader) > 7 && authHeader[:7] == "Bearer " {
			token = authHeader[7:]
		}
	}
	return token
}

// respondError writes an error response with the given status code.
func respondError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(ErrorResponse{
		Error:  message,
		Status: status,
	})
}

// respondSuccess writes a success response.
func respondSuccess(w http.ResponseWriter, message string, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	resp := SuccessResponse{
		Message: message,
	}
	// Handle different data types
	if data != nil {
		if key, ok := data.(*KeyResponse); ok {
			resp.Key = key
		} else {
			resp.Data = data
		}
	}
	json.NewEncoder(w).Encode(resp)
}

// respondJSON writes a JSON response with the given status code.
func respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

// getIDFromPath extracts the last path segment (key ID) from the URL.
// Example: /admin/api/keys/iw:abc123 → "iw:abc123"
func getIDFromPath(path string) string {
	parts := strings.Split(strings.TrimRight(path, "/"), "/")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return ""
}
