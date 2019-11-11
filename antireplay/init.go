package antireplay

import (
	"math"
	"sync"

	"mtg/config"
	"github.com/dgraph-io/ristretto"
)

var (
	Cache    cache
	initOnce sync.Once
)

func Init() {
	initOnce.Do(func() {
		cost := float64(config.C.AntiReplayMaxSize) / 32.0
		cost = math.Ceil(cost)

		c, err := ristretto.NewCache(&ristretto.Config{
			NumCounters: int64(cost) * 10,
			MaxCost:     config.C.AntiReplayMaxSize,
			BufferItems: 64,
			Metrics:     false,
		})
		if err != nil {
			panic(err)
		}

		Cache.data = c
	})
}
