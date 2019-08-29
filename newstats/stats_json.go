package newstats

import (
	"encoding/json"
	"net"
	"net/http"
	"sync/atomic"
	"time"

	"go.uber.org/zap"

	"github.com/9seconds/mtg/newprotocol"
)

type statsJSON struct {
	Connections statsJSONConnections `json:"connections"`
	Traffic     statsJSONTraffic     `json:"traffic"`
	Uptime      statsJSONUptime      `json:"uptime"`
	Crashes     uint32               `json:"crashes"`
	AntiReplays uint32               `json:"anti_replay_detected"`
}

type statsBaseJSONConnections struct {
	All          statsJSONConnectionType `json:"all"`
	Abridged     statsJSONConnectionType `json:"abridged"`
	Intermediate statsJSONConnectionType `json:"intermediate"`
	Secured      statsJSONConnectionType `json:"secured"`
}

type statsJSONConnections struct {
	statsBaseJSONConnections
}

type statsJSONConnectionType struct {
	IPv4 uint32 `json:"ipv4"`
	IPv6 uint32 `json:"ipv6"`
}

func (c statsJSONConnections) MarshalJSON() ([]byte, error) {
	c.All.IPv4 = c.Abridged.IPv4 + c.Intermediate.IPv4 + c.Secured.IPv4
	c.All.IPv6 = c.Abridged.IPv6 + c.Intermediate.IPv6 + c.Secured.IPv6

	return json.Marshal(c.statsBaseJSONConnections)
}

type statsJSONTraffic struct {
	Ingress uint64 `json:"ingress"`
	Egress  uint64 `json:"egress"`
}

type statsJSONUptime time.Time

func (s statsJSONUptime) MarshalJSON() ([]byte, error) {
	return json.Marshal(time.Since(time.Time(s)).Seconds())
}

func (s *statsJSON) IngressTraffic(traffic int) {
	atomic.AddUint64(&s.Traffic.Ingress, uint64(traffic))
}

func (s *statsJSON) EgressTraffic(traffic int) {
	atomic.AddUint64(&s.Traffic.Egress, uint64(traffic))
}

func (s *statsJSON) ClientConnected(connectionType newprotocol.ConnectionType, addr *net.TCPAddr) {
	s.changeConnections(connectionType, addr, 1)
}

func (s *statsJSON) ClientDisconnected(connectionType newprotocol.ConnectionType, addr *net.TCPAddr) {
	s.changeConnections(connectionType, addr, ^uint32(0))
}

func (s *statsJSON) changeConnections(connectionType newprotocol.ConnectionType, addr *net.TCPAddr, value uint32) {
	var connections *statsJSONConnectionType

	switch connectionType {
	case newprotocol.ConnectionTypeAbridged:
		connections = &s.Connections.Abridged
	case newprotocol.ConnectionTypeSecure:
		connections = &s.Connections.Secured
	default:
		connections = &s.Connections.Intermediate
	}

	if addr.IP.To4() == nil {
		atomic.AddUint32(&connections.IPv4, value)
	} else {
		atomic.AddUint32(&connections.IPv6, value)
	}
}

func (s *statsJSON) Crash() {
	atomic.AddUint32(&s.Crashes, 1)
}

func (s *statsJSON) AntiReplayDetected() {
	atomic.AddUint32(&s.AntiReplays, 1)
}

func newStatsJSON(mux *http.ServeMux) Stats {
	instance := &statsJSON{}
	logger := zap.S().Named("stats")

	mux.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		first, err := json.Marshal(instance)
		if err != nil {
			logger.Errorw("Cannot encode json", "error", err)
			http.Error(w, "Internal server error", http.StatusServiceUnavailable)
			return
		}

		interim := map[string]interface{}{}
		if err := json.Unmarshal(first, &interim); err != nil {
			panic(err)
		}

		encoder := json.NewEncoder(w)
		encoder.SetEscapeHTML(false)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(interim); err != nil {
			logger.Errorw("Cannot encode json", "error", err)
		}
	})

	return instance
}
