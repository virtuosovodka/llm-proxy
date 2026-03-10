package dashboard

import "time"

// StatsSnapshot is the complete JSON payload returned by GET /dashboard/api/stats.
type StatsSnapshot struct {
	GeneratedAt    time.Time              `json:"generated_at"`
	UptimeSeconds  float64                `json:"uptime_seconds"`
	ProviderHealth map[string]interface{} `json:"provider_health"`
	// RequestRate contains per-minute buckets for the last 10 minutes.
	// Each bucket includes both request count and token count.
	RequestRate    []BucketPoint        `json:"request_rate"`
	PerModel       []ProviderModelStats `json:"per_model"`
	RecentRequests []RequestRecord      `json:"recent_requests"`
	TotalRequests  int                  `json:"total_requests"`
	TotalTokens    int                  `json:"total_tokens"`
	// RateLimit is nil when rate limiting is disabled.
	RateLimit *RateLimitSnapshot `json:"rate_limit,omitempty"`
}

// BucketPoint is a one-minute resolution data point for the request rate chart.
type BucketPoint struct {
	Timestamp int64  `json:"ts"`    // Unix seconds, truncated to minute
	Label     string `json:"label"` // "HH:MM" for chart axis
	Requests  int    `json:"requests"`
	Tokens    int    `json:"tokens"`
}

// ProviderModelStats holds aggregated stats for a single provider+model pair.
type ProviderModelStats struct {
	Provider      string `json:"provider"`
	Model         string `json:"model"`
	RequestsTotal int    `json:"requests_total"`
	InputTokens   int    `json:"input_tokens"`
	OutputTokens  int    `json:"output_tokens"`
	Requests1m    int    `json:"requests_1m"`
	Requests1h    int    `json:"requests_1h"`
	Requests24h   int    `json:"requests_24h"`
}

// RequestRecord is one entry in the recent-requests ring buffer.
type RequestRecord struct {
	Timestamp    time.Time `json:"timestamp"`
	Provider     string    `json:"provider"`
	Model        string    `json:"model"`
	Endpoint     string    `json:"endpoint"`
	InputTokens  int       `json:"input_tokens"`
	OutputTokens int       `json:"output_tokens"`
	TotalTokens  int       `json:"total_tokens"`
	IsStreaming  bool      `json:"is_streaming"`
	FinishReason string    `json:"finish_reason"`
	UserID       string    `json:"user_id"`
	IPAddress    string    `json:"ip_address"`
}

// RateLimitSnapshot is a point-in-time view of current rate limit counters.
type RateLimitSnapshot struct {
	GlobalMinuteRequests int `json:"global_minute_requests"`
	GlobalMinuteTokens   int `json:"global_minute_tokens"`
	GlobalDayRequests    int `json:"global_day_requests"`
	GlobalDayTokens      int `json:"global_day_tokens"`
	ConfiguredRPM        int `json:"configured_rpm"`
	ConfiguredTPM        int `json:"configured_tpm"`
	ConfiguredRPD        int `json:"configured_rpd"`
	ConfiguredTPD        int `json:"configured_tpd"`
}
