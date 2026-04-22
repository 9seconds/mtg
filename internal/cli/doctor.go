package cli

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"net"
	"os"
	"slices"
	"strconv"
	"strings"
	"text/template"
	"time"

	"github.com/9seconds/mtg/v2/essentials"
	"github.com/9seconds/mtg/v2/internal/config"
	"github.com/9seconds/mtg/v2/internal/utils"
	"github.com/9seconds/mtg/v2/mtglib"
	"github.com/9seconds/mtg/v2/network/v2"
	"github.com/beevik/ntp"
)

var (
	tplError = template.Must(
		template.New("").Parse("  ‼️ {{ .description }}: {{ .error }}\n"),
	)

	tplWDeprecatedConfig = template.Must(
		template.New("").
			Parse(`  ⚠️ Option {{ .old | printf "%q" }}{{ if .old_section }} from section [{{ .old_section }}]{{ end }} is deprecated and will be removed in v{{ .when }}. Please use {{ .new | printf "%q" }}{{ if .new_section }} in [{{ .new_section }}] section{{ end }} instead.` + "\n"),
	)

	tplOTimeSkewness = template.Must(
		template.New("").
			Parse("  ✅ Time drift is {{ .drift }}, but tolerate-time-skewness is {{ .value }}\n"),
	)
	tplWTimeSkewness = template.Must(
		template.New("").
			Parse("  ⚠️ Time drift is {{ .drift }}, but tolerate-time-skewness is {{ .value }}. Please check ntp.\n"),
	)
	tplETimeSkewness = template.Must(
		template.New("").
			Parse("  ❌ Time drift is {{ .drift }}, but tolerate-time-skewness is {{ .value }}. You will get many rejected connections!\n"),
	)

	tplODCConnect = template.Must(
		template.New("").Parse("  ✅ DC {{ .dc }}\n"),
	)
	tplEDCConnect = template.Must(
		template.New("").Parse("  ❌ DC {{ .dc }}: {{ .error }}\n"),
	)

	tplODNSSNIMatch = template.Must(
		template.New("").Parse("  ✅ Secret hostname {{ .hostname }} matches our public IP ({{ .our }}); resolved: {{ .resolved }}\n"),
	)
	tplEDNSSNIMatch = template.Must(
		template.New("").Parse("  ❌ Secret hostname {{ .hostname }} resolves to {{ .resolved }} but our public IP is {{ .our }}{{ if .families }} (mismatched families: {{ .families }}){{ end }}\n"),
	)
	tplEDNSSNINoResolve = template.Must(
		template.New("").Parse("  ❌ Secret hostname {{ .hostname }} cannot be resolved to any address\n"),
	)

	tplOFrontingDomain = template.Must(
		template.New("").Parse("  ✅ {{ .address }} is reachable\n"),
	)
	tplEFrontingDomain = template.Must(
		template.New("").Parse("  ❌ {{ .address }}: {{ .error }}\n"),
	)
)

type Doctor struct {
	conf *config.Config

	ConfigPath string `kong:"arg,required,type='existingfile',help='Path to the configuration file.',name='config-path'"` //nolint: lll
}

func (d *Doctor) Run(cli *CLI, version string) error {
	conf, err := utils.ReadConfig(d.ConfigPath)
	if err != nil {
		return fmt.Errorf("cannot init config: %w", err)
	}

	d.conf = conf

	fmt.Println("Deprecated options")
	everythingOK := d.checkDeprecatedConfig()

	fmt.Println("Time skewness")
	everythingOK = d.checkTimeSkewness() && everythingOK

	resolver, err := network.GetDNS(conf.GetDNS())
	if err != nil {
		return fmt.Errorf("cannot create DNS resolver: %w", err)
	}

	base := network.New(
		resolver,
		"",
		conf.Network.Timeout.TCP.Get(10*time.Second),
		conf.Network.Timeout.HTTP.Get(0),
		conf.Network.Timeout.Idle.Get(0),
		net.KeepAliveConfig{
			Enable:   !conf.Network.KeepAlive.Disabled.Get(false),
			Idle:     conf.Network.KeepAlive.Idle.Get(0),
			Interval: conf.Network.KeepAlive.Interval.Get(0),
			Count:    int(conf.Network.KeepAlive.Count.Get(0)),
		},
	)

	fmt.Println("Validate native network connectivity")
	everythingOK = d.checkNetwork(base) && everythingOK

	for _, url := range conf.Network.Proxies {
		value, err := network.NewProxyNetwork(base, url.Get(nil))
		if err != nil {
			return err
		}

		fmt.Printf("Validate network connectivity with proxy %s\n", url.Get(nil))
		everythingOK = d.checkNetwork(value) && everythingOK
	}

	fmt.Println("Validate fronting domain connectivity")
	everythingOK = d.checkFrontingDomain(base) && everythingOK

	fmt.Println("Validate SNI-DNS match")
	everythingOK = d.checkSecretHost(resolver, base) && everythingOK

	if !everythingOK {
		os.Exit(1)
	}

	return nil
}

