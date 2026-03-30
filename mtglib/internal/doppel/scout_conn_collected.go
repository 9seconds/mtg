package doppel

import (
	"sync"
	"time"
)

const (
	ScoutConnCollectedPreallocSize = 100
)

type ScoutConnResult struct {
	timestamp  time.Time
	recordType byte
	payloadLen int
}

type ScoutConnCollected struct {
	mu         sync.Mutex
	data       []ScoutConnResult
	writeIndex int // index at which client first wrote post-handshake data; -1 if not set
}

func (s *ScoutConnCollected) Add(record byte, payloadLen int) {
	s.mu.Lock()
	s.data = append(s.data, ScoutConnResult{
		timestamp:  time.Now(),
		recordType: record,
		payloadLen: payloadLen,
	})
	s.mu.Unlock()
}

// MarkWrite records the current data length as the handshake boundary.
func (s *ScoutConnCollected) MarkWrite() {
	s.mu.Lock()
	if s.writeIndex < 0 {
		s.writeIndex = len(s.data)
	}
	s.mu.Unlock()
}

// Snapshot returns a copy of the collected data and the write index.
func (s *ScoutConnCollected) Snapshot() ([]ScoutConnResult, int) {
	s.mu.Lock()
	snapshot := make([]ScoutConnResult, len(s.data))
	copy(snapshot, s.data)
	writeIndex := s.writeIndex
	s.mu.Unlock()

	return snapshot, writeIndex
}

func NewScoutConnCollected() *ScoutConnCollected {
	return &ScoutConnCollected{
		data:       make([]ScoutConnResult, 0, ScoutConnCollectedPreallocSize),
		writeIndex: -1,
	}
}
