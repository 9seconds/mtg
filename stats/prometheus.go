package stats

import (
	"context"
	"net"
	"net/http"
	"strconv"
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

	p.factory.metricClientConnections.WithLabelValues(sInfo.GetClientIPType()).Inc()
}

func (p prometheusProcessor) EventConnectedToDC(evt mtglib.EventConnectedToDC) {
	sInfo, ok := p.streams[evt.StreamID()]
	if !ok {
		return
	}

	sInfo.remoteIP = evt.RemoteIP
	sInfo.dc = evt.DC

	p.factory.metricTelegramConnections.WithLabelValues(
		sInfo.GetRemoteIPType(),
		sInfo.remoteIP.String(),
		strconv.Itoa(sInfo.dc)).Inc()
}

func (p prometheusProcessor) EventTraffic(evt mtglib.EventTraffic) {
	sInfo, ok := p.streams[evt.StreamID()]
	if !ok {
		return
	}

	labels := []string{
		sInfo.GetRemoteIPType(),
		sInfo.remoteIP.String(),
		strconv.Itoa(sInfo.dc),
	}

	if evt.IsRead {
		sInfo.bytesRecvFromTelegram += evt.Traffic

		labels = append(labels, TagDirectionClient)
	} else {
		sInfo.bytesSentToTelegram += evt.Traffic

		labels = append(labels, TagDirectionTelegram)
	}

	p.factory.metricTraffic.WithLabelValues(labels...).Add(float64(evt.Traffic))
}

func (p prometheusProcessor) EventFinish(evt mtglib.EventFinish) {
	sInfo, ok := p.streams[evt.StreamID()]
	if !ok {
		return
	}

	defer delete(p.streams, evt.StreamID())

	duration := evt.CreatedAt.Sub(sInfo.createdAt)

	p.factory.metricClientConnections.WithLabelValues(sInfo.GetClientIPType()).Dec()
	p.factory.metricSessionDuration.Observe(float64(duration) / float64(time.Second))

	if sInfo.remoteIP == nil {
		return
	}

	labels := []string{
		sInfo.GetRemoteIPType(),
		sInfo.remoteIP.String(),
		strconv.Itoa(sInfo.dc),
	}

	p.factory.metricTelegramConnections.WithLabelValues(labels...).Dec()

	labels = append(labels, TagDirectionClient)
	p.factory.metricSessionTraffic.
		WithLabelValues(labels...).
		Observe(float64(sInfo.bytesRecvFromTelegram))

	labels[3] = TagDirectionTelegram
	p.factory.metricSessionTraffic.
		WithLabelValues(labels...).
		Observe(float64(sInfo.bytesSentToTelegram))
}

func (p prometheusProcessor) EventConcurrencyLimited(_ mtglib.EventConcurrencyLimited) {
	p.factory.metricConcurrencyLimited.Inc()
}

func (p prometheusProcessor) EventIPBlocklisted(evt mtglib.EventIPBlocklisted) {
	if evt.RemoteIP.To4() == nil {
		p.factory.metricIPBlocklisted.WithLabelValues(TagIPTypeIPv6).Inc()
	} else {
		p.factory.metricIPBlocklisted.WithLabelValues(TagIPTypeIPv4).Inc()
	}
}

func (p prometheusProcessor) Shutdown() {
	p.streams = make(map[string]*streamInfo)
}

type PrometheusFactory struct {
	httpServer *http.Server

	metricClientConnections   *prometheus.GaugeVec
	metricTelegramConnections *prometheus.GaugeVec
	metricTraffic             *prometheus.CounterVec
	metricIPBlocklisted       *prometheus.CounterVec
	metricSessionTraffic      *prometheus.HistogramVec
	metricConcurrencyLimited  prometheus.Counter
	metricSessionDuration     prometheus.Histogram
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
			Help:      "A number of connections under active processing.",
		}, []string{TagIPType}),
		metricTelegramConnections: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: metricPrefix,
			Name:      MetricTelegramConnections,
			Help:      "A number of connections to Telegram servers.",
		}, []string{TagIPType, TagTelegramIP, TagDC}),
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
		metricSessionTraffic: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: metricPrefix,
			Name:      MetricSessionTraffic,
			Help:      "A traffic size which flew via proxy within a single session.",
			Buckets: []float64{ // per 1mb
				1 * 1024 * 1024,
				2 * 1024 * 1024,
				3 * 1024 * 1024,
				4 * 1024 * 1024,
				5 * 1024 * 1024,
				6 * 1024 * 1024,
				7 * 1024 * 1024,
				8 * 1024 * 1024,
				9 * 1024 * 1024,
			},
		}, []string{TagIPType, TagTelegramIP, TagDC, TagDirection}),
		metricTraffic: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: metricPrefix,
			Name:      MetricTraffic,
			Help:      "Traffic which is sent through this proxy.",
		}, []string{TagIPType, TagTelegramIP, TagDC, TagDirection}),
		metricConcurrencyLimited: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: metricPrefix,
			Name:      MetricConcurrencyLimited,
			Help:      "A number of sessions that were rejected by concurrency limiter.",
		}),
		metricIPBlocklisted: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: metricPrefix,
			Name:      MetricIPBlocklisted,
			Help:      "A number of rejected sessions due to ip blocklisting",
		}, []string{TagIPType}),
	}

	registry.MustRegister(factory.metricClientConnections)
	registry.MustRegister(factory.metricTelegramConnections)
	registry.MustRegister(factory.metricTraffic)
	registry.MustRegister(factory.metricSessionTraffic)
	registry.MustRegister(factory.metricSessionDuration)
	registry.MustRegister(factory.metricConcurrencyLimited)
	registry.MustRegister(factory.metricIPBlocklisted)

	return factory
}
