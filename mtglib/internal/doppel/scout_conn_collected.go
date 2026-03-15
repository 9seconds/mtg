package doppel

import "time"

const (
	ScoutConnCollectedPreallocSize = 100
)

type ScoutConnResult struct {
	timestamp  time.Time
	recordType byte
}

type ScoutConnCollected struct {
	data []ScoutConnResult
}

func (s *ScoutConnCollected) Add(record byte) {
	s.data = append(s.data, ScoutConnResult{
		timestamp:  time.Now(),
		recordType: record,
	})
}

func NewScoutConnCollected() *ScoutConnCollected {
	return &ScoutConnCollected{
		data: make([]ScoutConnResult, 0, ScoutConnCollectedPreallocSize),
	}
}
