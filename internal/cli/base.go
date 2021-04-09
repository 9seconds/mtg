package cli

import (
	"fmt"
	"net"
	"net/url"
	"os"

	"github.com/9seconds/mtg/v2/internal/config"
	"github.com/9seconds/mtg/v2/mtglib"
	"github.com/9seconds/mtg/v2/network"
)

type base struct {
	ConfigPath string `kong:"arg,required,type='existingfile',help='Path to the configuration file.',name='config-path'"` // nolint: lll

	Network mtglib.Network `kong:"-"`
	Config  *config.Config `kong:"-"`
}

func (b *base) ReadConfig(version string) error {
	content, err := os.ReadFile(b.ConfigPath)
	if err != nil {
		return fmt.Errorf("cannot read config file: %w", err)
	}

	conf, err := config.Parse(content)
	if err != nil {
		return fmt.Errorf("cannot parse config: %w", err)
	}

	ntw, err := b.makeNetwork(conf, version)
	if err != nil {
		return fmt.Errorf("cannot build a network: %w", err)
	}

	b.Config = conf
	b.Network = ntw

	return nil
}

func (b *base) makeNetwork(conf *config.Config, version string) (mtglib.Network, error) {
	tcpTimeout := conf.Network.Timeout.TCP.Value(network.DefaultTimeout)
	httpTimeout := conf.Network.Timeout.HTTP.Value(network.DefaultHTTPTimeout)
	dohIP := conf.Network.DOHIP.Value(net.ParseIP(network.DefaultDOHHostname)).String()
	bufferSize := conf.TCPBuffer.Value(network.DefaultBufferSize)
	userAgent := "mtg/" + version

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
		return network.NewNetwork(baseDialer, userAgent, dohIP, httpTimeout) // nolint: wrapcheck
	case 1:
		socksDialer, err := network.NewSocks5Dialer(baseDialer, proxyURLs[0])
		if err != nil {
			return nil, fmt.Errorf("cannot build socks5 dialer: %w", err)
		}

		return network.NewNetwork(socksDialer, userAgent, dohIP, httpTimeout) // nolint: wrapcheck
	}

	socksDialer, err := network.NewLoadBalancedSocks5Dialer(baseDialer, proxyURLs)
	if err != nil {
		return nil, fmt.Errorf("cannot build socks5 dialer: %w", err)
	}

	return network.NewNetwork(socksDialer, userAgent, dohIP, httpTimeout) // nolint: wrapcheck
}
