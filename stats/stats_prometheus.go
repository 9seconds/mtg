package stats

import (
	"net"
	"net/http"
	"strconv"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/9seconds/mtg/config"
	"github.com/9seconds/mtg/conntypes"
)

type statsPrometheus struct {
	connections          *prometheus.GaugeVec
	telegramConnections  *prometheus.GaugeVec
	traffic              *prometheus.GaugeVec
	crashes              prometheus.Counter
	replayAttacks        prometheus.Counter
	authenticationFailed prometheus.Counter
	cloakedRequests      prometheus.Counter
}

func (s *statsPrometheus) IngressTraffic(traffic int) {
	s.traffic.WithLabelValues("ingress").Add(float64(traffic))
}

func (s *statsPrometheus) EgressTraffic(traffic int) {
	s.traffic.WithLabelValues("egress").Add(float64(traffic))
}

func (s *statsPrometheus) ClientConnected(connectionType conntypes.ConnectionType, addr *net.TCPAddr) {
	s.changeConnections(connectionType, addr, 1.0)
}

func (s *statsPrometheus) ClientDisconnected(connectionType conntypes.ConnectionType, addr *net.TCPAddr) {
	s.changeConnections(connectionType, addr, -1.0)
}

func (s *statsPrometheus) changeConnections(connectionType conntypes.ConnectionType,
	addr *net.TCPAddr,
	increment float64) {
	labels := [...]string{
		"intermediate",
		"ipv4",
	}

	switch connectionType {
	case conntypes.ConnectionTypeAbridged:
		labels[0] = "abridged"
	case conntypes.ConnectionTypeSecure:
		labels[0] = "secured"
	}

	if addr.IP.To4() == nil {
		labels[1] = "ipv6" // nolint: goconst
	}

	s.connections.WithLabelValues(labels[:]...).Add(increment)
}

func (s *statsPrometheus) TelegramConnected(dc conntypes.DC, addr *net.TCPAddr) {
	s.changeTelegramConnections(dc, addr, 1.0)
}

func (s *statsPrometheus) TelegramDisconnected(dc conntypes.DC, addr *net.TCPAddr) {
	s.changeTelegramConnections(dc, addr, -1.0)
}

func (s *statsPrometheus) changeTelegramConnections(dc conntypes.DC, addr *net.TCPAddr, increment float64) {
	labels := [...]string{
		strconv.Itoa(int(dc)),
		"ipv4",
	}

	if addr.IP.To4() == nil {
		labels[1] = "ipv6"
	}

	s.telegramConnections.WithLabelValues(labels[:]...).Add(increment)
}

func (s *statsPrometheus) Crash() {
	s.crashes.Inc()
}

func (s *statsPrometheus) ReplayDetected() {
	s.replayAttacks.Inc()
}

func (s *statsPrometheus) AuthenticationFailed() {
	s.authenticationFailed.Inc()
}

func (s *statsPrometheus) CloakedRequest() {
	s.cloakedRequests.Inc()
}

func newStatsPrometheus(mux *http.ServeMux) Interface {
	registry := prometheus.NewPedanticRegistry()

	instance := &statsPrometheus{
		connections: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: config.C.StatsNamespace,
			Name:      "connections",
			Help:      "Current number of client connections to the proxy.",
		}, []string{"type", "protocol"}),
		telegramConnections: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: config.C.StatsNamespace,
			Name:      "telegram_connections",
			Help:      "Current number of telegram connections established by this proxy.",
		}, []string{"dc", "protocol"}),
		traffic: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: config.C.StatsNamespace,
			Name:      "traffic",
			Help:      "Traffic passed through the proxy in bytes.",
		}, []string{"direction"}),
		crashes: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: config.C.StatsNamespace,
			Name:      "crashes",
			Help:      "How many crashes happened.",
		}),
		replayAttacks: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: config.C.StatsNamespace,
			Name:      "replay_attacks",
			Help:      "How many replay attacks were prevented.",
		}),
		authenticationFailed: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: config.C.StatsNamespace,
			Name:      "authentication_failed",
			Help:      "How many authentication failed events we've seen.",
		}),
		cloakedRequests: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: config.C.StatsNamespace,
			Name:      "cloaked_requests",
			Help:      "How many requests were proxified during cloaking.",
		}),
	}

	registry.MustRegister(instance.connections)
	registry.MustRegister(instance.telegramConnections)
	registry.MustRegister(instance.traffic)
	registry.MustRegister(instance.crashes)
	registry.MustRegister(instance.replayAttacks)
	registry.MustRegister(instance.authenticationFailed)
	registry.MustRegister(instance.cloakedRequests)

	handler := promhttp.HandlerFor(registry, promhttp.HandlerOpts{})
	mux.Handle("/", handler)

	return instance
}
