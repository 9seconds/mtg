package doppel

import "time"

const (
	ScoutConnCollectedPreallocSize = 100
)

type ScoutConnResult struct {
	timestamp  time.Time
	recordType byte
	payloadLen int
}

type ScoutConnCollected struct {
	data       []ScoutConnResult
	writeIndex int // index at which client first wrote post-handshake data; -1 if not set
}

func (s *ScoutConnCollected) Add(record byte, payloadLen int) {
	s.data = append(s.data, ScoutConnResult{
		timestamp:  time.Now(),
		recordType: record,
		payloadLen: payloadLen,
	})
}

// MarkWrite records the current data length as the handshake boundary.
func (s *ScoutConnCollected) MarkWrite() {
	if s.writeIndex < 0 {
		s.writeIndex = len(s.data)
	}
}

func NewScoutConnCollected() *ScoutConnCollected {
	return &ScoutConnCollected{
		data:       make([]ScoutConnResult, 0, ScoutConnCollectedPreallocSize),
		writeIndex: -1,
	}
}
