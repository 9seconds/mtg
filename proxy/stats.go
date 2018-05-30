package proxy

import (
	"encoding/json"
	"net"
	"net/http"
	"strconv"
	"sync/atomic"
	"time"
)

type statsUptime time.Time

func (s statsUptime) MarshalJSON() ([]byte, error) {
	uptime := int(time.Since(time.Time(s)).Seconds())
	return []byte(strconv.Itoa(uptime)), nil
}

type Stats struct {
	AllConnections    uint64 `json:"all_connections"`
	ActiveConnections uint32 `json:"active_connections"`
	Traffic           struct {
		Incoming uint64 `json:"incoming"`
		Outgoing uint64 `json:"outgoing"`
	} `json:"traffic"`
	Uptime statsUptime `json:"uptime"`
}

func (s *Stats) newConnection() {
	atomic.AddUint64(&s.AllConnections, 1)
	atomic.AddUint32(&s.ActiveConnections, 1)
}

func (s *Stats) closeConnection() {
	atomic.AddUint32(&s.ActiveConnections, ^uint32(0))
}

func (s *Stats) addIncomingTraffic(n int) {
	atomic.AddUint64(&s.Traffic.Incoming, uint64(n))
}

func (s *Stats) addOutgoingTraffic(n int) {
	atomic.AddUint64(&s.Traffic.Outgoing, uint64(n))
}

func (s *Stats) Serve(host net.IP, port uint16) {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(s)
	})

	addr := net.JoinHostPort(host.String(), strconv.Itoa(int(port)))
	http.ListenAndServe(addr, nil)
}

func NewStats() *Stats {
	return &Stats{Uptime: statsUptime(time.Now())}
}
