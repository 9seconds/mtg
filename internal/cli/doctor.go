package cli

import (
	"fmt"
	"os"
	"strings"
	"text/template"

	"github.com/9seconds/mtg/v2/internal/config"
	"github.com/9seconds/mtg/v2/internal/utils"
)

var (
	tplWDeprecatedConfig = template.Must(
		template.New("deprecated-config").
			Parse(`  ⚠️ Option {{ .old | printf "%q" }}{{ if .old_section }} from section [{{ .old_section }}]{{ end }} is deprecated and will be removed in v{{ .when }}. Please use {{ .new | printf "%q" }}{{ if .new_section }} in [{{ .new_section }}] section{{ end }} instead.`),
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
	if errs := d.checkDeprecatedConfig(); len(errs) > 0 {
		for _, err := range errs {
			fmt.Println(err)
			everythingOK = false
		}
	} else {
		fmt.Println("  ✅ All good")
	}

	if !everythingOK {
		os.Exit(1)
	}

	return nil
}

func (d *Doctor) checkDeprecatedConfig() []string {
	errors := []string{}

	if d.conf.DomainFrontingIP.Value != nil {
		errors = d.addError(errors, tplWDeprecatedConfig, map[string]string{
			"when":        "2.3.0",
			"old":         "domain-fronting-ip",
			"old_section": "",
			"new":         "ip",
			"new_section": "domain-fronting",
		})
	}

	if d.conf.DomainFrontingPort.Value != 0 {
		errors = d.addError(errors, tplWDeprecatedConfig, map[string]string{
			"when":        "2.3.0",
			"old":         "domain-fronting-port",
			"old_section": "",
			"new":         "port",
			"new_section": "domain-fronting",
		})
	}

	if d.conf.DomainFrontingProxyProtocol.Value {
		errors = d.addError(errors, tplWDeprecatedConfig, map[string]string{
			"when":        "2.3.0",
			"old":         "domain-fronting-proxy-protocol",
			"old_section": "",
			"new":         "proxy-protocol",
			"new_section": "domain-fronting",
		})
	}

	if d.conf.Network.DOHIP.Value != nil {
		errors = d.addError(errors, tplWDeprecatedConfig, map[string]string{
			"when":        "2.3.0",
			"old":         "doh-ip",
			"old_section": "network",
			"new":         "dns",
			"new_section": "network",
		})
	}

	return errors
}

func (d *Doctor) addError(messages []string, tpl *template.Template, context map[string]string) []string {
	value := &strings.Builder{}
	if err := tpl.Execute(value, context); err != nil {
		panic(err)
	}

	return append(messages, value.String())
}
