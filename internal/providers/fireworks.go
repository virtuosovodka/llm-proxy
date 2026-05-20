package providers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"

	"github.com/gorilla/mux"
)

const (
	// Fireworks API base URL
	fireworksBaseURL = "https://api.fireworks.ai/inference/v1"
)

// FireworksProxy handles Fireworks API requests and implements the Provider interface
type FireworksProxy struct {
	proxy *httputil.ReverseProxy
}

// NewFireworksProxy creates a new Fireworks reverse proxy
func NewFireworksProxy() *FireworksProxy {
	// Parse the Fireworks API URL
	targetURL, err := url.Parse(fireworksBaseURL)
	if err != nil {
		log.Fatalf("Failed to parse Fireworks API URL: %v", err)
	}

	// Create the reverse proxy
	proxy := httputil.NewSingleHostReverseProxy(targetURL)

	// Create the Fireworks proxy instance
	fireworksProxy := &FireworksProxy{proxy: proxy}

	// Use the generic director function to handle common proxy logic
	originalDirector := proxy.Director
	proxy.Director = CreateGenericDirector(fireworksProxy, targetURL, originalDirector)

	// Customize the transport for optimal streaming performance
	proxy.Transport = newProxyTransport()

	// Add custom response modifier for streaming support
	proxy.ModifyResponse = func(resp *http.Response) error {
		// Handle streaming responses
		if fireworksProxy.isStreamingResponse(resp) {
			log.Printf("Detected streaming response from Fireworks")

			// Ensure proper headers for streaming
			resp.Header.Set("Cache-Control", "no-cache")
			resp.Header.Set("Connection", "keep-alive")
			resp.Header.Set("X-Accel-Buffering", "no")

			// Remove content-length header for streaming
			resp.Header.Del("Content-Length")
		}

		return nil
	}

	// Add error handler with streaming-specific error handling
	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		log.Printf("Fireworks proxy error: %v", err)

		if fireworksProxy.IsStreamingRequest(r) {
			if w.Header().Get("Content-Type") == "" {
				w.Header().Set("Content-Type", "text/event-stream")
				w.Header().Set("Cache-Control", "no-cache")
				w.WriteHeader(http.StatusBadGateway)
				fmt.Fprintf(w, "data: {\"error\": \"Proxy error: %v\"}\n\n", err)
				fmt.Fprintf(w, "data: [DONE]\n\n")
			} else {
				log.Printf("Cannot send error response, headers already sent")
			}
		} else {
			w.WriteHeader(http.StatusBadGateway)
			fmt.Fprintf(w, "Fireworks proxy error: %v", err)
		}
	}

	return fireworksProxy
}

// GetName returns the name of the provider
func (f *FireworksProxy) GetName() string {
	return "fireworks"
}

// IsStreamingRequest checks if the request is likely to be a streaming request for Fireworks
func (f *FireworksProxy) IsStreamingRequest(req *http.Request) bool {
	// Check for streaming in the Accept header first (fast check)
	if strings.Contains(req.Header.Get("Accept"), "text/event-stream") {
		return true
	}

	// Only check Fireworks-specific endpoints
	if !strings.HasPrefix(req.URL.Path, "/fireworks/") {
		return false
	}

	// For completion endpoints, check the request body for stream: true
	if req.Method == "POST" && strings.Contains(req.URL.Path, "/chat/completions") {
		return f.checkStreamingInBody(req)
	}

	return false
}

// checkStreamingInBody reads the request body to check for "stream": true
func (f *FireworksProxy) checkStreamingInBody(req *http.Request) bool {
	if req.Body == nil {
		return false
	}

	var bodyBytes []byte
	var err error

	if req.GetBody != nil {
		bodyReader, err := req.GetBody()
		if err != nil {
			log.Printf("Error getting cached request body for streaming check: %v", err)
			return false
		}
		defer bodyReader.Close()
		bodyBytes, err = io.ReadAll(bodyReader)
		if err != nil {
			log.Printf("Error reading cached request body for streaming check: %v", err)
			return false
		}
	} else {
		bodyBytes, err = io.ReadAll(req.Body)
		if err != nil {
			log.Printf("Error reading request body for streaming check: %v", err)
			return false
		}

		req.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
		req.GetBody = func() (io.ReadCloser, error) {
			return io.NopCloser(bytes.NewBuffer(bodyBytes)), nil
		}
	}

	var requestData map[string]interface{}
	if err := json.Unmarshal(bodyBytes, &requestData); err != nil {
		log.Printf("Error parsing request body JSON for streaming check: %v", err)
		return false
	}

	if streamValue, exists := requestData["stream"]; exists {
		if streamBool, ok := streamValue.(bool); ok {
			return streamBool
		}
	}

	return false
}

// isStreamingResponse checks if the response is a streaming response
func (f *FireworksProxy) isStreamingResponse(resp *http.Response) bool {
	contentType := resp.Header.Get("Content-Type")
	return strings.Contains(contentType, "text/event-stream")
}

// Proxy returns the underlying HTTP handler
func (f *FireworksProxy) Proxy() http.Handler {
	return f.proxy
}

