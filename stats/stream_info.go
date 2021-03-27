package stats

import (
	"net"
	"strconv"
	"time"

	statsd "github.com/smira/go-statsd"
)

type streamInfo struct {
	startTime time.Time
	tags      map[string]string
}

func (s *streamInfo) SetStartTime(tme time.Time) {
	s.startTime = tme
}

func (s *streamInfo) SetClientIP(ip net.IP) {
	if ip.To4() != nil {
		s.tags[TagIPFamily] = TagIPFamilyIPv4
	} else {
		s.tags[TagIPFamily] = TagIPFamilyIPv6
	}
}

func (s *streamInfo) SetTelegramIP(ip net.IP) {
	s.tags[TagTelegramIP] = ip.String()
}

func (s *streamInfo) SetDC(dc int) {
	s.tags[TagDC] = strconv.Itoa(dc)
}

func (s *streamInfo) V(key string) string {
	return s.tags[key]
}

func (s *streamInfo) TV(key string) statsd.Tag {
	return statsd.StringTag(key, s.tags[key])
}

func (s *streamInfo) Reset() {
	s.startTime = time.Time{}

	for k := range s.tags {
		delete(s.tags, k)
	}
}

func getDirection(isRead bool) string {
	if isRead { // for telegram
		return TagDirectionToClient
	}

	return TagDirectionFromClient
}
