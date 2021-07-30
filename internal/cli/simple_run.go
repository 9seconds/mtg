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

	Debug               bool          `kong:"name='debug',short='d',help='Run in debug mode.'"`
	Concurrency         uint64        `kong:"name='concurrency',short='c',default='8192',help='Max number of concurrent connection to proxy.'"`
	TCPBuffer           string        `kong:"name='tcp-buffer',short='b',default='4KB',help='Size of TCP buffer to use.'"`
	PreferIP            string        `kong:"name='prefer-ip',short='i',default='prefer-ipv6',help='IP preference. By default we prefer IPv6 with fallback to IPv4.'"`
	DomainFrontingPort  uint64        `kong:"name='domain-fronting-port',short='p',default='443',help='A port to access for domain fronting.'"`
	DOHIP               net.IP        `kong:"name='doh-ip',short='d',default='9.9.9.9',help='IP address of DNS-over-HTTP to use.'"`
	Timeout             time.Duration `kong:"name='timeout',short='t',default='10s',help='Network timeout to use'"`
	AntiReplayCacheSize string        `kong:"name='antireplay-cache-size',short='a',default='1MB',help='A size of anti-replay cache to use.'"`
}

func (s *SimpleRun) Run(cli *CLI, version string) error {
	conf := &config.Config{}

	if err := conf.BindTo.Set(s.BindTo); err != nil {
		return fmt.Errorf("incorrect bind-to parameter: %w", err)
	}

	if err := conf.Secret.Set(s.Secret); err != nil {
		return fmt.Errorf("incorrect secret: %w", err)
	}

	if err := conf.Concurrency.Set(strconv.FormatUint(s.Concurrency, 10)); err != nil {
		return fmt.Errorf("incorrect concurrency: %w", err)
	}

	if err := conf.TCPBuffer.Set(s.TCPBuffer); err != nil {
		return fmt.Errorf("incorrect tcp-buffer: %w", err)
	}

	if err := conf.PreferIP.Set(s.PreferIP); err != nil {
		return fmt.Errorf("incorrect prefer-ip: %w", err)
	}

	if err := conf.DomainFrontingPort.Set(strconv.FormatUint(s.DomainFrontingPort, 10)); err != nil {
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

	conf.Debug.Value = s.Debug
	conf.Defense.AntiReplay.Enabled.Value = true
	conf.Defense.Blocklist.Enabled.Value = false
	conf.Stats.StatsD.Enabled.Value = false
	conf.Stats.Prometheus.Enabled.Value = false

	if err := conf.Validate(); err != nil {
		return fmt.Errorf("invalid result configuration: %w", err)
	}

	return runProxy(conf, version)
}
