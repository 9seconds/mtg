//go:build linux

package desync

import (
	"encoding/binary"
	"fmt"
	"io"
	"sync"
	"time"

	"golang.org/x/sys/unix"
)

const (
	tcpFlagFin = 0x01
	tcpFlagSyn = 0x02
	tcpFlagPsh = 0x08
	tcpFlagAck = 0x10

	desyncStateTTL     = 30 * time.Second
	desyncCleanupEvery = time.Second
	ethernetHeaderLen  = 14
	vlanHeaderLen      = 4
	ipv4HeaderLen      = 20
	tcpHeaderLen       = 20
)

var fakeTLSAlert = []byte{0x15, 0x03, 0x03, 0x00, 0x02, 0x02, 0x28}

type runner struct {
	port      uint16
	packetFD  int
	rawFD     int
	wg        sync.WaitGroup
	closeOnce sync.Once
	state     map[connKey]*connState
	ident     uint16
	cleanup   time.Time
}

type connKey struct {
	clientIP   [4]byte
	localIP    [4]byte
	clientPort uint16
}

type connState struct {
	clientSeq uint32
	serverSeq uint32
	expiresAt time.Time
	sentMask  uint8
}

type tcpPacket struct {
	srcIP   [4]byte
	dstIP   [4]byte
	srcPort uint16
	dstPort uint16
	seq     uint32
	ack     uint32
	flags   byte
}

func Start(port int) (io.Closer, error) {
	if port <= 0 || port > 65535 {
		return nil, fmt.Errorf("invalid desync port: %d", port)
	}

	packetFD, err := unix.Socket(unix.AF_PACKET, unix.SOCK_RAW|unix.SOCK_CLOEXEC, int(htons(unix.ETH_P_IP)))
	if err != nil {
		return nil, fmt.Errorf("cannot open packet socket: %w", err)
	}

	rawFD, err := unix.Socket(unix.AF_INET, unix.SOCK_RAW|unix.SOCK_CLOEXEC, unix.IPPROTO_RAW)
	if err != nil {
		unix.Close(packetFD) //nolint: errcheck

		return nil, fmt.Errorf("cannot open raw ipv4 socket: %w", err)
	}
	if err := unix.SetsockoptInt(rawFD, unix.IPPROTO_IP, unix.IP_HDRINCL, 1); err != nil {
		unix.Close(packetFD) //nolint: errcheck
		unix.Close(rawFD)    //nolint: errcheck

		return nil, fmt.Errorf("cannot enable IP_HDRINCL: %w", err)
	}

	r := &runner{
		port:     uint16(port),
		packetFD: packetFD,
		rawFD:    rawFD,
		state:    map[connKey]*connState{},
	}

	r.wg.Add(1)
	go r.loop()

	return r, nil
}

func (r *runner) Close() error {
	r.closeOnce.Do(func() {
		unix.Close(r.packetFD) //nolint: errcheck
		unix.Close(r.rawFD)    //nolint: errcheck
		r.wg.Wait()
	})

	return nil
}

func (r *runner) loop() {
	defer r.wg.Done()

	buf := make([]byte, 64*1024)
	for {
		n, _, err := unix.Recvfrom(r.packetFD, buf, 0)
		if err != nil {
			if err == unix.EBADF || err == unix.EINVAL {
				return
			}

			continue
		}

		packet, ok := parseIPv4TCP(buf[:n])
		if !ok {
			continue
		}

		switch {
		case packet.dstPort == r.port:
			r.handleInbound(packet)
		case packet.srcPort == r.port:
			r.handleOutbound(packet)
		}
	}
}

func (r *runner) handleInbound(packet tcpPacket) {
	key := connKey{
		clientIP:   packet.srcIP,
		localIP:    packet.dstIP,
		clientPort: packet.srcPort,
	}

	now := time.Now()

	r.cleanupExpired(now)

	state := r.state[key]
	if packet.flags&tcpFlagSyn != 0 && packet.flags&tcpFlagAck == 0 {
		if state == nil {
			state = &connState{}
			r.state[key] = state
		}
		state.clientSeq = packet.seq
		state.expiresAt = now.Add(desyncStateTTL)

		return
	}

	if state == nil {
		return
	}

	state.expiresAt = now.Add(desyncStateTTL)

	if packet.flags&tcpFlagFin != 0 {
		delete(r.state, key)

		return
	}

	if packet.flags&tcpFlagAck != 0 &&
		state.clientSeq != 0 &&
		state.serverSeq != 0 &&
		packet.seq == state.clientSeq+1 &&
		packet.ack == state.serverSeq+1 {
		r.sendFake(key, state, 1)
		delete(r.state, key)
	}
}

func (r *runner) handleOutbound(packet tcpPacket) {
	if packet.flags&tcpFlagSyn == 0 || packet.flags&tcpFlagAck == 0 {
		return
	}

	key := connKey{
		clientIP:   packet.dstIP,
		localIP:    packet.srcIP,
		clientPort: packet.dstPort,
	}

	now := time.Now()

	r.cleanupExpired(now)

	state := r.state[key]
	if state == nil {
		state = &connState{}
		r.state[key] = state
	}

	state.serverSeq = packet.seq
	state.expiresAt = now.Add(desyncStateTTL)

	if state.clientSeq != 0 {
		r.sendFake(key, state, 0)
	}
}

func (r *runner) sendFake(key connKey, state *connState, phase uint8) {
	mask := uint8(1) << phase
	if state.sentMask&mask != 0 {
		return
	}
	state.sentMask |= mask
	r.ident++

	packet := buildIPv4TCPPacket(key.localIP, key.clientIP,
		r.port, key.clientPort, state.serverSeq+1, state.clientSeq+1, r.ident)

	unix.Sendto(r.rawFD, packet, 0, &unix.SockaddrInet4{Addr: key.clientIP}) //nolint: errcheck
}

