//go:build linux

package mtglib_test

import (
	"encoding/binary"
	"errors"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"golang.org/x/sys/unix"
)

// capturedSegment is one TCP data segment seen on the loopback interface.
type capturedSegment struct {
	srcPort int
	dstPort int
	payload int
	options int
}

// loopbackCapture records TCP data segments of the loopback interface with an
// AF_PACKET socket. On lo every packet is seen twice (outgoing and incoming),
// only the incoming copy is kept.
type loopbackCapture struct {
	fd   int
	stop chan struct{}
	done chan struct{}

	mu       sync.Mutex
	segments []capturedSegment
}

// startLoopbackCapture starts capturing or skips the test if the process has
// no CAP_NET_RAW.
func startLoopbackCapture(t *testing.T) *loopbackCapture {
	t.Helper()

	fd, err := unix.Socket(unix.AF_PACKET, unix.SOCK_DGRAM|unix.SOCK_CLOEXEC, int(htons(unix.ETH_P_ALL)))
	if errors.Is(err, unix.EPERM) || errors.Is(err, unix.EACCES) {
		t.Skipf("capturing packets needs CAP_NET_RAW: %v", err)
	}

	require.NoError(t, err)

	lo := loopbackInterface(t)

	require.NoError(t, unix.SetsockoptInt(fd, unix.SOL_SOCKET, unix.SO_RCVBUF, 4<<20))
	require.NoError(t, unix.SetsockoptTimeval(fd, unix.SOL_SOCKET, unix.SO_RCVTIMEO,
		&unix.Timeval{Usec: 20_000}))
	require.NoError(t, unix.Bind(fd, &unix.SockaddrLinklayer{
		Protocol: htons(unix.ETH_P_ALL),
		Ifindex:  lo.Index,
	}))

	capture := &loopbackCapture{
		fd:   fd,
		stop: make(chan struct{}),
		done: make(chan struct{}),
	}

	go capture.run()

	t.Cleanup(capture.close)

	return capture
}

func loopbackInterface(t *testing.T) net.Interface {
	t.Helper()

	ifaces, err := net.Interfaces()
	require.NoError(t, err)

	for _, iface := range ifaces {
		if iface.Flags&net.FlagLoopback != 0 {
			return iface
		}
	}

	t.Skip("no loopback interface")

	return net.Interface{}
}

func (c *loopbackCapture) run() {
	defer close(c.done)

	buf := make([]byte, 65536)

	for {
		select {
		case <-c.stop:
			return
		default:
		}

		n, from, err := unix.Recvfrom(c.fd, buf, 0)
		if err != nil {
			continue // EAGAIN on SO_RCVTIMEO, EINTR
		}

		if sll, ok := from.(*unix.SockaddrLinklayer); !ok || sll.Pkttype != unix.PACKET_HOST {
			continue
		}

		if seg, ok := parseTCPv4(buf[:n]); ok && seg.payload > 0 {
			c.mu.Lock()
			c.segments = append(c.segments, seg)
			c.mu.Unlock()
		}
	}
}

func (c *loopbackCapture) close() {
	select {
	case <-c.stop:
		return
	default:
		close(c.stop)
	}

	<-c.done
	unix.Close(c.fd) //nolint: errcheck
}

// serverSegments waits until the captured traffic settles and returns the
// data segments sent to the given client port.
func (c *loopbackCapture) serverSegments(t *testing.T, serverPort int) []capturedSegment {
	t.Helper()

	// The client has read everything, so all segments have passed lo; give
	// the capture socket a moment to drain its queue.
	time.Sleep(100 * time.Millisecond)
	c.close()

	c.mu.Lock()
	defer c.mu.Unlock()

	var segments []capturedSegment

	for _, seg := range c.segments {
		if seg.srcPort == serverPort {
			segments = append(segments, seg)
		}
	}

	return segments
}

// parseTCPv4 parses an IPv4 packet (SOCK_DGRAM gives it without the link
// header) carrying TCP.
func parseTCPv4(packet []byte) (capturedSegment, bool) {
	if len(packet) < 20 || packet[0]>>4 != 4 || packet[9] != unix.IPPROTO_TCP {
		return capturedSegment{}, false
	}

	ipHeader := int(packet[0]&0x0f) * 4
	total := int(binary.BigEndian.Uint16(packet[2:4]))

	if total > len(packet) || ipHeader+20 > total {
		return capturedSegment{}, false
	}

	tcp := packet[ipHeader:total]
	tcpHeader := int(tcp[12]>>4) * 4

	if tcpHeader < 20 || tcpHeader > len(tcp) {
		return capturedSegment{}, false
	}

	return capturedSegment{
		srcPort: int(binary.BigEndian.Uint16(tcp[0:2])),
		dstPort: int(binary.BigEndian.Uint16(tcp[2:4])),
		payload: len(tcp) - tcpHeader,
		options: tcpHeader - 20,
	}, true
}

func htons(v uint16) uint16 {
	return v<<8 | v>>8
}
