package cli

import (
	"fmt"
	"io/ioutil"
	"net"
	"net/url"

	"github.com/9seconds/mtg/v2/config"
	"github.com/9seconds/mtg/v2/mtglib/network"
)

type base struct {
	network network.Network
	conf    *config.Config
}

func (b *base) ReadConfig(path string) error {
	content, err := ioutil.ReadFile(path)
	if err != nil {
		return fmt.Errorf("cannot read config file: %w", err)
	}

	conf, err := config.Parse(content)
	if err != nil {
		return fmt.Errorf("cannot parse config: %w", err)
	}

	ntw, err := b.makeNetwork(conf)
	if err != nil {
		return fmt.Errorf("cannot build a network: %w", err)
	}

	b.conf = conf
	b.network = ntw

	return nil
}

func (b *base) makeNetwork(conf *config.Config) (network.Network, error) {
	tcpTimeout := conf.Network.Timeout.TCP.Value(network.DefaultTimeout)
	idleTimeout := conf.Network.Timeout.Idle.Value(network.DefaultIdleTimeout)
	dohIP := conf.Network.DOHIP.Value(net.ParseIP(network.DefaultDOHHostname)).String()
	bufferSize := conf.TCPBuffer.Value(network.DefaultBufferSize)

	baseDialer, err := network.NewDefaultDialer(tcpTimeout, int(bufferSize))
	if err != nil {
		return nil, fmt.Errorf("cannot build a default dialer: %w", err)
	}

	proxyURLs := make([]*url.URL, 0, len(conf.Network.Proxies))

	for _, v := range conf.Network.Proxies {
		if value := v.Value(nil); value != nil {
			proxyURLs = append(proxyURLs, v.Value(nil))
		}
	}

	switch len(proxyURLs) {
	case 0:
		return network.NewNetwork(baseDialer, dohIP, idleTimeout)
	case 1:
		socksDialer, err := network.NewSocks5Dialer(baseDialer, proxyURLs[0])
		if err != nil {
			return nil, fmt.Errorf("cannot build socks5 dialer: %w", err)
		}

		return network.NewNetwork(socksDialer, dohIP, idleTimeout)
	}

	socksDialer, err := network.NewLoadBalancedSocks5Dialer(baseDialer, proxyURLs)
	if err != nil {
		return nil, fmt.Errorf("cannot build socks5 dialer: %w", err)
	}

	return network.NewNetwork(socksDialer, dohIP, idleTimeout)
}
