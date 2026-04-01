package mtglib

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewProxyStats(t *testing.T) {
	t.Parallel()

	stats := NewProxyStats()
	assert.NotNil(t, stats)
	assert.NotNil(t, stats.users)
	assert.False(t, stats.startedAt.IsZero())
}

func TestPreRegister(t *testing.T) {
	t.Parallel()

	stats := NewProxyStats()
	stats.PreRegister("alice")
	stats.PreRegister("bob")

	stats.mu.RLock()
	defer stats.mu.RUnlock()

	assert.Contains(t, stats.users, "alice")
	assert.Contains(t, stats.users, "bob")
	assert.Equal(t, int64(0), stats.users["alice"].connections.Load())
}

func TestOnConnectDisconnect(t *testing.T) {
	t.Parallel()

	stats := NewProxyStats()
	stats.PreRegister("alice")

	stats.OnConnect("alice")
	assert.Equal(t, int64(1), stats.users["alice"].connections.Load())

	stats.OnConnect("alice")
	assert.Equal(t, int64(2), stats.users["alice"].connections.Load())

	stats.OnDisconnect("alice")
	assert.Equal(t, int64(1), stats.users["alice"].connections.Load())

	stats.OnDisconnect("alice")
	assert.Equal(t, int64(0), stats.users["alice"].connections.Load())
}

func TestAddBytes(t *testing.T) {
	t.Parallel()

	stats := NewProxyStats()
	stats.PreRegister("alice")

	stats.AddBytesIn("alice", 100)
	stats.AddBytesIn("alice", 200)
	stats.AddBytesOut("alice", 50)

	st := stats.users["alice"]
	assert.Equal(t, int64(300), st.bytesIn.Load())
	assert.Equal(t, int64(50), st.bytesOut.Load())
}

func TestUpdateLastSeen(t *testing.T) {
	t.Parallel()

	stats := NewProxyStats()
	stats.PreRegister("alice")

	before := time.Now()
	stats.UpdateLastSeen("alice")
	after := time.Now()

	lastSeen := stats.users["alice"].lastSeen.Load().(time.Time)
	assert.False(t, lastSeen.Before(before))
	assert.False(t, lastSeen.After(after))
}

func TestGetOrCreateLazy(t *testing.T) {
	t.Parallel()

	stats := NewProxyStats()

	// getOrCreate should create a new entry on first access.
	stats.OnConnect("new-user")
	assert.Equal(t, int64(1), stats.users["new-user"].connections.Load())
}

func TestServeHTTPBasic(t *testing.T) {
	t.Parallel()

	stats := NewProxyStats()
	stats.PreRegister("alice")
	stats.PreRegister("bob")

	stats.OnConnect("alice")
	stats.OnConnect("alice")
	stats.OnConnect("bob")
	stats.AddBytesIn("alice", 1024)
	stats.AddBytesOut("alice", 512)
	stats.UpdateLastSeen("alice")

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/stats", nil)

	stats.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "application/json", rec.Header().Get("Content-Type"))

	var resp StatsResponse
	err := json.Unmarshal(rec.Body.Bytes(), &resp)
	require.NoError(t, err)

	assert.Equal(t, int64(3), resp.TotalConnections)
	assert.False(t, resp.StartedAt.IsZero())
	assert.GreaterOrEqual(t, resp.UptimeSeconds, int64(0))

	alice, ok := resp.Users["alice"]
	require.True(t, ok)
	assert.Equal(t, int64(2), alice.Connections)
	assert.Equal(t, int64(1024), alice.BytesIn)
	assert.Equal(t, int64(512), alice.BytesOut)
	assert.NotNil(t, alice.LastSeen)

	bob, ok := resp.Users["bob"]
	require.True(t, ok)
	assert.Equal(t, int64(1), bob.Connections)
	assert.Equal(t, int64(0), bob.BytesIn)
	assert.Equal(t, int64(0), bob.BytesOut)
	assert.Nil(t, bob.LastSeen)
}

func TestServeHTTPEmpty(t *testing.T) {
	t.Parallel()

	stats := NewProxyStats()

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/stats", nil)

	stats.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var resp StatsResponse
	err := json.Unmarshal(rec.Body.Bytes(), &resp)
	require.NoError(t, err)

	assert.Empty(t, resp.Users)
	assert.Equal(t, int64(0), resp.TotalConnections)
}

func TestComputeFairCaps_NoThrottle(t *testing.T) {
	t.Parallel()

	caps := computeFairCaps(map[string]int64{
		"a": 10,
		"b": 20,
	}, 100)

	assert.Nil(t, caps)
}

