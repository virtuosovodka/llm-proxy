package dashboard

import (
	"net/http"
	"sort"
	"sync"
	"time"

	"github.com/Instawork/llm-proxy/internal/middleware"
	"github.com/Instawork/llm-proxy/internal/providers"
)

// ringBuffer is a fixed-size circular buffer of RequestRecord values.
type ringBuffer struct {
	mu      sync.Mutex
	entries []RequestRecord
	size    int
	next    int
	count   int
}

func newRingBuffer(size int) *ringBuffer {
	return &ringBuffer{entries: make([]RequestRecord, size), size: size}
}

// push adds a new entry, overwriting the oldest when full.
func (rb *ringBuffer) push(r RequestRecord) {
	rb.mu.Lock()
	defer rb.mu.Unlock()
	rb.entries[rb.next] = r
	rb.next = (rb.next + 1) % rb.size
	if rb.count < rb.size {
		rb.count++
	}
}

// snapshot returns a copy of all entries in insertion order (oldest first).
func (rb *ringBuffer) snapshot() []RequestRecord {
	rb.mu.Lock()
	defer rb.mu.Unlock()
	if rb.count == 0 {
		return nil
	}
	result := make([]RequestRecord, rb.count)
	if rb.count < rb.size {
		copy(result, rb.entries[:rb.count])
	} else {
		// buffer is full; oldest entry is at rb.next
		n := copy(result, rb.entries[rb.next:])
		copy(result[n:], rb.entries[:rb.next])
	}
	return result
}

// timeBucket holds aggregated counters for one one-minute window.
type timeBucket struct {
	BucketStart  time.Time
	Requests     int
	InputTokens  int
	OutputTokens int
}

type providerModelKey struct {
	Provider string
	Model    string
}

// StatsCollector aggregates per-request statistics in memory.
// It is thread-safe and designed to be long-lived (created once at startup).
type StatsCollector struct {
	mu            sync.RWMutex
	buckets       map[int64]*timeBucket        // keyed by unix-minute; evicted after 25h
	perModel      map[providerModelKey]*ProviderModelStats
	recent        *ringBuffer // last 200 completed requests
	startedAt     time.Time
	totalRequests int
	totalTokens   int
}

// NewStatsCollector creates a zeroed collector ready for use.
func NewStatsCollector() *StatsCollector {
	return &StatsCollector{
		buckets:   make(map[int64]*timeBucket),
		perModel:  make(map[providerModelKey]*ProviderModelStats),
		recent:    newRingBuffer(200),
		startedAt: time.Now(),
	}
}

// Callback returns a MetadataCallback that feeds completed request data into
// the collector. Pass the returned function to TokenParsingMiddleware.
func (sc *StatsCollector) Callback(pm *providers.ProviderManager) middleware.MetadataCallback {
	return func(r *http.Request, meta *providers.LLMResponseMetadata) {
		if meta == nil {
			return
		}
		provider := middleware.GetProviderFromRequest(pm, r)
		userID := middleware.ExtractUserIDFromRequest(r, provider)
		ipAddress := middleware.ExtractIPAddressFromRequest(r)

		rec := RequestRecord{
			Timestamp:    time.Now(),
			Provider:     meta.Provider,
			Model:        meta.Model,
			Endpoint:     r.URL.Path,
			InputTokens:  meta.InputTokens,
			OutputTokens: meta.OutputTokens,
			TotalTokens:  meta.TotalTokens,
			IsStreaming:  meta.IsStreaming,
			FinishReason: meta.FinishReason,
			UserID:       userID,
			IPAddress:    ipAddress,
		}

		sc.recent.push(rec)
		sc.record(time.Now(), meta.Provider, meta.Model, meta.InputTokens, meta.OutputTokens, meta.TotalTokens)
	}
}

