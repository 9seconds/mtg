package stats

import (
	"context"
	"net"
	"net/http"

	"github.com/9seconds/mtg/v2/events"
	"github.com/9seconds/mtg/v2/mtglib"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type prometheusProcessor struct {
	streams map[string]*streamInfo
	factory *PrometheusFactory
}

func (p prometheusProcessor) EventStart(evt mtglib.EventStart) {
	info := acquireStreamInfo()
	info.SetStartTime(evt.CreatedAt)
	info.SetClientIP(evt.RemoteIP)
	p.streams[evt.StreamID()] = info

	p.factory.metricClientConnections.
		WithLabelValues(info.V(TagIPFamily)).
		Inc()
}

func (p prometheusProcessor) EventConnectedToDC(evt mtglib.EventConnectedToDC) {
	info, ok := p.streams[evt.StreamID()]
	if !ok {
		return
	}

	info.SetTelegramIP(evt.RemoteIP)
	info.SetDC(evt.DC)

	p.factory.metricTelegramConnections.
		WithLabelValues(info.V(TagTelegramIP), info.V(TagDC)).
		Inc()
}

func (p prometheusProcessor) EventTraffic(evt mtglib.EventTraffic) {
	info, ok := p.streams[evt.StreamID()]
	if !ok {
		return
	}

	p.factory.metricTelegramTraffic.
		WithLabelValues(info.V(TagTelegramIP), info.V(TagDC), getDirection(evt.IsRead)).
		Add(float64(evt.Traffic))
}

func (p prometheusProcessor) EventFinish(evt mtglib.EventFinish) {
	info, ok := p.streams[evt.StreamID()]
	if !ok {
		return
	}

	defer func() {
		delete(p.streams, evt.StreamID())
		releaseStreamInfo(info)
	}()

	p.factory.metricClientConnections.
		WithLabelValues(info.V(TagIPFamily)).
		Dec()

	if info.V(TagTelegramIP) != "" {
		p.factory.metricTelegramConnections.
			WithLabelValues(info.V(TagTelegramIP), info.V(TagDC)).
			Dec()
	}
}

func (p prometheusProcessor) EventConcurrencyLimited(_ mtglib.EventConcurrencyLimited) {
	p.factory.metricConcurrencyLimited.Inc()
}

func (p prometheusProcessor) EventIPBlocklisted(evt mtglib.EventIPBlocklisted) {
	p.factory.metricIPBlocklisted.Inc()
}

func (p prometheusProcessor) Shutdown() {
	p.streams = make(map[string]*streamInfo)
}

type PrometheusFactory struct {
	httpServer *http.Server

	metricClientConnections           *prometheus.GaugeVec
	metricTelegramConnections         *prometheus.GaugeVec
	metricDomainDisguisingConnections *prometheus.GaugeVec

	metricTelegramTraffic         *prometheus.CounterVec
	metricDomainDisguisingTraffic *prometheus.CounterVec

	metricDomainDisguising   prometheus.Counter
	metricConcurrencyLimited prometheus.Counter
	metricIPBlocklisted      prometheus.Counter
	metricReplayAttacks      prometheus.Counter
}

func (p *PrometheusFactory) Make() events.Observer {
	return prometheusProcessor{
		streams: make(map[string]*streamInfo),
		factory: p,
	}
}

func (p *PrometheusFactory) Serve(listener net.Listener) error {
	return p.httpServer.Serve(listener)
}

func (p *PrometheusFactory) Close() error {
	return p.httpServer.Shutdown(context.Background())
}

func NewPrometheus(metricPrefix, httpPath string) *PrometheusFactory { // nolint: funlen
	registry := prometheus.NewPedanticRegistry()
	httpHandler := promhttp.HandlerFor(registry, promhttp.HandlerOpts{
		EnableOpenMetrics: true,
	})
	mux := http.NewServeMux()

	mux.Handle(httpPath, httpHandler)

	factory := &PrometheusFactory{
		httpServer: &http.Server{
			Handler: mux,
		},

		metricClientConnections: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: metricPrefix,
			Name:      MetricClientConnections,
			Help:      "A number of actively processing client connections.",
		}, []string{TagIPFamily}),
		metricTelegramConnections: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: metricPrefix,
			Name:      MetricTelegramConnections,
			Help:      "A number of connections to Telegram servers.",
		}, []string{TagTelegramIP, TagDC}),
		metricDomainDisguisingConnections: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: metricPrefix,
			Name:      MetricDomainDisguisingConnections,
			Help:      "A number of connections which talk with disguising domain.",
		}, []string{TagIPFamily}),

		metricTelegramTraffic: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: metricPrefix,
			Name:      MetricTelegramTraffic,
			Help:      "Traffic which is generated talking with Telegram servers.",
		}, []string{TagTelegramIP, TagDC, TagDirection}),
		metricDomainDisguisingTraffic: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: metricPrefix,
			Name:      MetricDomainDisguisingTraffic,
			Help:      "Traffic which is generated talking with disguising domain.",
		}, []string{TagDirection}),

		metricDomainDisguising: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: metricPrefix,
			Name:      MetricDomainDisguising,
			Help:      "A number of routings to disguising domain.",
		}),
		metricConcurrencyLimited: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: metricPrefix,
			Name:      MetricConcurrencyLimited,
			Help:      "A number of sessions that were rejected by concurrency limiter.",
		}),
		metricIPBlocklisted: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: metricPrefix,
			Name:      MetricIPBlocklisted,
			Help:      "A number of rejected sessions due to ip blocklisting.",
		}),
		metricReplayAttacks: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: metricPrefix,
			Name:      MetricReplayAttacks,
			Help:      "A number of detected replay attacks.",
		}),
	}

	registry.MustRegister(factory.metricClientConnections)
	registry.MustRegister(factory.metricTelegramConnections)
	registry.MustRegister(factory.metricDomainDisguisingConnections)

	registry.MustRegister(factory.metricTelegramTraffic)
	registry.MustRegister(factory.metricDomainDisguisingTraffic)

	registry.MustRegister(factory.metricDomainDisguising)
	registry.MustRegister(factory.metricConcurrencyLimited)
	registry.MustRegister(factory.metricIPBlocklisted)
	registry.MustRegister(factory.metricReplayAttacks)

	return factory
}
