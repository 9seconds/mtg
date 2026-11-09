//go:build linux

package mtglib

import (
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"syscall"
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

// sendmsgFunc is the signature of unix.SendmsgN. sendChunks takes it as an
// argument so that tests can check the flags and the bounds of every call.
type sendmsgFunc func(fd int, p, oob []byte, to unix.Sockaddr, flags int) (int, error)

// writeFragmented writes the first split bytes of data to a TCP connection in
// chunks, each of which leaves in its own segment with a payload no larger
// than at MSS = mss, and the rest of data in one go, in regular segments. If
// the connection is not TCP or mss <= 0, data is written in one go.
//
// Each chunk is sent with sendmsg(MSG_EOR). The kernel never appends data to
// an skb marked EOR and never collapses it with the next one on
// retransmission (tcp_retrans_collapse), so chunks are not merged when they
// wait in the queue for the congestion window, the receive window or
// autocorking, and a retransmitted ServerHello still leaves in small
// segments. Without it, 4 KB written in 80-byte chunks at 100 ms RTT leave in
// 11-13 segments instead of 52 (kernel 6.12, netem).
//
// There is no polling: a write blocks only when the send buffer is full, and
// then it waits for the socket to become writable. The wait is bounded by
// deadline, which becomes the write deadline of the connection; a zero
// deadline keeps the write deadline the connection already has.
func writeFragmented(conn net.Conn, data []byte, mss, split int, deadline time.Time) (int, error) {
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

	if !deadline.IsZero() {
		if err := tcp.SetWriteDeadline(deadline); err != nil {
			return 0, fmt.Errorf("cannot set write deadline: %w", err)
		}
	}

	info, err := getTCPInfo(rawConn)
	if err != nil {
		return 0, err
	}

	return sendChunks(rawConn, data, segmentPayload(mss, info), split, unix.SendmsgN)
}

// sendChunks sends the first split bytes of data in chunks of chunk bytes and
// the rest as one more chunk, each one with MSG_EOR (the EOR of the last small
// chunk keeps the rest from being appended to it).
//
// A partial send (the send buffer filled up in the middle of a chunk) is not
// marked EOR by the kernel, so the rest of the same chunk, sent next with
// MSG_EOR, joins it in the same skb and the chunk still leaves as one
// segment. Chunk boundaries are therefore counted from the start of data, not
// from the last write.
func sendChunks(rawConn syscall.RawConn, data []byte, chunk, split int, sendmsg sendmsgFunc) (int, error) {
	var (
		written int
		sendErr error
	)

	err := rawConn.Write(func(fd uintptr) bool {
		for written < len(data) {
			end := len(data)
			if written < split {
				end = min((written/chunk+1)*chunk, split, len(data))
			}

			n, err := sendmsg(int(fd), data[written:end], nil, nil, unix.MSG_EOR|unix.MSG_NOSIGNAL)

			switch {
			case errors.Is(err, unix.EINTR):
				continue
			case errors.Is(err, unix.EAGAIN):
				// The send buffer is full: wait until the socket is
				// writable (or the write deadline passes).
				return false
			case err != nil:
				sendErr = os.NewSyscallError("sendmsg", err)

				return true
			case n <= 0:
				sendErr = io.ErrShortWrite

				return true
			}

			written += n
		}

		return true
	})
	if err != nil {
		return written, fmt.Errorf("cannot send server hello: %w", err)
	}

	return written, sendErr
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
