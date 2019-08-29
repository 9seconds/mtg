package newstats

import (
	"net"
	"strings"

	"gopkg.in/alexcesaro/statsd.v2"

	"github.com/9seconds/mtg/newconfig"
	"github.com/9seconds/mtg/newprotocol"
	"github.com/juju/errors"
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

func (s *statsStatsd) ClientConnected(connectionType newprotocol.ConnectionType, addr *net.TCPAddr) {
	s.changeConnections(connectionType, addr, 1)
}

func (s *statsStatsd) ClientDisconnected(connectionType newprotocol.ConnectionType, addr *net.TCPAddr) {
	s.changeConnections(connectionType, addr, -1)
}

func (s *statsStatsd) changeConnections(connectionType newprotocol.ConnectionType, addr *net.TCPAddr, value int) {
	var labels [3]string

	labels[0] = "connections"
	switch connectionType {
	case newprotocol.ConnectionTypeAbridged:
		labels[1] = "abridged"
	case newprotocol.ConnectionTypeSecure:
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
		statsd.Prefix(newconfig.C.StatsdStats.Prefix),
		statsd.Network(newconfig.C.StatsdStats.Addr.Network()),
		statsd.Address(newconfig.C.StatsdStats.Addr.String()),
		statsd.TagsFormat(newconfig.C.StatsdStats.TagsFormat),
	}

	if len(newconfig.C.StatsdStats.Tags) > 0 {
		tags := make([]string, len(newconfig.C.StatsdStats.Tags)*2)
		for k, v := range newconfig.C.StatsdStats.Tags {
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
