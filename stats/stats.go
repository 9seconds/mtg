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
	All          uint32 `json:"all"`
	Abridged     uint32 `json:"abridged"`
	Intermediate uint32 `json:"intermediate"`
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
	URLs              config.IPURLs `json:"urls"`
	ActiveConnections connections   `json:"active_connections"`
	AllConnections    connections   `json:"all_connections"`
	Traffic           traffic       `json:"traffic"`
	Speed             speed         `json:"speed"`
	Uptime            uptime        `json:"uptime"`

	speedCurrent *speed
	mutex        *sync.RWMutex
}
