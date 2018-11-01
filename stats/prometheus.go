package stats

import (
	"net/http"
	"time"

	"github.com/juju/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/9seconds/mtg/config"
)

const prometheusPollTime = time.Second

type prometheusExporter struct {
	registry prometheus.Gatherer

	connections *prometheus.GaugeVec
	traffic     *prometheus.GaugeVec
	speed       *prometheus.GaugeVec
	crashes     prometheus.Gauge
}

func (p *prometheusExporter) run() {
	for range time.Tick(prometheusPollTime) {
		instance := GetStats()

		p.connections.WithLabelValues("abridged", "v4").Set(float64(instance.Connections.Abridged.IPv4))
		p.connections.WithLabelValues("abridged", "v6").Set(float64(instance.Connections.Abridged.IPv6))
		p.connections.WithLabelValues("intermediate", "v4").Set(float64(instance.Connections.Intermediate.IPv4))
		p.connections.WithLabelValues("intermediate", "v6").Set(float64(instance.Connections.Intermediate.IPv6))
		p.connections.WithLabelValues("secure", "v4").Set(float64(instance.Connections.Secure.IPv4))
		p.connections.WithLabelValues("secure", "v6").Set(float64(instance.Connections.Secure.IPv6))
		p.traffic.WithLabelValues("ingress").Set(float64(instance.Traffic.ingress))
		p.traffic.WithLabelValues("egress").Set(float64(instance.Traffic.egress))
		p.speed.WithLabelValues("ingress").Set(float64(instance.Speed.ingress))
		p.speed.WithLabelValues("egress").Set(float64(instance.Speed.egress))
		p.crashes.Set(float64(instance.Crashes))
	}
}

func (p *prometheusExporter) getHTTPHandler() http.Handler {
	return promhttp.HandlerFor(p.registry, promhttp.HandlerOpts{})
}

func newPrometheus(conf *config.Config) (*prometheusExporter, error) {
	registry := prometheus.NewRegistry()

	connections := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: conf.Prometheus.Prefix,
		Name:      "connections",
		Help:      "Current number of connections to the proxy.",
	}, []string{"type", "protocol"})
	traffic := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: conf.Prometheus.Prefix,
		Name:      "traffic",
		Help:      "Traffic passed through the proxy in bytes.",
	}, []string{"direction"})
	speed := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: conf.Prometheus.Prefix,
		Name:      "speed",
		Help:      "Current throughput in bytes per second.",
	}, []string{"direction"})
	crashes := prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: conf.Prometheus.Prefix,
		Name:      "crashes",
		Help:      "How many crashes happened.",
	})

	if err := registry.Register(connections); err != nil {
		return nil, errors.Annotate(err, "Cannot register connections collector")
	}
	if err := registry.Register(traffic); err != nil {
		return nil, errors.Annotate(err, "cannot register traffic collector")
	}
	if err := registry.Register(speed); err != nil {
		return nil, errors.Annotate(err, "cannot register speed collector")
	}
	if err := registry.Register(crashes); err != nil {
		return nil, errors.Annotate(err, "cannot register crashes collector")
	}

	return &prometheusExporter{
		registry:    registry,
		connections: connections,
		traffic:     traffic,
		speed:       speed,
		crashes:     crashes,
	}, nil
}