// record updates all in-memory counters. Called under no lock; acquires write lock internally.
func (sc *StatsCollector) record(t time.Time, provider, model string, in, out, total int) {
	sc.mu.Lock()
	defer sc.mu.Unlock()

	sc.totalRequests++
	sc.totalTokens += total

	// Update the per-minute time bucket.
	bucketKey := t.Truncate(time.Minute).Unix()
	if b, ok := sc.buckets[bucketKey]; ok {
		b.Requests++
		b.InputTokens += in
		b.OutputTokens += out
	} else {
		sc.buckets[bucketKey] = &timeBucket{
			BucketStart:  t.Truncate(time.Minute),
			Requests:     1,
			InputTokens:  in,
			OutputTokens: out,
		}
	}

	// Evict buckets older than 25 hours to bound memory usage.
	cutoff := t.Add(-25 * time.Hour).Unix()
	for k := range sc.buckets {
		if k < cutoff {
			delete(sc.buckets, k)
		}
	}

	// Update all-time per-model totals.
	key := providerModelKey{Provider: provider, Model: model}
	if s, ok := sc.perModel[key]; ok {
		s.RequestsTotal++
		s.InputTokens += in
		s.OutputTokens += out
	} else {
		sc.perModel[key] = &ProviderModelStats{
			Provider:      provider,
			Model:         model,
			RequestsTotal: 1,
			InputTokens:   in,
			OutputTokens:  out,
		}
	}
}

// Snapshot returns a consistent point-in-time copy of all aggregated data.
// It is safe to call concurrently and is designed for low latency (<1ms typical).
func (sc *StatsCollector) Snapshot() StatsSnapshot {
	now := time.Now()

	// Capture ring buffer before acquiring the main lock (it has its own mutex).
	recentAll := sc.recent.snapshot()

	sc.mu.RLock()
	defer sc.mu.RUnlock()

	snap := StatsSnapshot{
		GeneratedAt:   now,
		UptimeSeconds: now.Sub(sc.startedAt).Seconds(),
		TotalRequests: sc.totalRequests,
		TotalTokens:   sc.totalTokens,
	}

	// Build request rate from the last 10 minutes of per-minute buckets.
	tenMinAgo := now.Add(-10 * time.Minute).Truncate(time.Minute)

	type bucketEntry struct {
		key int64
		b   *timeBucket
	}
	var sorted []bucketEntry
	for k, b := range sc.buckets {
		if time.Unix(k, 0).Equal(tenMinAgo) || time.Unix(k, 0).After(tenMinAgo) {
			sorted = append(sorted, bucketEntry{k, b})
		}
	}
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].key < sorted[j].key })

	for _, e := range sorted {
		snap.RequestRate = append(snap.RequestRate, BucketPoint{
			Timestamp: e.key,
			Label:     e.b.BucketStart.Format("15:04"),
			Requests:  e.b.Requests,
			Tokens:    e.b.InputTokens + e.b.OutputTokens,
		})
	}

	// Compute per-model window request counts from the ring buffer.
	oneMinAgo := now.Add(-time.Minute)
	oneHourAgo := now.Add(-time.Hour)
	oneDayAgo := now.Add(-24 * time.Hour)

	type windowCnt struct{ m1, h1, d1 int }
	modelCounts := make(map[providerModelKey]*windowCnt)
	for _, rec := range recentAll {
		k := providerModelKey{Provider: rec.Provider, Model: rec.Model}
		if _, ok := modelCounts[k]; !ok {
			modelCounts[k] = &windowCnt{}
		}
		if rec.Timestamp.After(oneMinAgo) {
			modelCounts[k].m1++
		}
		if rec.Timestamp.After(oneHourAgo) {
			modelCounts[k].h1++
		}
		if rec.Timestamp.After(oneDayAgo) {
			modelCounts[k].d1++
		}
	}

	for key, s := range sc.perModel {
		stat := ProviderModelStats{
			Provider:      s.Provider,
			Model:         s.Model,
			RequestsTotal: s.RequestsTotal,
			InputTokens:   s.InputTokens,
			OutputTokens:  s.OutputTokens,
		}
		if wc, ok := modelCounts[key]; ok {
			stat.Requests1m = wc.m1
			stat.Requests1h = wc.h1
			stat.Requests24h = wc.d1
		}
		snap.PerModel = append(snap.PerModel, stat)
	}
	sort.Slice(snap.PerModel, func(i, j int) bool {
		return snap.PerModel[i].RequestsTotal > snap.PerModel[j].RequestsTotal
	})

	// Recent requests: last 20, newest first.
	recent := recentAll
	if len(recent) > 20 {
		recent = recent[len(recent)-20:]
	}
	for i, j := 0, len(recent)-1; i < j; i, j = i+1, j-1 {
		recent[i], recent[j] = recent[j], recent[i]
	}
	snap.RecentRequests = recent

	return snap
}
