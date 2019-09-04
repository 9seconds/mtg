package stats

import (
	"net"
	"strings"

	"github.com/juju/errors"
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
	var labels [3]string

	labels[0] = "connections"
	switch connectionType {
	case conntypes.ConnectionTypeAbridged:
		labels[1] = "abridged"
	case conntypes.ConnectionTypeSecure:
		labels[1] = "secured"
	default:
		labels[1] = "intermediate"
	}

	labels[2] = "ipv4"
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

func newStatsStatsd() (Stats, error) {
	options := []statsd.Option{
		statsd.Prefix(config.C.StatsdStats.Prefix),
		statsd.Network(config.C.StatsdStats.Addr.Network()),
		statsd.Address(config.C.StatsdStats.Addr.String()),
		statsd.TagsFormat(config.C.StatsdStats.TagsFormat),
	}

	if len(config.C.StatsdStats.Tags) > 0 {
		tags := make([]string, len(config.C.StatsdStats.Tags)*2)
		for k, v := range config.C.StatsdStats.Tags {
			tags = append(tags, k, v)
		}
		options = append(options, statsd.Tags(tags...))
	}

	client, err := statsd.New(options...)
	if err != nil {
		return nil, errors.Annotate(err, "Cannot initialize a client")
	}

	return &statsStatsd{
		client: client,
	}, nil
}
