package stats

import (
	"github.com/juju/errors"

	"github.com/9seconds/mtg/config"
)

// Init initializes stats subsystem.
func Init(conf *config.Config) error {
	if conf.StatsD.Enabled {
		client, err := newStatsd(conf)
		if err != nil {
			return errors.Annotate(err, "Cannot initialize statsd client")
		}
		go client.run()
	}
	prometheus, err := newPrometheus(conf)
	if err != nil {
		return errors.Annotate(err, "Cannot initialize prometheus client")
	}
	go prometheus.run()

	go NewStats(conf).start()
	go startServer(conf)

	return nil
}
