package newstats

import (
	"net"
	"net/http"

	"github.com/juju/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/9seconds/mtg/newconfig"
	"github.com/9seconds/mtg/newprotocol"
)

type statsPrometheus struct {
	connections *prometheus.GaugeVec
	traffic     *prometheus.GaugeVec
	crashes     prometheus.Gauge
	antiReplays prometheus.Gauge
}

func (s *statsPrometheus) IngressTraffic(traffic int) {
	s.traffic.WithLabelValues("ingress").Add(float64(traffic))
}

func (s *statsPrometheus) EgressTraffic(traffic int) {
	s.traffic.WithLabelValues("egress").Add(float64(traffic))
}

func (s *statsPrometheus) ClientConnected(connectionType newprotocol.ConnectionType, addr *net.TCPAddr) {
	s.changeConnections(connectionType, addr, 1.0)
}

func (s *statsPrometheus) ClientDisconnected(connectionType newprotocol.ConnectionType, addr *net.TCPAddr) {
	s.changeConnections(connectionType, addr, -1.0)
}

func (s *statsPrometheus) changeConnections(connectionType newprotocol.ConnectionType,
	addr *net.TCPAddr,
	increment float64) {
	var labels [2]string

	switch connectionType {
	case newprotocol.ConnectionTypeAbridged:
		labels[0] = "abridged"
	case newprotocol.ConnectionTypeSecure:
		labels[0] = "secured"
	default:
		labels[0] = "intermediate"
	}

	labels[1] = "ipv4"
	if addr.IP.To4() == nil {
		labels[1] = "ipv6"
	}

	s.connections.WithLabelValues(labels[:]...).Add(increment)
}

func (s *statsPrometheus) Crash() {
	s.crashes.Inc()
}

func (s *statsPrometheus) AntiReplayDetected() {
	s.antiReplays.Inc()
}

func newStatsPrometheus(mux *http.ServeMux) (Stats, error) {
	registry := prometheus.NewRegistry()
	instance := &statsPrometheus{
		connections: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: newconfig.C.PrometheusStats.Prefix,
			Name:      "connections",
			Help:      "Current number of connections to the proxy.",
		}, []string{"type", "protocol"}),
		traffic: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: newconfig.C.PrometheusStats.Prefix,
			Name:      "traffic",
			Help:      "Traffic passed through the proxy in bytes.",
		}, []string{"direction"}),
		crashes: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: newconfig.C.PrometheusStats.Prefix,
			Name:      "crashes",
			Help:      "How many crashes happened.",
		}),
		antiReplays: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: newconfig.C.PrometheusStats.Prefix,
			Name:      "anti_replays",
			Help:      "How many anti replay attacks were prevented.",
		}),
	}

	if err := registry.Register(instance.connections); err != nil {
		return nil, errors.Annotate(err, "Cannot register metrics for connections")
	}
	if err := registry.Register(instance.traffic); err != nil {
		return nil, errors.Annotate(err, "Cannot register metrics for traffic")
	}
	if err := registry.Register(instance.crashes); err != nil {
		return nil, errors.Annotate(err, "Cannot register metrics for crashes")
	}
	if err := registry.Register(instance.antiReplays); err != nil {
		return nil, errors.Annotate(err, "Cannot register metrics for anti replays")
	}

	handler := promhttp.HandlerFor(registry, promhttp.HandlerOpts{})
	mux.Handle("/prometheus", handler)

	return instance, nil
}
