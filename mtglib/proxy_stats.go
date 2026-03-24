package mtglib

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"sync"
	"sync/atomic"
	"time"
)

type secretStats struct {
	connections atomic.Int64
	bytesIn     atomic.Int64
	bytesOut    atomic.Int64
	lastSeen    atomic.Value // stores time.Time
}

// ProxyStats tracks per-secret connection stats with atomic counters.
// Thread-safe for concurrent access from proxy goroutines.
type ProxyStats struct {
	mu        sync.RWMutex
	users     map[string]*secretStats
	startedAt time.Time
}

// NewProxyStats creates a new ProxyStats instance.
func NewProxyStats() *ProxyStats {
	return &ProxyStats{
		users:     make(map[string]*secretStats),
		startedAt: time.Now(),
	}
}

func (s *ProxyStats) getOrCreate(name string) *secretStats {
	s.mu.RLock()
	st, ok := s.users[name]
	s.mu.RUnlock()

	if ok {
		return st
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if st, ok = s.users[name]; ok {
		return st
	}

	st = &secretStats{}
	st.lastSeen.Store(time.Time{})
	s.users[name] = st

	return st
}

// PreRegister adds a secret name to the stats map so it appears in output
// even if no connections have been made yet.
func (s *ProxyStats) PreRegister(name string) {
	s.getOrCreate(name)
}

// OnConnect increments the active connection count for the given secret.
func (s *ProxyStats) OnConnect(name string) {
	s.getOrCreate(name).connections.Add(1)
}

// OnDisconnect decrements the active connection count for the given secret.
func (s *ProxyStats) OnDisconnect(name string) {
	s.getOrCreate(name).connections.Add(-1)
}

// AddBytesIn adds to the bytes-in counter for the given secret.
func (s *ProxyStats) AddBytesIn(name string, n int64) {
	s.getOrCreate(name).bytesIn.Add(n)
}

// AddBytesOut adds to the bytes-out counter for the given secret.
func (s *ProxyStats) AddBytesOut(name string, n int64) {
	s.getOrCreate(name).bytesOut.Add(n)
}

// UpdateLastSeen sets the last-seen timestamp for the given secret to now.
func (s *ProxyStats) UpdateLastSeen(name string) {
	s.getOrCreate(name).lastSeen.Store(time.Now())
}

// StatsResponse is the JSON response for the stats endpoint.
type StatsResponse struct {
	StartedAt        time.Time                `json:"started_at"`
	UptimeSeconds    int64                    `json:"uptime_seconds"`
	TotalConnections int64                    `json:"total_connections"`
	Users            map[string]UserStatsJSON `json:"users"`
}

// UserStatsJSON is the per-user portion of the stats JSON response.
type UserStatsJSON struct {
	Connections int64      `json:"connections"`
	BytesIn     int64      `json:"bytes_in"`
	BytesOut    int64      `json:"bytes_out"`
	LastSeen    *time.Time `json:"last_seen"`
}

func (s *ProxyStats) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var totalConns int64

	users := make(map[string]UserStatsJSON, len(s.users))

	for name, st := range s.users {
		conns := st.connections.Load()
		totalConns += conns

		lastSeen := st.lastSeen.Load().(time.Time) //nolint: forcetypeassert
		var lastSeenPtr *time.Time
		if !lastSeen.IsZero() {
			lastSeenPtr = &lastSeen
		}

		users[name] = UserStatsJSON{
			Connections: conns,
			BytesIn:     st.bytesIn.Load(),
			BytesOut:    st.bytesOut.Load(),
			LastSeen:    lastSeenPtr,
		}
	}

	resp := StatsResponse{
		StartedAt:        s.startedAt,
		UptimeSeconds:    int64(time.Since(s.startedAt).Seconds()),
		TotalConnections: totalConns,
		Users:            users,
	}

	w.Header().Set("Content-Type", "application/json")

	if err := json.NewEncoder(w).Encode(resp); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// StartServer starts an HTTP server for the stats API in a background goroutine.
// The server is shut down when ctx is cancelled.
func (s *ProxyStats) StartServer(ctx context.Context, bindTo string, logger Logger) {
	mux := http.NewServeMux()
	mux.Handle("/stats", s)

	srv := &http.Server{
		Addr:    bindTo,
		Handler: mux,
	}

	ln, err := net.Listen("tcp", bindTo)
	if err != nil {
		logger.WarningError("cannot start stats API listener", err)
		return
	}

	go func() {
		if err := srv.Serve(ln); err != nil && err != http.ErrServerClosed {
			logger.WarningError("stats API server error", err)
		}
	}()

	go func() {
		<-ctx.Done()

		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second) //nolint: mnd
		defer cancel()

		srv.Shutdown(shutdownCtx) //nolint: errcheck
	}()

	logger.BindStr("bind", bindTo).Info("Stats API server started")
}
