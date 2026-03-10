package dashboard

import (
	_ "embed"
	"encoding/json"
	"net/http"

	"github.com/Instawork/llm-proxy/internal/config"
	"github.com/Instawork/llm-proxy/internal/providers"
	"github.com/Instawork/llm-proxy/internal/ratelimit"
	"github.com/gorilla/mux"
)

//go:embed dashboard.html
var dashboardHTML []byte

// Handler owns the HTTP handlers for the dashboard endpoints.
type Handler struct {
	collector       *StatsCollector
	providerManager *providers.ProviderManager
	rateLimiter     ratelimit.RateLimiter // may be nil
	config          *config.YAMLConfig
}

// NewHandler constructs the dashboard handler.
func NewHandler(
	c *StatsCollector,
	pm *providers.ProviderManager,
	rl ratelimit.RateLimiter,
	cfg *config.YAMLConfig,
) *Handler {
	return &Handler{
		collector:       c,
		providerManager: pm,
		rateLimiter:     rl,
		config:          cfg,
	}
}

// RegisterRoutes wires the dashboard routes into the given mux router.
// Call this before provider catch-all PathPrefix routes are registered.
func (h *Handler) RegisterRoutes(router *mux.Router) {
	router.HandleFunc("/dashboard", h.serveDashboard).Methods("GET", "HEAD")
	router.HandleFunc("/dashboard/", h.serveDashboard).Methods("GET", "HEAD")
	router.HandleFunc("/dashboard/api/stats", h.serveStats).Methods("GET")
}

// serveDashboard serves the embedded single-page HTML dashboard.
func (h *Handler) serveDashboard(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	w.Write(dashboardHTML)
}

// serveStats returns a JSON StatsSnapshot assembled from in-process data.
// Typical response latency is well under 1ms.
func (h *Handler) serveStats(w http.ResponseWriter, r *http.Request) {
	snap := h.buildSnapshot()
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	json.NewEncoder(w).Encode(snap)
}

// buildSnapshot assembles the full snapshot.
func (h *Handler) buildSnapshot() StatsSnapshot {
	snap := h.collector.Snapshot()

	// Inject live provider health (GetHealthStatus returns a fresh copy).
	snap.ProviderHealth = h.providerManager.GetHealthStatus()

	// Inject rate limit counters if the underlying limiter supports it.
	if h.rateLimiter != nil {
		if snapshotter, ok := h.rateLimiter.(ratelimit.Snapshotter); ok {
			counters := snapshotter.Snapshot()
			rl := &RateLimitSnapshot{
				GlobalMinuteRequests: counters.GlobalMinuteRequests,
				GlobalMinuteTokens:   counters.GlobalMinuteTokens,
				GlobalDayRequests:    counters.GlobalDayRequests,
				GlobalDayTokens:      counters.GlobalDayTokens,
			}
			// Fill in configured limits from config if available.
			if h.config != nil {
				lim := h.config.Features.RateLimiting.Limits
				rl.ConfiguredRPM = lim.RequestsPerMinute
				rl.ConfiguredTPM = lim.TokensPerMinute
				rl.ConfiguredRPD = lim.RequestsPerDay
				rl.ConfiguredTPD = lim.TokensPerDay
			}
			snap.RateLimit = rl
		}
	}

	return snap
}
