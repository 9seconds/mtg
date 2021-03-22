package stats

import (
	"net"
	"time"
)

type streamInfo struct {
	createdAt             time.Time
	clientIP              net.IP
	remoteIP              net.IP
	dc                    int
	bytesSentToTelegram   uint
	bytesRecvFromTelegram uint
}

func (s *streamInfo) GetClientIPType() string {
	return s.getIPType(s.clientIP)
}

func (s *streamInfo) GetRemoteIPType() string {
	return s.getIPType(s.remoteIP)
}

func (s *streamInfo) getIPType(ip net.IP) string {
	if ip.To4() == nil {
		return TagIPTypeIPv6
	}

	return TagIPTypeIPv4
}
