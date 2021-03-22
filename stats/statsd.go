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
	sInfo := &streamInfo{
		createdAt: evt.CreatedAt,
		clientIP:  evt.RemoteIP,
	}
	s.streams[evt.StreamID()] = sInfo

	s.client.GaugeDelta(MetricClientConnections,
		1,
		statsd.StringTag(TagIPType, sInfo.GetClientIPType()))
}

func (s statsdProcessor) EventConnectedToDC(evt mtglib.EventConnectedToDC) {
	sInfo, ok := s.streams[evt.StreamID()]
	if !ok {
		return
	}

	sInfo.remoteIP = evt.RemoteIP
	sInfo.dc = evt.DC

	s.client.GaugeDelta(MetricTelegramConnections,
		1,
		statsd.StringTag(TagIPType, sInfo.GetRemoteIPType()),
		statsd.StringTag(TagTelegramIP, sInfo.remoteIP.String()),
		statsd.IntTag(TagDC, sInfo.dc))
}

func (s statsdProcessor) EventTraffic(evt mtglib.EventTraffic) {
	sInfo, ok := s.streams[evt.StreamID()]
	if !ok {
		return
	}

	tags := []statsd.Tag{
		statsd.StringTag(TagIPType, sInfo.GetRemoteIPType()),
		statsd.StringTag(TagTelegramIP, sInfo.remoteIP.String()),
		statsd.IntTag(TagDC, sInfo.dc),
	}

	if evt.IsRead {
		tags = append(tags, statsd.StringTag(TagDirection, TagDirectionClient))
		s.client.Incr(MetricTraffic, int64(evt.Traffic), tags...)
	} else {
		tags = append(tags, statsd.StringTag(TagDirection, TagDirectionTelegram))
		s.client.Incr(MetricTraffic, int64(evt.Traffic), tags...)
	}
}

func (s statsdProcessor) EventFinish(evt mtglib.EventFinish) {
	sInfo, ok := s.streams[evt.StreamID()]
	if !ok {
		return
	}

	defer delete(s.streams, evt.StreamID())

	s.client.GaugeDelta(MetricClientConnections,
		-1,
		statsd.StringTag(TagIPType, sInfo.GetClientIPType()))
	s.client.PrecisionTiming(MetricSessionDuration,
		evt.CreatedAt.Sub(sInfo.createdAt))

	if sInfo.remoteIP != nil {
		s.client.GaugeDelta(MetricTelegramConnections,
			-1,
			statsd.StringTag(TagIPType, sInfo.GetRemoteIPType()),
			statsd.StringTag(TagTelegramIP, sInfo.remoteIP.String()),
			statsd.IntTag(TagDC, sInfo.dc))
	}
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
