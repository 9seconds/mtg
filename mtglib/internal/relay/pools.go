package relay

import "sync"

type eastWest struct {
	east []byte
	west []byte
}

var eastWestPool = sync.Pool{
	New: func() interface{} {
		return &eastWest{}
	},
}

func acquireEastWest(bufferSize int) *eastWest {
	wanted := eastWestPool.Get().(*eastWest) // nolint: forcetypeassert

	if len(wanted.east) != bufferSize {
		wanted.east = make([]byte, bufferSize)
	}

	if len(wanted.west) != bufferSize {
		wanted.west = make([]byte, bufferSize)
	}

	return wanted
}

func releaseEastWest(ew *eastWest) {
	eastWestPool.Put(ew)
}
