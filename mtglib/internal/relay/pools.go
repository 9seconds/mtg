package relay

import "sync"

type eastWest struct {
	east []byte
	west []byte
}

var eastWestPool = sync.Pool{
	New: func() interface{} {
		return &eastWest{
			east: make([]byte, bufferSize),
			west: make([]byte, bufferSize),
		}
	},
}

func acquireEastWest() *eastWest {
	wanted := eastWestPool.Get().(*eastWest) // nolint: forcetypeassert

	return wanted
}

func releaseEastWest(ew *eastWest) {
	eastWestPool.Put(ew)
}
