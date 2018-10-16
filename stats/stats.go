package stats

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	humanize "github.com/dustin/go-humanize"

	"github.com/9seconds/mtg/config"
	"github.com/9seconds/mtg/mtproto"
)

type uptime time.Time

func (u uptime) MarshalJSON() ([]byte, error) {
	duration := time.Since(time.Time(u))
	value := map[string]string{
		"seconds": strconv.Itoa(int(duration.Seconds())),
		"human":   humanize.Time(time.Time(u)),
	}

	return json.Marshal(value)
}

type connectionType struct {
	IPv6 uint32 `json:"ipv6"`
	IPv4 uint32 `json:"ipv4"`
}

type baseConnections struct {
	All          connectionType `json:"all"`
	Abridged     connectionType `json:"abridged"`
	Intermediate connectionType `json:"intermediate"`
	Secure       connectionType `json:"secure"`
}

type connections struct {
	baseConnections
}

func (c connections) MarshalJSON() ([]byte, error) {
	c.All.IPv4 = c.Abridged.IPv4 + c.Intermediate.IPv4 + c.Secure.IPv4
	c.All.IPv6 = c.Abridged.IPv6 + c.Intermediate.IPv6 + c.Secure.IPv6

	return json.Marshal(c.baseConnections)
}

type traffic struct {
	ingress uint64
	egress  uint64
}

func (t *traffic) dumpValue(value uint64) map[string]interface{} {
	return map[string]interface{}{
		"bytes": value,
		"human": humanize.Bytes(value),
	}
}

func (t traffic) MarshalJSON() ([]byte, error) {
	value := map[string]map[string]interface{}{
		"ingress": t.dumpValue(t.ingress),
		"egress":  t.dumpValue(t.egress),
	}

	return json.Marshal(value)
}

type speed struct {
	ingress uint64
	egress  uint64
}

func (s *speed) dumpValue(value uint64) map[string]interface{} {
	return map[string]interface{}{
		"bytes/s": value,
		"human":   fmt.Sprintf("%s/s", humanize.Bytes(value)),
	}
}

func (s speed) MarshalJSON() ([]byte, error) {
	value := map[string]map[string]interface{}{
		"ingress": s.dumpValue(s.ingress),
		"egress":  s.dumpValue(s.egress),
	}

	return json.Marshal(value)
}

type Stats struct {
	URLs        config.IPURLs `json:"urls"`
	Connections connections   `json:"connections"`
	Traffic     traffic       `json:"traffic"`
	Speed       speed         `json:"speed"`
	Uptime      uptime        `json:"uptime"`
	Crashes     uint32        `json:"crashes"`

	previousTraffic traffic
}

func (s *Stats) start() {
	speedChan := time.Tick(time.Second)

	for {
		select {
		case <-speedChan:
			s.handleSpeed()
		case event := <-trafficChan:
			s.handleTraffic(event)
		case event := <-connectionsChan:
			s.handleConnection(event)
		case getStatsChan := <-statsChan:
			s.handleGetStats(getStatsChan)
		case <-crashesChan:
			s.handleCrash()
		}
	}
}

func (s *Stats) handleTraffic(evt trafficData) {
	if evt.ingress {
		s.Traffic.ingress += uint64(evt.traffic)
	} else {
		s.Traffic.egress += uint64(evt.traffic)
	}
}

func (s *Stats) handleSpeed() {
	s.Speed.ingress = s.Traffic.ingress - s.previousTraffic.ingress
	s.Speed.egress = s.Traffic.egress - s.previousTraffic.egress
	s.previousTraffic.ingress = s.Traffic.ingress
	s.previousTraffic.egress = s.Traffic.egress
}

func (s *Stats) handleConnection(evt connectionData) {
	var inc uint32 = 1
	if !evt.connected {
		inc = ^uint32(0)
	}

	var conn *connectionType
	switch evt.connectionType {
	case mtproto.ConnectionTypeAbridged:
		conn = &s.Connections.Abridged
	case mtproto.ConnectionTypeSecure:
		conn = &s.Connections.Secure
	default:
		conn = &s.Connections.Intermediate
	}

	if evt.addr.IP.To4() != nil {
		conn.IPv4 += inc
	} else {
		conn.IPv6 += inc
	}
}

func (s *Stats) handleGetStats(getStatsChan chan<- Stats) {
	getStatsChan <- *s
}

func (s *Stats) handleCrash() {
	s.Crashes++
}

func NewStats(conf *config.Config) *Stats {
	return &Stats{
		URLs:   conf.GetURLs(),
		Uptime: uptime(time.Now()),
	}
}
