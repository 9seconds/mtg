package proxy

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"sync/atomic"
	"time"
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
	URLs struct {
		TG        string `json:"tg_url"`
		TMe       string `json:"tme_url"`
		TGQRCode  string `json:"tg_qrcode"`
		TMeQRCode string `json:"tme_qrcode"`
	} `json:"urls"`
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

// Serve runs statistics HTTP server.
func (s *Stats) Serve(host fmt.Stringer, port uint16) {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		encoder := json.NewEncoder(w)
		encoder.SetEscapeHTML(false)
		encoder.SetIndent("", "  ")
		encoder.Encode(s) // nolint: errcheck, gas
	})

	addr := net.JoinHostPort(host.String(), strconv.Itoa(int(port)))
	http.ListenAndServe(addr, nil) // nolint: errcheck, gas
}

// NewStats returns new instance of statistics datastructure.
func NewStats(serverName string, port uint16, secret string) *Stats {
	urlQuery := makeURLQuery(serverName, port, secret)

	stat := &Stats{Uptime: statsUptime(time.Now())}
	stat.URLs.TG = makeTGURL(urlQuery)
	stat.URLs.TMe = makeTMeURL(urlQuery)
	stat.URLs.TGQRCode = makeQRCodeURL(stat.URLs.TG)
	stat.URLs.TMeQRCode = makeQRCodeURL(stat.URLs.TMe)

	return stat
}

func makeURLQuery(serverName string, port uint16, secret string) url.Values {
	values := url.Values{}
	values.Set("server", serverName)
	values.Set("port", strconv.Itoa(int(port)))
	values.Set("secret", secret)

	return values
}

func makeTGURL(values url.Values) string {
	tgURL := url.URL{
		Scheme:   "tg",
		Host:     "proxy",
		RawQuery: values.Encode(),
	}

	return tgURL.String()
}

func makeTMeURL(values url.Values) string {
	tMeURL := url.URL{
		Scheme:   "https",
		Host:     "t.me",
		Path:     "proxy",
		RawQuery: values.Encode(),
	}

	return tMeURL.String()
}

func makeQRCodeURL(data string) string {
	QRURL := url.URL{
		Scheme: "https",
		Host:   "api.qrserver.com",
		Path:   "v1/create-qr-code",
	}

	values := url.Values{}
	values.Set("qzone", "4")
	values.Set("format", "svg")
	values.Set("data", data)
	QRURL.RawQuery = values.Encode()

	return QRURL.String()
}
