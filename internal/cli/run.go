package cli

import (
	"fmt"

	"github.com/9seconds/mtg/v2/internal/utils"
)

type Run struct {
	ConfigPath string `kong:"arg,required,type='existingfile',help='Path to the configuration file.',name='config-path'"` //nolint: lll
}

func (r *Run) Run(cli *CLI, version string) error {
	conf, err := utils.ReadConfig(r.ConfigPath)
	if err != nil {
		return fmt.Errorf("cannot init config: %w", err)
	}

	return runProxy(conf, version)
}
