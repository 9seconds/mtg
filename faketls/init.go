package faketls

import (
	"container/ring"
	"context"
	"sync"
	"time"

	"github.com/9seconds/mtg/config"
)

var (
	connectionServerInstance connectionServer
	connectionServerInitOnce sync.Once
)

const (
	connectionServerKeepCertificates = 5
	connectionServerUpdateEvery      = 10 * time.Minute
)

func Init(ctx context.Context) {
	connectionServerInitOnce.Do(func() {
		if config.C.CloakHost == "" {
			return
		}

		connectionServerInstance = connectionServer{
			channelGet: make(chan chan<- []byte),
			ctx:        ctx,
		}

		cert, err := connectionServerInstance.fetch()
		if err != nil {
			panic(err)
		}

		r := ring.New(connectionServerKeepCertificates)

		for i := 0; i < connectionServerKeepCertificates; i++ {
			r.Value = cert
			r = r.Next()
		}

		connectionServerInstance.nextWriteItem = r
		connectionServerInstance.nextReadItem = r

		go connectionServerInstance.run(connectionServerUpdateEvery)
	})
}
