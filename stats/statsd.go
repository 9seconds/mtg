package stats

import (
	"fmt"
	"strings"
	"time"

	"github.com/9seconds/mtg/v2/events"
	"github.com/9seconds/mtg/v2/logger"
	"github.com/9seconds/mtg/v2/mtglib"
	statsd "github.com/smira/go-statsd"
)

type statsdProcessor struct {
	streams map[string]*streamInfo
	client  *statsd.Client
}

func (s statsdProcessor) EventStart(evt mtglib.EventStart) {
	info := acquireStreamInfo()
	info.SetStartTime(evt.CreatedAt)
	info.SetClientIP(evt.RemoteIP)
	s.streams[evt.StreamID()] = info

	s.client.GaugeDelta(MetricClientConnections,
		1,
		info.TV(TagIPFamily))
}

func (s statsdProcessor) EventConnectedToDC(evt mtglib.EventConnectedToDC) {
	info, ok := s.streams[evt.StreamID()]
	if !ok {
		return
	}

	info.SetTelegramIP(evt.RemoteIP)
	info.SetDC(evt.DC)

	s.client.GaugeDelta(MetricTelegramConnections,
		1,
		info.TV(TagTelegramIP),
		info.TV(TagDC))
}

func (s statsdProcessor) EventTraffic(evt mtglib.EventTraffic) {
	info, ok := s.streams[evt.StreamID()]
	if !ok {
		return
	}

	s.client.Incr(MetricTelegramTraffic,
		int64(evt.Traffic),
		info.TV(TagTelegramIP),
		info.TV(TagDC),
		statsd.StringTag(TagDirection, getDirection(evt.IsRead)))
}

func (s statsdProcessor) EventFinish(evt mtglib.EventFinish) {
	info, ok := s.streams[evt.StreamID()]
	if !ok {
		return
	}

	defer func() {
		delete(s.streams, evt.StreamID())
		releaseStreamInfo(info)
	}()

	s.client.GaugeDelta(MetricClientConnections,
		-1,
		info.TV(TagIPFamily))

	if info.V(TagTelegramIP) != "" {
		s.client.GaugeDelta(MetricTelegramConnections,
			-1,
			info.TV(TagTelegramIP),
			info.TV(TagDC))
	}
}

func (s statsdProcessor) EventConcurrencyLimited(_ mtglib.EventConcurrencyLimited) {
	s.client.Incr(MetricConcurrencyLimited, 1)
}

func (s statsdProcessor) EventIPBlocklisted(evt mtglib.EventIPBlocklisted) {
	s.client.Incr(MetricIPBlocklisted, 1)
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

func NewStatsd(address string, log logger.StdLikeLogger,
	metricPrefix, tagFormat string) (StatsdFactory, error) {
	options := []statsd.Option{
		statsd.MetricPrefix(metricPrefix),
		statsd.Logger(log),
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
