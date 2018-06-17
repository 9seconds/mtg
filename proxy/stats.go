package proxy

import (
	"encoding/json"
	"net/http"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/9seconds/mtg/config"
)

type statsUptime time.Time

func (s statsUptime) MarshalJSON() ([]byte, error) {
	uptime := int(time.Since(time.Time(s)).Seconds())
	return []byte(strconv.Itoa(uptime)), nil
}

// Stats is a datastructure for statistics on work of this proxy.
type Stats struct {
	AllConnections    uint64 `json:"all_connections"`
	ActiveConnections uint32 `json:"active_connections"`
	Traffic           struct {
		Incoming uint64 `json:"incoming"`
		Outgoing uint64 `json:"outgoing"`
	} `json:"traffic"`
	URLs   config.IPURLs `json:"urls"`
	Uptime statsUptime   `json:"uptime"`

	conf *config.Config
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

// Serve runs statistics HTTP server.
func (s *Stats) Serve() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		encoder := json.NewEncoder(w)
		encoder.SetEscapeHTML(false)
		encoder.SetIndent("", "  ")
		encoder.Encode(s) // nolint: errcheck, gas
	})

	http.ListenAndServe(s.conf.StatAddr(), nil) // nolint: errcheck, gas
}

// NewStats returns new instance of statistics datastructure.
func NewStats(conf *config.Config) *Stats {
	stat := &Stats{
		Uptime: statsUptime(time.Now()),
		conf:   conf,
	}
	stat.URLs = conf.GetURLs()

	return stat
}
