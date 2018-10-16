package stats

import (
	"github.com/juju/errors"

	"github.com/9seconds/mtg/config"
)

func Init(conf *config.Config) error {
	if conf.StatsD.Enabled {
		client, err := newStatsd(conf)
		if err != nil {
			return errors.Annotate(err, "Cannot initialize statsd client")
		}
		go client.run()
	}

	go NewStats(conf).start()
	go startServer(conf)

	return nil
}
