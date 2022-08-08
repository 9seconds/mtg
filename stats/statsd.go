package stats

import (
	"fmt"
	"strconv"
	"strings"

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

	if evt.RemoteIP.To4() != nil {
		info.tags[TagIPFamily] = TagIPFamilyIPv4
	} else {
		info.tags[TagIPFamily] = TagIPFamilyIPv6
	}

	s.streams[evt.StreamID()] = info

	s.client.GaugeDelta(MetricClientConnections,
		1,
		info.T(TagIPFamily))
}

func (s statsdProcessor) EventConnectedToDC(evt mtglib.EventConnectedToDC) {
	info, ok := s.streams[evt.StreamID()]
	if !ok {
		return
	}

	info.tags[TagTelegramIP] = evt.RemoteIP.String()
	info.tags[TagDC] = strconv.Itoa(evt.DC)

	s.client.GaugeDelta(MetricTelegramConnections,
		1,
		info.T(TagTelegramIP),
		info.T(TagDC))
}

func (s statsdProcessor) EventDomainFronting(evt mtglib.EventDomainFronting) {
	info, ok := s.streams[evt.StreamID()]
	if !ok {
		return
	}

	info.isDomainFronted = true

	s.client.Incr(MetricDomainFronting, 1)
	s.client.GaugeDelta(MetricDomainFrontingConnections,
		1,
		info.T(TagIPFamily))
}

func (s statsdProcessor) EventTraffic(evt mtglib.EventTraffic) {
	info, ok := s.streams[evt.StreamID()]
	if !ok {
		return
	}

	directionTag := statsd.StringTag(TagDirection, getDirection(evt.IsRead))

	if info.isDomainFronted {
		s.client.Incr(MetricDomainFrontingTraffic,
			int64(evt.Traffic),
			directionTag)
	} else {
		s.client.Incr(MetricTelegramTraffic,
			int64(evt.Traffic),
			info.T(TagTelegramIP),
			info.T(TagDC),
			directionTag)
	}
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
		info.T(TagIPFamily))

	if info.isDomainFronted {
		s.client.GaugeDelta(MetricDomainFrontingConnections,
			-1,
			info.T(TagIPFamily))
	} else if _, ok := info.tags[TagTelegramIP]; ok {
		s.client.GaugeDelta(MetricTelegramConnections,
			-1,
			info.T(TagTelegramIP),
			info.T(TagDC))
	}
}

func (s statsdProcessor) EventConcurrencyLimited(_ mtglib.EventConcurrencyLimited) {
	s.client.Incr(MetricConcurrencyLimited, 1)
}

func (s statsdProcessor) EventIPBlocklisted(evt mtglib.EventIPBlocklisted) {
	tag := TagIPListBlock
	if !evt.IsBlockList {
		tag = TagIPListAllow
	}

	s.client.Incr(MetricIPBlocklisted, 1, statsd.StringTag(TagIPList, tag))
}

func (s statsdProcessor) EventReplayAttack(_ mtglib.EventReplayAttack) {
	s.client.Incr(MetricReplayAttacks, 1)
}

func (s statsdProcessor) EventIPListSize(evt mtglib.EventIPListSize) {
	tag := TagIPListBlock
	if !evt.IsBlockList {
		tag = TagIPListAllow
	}

	s.client.Gauge(MetricIPListSize, int64(evt.Size), statsd.StringTag(TagIPList, tag))
}

func (s statsdProcessor) Shutdown() {
	events := make([]mtglib.EventFinish, 0, len(s.streams))

	for k := range s.streams {
		events = append(events, mtglib.NewEventFinish(k))
	}

	for i := range events {
		s.EventFinish(events[i])
	}
}

// StatsdFactory is a factory of [events.Observer] which dumps information to
// statsd.
//
// Please beware that we support ONLY UDP endpoints there. And this factory
// won't use [mtglib.Network] so it won't use a proxy if you provide any. If
// you need it, I would recommend starting a local statsd and route metrics
// further by features of the chosen server.
type StatsdFactory struct {
	client *statsd.Client
}

// Close stops sending requests to statsd.
func (s StatsdFactory) Close() error {
	return s.client.Close() //nolint: wrapcheck
}

// Make build a new observer.
func (s StatsdFactory) Make() events.Observer {
	return statsdProcessor{
		client:  s.client,
		streams: make(map[string]*streamInfo),
	}
}

// NewStatsd builds an [events.ObserverFactory] that sends events to statsd.
//
// Valid tagFormats are 'datadog', 'influxdb' and 'graphite'.
func NewStatsd(address string, log logger.StdLikeLogger,
	metricPrefix, tagFormat string,
) (StatsdFactory, error) {
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
