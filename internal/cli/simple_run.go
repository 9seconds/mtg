package cli

import (
	"fmt"
	"net"
	"strconv"
	"time"

	"github.com/9seconds/mtg/v2/internal/config"
)

type SimpleRun struct {
	BindTo string `kong:"arg,required,name='bind-to',help='A host:port to bind proxy to.'"`
	Secret string `kong:"arg,required,name='secret',help='Proxy secret.'"`

	Debug               bool          `kong:"name='debug',short='d',help='Run in debug mode.'"`                                                                        //nolint: lll
	Concurrency         uint64        `kong:"name='concurrency',short='c',default='8192',help='Max number of concurrent connection to proxy.'"`                        //nolint: lll
	TCPBuffer           string        `kong:"name='tcp-buffer',short='b',default='4KB',help='Deprecated and ignored'"`                                                 //nolint: lll
	PreferIP            string        `kong:"name='prefer-ip',short='i',default='prefer-ipv6',help='IP preference. By default we prefer IPv6 with fallback to IPv4.'"` //nolint: lll
	DomainFrontingPort  uint64        `kong:"name='domain-fronting-port',short='p',default='443',help='A port to access for domain fronting.'"`                        //nolint: lll
	DOHIP               net.IP        `kong:"name='doh-ip',short='n',default='9.9.9.9',help='IP address of DNS-over-HTTP to use.'"`                                    //nolint: lll
	Timeout             time.Duration `kong:"name='timeout',short='t',default='10s',help='Network timeout to use'"`                                                    //nolint: lll
	Socks5Proxies       []string      `kong:"name='socks5-proxy',short='s',help='Socks5 proxies to use for network access.'"`                                          //nolint: lll
	AntiReplayCacheSize string        `kong:"name='antireplay-cache-size',short='a',default='1MB',help='A size of anti-replay cache to use.'"`                         //nolint: lll
}

func (s *SimpleRun) Run(cli *CLI, version string) error { //nolint: cyclop,funlen
	conf := &config.Config{}

	if err := conf.BindTo.Set(s.BindTo); err != nil {
		return fmt.Errorf("incorrect bind-to parameter: %w", err)
	}

	if err := conf.Secret.Set(s.Secret); err != nil {
		return fmt.Errorf("incorrect secret: %w", err)
	}

	if err := conf.Concurrency.Set(strconv.FormatUint(s.Concurrency, 10)); err != nil { //nolint: gomnd
		return fmt.Errorf("incorrect concurrency: %w", err)
	}

	if err := conf.PreferIP.Set(s.PreferIP); err != nil {
		return fmt.Errorf("incorrect prefer-ip: %w", err)
	}

	if err := conf.DomainFrontingPort.Set(strconv.FormatUint(s.DomainFrontingPort, 10)); err != nil { //nolint: gomnd
		return fmt.Errorf("incorrect domain-fronting-port: %w", err)
	}

	if err := conf.Network.DOHIP.Set(s.DOHIP.String()); err != nil {
		return fmt.Errorf("incorrect doh-ip: %w", err)
	}

	if err := conf.Network.Timeout.TCP.Set(s.Timeout.String()); err != nil {
		return fmt.Errorf("incorrect timeout: %w", err)
	}

	if err := conf.Network.Timeout.HTTP.Set(s.Timeout.String()); err != nil {
		return fmt.Errorf("incorrect timeout: %w", err)
	}

	if err := conf.Network.Timeout.Idle.Set(s.Timeout.String()); err != nil {
		return fmt.Errorf("incorrect timeout: %w", err)
	}

	if err := conf.Defense.AntiReplay.MaxSize.Set(s.AntiReplayCacheSize); err != nil {
		return fmt.Errorf("incorrect antireplay-cache-size: %w", err)
	}

	for _, v := range s.Socks5Proxies {
		proxyURL := config.TypeProxyURL{}

		if err := proxyURL.Set(v); err != nil {
			return fmt.Errorf("incorrect socks5 proxy URL: %w", err)
		}

		conf.Network.Proxies = append(conf.Network.Proxies, proxyURL)
	}

	conf.Debug.Value = s.Debug
	conf.AllowFallbackOnUnknownDC.Value = true
	conf.Defense.AntiReplay.Enabled.Value = true

	if err := conf.Validate(); err != nil {
		return fmt.Errorf("invalid result configuration: %w", err)
	}

	return runProxy(conf, version)
}
