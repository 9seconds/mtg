package stats

import (
	"net"
	"time"
)

type streamInfo struct {
	createdAt time.Time
	clientIP  net.IP
}

func (s *streamInfo) IPType() string {
	if s.clientIP.To4() == nil {
		return TagIPTypeIPv6
	}

	return TagIPTypeIPv4
}
