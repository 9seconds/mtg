package utils

import (
	"fmt"
	"os"

	"github.com/9seconds/mtg/v2/internal/config"
)

func ReadConfig(path string) (*config.Config, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("cannot read config file: %w", err)
	}

	conf, err := config.Parse(content)
	if err != nil {
		return nil, fmt.Errorf("cannot parse config: %w", err)
	}

	if err := conf.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	return conf, nil
}
