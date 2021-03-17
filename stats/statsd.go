package stats

import (
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/9seconds/mtg/v2/events"
	"github.com/9seconds/mtg/v2/mtglib"
	statsd "github.com/smira/go-statsd"
)

type statsdFakeLogger struct{}

func (s statsdFakeLogger) Printf(msg string, args ...interface{}) {}

type statsdStreamInfo struct {
	createdAt time.Time
	clientIP  net.IP
}

func (s *statsdStreamInfo) ClientIPTag() statsd.Tag {
	if s.clientIP.To4() == nil {
		return statsd.StringTag(TagIPType, TagIPTypeIPv6)
	} else {
		return statsd.StringTag(TagIPType, TagIPTypeIPv4)
	}
}

type statsdProcessor struct {
	streams map[string]*statsdStreamInfo
	client  *statsd.Client
}

func (s statsdProcessor) EventStart(evt mtglib.EventStart) {
	clientInfo := &statsdStreamInfo{
		createdAt: evt.CreatedAt,
		clientIP:  evt.RemoteIP,
	}
	s.streams[evt.StreamID()] = clientInfo

	s.client.GaugeDelta(MetricActiveConnection, 1, clientInfo.ClientIPTag())
}

func (s statsdProcessor) EventFinish(evt mtglib.EventFinish) {
	clientInfo, ok := s.streams[evt.StreamID()]
	if !ok {
		return
	}

	defer delete(s.streams, evt.StreamID())

	duration := evt.CreatedAt.Sub(clientInfo.createdAt)

	s.client.GaugeDelta(MetricActiveConnection, -1, clientInfo.ClientIPTag())
	s.client.PrecisionTiming(MetricSessionDuration, duration)
}

func (s statsdProcessor) EventConcurrencyLimited(_ mtglib.EventConcurrencyLimited) {
	s.client.Incr(MetricConcurrencyLimited, 1)
}

func (s statsdProcessor) Shutdown() {
	now := time.Now()
	events := make([]mtglib.EventFinish, 0, len(s.streams))

	for k := range s.streams {
		events = append(events, mtglib.EventFinish{
			CreatedAt: now,
			ConnID:    k,
		})
	}

	for i := range events {
		s.EventFinish(events[i])
	}
}

type StatsdFactory struct {
	client *statsd.Client
}

func (s StatsdFactory) Close() error {
	return s.client.Close()
}

func (s StatsdFactory) Make() events.Observer {
	return statsdProcessor{
		client:  s.client,
		streams: make(map[string]*statsdStreamInfo),
	}
}

func NewStatsd(address, metricPrefix, tagFormat string) (StatsdFactory, error) {
	options := []statsd.Option{
		statsd.MetricPrefix(metricPrefix),
		statsd.Logger(statsdFakeLogger{}),
	}

	switch strings.ToLower(tagFormat) {
	case "datadog":
		options = append(options, statsd.TagStyle(statsd.TagFormatDatadog))
	case "influxdb":
		options = append(options, statsd.TagStyle(statsd.TagFormatInfluxDB))
	case "graphite":
		options = append(options, statsd.TagStyle(statsd.TagFormatGraphite))
	default:
		return StatsdFactory{}, fmt.Errorf("unknown tag format %s", tagFormat)
	}

	return StatsdFactory{
		client: statsd.NewClient(address, options...),
	}, nil
}