// GetHealthStatus returns health status information
func (f *FireworksProxy) GetHealthStatus() map[string]interface{} {
	return map[string]interface{}{
		"status": "healthy",
		"provider": "fireworks",
	}
}

// UserIDFromRequest extracts user ID from the request
func (f *FireworksProxy) UserIDFromRequest(req *http.Request) string {
	// Fireworks uses OpenAI-compatible format, so extract user ID from request body if present
	if req.Body == nil || req.Method != "POST" {
		return ""
	}

	if !strings.HasPrefix(req.URL.Path, "/fireworks/") {
		return ""
	}

	bodyBytes, err := f.readRequestBodyForUserID(req)
	if err != nil {
		return ""
	}

	if len(bodyBytes) == 0 {
		return ""
	}

	var requestData map[string]interface{}
	if err := json.Unmarshal(bodyBytes, &requestData); err != nil {
		return ""
	}

	if userID, exists := requestData["user"]; exists {
		return fmt.Sprintf("%v", userID)
	}

	return ""
}

// readRequestBodyForUserID reads the request body while preserving it for later use
func (f *FireworksProxy) readRequestBodyForUserID(req *http.Request) ([]byte, error) {
	if req.GetBody != nil {
		bodyReader, err := req.GetBody()
		if err != nil {
			return nil, err
		}
		defer bodyReader.Close()
		return io.ReadAll(bodyReader)
	}

	bodyBytes, err := io.ReadAll(req.Body)
	if err != nil {
		return nil, err
	}

	req.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
	req.GetBody = func() (io.ReadCloser, error) {
		return io.NopCloser(bytes.NewBuffer(bodyBytes)), nil
	}

	return bodyBytes, nil
}

// RegisterExtraRoutes registers any provider-specific routes
func (f *FireworksProxy) RegisterExtraRoutes(router *mux.Router) {
	// Fireworks doesn't need extra routes
}

// ValidateAPIKey validates the API key for Fireworks
func (f *FireworksProxy) ValidateAPIKey(req *http.Request, keyStore APIKeyStore) error {
	// Get the API key from the Authorization header
	authHeader := req.Header.Get("Authorization")
	if authHeader == "" {
		return fmt.Errorf("missing Authorization header")
	}

	// Extract the key from "Bearer <key>" format
	if !strings.HasPrefix(authHeader, "Bearer ") {
		return fmt.Errorf("invalid Authorization header format")
	}

	key := strings.TrimPrefix(authHeader, "Bearer ")

	// If it's a proxy key (starts with "my:"), validate it
	if strings.HasPrefix(key, "my:") {
		if keyStore == nil {
			return fmt.Errorf("key store not available")
		}
		_, _, err := keyStore.ValidateAndGetActualKey(req.Context(), key)
		return err
	}

	// If it's a direct key, make sure it looks valid (non-empty)
	if key == "" {
		return fmt.Errorf("empty API key")
	}

	return nil
}

// ExtractRequestModelAndMessages extracts the model and messages from the request
func (f *FireworksProxy) ExtractRequestModelAndMessages(req *http.Request) (string, []string) {
	if req.Body == nil {
		return "", []string{}
	}

	var bodyBytes []byte
	var err error

	if req.GetBody != nil {
		bodyReader, err := req.GetBody()
		if err != nil {
			return "", []string{}
		}
		defer bodyReader.Close()
		bodyBytes, _ = io.ReadAll(bodyReader)
	} else {
		bodyBytes, err = io.ReadAll(req.Body)
		if err != nil {
			return "", []string{}
		}
		req.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
		req.GetBody = func() (io.ReadCloser, error) {
			return io.NopCloser(bytes.NewBuffer(bodyBytes)), nil
		}
	}

	var requestData map[string]interface{}
	if err := json.Unmarshal(bodyBytes, &requestData); err != nil {
		return "", []string{}
	}

	model := ""
	if m, ok := requestData["model"]; ok {
		model = fmt.Sprintf("%v", m)
	}

	return model, []string{}
}

// ParseResponseMetadata parses Fireworks response to extract usage metadata
func (f *FireworksProxy) ParseResponseMetadata(responseBody io.Reader, isStreaming bool) (*LLMResponseMetadata, error) {
	if isStreaming {
		// For streaming, we don't have full metadata until the end
		return &LLMResponseMetadata{}, nil
	}

	// Parse non-streaming response
	var response map[string]interface{}
	if err := json.NewDecoder(responseBody).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to parse Fireworks response: %w", err)
	}

	metadata := &LLMResponseMetadata{}

	// Extract usage if available
	if usage, ok := response["usage"].(map[string]interface{}); ok {
		if promptTokens, ok := usage["prompt_tokens"].(float64); ok {
			metadata.InputTokens = int(promptTokens)
		}
		if completionTokens, ok := usage["completion_tokens"].(float64); ok {
			metadata.OutputTokens = int(completionTokens)
		}
		if totalTokens, ok := usage["total_tokens"].(float64); ok {
			metadata.TotalTokens = int(totalTokens)
		}
	}

	return metadata, nil
}

// init registers the Fireworks provider if the environment is configured
func init() {
	// Fireworks provider is registered in main.go
}
