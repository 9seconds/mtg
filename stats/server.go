package stats

import (
	"encoding/json"
	"net/http"

	"go.uber.org/zap"

	"github.com/9seconds/mtg/config"
)

func startServer(conf *config.Config) {
	log := zap.S().Named("stats")

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		first, err := json.Marshal(GetStats())
		if err != nil {
			log.Errorw("Cannot encode json", "error", err)
			http.Error(w, "Internal server error", 500)
			return
		}

		interm := map[string]interface{}{}
		json.Unmarshal(first, &interm) // nolint: errcheck, gosec

		encoder := json.NewEncoder(w)
		encoder.SetEscapeHTML(false)
		encoder.SetIndent("", "  ")
		if err = encoder.Encode(interm); err != nil {
			log.Errorw("Cannot encode json", "error", err)
		}
	})

	if err := http.ListenAndServe(conf.StatAddr(), nil); err != nil {
		log.Fatalw("Stats server has been stopped", "error", err)
	}
}
