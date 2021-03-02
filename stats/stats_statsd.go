package stats

import (
	"fmt"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/9seconds/mtg/config"
	"github.com/9seconds/mtg/conntypes"
	statsd "github.com/smira/go-statsd"
	"go.uber.org/zap"
)

var (
	tagTrafficIngress = &statsStatsdTag{
		name: "ingress",
		tag:  statsd.StringTag("type", "ingress"),
	}
	tagTrafficEgress = &statsStatsdTag{
		name: "egress",
		tag:  statsd.StringTag("type", "egress"),
	}

	tagConnectionTypeAbridged = &statsStatsdTag{
		name: "abridged",
		tag:  statsd.StringTag("type", "abridged"),
	}
	tagConnectionTypeIntermediate = &statsStatsdTag{
		name: "intermediate",
		tag:  statsd.StringTag("type", "intermediate"),
	}
	tagConnectionTypeSecured = &statsStatsdTag{
		name: "secured",
		tag:  statsd.StringTag("type", "secured"),
	}

	tagConnectionProtocol4 = &statsStatsdTag{
		name: "ipv4",
		tag:  statsd.StringTag("protocol", "ipv4"),
	}
	tagConnectionProtocol6 = &statsStatsdTag{
		name: "ipv6",
		tag:  statsd.StringTag("protocol", "ipv6"),
	}
)

type statsStatsdTag struct {
	tag  statsd.Tag
	name string
}

type statsStatsdLogger struct {
	log *zap.SugaredLogger
}

func (s statsStatsdLogger) Printf(msg string, args ...interface{}) {
	s.log.Debugw(fmt.Sprintf(msg, args...))
}

type statsStatsd struct {
	seen      map[string]struct{}
	seenMutex sync.RWMutex
	client    *statsd.Client
}

func (s *statsStatsd) IngressTraffic(traffic int) {
	s.gauge("traffic", int64(traffic), tagTrafficIngress)
}

func (s *statsStatsd) EgressTraffic(traffic int) {
	s.gauge("traffic", int64(traffic), tagTrafficEgress)
}

func (s *statsStatsd) ClientConnected(connectionType conntypes.ConnectionType, addr *net.TCPAddr) {
	s.changeConnections(connectionType, addr, 1)
}

func (s *statsStatsd) ClientDisconnected(connectionType conntypes.ConnectionType, addr *net.TCPAddr) {
	s.changeConnections(connectionType, addr, -1)
}

func (s *statsStatsd) changeConnections(connectionType conntypes.ConnectionType, addr *net.TCPAddr, increment int64) {
	tags := make([]*statsStatsdTag, 0, 2)

	switch connectionType {
	case conntypes.ConnectionTypeAbridged:
		tags = append(tags, tagConnectionTypeAbridged)
	case conntypes.ConnectionTypeIntermediate:
		tags = append(tags, tagConnectionTypeIntermediate)
	case conntypes.ConnectionTypeSecure:
		tags = append(tags, tagConnectionTypeSecured)
	case conntypes.ConnectionTypeUnknown:
		panic("Unknown connection type")
	}

	if addr.IP.To4() == nil {
		tags = append(tags, tagConnectionProtocol6)
	} else {
		tags = append(tags, tagConnectionProtocol4)
	}

	s.gauge("connections", increment, tags...)
}

func (s *statsStatsd) TelegramConnected(dc conntypes.DC, addr *net.TCPAddr) {
	s.changeTelegramConnections(dc, addr, 1)
}

func (s *statsStatsd) TelegramDisconnected(dc conntypes.DC, addr *net.TCPAddr) {
	s.changeTelegramConnections(dc, addr, -1)
}

func (s *statsStatsd) changeTelegramConnections(dc conntypes.DC, addr *net.TCPAddr, increment int64) {
	tags := []*statsStatsdTag{
		{
			name: "dc" + strconv.Itoa(int(dc)),
			tag:  statsd.IntTag("dc", int(dc)),
		},
	}

	if addr.IP.To4() == nil {
		tags = append(tags, tagConnectionProtocol6)
	} else {
		tags = append(tags, tagConnectionProtocol4)
	}

	s.gauge("telegram_connections", increment, tags...)
}

func (s *statsStatsd) Crash() {
	s.gauge("crashes", 1)
}

func (s *statsStatsd) ReplayDetected() {
	s.gauge("replay_attacks", 1)
}

func (s *statsStatsd) AuthenticationFailed() {
	s.gauge("authentication_failed", 1)
}

func (s *statsStatsd) CloakedRequest() {
	s.gauge("cloaked_requests", 1)
}

func (s *statsStatsd) gauge(metric string, value int64, tags ...*statsStatsdTag) {
	key, tagList := s.prepareVals(metric, tags)
	s.initGauge(metric, key, tagList)
	s.client.GaugeDelta(metric, value, tagList...)
}

func (s *statsStatsd) prepareVals(metric string, tags []*statsStatsdTag) (string, []statsd.Tag) {
	tagList := make([]statsd.Tag, len(tags))
	builder := strings.Builder{}
	builder.WriteString(metric)

	for i, v := range tags {
		builder.WriteRune('.')
		builder.WriteString(v.name)
		tagList[i] = v.tag
	}

	return builder.String(), tagList
}

func (s *statsStatsd) initGauge(metric, key string, tags []statsd.Tag) {
	s.seenMutex.RLock()
	if _, ok := s.seen[key]; ok {
		s.seenMutex.RUnlock()

		return
	} else { // nolint: golint,revive
		s.seenMutex.RUnlock()
	}

	s.seenMutex.Lock()
	defer s.seenMutex.Unlock()

	if _, ok := s.seen[key]; !ok {
		s.seen[key] = struct{}{}
		s.client.Gauge(metric, 0, tags...)
	}
}

func newStatsStatsd() Interface {
	prefix := strings.TrimSuffix(config.C.StatsNamespace, ".") + "."
	logger := statsStatsdLogger{
		log: zap.S().Named("stats").Named("statsd"),
	}

	return &statsStatsd{
		seen: make(map[string]struct{}),
		client: statsd.NewClient(config.C.StatsdAddr.String(),
			statsd.SendLoopCount(2),
			statsd.ReconnectInterval(10*time.Second),
			statsd.Logger(logger),
			statsd.MetricPrefix(prefix),
			statsd.TagStyle(config.C.StatsdTagsFormat),
		),
	}
}