func (d *Doctor) checkDeprecatedConfig() bool {
	ok := true

	if d.conf.DomainFrontingIP.Value != nil {
		ok = false
		tplWDeprecatedConfig.Execute(os.Stdout, map[string]string{ //nolint: errcheck
			"when":        "2.3.0",
			"old":         "domain-fronting-ip",
			"old_section": "",
			"new":         "ip",
			"new_section": "domain-fronting",
		})
	}

	if d.conf.DomainFrontingPort.Value != 0 {
		ok = false
		tplWDeprecatedConfig.Execute(os.Stdout, map[string]string{ //nolint: errcheck
			"when":        "2.3.0",
			"old":         "domain-fronting-port",
			"old_section": "",
			"new":         "port",
			"new_section": "domain-fronting",
		})
	}

	if d.conf.DomainFrontingProxyProtocol.Value {
		ok = false
		tplWDeprecatedConfig.Execute(os.Stdout, map[string]string{ //nolint: errcheck
			"when":        "2.3.0",
			"old":         "domain-fronting-proxy-protocol",
			"old_section": "",
			"new":         "proxy-protocol",
			"new_section": "domain-fronting",
		})
	}

	if d.conf.Network.DOHIP.Value != nil {
		ok = false
		tplWDeprecatedConfig.Execute(os.Stdout, map[string]string{ //nolint: errcheck
			"when":        "2.3.0",
			"old":         "doh-ip",
			"old_section": "network",
			"new":         "dns",
			"new_section": "network",
		})
	}

	if ok {
		fmt.Println("  ✅ All good")
	}

	return ok
}

func (d *Doctor) checkTimeSkewness() bool {
	response, err := ntp.Query("0.pool.ntp.org")
	if err != nil {
		tplError.Execute(os.Stdout, map[string]any{ //nolint: errcheck
			"description": "cannot access ntp pool",
			"error":       err,
		})
		return false
	}

	skewness := response.ClockOffset.Abs()
	confValue := d.conf.TolerateTimeSkewness.Get(mtglib.DefaultTolerateTimeSkewness)
	diff := float64(skewness) / float64(confValue)
	tplData := map[string]any{
		"drift": response.ClockOffset,
		"value": confValue,
	}

	switch {
	case diff < 0.3:
		tplOTimeSkewness.Execute(os.Stdout, tplData) //nolint: errcheck
		return true
	case diff < 0.7:
		tplWTimeSkewness.Execute(os.Stdout, tplData) //nolint: errcheck
	default:
		tplETimeSkewness.Execute(os.Stdout, tplData) //nolint: errcheck
	}

	return false
}

func (d *Doctor) checkNetwork(ntw mtglib.Network) bool {
	dcs := slices.Collect(maps.Keys(essentials.TelegramCoreAddresses))
	slices.Sort(dcs)

	ok := true

	for _, dc := range dcs {
		err := d.checkNetworkAddresses(ntw, essentials.TelegramCoreAddresses[dc])
		if err == nil {
			tplODCConnect.Execute(os.Stdout, map[string]any{ //nolint: errcheck
				"dc": dc,
			})
		} else {
			tplEDCConnect.Execute(os.Stdout, map[string]any{ //nolint: errcheck
				"dc":    dc,
				"error": err,
			})
			ok = false
		}
	}

	return ok
}

