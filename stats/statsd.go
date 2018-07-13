package stats

import (
	"time"

	"github.com/juju/errors"
	statsd "gopkg.in/alexcesaro/statsd.v2"

	"github.com/9seconds/mtg/config"
)

const (
	statsdConnectionsAbridgedV4 = "connections.abridged.ipv4"
	statsdConnectionsAbridgedV6 = "connections.abridged.ipv6"

	statsdConnectionsIntermediateV4 = "connections.intermediate.ipv4"
	statsdConnectionsIntermediateV6 = "connections.intermediate.ipv6"

	statsdConnectionsSecureV4 = "connections.secure.ipv4"
	statsdConnectionsSecureV6 = "connections.secure.ipv6"

	statsdTrafficIngress = "traffic.ingress"
	statsdTrafficEgress  = "traffic.egress"

	statsdSpeedIngress = "speed.ingress"
	statsdSpeedEgress  = "speed.egress"

	statsdCrashes = "crashes"
)

const statsdPollTime = time.Second

type statsdExporter struct {
	client *statsd.Client
}

func (s *statsdExporter) run() {
	for range time.Tick(statsdPollTime) {
		instance.mutex.Lock()

		s.client.Gauge(statsdConnectionsAbridgedV4, instance.Connections.Abridged.IPv4)
		s.client.Gauge(statsdConnectionsAbridgedV6, instance.Connections.Abridged.IPv6)
		s.client.Gauge(statsdConnectionsIntermediateV4, instance.Connections.Intermediate.IPv4)
		s.client.Gauge(statsdConnectionsIntermediateV6, instance.Connections.Intermediate.IPv6)
		s.client.Gauge(statsdConnectionsSecureV4, instance.Connections.Secure.IPv4)
		s.client.Gauge(statsdConnectionsSecureV6, instance.Connections.Secure.IPv6)
		s.client.Gauge(statsdTrafficIngress, uint64(instance.Traffic.Ingress))
		s.client.Gauge(statsdTrafficEgress, uint64(instance.Traffic.Egress))
		s.client.Gauge(statsdSpeedIngress, uint64(instance.Speed.Ingress))
		s.client.Gauge(statsdSpeedEgress, uint64(instance.Speed.Egress))
		s.client.Gauge(statsdCrashes, instance.Crashes)

		instance.mutex.Unlock()
	}
}

func newStatsd(conf *config.Config) (*statsdExporter, error) {
	options := []statsd.Option{
		statsd.Network(conf.StatsD.Addr.Network()),
		statsd.Address(conf.StatsD.Addr.String()),
		statsd.Prefix(conf.StatsD.Prefix),
	}

	if conf.StatsD.TagsFormat > 0 {
		options = append(options, statsd.TagsFormat(conf.StatsD.TagsFormat))
		tags := make([]string, len(conf.StatsD.Tags)*2)
		for k, v := range conf.StatsD.Tags {
			tags = append(tags, k, v)
		}
		options = append(options, statsd.Tags(tags...))
	}

	client, err := statsd.New(options...)
	if err != nil {
		return nil, errors.Annotate(err, "Cannot create statsd client")
	}

	return &statsdExporter{client: client}, nil
}
