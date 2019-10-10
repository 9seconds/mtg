package stats

import (
	"fmt"
	"net"
	"strconv"
	"strings"

	"gopkg.in/alexcesaro/statsd.v2"

	"github.com/9seconds/mtg/config"
	"github.com/9seconds/mtg/conntypes"
)

type statsStatsd struct {
	client *statsd.Client
}

func (s *statsStatsd) IngressTraffic(traffic int) {
	s.client.Count("traffic.ingress", traffic)
}

func (s *statsStatsd) EgressTraffic(traffic int) {
	s.client.Count("traffic.egress", traffic)
}

func (s *statsStatsd) ClientConnected(connectionType conntypes.ConnectionType, addr *net.TCPAddr) {
	s.changeConnections(connectionType, addr, 1)
}

func (s *statsStatsd) ClientDisconnected(connectionType conntypes.ConnectionType, addr *net.TCPAddr) {
	s.changeConnections(connectionType, addr, -1)
}

func (s *statsStatsd) changeConnections(connectionType conntypes.ConnectionType, addr *net.TCPAddr, value int) {
	labels := [...]string{
		"connections",
		"intermediate",
		"ipv4",
	}

	switch connectionType {
	case conntypes.ConnectionTypeAbridged:
		labels[1] = "abridged"
	case conntypes.ConnectionTypeSecure:
		labels[1] = "secured"
	}

	if addr.IP.To4() == nil {
		labels[2] = "ipv6"
	}

	s.client.Count(strings.Join(labels[:], "."), value)
}

func (s *statsStatsd) TelegramConnected(dc conntypes.DC, addr *net.TCPAddr) {
	s.changeTelegramConnections(dc, addr, 1)
}

func (s *statsStatsd) TelegramDisconnected(dc conntypes.DC, addr *net.TCPAddr) {
	s.changeTelegramConnections(dc, addr, -1)
}

func (s *statsStatsd) changeTelegramConnections(dc conntypes.DC, addr *net.TCPAddr, value int) {
	labels := [...]string{
		"telegram_connections",
		strconv.Itoa(int(dc)),
		"ipv4",
	}

	if addr.IP.To4() == nil {
		labels[2] = "ipv6"
	}

	s.client.Count(strings.Join(labels[:], "."), value)
}

func (s *statsStatsd) Crash() {
	s.client.Increment("crashes")
}

func (s *statsStatsd) AntiReplayDetected() {
	s.client.Increment("anti_replays")
}

func newStatsStatsd() (Interface, error) {
	options := []statsd.Option{
		statsd.Prefix(config.C.StatsNamespace),
		statsd.Network(config.C.StatsdNetwork),
		statsd.Address(config.C.StatsBind.String()),
		statsd.TagsFormat(config.C.StatsdTagsFormat),
	}

	if len(config.C.StatsdTags) > 0 {
		tags := make([]string, len(config.C.StatsdTags)*2)
		for k, v := range config.C.StatsdTags {
			tags = append(tags, k, v)
		}
		options = append(options, statsd.Tags(tags...))
	}

	client, err := statsd.New(options...)
	if err != nil {
		return nil, fmt.Errorf("cannot initialize a client: %w", err)
	}

	return &statsStatsd{
		client: client,
	}, nil
}
