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
	return eastWestPool.Get().(*eastWest)
}

func releaseEastWest(ew *eastWest) {
	eastWestPool.Put(ew)
}