func (d *Doctor) checkNetworkAddresses(ntw mtglib.Network, addresses []string) error {
	checkAddresses := []string{}

	switch d.conf.PreferIP.Get("prefer-ip4") {
	case "only-ipv4":
		for _, addr := range addresses {
			host, _, err := net.SplitHostPort(addr)
			if err != nil {
				panic(err)
			}

			if ip := net.ParseIP(host); ip != nil && ip.To4() != nil {
				checkAddresses = append(checkAddresses, addr)
			}
		}
	case "only-ipv6":
		for _, addr := range addresses {
			host, _, err := net.SplitHostPort(addr)
			if err != nil {
				panic(err)
			}

			if ip := net.ParseIP(host); ip != nil && ip.To4() == nil {
				checkAddresses = append(checkAddresses, addr)
			}
		}
	default:
		checkAddresses = addresses
	}

	if len(checkAddresses) == 0 {
		return fmt.Errorf("no suitable addresses after IP version filtering")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var (
		conn net.Conn
		err  error
	)

	for _, addr := range checkAddresses {
		conn, err = ntw.DialContext(ctx, "tcp", addr)
		if err != nil {
			continue
		}

		conn.Close() //nolint: errcheck

		return nil
	}

	return err
}

func (d *Doctor) checkFrontingDomain(ntw mtglib.Network) bool {
	host := d.conf.Secret.Host
	if ip := d.conf.GetDomainFrontingIP(nil); ip != "" {
		host = ip
	}

	port := d.conf.GetDomainFrontingPort(mtglib.DefaultDomainFrontingPort)
	address := net.JoinHostPort(host, strconv.Itoa(int(port)))

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	dialer := ntw.NativeDialer()

	conn, err := dialer.DialContext(ctx, "tcp", address)
	if err != nil {
		tplEFrontingDomain.Execute(os.Stdout, map[string]any{ //nolint: errcheck
			"address": address,
			"error":   err,
		})
		return false
	}

	conn.Close() //nolint: errcheck

	tplOFrontingDomain.Execute(os.Stdout, map[string]any{ //nolint: errcheck
		"address": address,
	})

	return true
}

func (d *Doctor) checkSecretHost(resolver *net.Resolver, ntw mtglib.Network) bool {
	res := runSNICheck(context.Background(), resolver, d.conf, ntw)

	if res.ResolveErr != nil {
		tplError.Execute(os.Stdout, map[string]any{ //nolint: errcheck
			"description": fmt.Sprintf("cannot resolve DNS name of %s", res.Host),
			"error":       res.ResolveErr,
		})
		return false
	}

	if !res.Known() {
		tplError.Execute(os.Stdout, map[string]any{ //nolint: errcheck
			"description": "cannot detect public IP address",
			"error":       errors.New("cannot detect automatically and public-ipv4/public-ipv6 are not set in config"),
		})
		return false
	}

	if len(res.Resolved) == 0 {
		tplEDNSSNINoResolve.Execute(os.Stdout, map[string]any{ //nolint: errcheck
			"hostname": res.Host,
		})
		return false
	}

	resolved := make([]string, 0, len(res.Resolved))
	for _, ip := range res.Resolved {
		resolved = append(resolved, `"`+ip.String()+`"`)
	}

	our := ""
	if res.OurIPv4 != nil {
		our = res.OurIPv4.String()
	}

	if res.OurIPv6 != nil {
		if our != "" {
			our += "/"
		}

		our += res.OurIPv6.String()
	}

	if res.OK() {
		tplODNSSNIMatch.Execute(os.Stdout, map[string]any{ //nolint: errcheck
			"hostname": res.Host,
			"resolved": strings.Join(resolved, ", "),
			"our":      our,
		})
		return true
	}

	mismatched := []string{}

	if res.OurIPv4 != nil && !res.IPv4Match {
		mismatched = append(mismatched, "IPv4")
	}

	if res.OurIPv6 != nil && !res.IPv6Match {
		mismatched = append(mismatched, "IPv6")
	}

	tplEDNSSNIMatch.Execute(os.Stdout, map[string]any{ //nolint: errcheck
		"hostname": res.Host,
		"resolved": strings.Join(resolved, ", "),
		"our":      our,
		"families": strings.Join(mismatched, ", "),
	})

	return false
}
