package cli

import (
	"fmt"

	"github.com/9seconds/mtg/v2/internal/utils"
)

type Proxy struct {
	ConfigPath string `kong:"arg,required,type='existingfile',help='Path to the configuration file.',name='config-path'"` // nolint: lll
}

func (c *Proxy) Run(cli *CLI, version string) error {
	conf, err := utils.ReadConfig(c.ConfigPath)
	if err != nil {
		return fmt.Errorf("cannot init config: %w", err)
	}

    return runProxy(conf, version)
}
