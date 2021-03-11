package main

import (
	"fmt"
	"io/ioutil"

	"github.com/9seconds/mtg/v2/config"
	"github.com/9seconds/mtg/v2/mtglib/network"
)

type cli struct {
	network network.Network
	conf    *config.Config
}

func (c *cli) ReadConfig(path string) error {
    content, err := ioutil.ReadFile(path)
    if err != nil {
		return fmt.Errorf("cannot read config file: %w", err)
    }

	conf, err := config.Parse(content)
	if err != nil {
		return fmt.Errorf("cannot parse config: %w", err)
	}

	ntw, err := makeNetwork(conf)
	if err != nil {
		return fmt.Errorf("cannot build a network: %w", err)
	}

	c.conf = conf
	c.network = ntw

	return nil
}
