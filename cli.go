package main

import (
	"fmt"
	"os"

	"github.com/9seconds/mtg/v2/mtglib/network"
)

type cli struct {
	conf    *config
	network *network.Network
}

func (c *cli) ReadConfig(path string) error {
	filefp, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("cannot open config file: %w", err)
	}

	defer filefp.Close()

	conf, err := parseConfig(filefp)
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
