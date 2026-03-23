package cli

import (
	"fmt"
	"os"
	"text/template"

	"github.com/9seconds/mtg/v2/internal/config"
	"github.com/9seconds/mtg/v2/internal/utils"
	"github.com/9seconds/mtg/v2/mtglib"
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
	everythingOK := true

	fmt.Println("Deprecated options")
	if !d.checkDeprecatedConfig() {
		everythingOK = false
	} else {
		fmt.Println("  ✅ All good")
	}

	fmt.Println("Time skewness")
	if !d.checkTimeSkewness() {
		everythingOK = false
	}

	if !everythingOK {
		os.Exit(1)
	}

	return nil
}

func (d *Doctor) checkDeprecatedConfig() bool {
	ok := true

	if d.conf.DomainFrontingIP.Value != nil {
		ok = false
		tplWDeprecatedConfig.Execute(os.Stdout, map[string]string{
			"when":        "2.3.0",
			"old":         "domain-fronting-ip",
			"old_section": "",
			"new":         "ip",
			"new_section": "domain-fronting",
		})
	}

	if d.conf.DomainFrontingPort.Value != 0 {
		ok = false
		tplWDeprecatedConfig.Execute(os.Stdout, map[string]string{
			"when":        "2.3.0",
			"old":         "domain-fronting-port",
			"old_section": "",
			"new":         "port",
			"new_section": "domain-fronting",
		})
	}

	if d.conf.DomainFrontingProxyProtocol.Value {
		ok = false
		tplWDeprecatedConfig.Execute(os.Stdout, map[string]string{
			"when":        "2.3.0",
			"old":         "domain-fronting-proxy-protocol",
			"old_section": "",
			"new":         "proxy-protocol",
			"new_section": "domain-fronting",
		})
	}

	if d.conf.Network.DOHIP.Value != nil {
		ok = false
		tplWDeprecatedConfig.Execute(os.Stdout, map[string]string{
			"when":        "2.3.0",
			"old":         "doh-ip",
			"old_section": "network",
			"new":         "dns",
			"new_section": "network",
		})
	}

	return ok
}

func (d *Doctor) checkTimeSkewness() bool {
	response, err := ntp.Query("0.pool.ntp.org")
	if err != nil {
		tplError.Execute(os.Stdout, map[string]any{
			"description": "cannot access ntp pool",
			"error":       err,
		})
		return false
	}

	skewness := response.ClockOffset.Abs()
	confValue := d.conf.TolerateTimeSkewness.Get(mtglib.DefaultTolerateTimeSkewness)
	diff := float64(skewness) / float64(confValue)
	context := map[string]any{
		"drift": response.ClockOffset,
		"value": confValue,
	}

	switch {
	case diff < 0.3:
		tplOTimeSkewness.Execute(os.Stdout, context)
		return true
	case diff < 0.7:
		tplWTimeSkewness.Execute(os.Stdout, context)
	default:
		tplETimeSkewness.Execute(os.Stdout, context)
	}

	return false
}
