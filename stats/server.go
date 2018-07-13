package stats

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"go.uber.org/zap"

	"github.com/9seconds/mtg/config"
)

var instance *stats

// Start starts new statisitcs server.
func Start(conf *config.Config) error {
	log := zap.S().Named("stats")

	instance = &stats{
		URLs:   conf.GetURLs(),
		Uptime: uptime(time.Now()),
		mutex:  &sync.RWMutex{},
	}

	if conf.StatsD.Enabled {
		client, err := newStatsd(conf)
		if err != nil {
			return err
		}
		go client.run()
	}

	go crashManager()
	go connectionManager()
	go trafficManager()

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		instance.mutex.Lock()
		first, err := json.Marshal(instance)
		instance.mutex.Unlock()

		if err != nil {
			log.Errorw("Cannot encode json", "error", err)
			http.Error(w, "Internal server error", 500)
			return
		}

		interm := map[string]interface{}{}
		json.Unmarshal(first, &interm) // nolint: errcheck

		encoder := json.NewEncoder(w)
		encoder.SetEscapeHTML(false)
		encoder.SetIndent("", "  ")
		if err = encoder.Encode(interm); err != nil {
			log.Errorw("Cannot encode json", "error", err)
		}
	})

	go func() {
		if err := http.ListenAndServe(conf.StatAddr(), nil); err != nil {
			log.Fatalw("Stats server has been stopped", "error", err)
		}
	}()

	return nil
}
