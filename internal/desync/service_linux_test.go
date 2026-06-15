//go:build linux

package desync

import (
	"encoding/binary"
	"testing"

	"github.com/stretchr/testify/require"
	"golang.org/x/sys/unix"
)

func TestBuildAndParseIPv4TCPPacket(t *testing.T) {
	src := [4]byte{192, 0, 2, 10}
	dst := [4]byte{198, 51, 100, 20}

	ipPacket := buildIPv4TCPPacket(src, dst, 12345, 9595, 100, 200, 1)

	require.Len(t, ipPacket, 20+20+len(fakeTLSAlert))
	require.Equal(t, byte(64), ipPacket[8])
	require.Equal(t, byte(unix.IPPROTO_TCP), ipPacket[9])
	require.Equal(t, uint16(12345), binary.BigEndian.Uint16(ipPacket[20:22]))
	require.Equal(t, uint16(9595), binary.BigEndian.Uint16(ipPacket[22:24]))
	require.Equal(t, uint32(100), binary.BigEndian.Uint32(ipPacket[24:28]))
	require.Equal(t, uint32(200), binary.BigEndian.Uint32(ipPacket[28:32]))
	require.Equal(t, byte(tcpFlagPsh|tcpFlagAck), ipPacket[33])

	wireSum := binary.BigEndian.Uint16(ipPacket[36:38])
	tcp := append([]byte(nil), ipPacket[20:40]...)
	binary.BigEndian.PutUint16(tcp[16:18], 0)
	validSum := tcpChecksum(src, dst, tcp)
	require.Equal(t, validSum^0xffff, wireSum)

	frame := make([]byte, 14+len(ipPacket))
	binary.BigEndian.PutUint16(frame[12:14], unix.ETH_P_IP)
	copy(frame[14:], ipPacket)

	packet, ok := parseIPv4TCP(frame)

	require.True(t, ok)
	require.Equal(t, [4]byte{192, 0, 2, 10}, packet.srcIP)
	require.Equal(t, [4]byte{198, 51, 100, 20}, packet.dstIP)
	require.Equal(t, uint16(12345), packet.srcPort)
	require.Equal(t, uint16(9595), packet.dstPort)
	require.Equal(t, uint32(100), packet.seq)
	require.Equal(t, uint32(200), packet.ack)
	require.Equal(t, byte(tcpFlagPsh|tcpFlagAck), packet.flags)
}
