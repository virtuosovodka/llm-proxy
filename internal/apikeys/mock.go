package apikeys

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
	"sync"
	"time"
)

// MockStore is an in-memory implementation for local testing
type MockStore struct {
	mu   sync.RWMutex
	keys map[string]*APIKey
}

// NewMockStore creates a new in-memory API key store for testing
func NewMockStore() *MockStore {
	return &MockStore{
		keys: make(map[string]*APIKey),
	}
}

// CreateKey adds a new API key (Admin interface method)
func (m *MockStore) CreateKey(ctx context.Context, provider, actualKey, description string, dailyCostLimit int64, tags map[string]string) (*APIKey, error) {
	// Generate a random key ID
	randBytes := make([]byte, KeyLength/2)
	if _, err := rand.Read(randBytes); err != nil {
		return nil, err
	}
	keyID := KeyPrefix + hex.EncodeToString(randBytes)

	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()
	key := &APIKey{
		PK:             keyID,
		Provider:       provider,
		ActualKey:      actualKey,
		DailyCostLimit: dailyCostLimit,
		Description:    description,
		CreatedAt:      now,
		UpdatedAt:      now,
		Enabled:        true,
		Tags:           tags,
	}
	m.keys[keyID] = key

	return key, nil
}

// GetKey retrieves an API key by ID (Admin interface method)
func (m *MockStore) GetKey(ctx context.Context, keyID string) (*APIKey, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	key, exists := m.keys[keyID]
	if !exists {
		return nil, fmt.Errorf("key not found: %s", keyID)
	}

	return key, nil
}

// ListKeys returns all API keys, optionally filtered by provider (Admin interface method)
func (m *MockStore) ListKeys(ctx context.Context, provider string) ([]*APIKey, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var result []*APIKey
	for _, key := range m.keys {
		if provider != "" && key.Provider != provider {
			continue
		}
		result = append(result, key)
	}

	return result, nil
}

// UpdateKey updates an existing API key (Admin interface method)
func (m *MockStore) UpdateKey(ctx context.Context, keyID string, updates map[string]interface{}) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	key, exists := m.keys[keyID]
	if !exists {
		return fmt.Errorf("key not found: %s", keyID)
	}

	// Apply updates
	if description, ok := updates["description"]; ok {
		key.Description = description.(string)
	}
	if dailyCostLimit, ok := updates["daily_cost_limit"]; ok {
		key.DailyCostLimit = int64(dailyCostLimit.(float64))
	}
	if enabled, ok := updates["enabled"]; ok {
		key.Enabled = enabled.(bool)
	}
	if expiresAt, ok := updates["expires_at"]; ok {
		if t, ok := expiresAt.(time.Time); ok {
			key.ExpiresAt = &t
		}
	}

	key.UpdatedAt = time.Now()
	m.keys[keyID] = key

	return nil
}

// DeleteKey removes an API key (Admin interface method)
func (m *MockStore) DeleteKey(ctx context.Context, keyID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.keys[keyID]; !exists {
		return fmt.Errorf("key not found: %s", keyID)
	}

	delete(m.keys, keyID)
	return nil
}

// ValidateAndGetActualKey validates a key and returns the actual provider key
// If the key doesn't start with "iw:", it returns the key as-is
// Returns (actualKey, provider, error)
func (m *MockStore) ValidateAndGetActualKey(ctx context.Context, key string) (string, string, error) {
	// If it's not an internal key, return it as-is
	if !strings.HasPrefix(key, KeyPrefix) {
		return key, "", nil
	}

	// Look up the internal key
	m.mu.RLock()
	defer m.mu.RUnlock()

	apiKey, exists := m.keys[key]
	if !exists {
		return "", "", fmt.Errorf("key not found: %s", key)
	}

	// Check if key is expired
	if apiKey.ExpiresAt != nil && time.Now().After(*apiKey.ExpiresAt) {
		return "", "", fmt.Errorf("key has expired: %s", key)
	}

	// Check if key is enabled
	if !apiKey.Enabled {
		return "", "", fmt.Errorf("key is disabled: %s", key)
	}

	return apiKey.ActualKey, apiKey.Provider, nil
}

// GetKeysByTag retrieves all API keys with a specific tag value
func (m *MockStore) GetKeysByTag(ctx context.Context, tagKey, tagValue string) ([]*APIKey, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var result []*APIKey
	for _, key := range m.keys {
		if key.Tags != nil {
			if val, ok := key.Tags[tagKey]; ok && val == tagValue {
				result = append(result, key)
			}
		}
	}

	return result, nil
}

// UpdateKeysByTag updates all keys with a specific tag to use a new actual key
func (m *MockStore) UpdateKeysByTag(ctx context.Context, tagKey, tagValue, newActualKey string) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	updated := 0
	for _, key := range m.keys {
		if key.Tags != nil {
			if val, ok := key.Tags[tagKey]; ok && val == tagValue {
				key.ActualKey = newActualKey
				key.UpdatedAt = time.Now()
				updated++
			}
		}
	}

	return updated, nil
}

// DeleteKeysByTag deletes all keys with a specific tag
func (m *MockStore) DeleteKeysByTag(ctx context.Context, tagKey, tagValue string) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	deleted := 0
	for keyID, key := range m.keys {
		if key.Tags != nil {
			if val, ok := key.Tags[tagKey]; ok && val == tagValue {
				delete(m.keys, keyID)
				deleted++
			}
		}
	}

	return deleted, nil
}