func TestComputeFairCaps_ExactLimit(t *testing.T) {
	t.Parallel()

	caps := computeFairCaps(map[string]int64{
		"a": 50,
		"b": 50,
	}, 100)

	assert.Nil(t, caps)
}

func TestComputeFairCaps_UserExample(t *testing.T) {
	t.Parallel()

	// The user's exact example: limit=100, users=[1, 1, 90, 110]
	// Small users keep their 1+1=2, remaining budget=98, split among 2 → 49 each
	caps := computeFairCaps(map[string]int64{
		"a": 1,
		"b": 1,
		"c": 90,
		"d": 110,
	}, 100)

	assert.Len(t, caps, 2)
	assert.Equal(t, int64(49), caps["c"])
	assert.Equal(t, int64(49), caps["d"])
	// "a" and "b" should not appear in caps (they're under the fair share)
	_, hasA := caps["a"]
	_, hasB := caps["b"]
	assert.False(t, hasA)
	assert.False(t, hasB)
}

func TestComputeFairCaps_AllOverLimit(t *testing.T) {
	t.Parallel()

	caps := computeFairCaps(map[string]int64{
		"a": 100,
		"b": 100,
	}, 50)

	assert.Len(t, caps, 2)
	assert.Equal(t, int64(25), caps["a"])
	assert.Equal(t, int64(25), caps["b"])
}

func TestComputeFairCaps_SingleHeavyUser(t *testing.T) {
	t.Parallel()

	caps := computeFairCaps(map[string]int64{
		"light": 5,
		"heavy": 200,
	}, 100)

	// light(5) < fairShare(50), keeps 5. Budget = 95. Heavy capped to 95.
	assert.Len(t, caps, 1)
	assert.Equal(t, int64(95), caps["heavy"])
}

func TestCanConnect_NoThrottle(t *testing.T) {
	t.Parallel()

	stats := NewProxyStats()
	// throttleLimit = 0 (default), so CanConnect always returns true
	assert.True(t, stats.CanConnect("anyone"))
}

func TestCanConnect_WithCap(t *testing.T) {
	t.Parallel()

	stats := NewProxyStats()
	stats.PreRegister("heavy")
	stats.SetThrottle(100, 5*time.Second)

	// Simulate 50 connections
	for range 50 {
		stats.OnConnect("heavy")
	}

	// Set cap to 50
	stats.throttleMu.Lock()
	stats.throttleCaps = map[string]int64{"heavy": 50}
	stats.throttleActive.Store(true)
	stats.throttleMu.Unlock()

	// At exactly the cap → reject
	assert.False(t, stats.CanConnect("heavy"))

	// Disconnect one → allow
	stats.OnDisconnect("heavy")
	assert.True(t, stats.CanConnect("heavy"))
}

func TestCanConnect_NoCap(t *testing.T) {
	t.Parallel()

	stats := NewProxyStats()
	stats.SetThrottle(100, 5*time.Second)

	// User not in caps map → always allowed
	assert.True(t, stats.CanConnect("uncapped-user"))
}

func TestServeHTTPThrottleInfo(t *testing.T) {
	t.Parallel()

	stats := NewProxyStats()
	stats.PreRegister("alice")
	stats.SetThrottle(100, 5*time.Second)

	stats.throttleMu.Lock()
	stats.throttleCaps = map[string]int64{"alice": 50}
	stats.throttleActive.Store(true)
	stats.throttleMu.Unlock()

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/stats", nil)

	stats.ServeHTTP(rec, req)

	var resp StatsResponse
	err := json.Unmarshal(rec.Body.Bytes(), &resp)
	require.NoError(t, err)

	require.NotNil(t, resp.Throttle)
	assert.True(t, resp.Throttle.Active)
	assert.Equal(t, int64(100), resp.Throttle.Limit)
	assert.Equal(t, int64(50), resp.Throttle.Caps["alice"])
}

func TestServeHTTPNoThrottle(t *testing.T) {
	t.Parallel()

	stats := NewProxyStats()

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/stats", nil)

	stats.ServeHTTP(rec, req)

	var resp StatsResponse
	err := json.Unmarshal(rec.Body.Bytes(), &resp)
	require.NoError(t, err)

	assert.Nil(t, resp.Throttle)
}

func TestServeHTTPLastSeenZeroIsNull(t *testing.T) {
	t.Parallel()

	stats := NewProxyStats()
	stats.PreRegister("alice")

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/stats", nil)

	stats.ServeHTTP(rec, req)

	var raw map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &raw))

	var users map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(raw["users"], &users))

	var aliceRaw map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(users["alice"], &aliceRaw))

	assert.Equal(t, "null", string(aliceRaw["last_seen"]))
}
