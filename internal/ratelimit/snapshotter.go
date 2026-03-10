package ratelimit

// Snapshotter is an optional interface that rate limiters may implement
// to expose current window counters to the dashboard.
// The dashboard checks for this interface via a type assertion - no change
// to the RateLimiter interface is required.
type Snapshotter interface {
	Snapshot() RateLimitCounters
}

// RateLimitCounters is a point-in-time copy of the global window counters.
type RateLimitCounters struct {
	GlobalMinuteRequests int
	GlobalMinuteTokens   int
	GlobalDayRequests    int
	GlobalDayTokens      int
}

// Snapshot implements Snapshotter for memoryLimiter.
// It reads the "global" scope counters from both minute and day windows.
func (m *memoryLimiter) Snapshot() RateLimitCounters {
	m.mu.Lock()
	defer m.mu.Unlock()
	var c RateLimitCounters
	if minC, ok := m.minute["global"]; ok {
		c.GlobalMinuteRequests = minC.Requests
		c.GlobalMinuteTokens = minC.Tokens
	}
	if dayC, ok := m.day["global"]; ok {
		c.GlobalDayRequests = dayC.Requests
		c.GlobalDayTokens = dayC.Tokens
	}
	return c
}