func (r *runner) cleanupExpired(now time.Time) {
	if now.Before(r.cleanup) {
		return
	}
	r.cleanup = now.Add(desyncCleanupEvery)

	for key, state := range r.state {
		if now.After(state.expiresAt) {
			delete(r.state, key)
		}
	}
}

func parseIPv4TCP(frame []byte) (tcpPacket, bool) {
	if len(frame) < ethernetHeaderLen {
		return tcpPacket{}, false
	}

	etherType := binary.BigEndian.Uint16(frame[12:14])
	ipOffset := ethernetHeaderLen
	if etherType == 0x8100 || etherType == 0x88a8 {
		if len(frame) < ethernetHeaderLen+vlanHeaderLen {
			return tcpPacket{}, false
		}
		etherType = binary.BigEndian.Uint16(frame[16:18])
		ipOffset += vlanHeaderLen
	}
	if etherType != unix.ETH_P_IP {
		return tcpPacket{}, false
	}
	if len(frame) < ipOffset+ipv4HeaderLen {
		return tcpPacket{}, false
	}

	ip := frame[ipOffset:]
	ihl := int(ip[0]&0x0f) * 4
	if ihl < ipv4HeaderLen || len(ip) < ihl+tcpHeaderLen {
		return tcpPacket{}, false
	}
	if ip[0]>>4 != 4 || ip[9] != unix.IPPROTO_TCP {
		return tcpPacket{}, false
	}

	fragment := binary.BigEndian.Uint16(ip[6:8])
	if fragment&0x1fff != 0 {
		return tcpPacket{}, false
	}

	totalLen := int(binary.BigEndian.Uint16(ip[2:4]))
	if totalLen < ihl+tcpHeaderLen || totalLen > len(ip) {
		return tcpPacket{}, false
	}

	tcp := ip[ihl:totalLen]
	dataOffset := int(tcp[12]>>4) * 4
	if dataOffset < tcpHeaderLen || len(tcp) < dataOffset {
		return tcpPacket{}, false
	}

	var packet tcpPacket
	copy(packet.srcIP[:], ip[12:16])
	copy(packet.dstIP[:], ip[16:20])
	packet.srcPort = binary.BigEndian.Uint16(tcp[0:2])
	packet.dstPort = binary.BigEndian.Uint16(tcp[2:4])
	packet.seq = binary.BigEndian.Uint32(tcp[4:8])
	packet.ack = binary.BigEndian.Uint32(tcp[8:12])
	packet.flags = tcp[13]

	return packet, true
}

func buildIPv4TCPPacket(
	srcIP [4]byte,
	dstIP [4]byte,
	srcPort uint16,
	dstPort uint16,
	seq uint32,
	ack uint32,
	ident uint16,
) []byte {
	packet := make([]byte, ipv4HeaderLen+tcpHeaderLen+len(fakeTLSAlert))
	ip := packet[:ipv4HeaderLen]
	tcp := packet[ipv4HeaderLen : ipv4HeaderLen+tcpHeaderLen]

	ip[0] = 0x45
	binary.BigEndian.PutUint16(ip[2:4], uint16(len(packet)))
	binary.BigEndian.PutUint16(ip[4:6], ident)
	binary.BigEndian.PutUint16(ip[6:8], 0x4000)
	ip[8] = 64
	ip[9] = unix.IPPROTO_TCP
	copy(ip[12:16], srcIP[:])
	copy(ip[16:20], dstIP[:])
	binary.BigEndian.PutUint16(ip[10:12], checksum(ip))

	binary.BigEndian.PutUint16(tcp[0:2], srcPort)
	binary.BigEndian.PutUint16(tcp[2:4], dstPort)
	binary.BigEndian.PutUint32(tcp[4:8], seq)
	binary.BigEndian.PutUint32(tcp[8:12], ack)
	tcp[12] = 5 << 4
	tcp[13] = tcpFlagPsh | tcpFlagAck
	binary.BigEndian.PutUint16(tcp[14:16], 65535)
	copy(packet[ipv4HeaderLen+tcpHeaderLen:], fakeTLSAlert)

	// The checksum is deliberately invalid: DPI can still inspect the fake TLS
	// alert, but the client TCP stack should drop it.
	sum := tcpChecksum(srcIP, dstIP, tcp) ^ 0xffff
	if sum == 0 {
		sum = 0xffff
	}
	binary.BigEndian.PutUint16(tcp[16:18], sum)

	return packet
}

func tcpChecksum(srcIP, dstIP [4]byte, tcp []byte) uint16 {
	pseudo := make([]byte, 12+len(tcp)+len(fakeTLSAlert))
	copy(pseudo[0:4], srcIP[:])
	copy(pseudo[4:8], dstIP[:])
	pseudo[9] = unix.IPPROTO_TCP
	binary.BigEndian.PutUint16(pseudo[10:12], uint16(len(tcp)+len(fakeTLSAlert)))
	copy(pseudo[12:], tcp)
	copy(pseudo[12+len(tcp):], fakeTLSAlert)

	return checksum(pseudo)
}

func checksum(data []byte) uint16 {
	var sum uint32
	for len(data) >= 2 {
		sum += uint32(binary.BigEndian.Uint16(data[:2]))
		data = data[2:]
	}
	if len(data) == 1 {
		sum += uint32(data[0]) << 8
	}

	for sum>>16 != 0 {
		sum = (sum & 0xffff) + (sum >> 16)
	}

	return ^uint16(sum)
}

func htons(value uint16) uint16 {
	return (value << 8) | (value >> 8)
}
