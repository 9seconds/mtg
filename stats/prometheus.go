package stats

import (
	"context"
	"net"
	"net/http"
	"time"

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
	sInfo := &streamInfo{
		createdAt: evt.CreatedAt,
		clientIP:  evt.RemoteIP,
	}
	p.streams[evt.StreamID()] = sInfo

	p.factory.metricActiveConnections.WithLabelValues(sInfo.IPType()).Inc()
}

func (p prometheusProcessor) EventFinish(evt mtglib.EventFinish) {
	sInfo, ok := p.streams[evt.StreamID()]
	if !ok {
		return
	}

	defer delete(p.streams, evt.StreamID())

	duration := evt.CreatedAt.Sub(sInfo.createdAt)

	p.factory.metricActiveConnections.WithLabelValues(sInfo.IPType()).Dec()
	p.factory.metricSessionDuration.Observe(float64(duration) / float64(time.Second))
}

func (p prometheusProcessor) EventConcurrencyLimited(evt mtglib.EventConcurrencyLimited) {
	p.factory.metricConcurrencyLimited.Inc()
}

func (p prometheusProcessor) Shutdown() {
	p.streams = make(map[string]*streamInfo)
}

type PrometheusFactory struct {
	httpServer *http.Server

	metricActiveConnections  *prometheus.GaugeVec
	metricConcurrencyLimited prometheus.Counter
	metricSessionDuration    prometheus.Histogram
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

func NewPrometheus(metricPrefix, httpPath string) *PrometheusFactory {
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

		metricActiveConnections: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: metricPrefix,
			Name:      MetricActiveConnection,
			Help:      "A number of connections under active processing.",
		}, []string{TagIPType}),
		metricSessionDuration: prometheus.NewHistogram(prometheus.HistogramOpts{
			Namespace: metricPrefix,
			Name:      MetricSessionDuration,
			Help:      "Session duration.",
			Buckets: []float64{ // per 30 seconds
				30,
				60,
				90,
				120,
				150,
				180,
				210,
				240,
				270,
				300,
			},
		}),
		metricConcurrencyLimited: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: metricPrefix,
			Name:      MetricConcurrencyLimited,
			Help:      "A number of sessions that were rejected by concurrency limiter.",
		}),
	}

	registry.MustRegister(factory.metricActiveConnections)
	registry.MustRegister(factory.metricSessionDuration)
	registry.MustRegister(factory.metricConcurrencyLimited)

	return factory
}
