package stats

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/9seconds/mtg/config"
)

var instance *stats

func Start(conf *config.Config) {
	instance = &stats{
		URLs:         conf.GetURLs(),
		Uptime:       uptime(time.Now()),
		speedCurrent: &speed{},
		mutex:        &sync.RWMutex{},
	}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		instance.mutex.Lock()
		first, _ := json.Marshal(instance)
		instance.mutex.Unlock()

		interm := map[string]interface{}{}
		json.Unmarshal(first, &interm)

		encoder := json.NewEncoder(w)
		encoder.SetEscapeHTML(false)
		encoder.SetIndent("", "  ")
		encoder.Encode(interm)
	})

	http.ListenAndServe(conf.StatAddr(), nil)
}
