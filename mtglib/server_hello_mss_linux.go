//go:build linux

package mtglib

import (
	"fmt"
	"net"
	"os"
	"time"

	"golang.org/x/sys/unix"
)

// tcpTimestampsOverhead is the size of the TCP timestamps option (with
// padding): the segment payload is that much smaller than the MSS when
// timestamps are negotiated.
const tcpTimestampsOverhead = 12

// tcpiOptTimestamps is TCPI_OPT_TIMESTAMPS of tcpi_options (linux/tcp.h);
// golang.org/x/sys/unix does not define it.
const tcpiOptTimestamps = 1

// serverHelloWaitMax caps the wait for one ServerHello to be sent when no
// handshake deadline is given.
const serverHelloWaitMax = 10 * time.Second

const (
	notsentPollMin = 100 * time.Microsecond
	notsentPollMax = 5 * time.Millisecond
)

// writeFragmented writes data to a TCP connection in chunks, each of which
// leaves in its own segment with a payload no larger than at MSS = mss. If the
// connection is not TCP or mss <= 0, data is written in one go.
func writeFragmented(conn net.Conn, data []byte, mss int, deadline time.Time) (int, error) {
	tcp := tcpConnOf(conn)
	if tcp == nil || mss <= 0 {
		return conn.Write(data) //nolint: wrapcheck
	}

	rawConn, err := tcp.SyscallConn()
	if err != nil {
		return 0, fmt.Errorf("cannot get raw connection: %w", err)
	}

	// Every chunk must leave immediately, without Nagle. Go enables
	// TCP_NODELAY by default, but do not rely on it.
	if err := tcp.SetNoDelay(true); err != nil {
		return 0, fmt.Errorf("cannot set TCP_NODELAY: %w", err)
	}

	info, err := getTCPInfo(rawConn)
	if err != nil {
		return 0, err
	}

	chunk := segmentPayload(mss, info)

	if deadline.IsZero() {
		deadline = time.Now().Add(serverHelloWaitMax)
	}

	written := 0

	for written < len(data) {
		if err := waitNotSent(rawConn, deadline); err != nil {
			return written, err
		}

		end := min(written+chunk, len(data))

		n, err := tcp.Write(data[written:end])
		written += n

		if err != nil {
			return written, err //nolint: wrapcheck
		}
	}

	// Wait for the last chunk as well, otherwise the next write may be glued
	// to it.
	return written, waitNotSent(rawConn, deadline)
}

// segmentPayload is the segment payload at MSS = mss: minus the timestamps
// option if it is negotiated, and no more than the current MSS of the
// connection.
func segmentPayload(mss int, info *unix.TCPInfo) int {
	payload := mss
	if info.Options&tcpiOptTimestamps != 0 {
		payload -= tcpTimestampsOverhead
	}

	if sndMSS := int(info.Snd_mss); sndMSS > 0 && payload > sndMSS {
		payload = sndMSS
	}

	return max(payload, 1)
}

type rawControl interface {
	Control(f func(fd uintptr)) error
}

func getTCPInfo(rawConn rawControl) (*unix.TCPInfo, error) {
	var (
		info   *unix.TCPInfo
		sysErr error
	)

	if err := rawConn.Control(func(fd uintptr) {
		info, sysErr = unix.GetsockoptTCPInfo(int(fd), unix.IPPROTO_TCP, unix.TCP_INFO)
	}); err != nil {
		return nil, fmt.Errorf("cannot access socket: %w", err)
	}

	if sysErr != nil {
		return nil, fmt.Errorf("cannot get TCP_INFO: %w", sysErr)
	}

	return info, nil
}

// waitNotSent waits until the socket queue has no unsent bytes (bytes already
// sent but not yet acknowledged do not count). It polls with a growing pause:
// waiting is needed only when the congestion window is full, which costs a
// few RTTs per handshake.
func waitNotSent(rawConn rawControl, deadline time.Time) error {
	pause := notsentPollMin

	for {
		info, err := getTCPInfo(rawConn)
		if err != nil {
			return err
		}

		if info.Notsent_bytes == 0 {
			return nil
		}

		if !time.Now().Before(deadline) {
			return fmt.Errorf("server hello is not sent in time: %w", os.ErrDeadlineExceeded)
		}

		time.Sleep(pause)

		pause = min(pause*2, notsentPollMax)
	}
}

// tcpConnOf returns the underlying TCP connection of the client: directly or
// from under the PROXY protocol wrapper. nil means it is not TCP.
func tcpConnOf(conn net.Conn) *net.TCPConn {
	switch c := conn.(type) {
	case *net.TCPConn:
		return c
	case interface{ TCPConn() (*net.TCPConn, bool) }:
		if tcp, ok := c.TCPConn(); ok {
			return tcp
		}
	}

	return nil
}
