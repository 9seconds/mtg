package stats

import (
	"fmt"
	"strings"
	"time"

	"github.com/9seconds/mtg/v2/events"
	"github.com/9seconds/mtg/v2/mtglib"
	statsd "github.com/smira/go-statsd"
)

type statsdFakeLogger struct{}

func (s statsdFakeLogger) Printf(msg string, args ...interface{}) {}

type statsdProcessor struct {
	streams map[string]*streamInfo
	client  *statsd.Client
}

func (s statsdProcessor) EventStart(evt mtglib.EventStart) {
	sInfo := &streamInfo{
		createdAt: evt.CreatedAt,
		clientIP:  evt.RemoteIP,
	}
	s.streams[evt.StreamID()] = sInfo
	ipTypeTag := statsd.StringTag(TagIPType, sInfo.IPType())

	s.client.GaugeDelta(MetricActiveConnection, 1, ipTypeTag)
}

func (s statsdProcessor) EventFinish(evt mtglib.EventFinish) {
	sInfo, ok := s.streams[evt.StreamID()]
	if !ok {
		return
	}

	defer delete(s.streams, evt.StreamID())

	duration := evt.CreatedAt.Sub(sInfo.createdAt)
	ipTypeTag := statsd.StringTag(TagIPType, sInfo.IPType())

	s.client.GaugeDelta(MetricActiveConnection, -1, ipTypeTag)
	s.client.PrecisionTiming(MetricSessionDuration, duration)
}

func (s statsdProcessor) EventConcurrencyLimited(_ mtglib.EventConcurrencyLimited) {
	s.client.Incr(MetricConcurrencyLimited, 1)
}

func (s statsdProcessor) EventIPBlocklisted(evt mtglib.EventIPBlocklisted) {
	var tag statsd.Tag

	if evt.RemoteIP.To4() == nil {
		tag = statsd.StringTag(TagIPType, TagIPTypeIPv6)
	} else {
		tag = statsd.StringTag(TagIPType, TagIPTypeIPv4)
	}

	s.client.Incr(MetricIPBlocklisted, 1, tag)
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
		streams: make(map[string]*streamInfo),
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
