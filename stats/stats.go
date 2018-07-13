package stats

import (
	"encoding/json"
	"fmt"
	"strconv"
	"sync"
	"time"

	humanize "github.com/dustin/go-humanize"

	"github.com/9seconds/mtg/config"
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

type trafficValue uint64

func (t trafficValue) MarshalJSON() ([]byte, error) {
	tv := uint64(t)
	value := map[string]interface{}{
		"bytes": tv,
		"human": humanize.Bytes(tv),
	}

	return json.Marshal(value)
}

type trafficSpeedValue uint64

func (t trafficSpeedValue) MarshalJSON() ([]byte, error) {
	speed := uint64(t)
	value := map[string]interface{}{
		"bytes/s": speed,
		"human":   fmt.Sprintf("%s/S", humanize.Bytes(speed)),
	}

	return json.Marshal(value)
}

type connections struct {
	All          connectionType `json:"all"`
	Abridged     connectionType `json:"abridged"`
	Intermediate connectionType `json:"intermediate"`
	Secure       connectionType `json:"secure"`
}

func (c connections) MarshalJSON() ([]byte, error) {
	c.All.IPv4 = c.Abridged.IPv4 + c.Intermediate.IPv4 + c.Secure.IPv4
	c.All.IPv6 = c.Abridged.IPv6 + c.Intermediate.IPv6 + c.Secure.IPv6

	value := struct {
		All          connectionType `json:"all"`
		Abridged     connectionType `json:"abridged"`
		Intermediate connectionType `json:"intermediate"`
		Secure       connectionType `json:"secure"`
	}{
		All:          c.All,
		Abridged:     c.Abridged,
		Intermediate: c.Intermediate,
		Secure:       c.Secure,
	}

	return json.Marshal(value)
}

type connectionType struct {
	IPv6 uint32 `json:"ipv6"`
	IPv4 uint32 `json:"ipv4"`
}

type traffic struct {
	Ingress trafficValue `json:"ingress"`
	Egress  trafficValue `json:"egress"`
}

type speed struct {
	Ingress trafficSpeedValue `json:"ingress"`
	Egress  trafficSpeedValue `json:"egress"`
}

type stats struct {
	URLs        config.IPURLs `json:"urls"`
	Connections connections   `json:"connections"`
	Traffic     traffic       `json:"traffic"`
	Speed       speed         `json:"speed"`
	Uptime      uptime        `json:"uptime"`
	Crashes     uint32        `json:"crashes"`

	speedCurrent speed
	mutex        *sync.RWMutex
}
