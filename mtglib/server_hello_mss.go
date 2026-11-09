package mtglib

import (
	"encoding/binary"
	"net"
	"time"

	"github.com/9seconds/mtg/v2/mtglib/internal/tls"
)

// Small-segment FakeTLS ServerHello (network.client-mss).
//
// Some DPI boxes recognise the FakeTLS ServerHello when it arrives in
// full-size TCP segments, but not when it is split into small ones (as with a
// client MSS of 92). Keeping a small MSS for the whole session is slow and
// costs a lot of packets per second, so only the ServerHello is split; the
// rest of the session uses full-size segments.
//
// Why not "a small TCP_MAXSEG on the listener, then setsockopt a larger one":
// Linux fixes mss_clamp during the TCP handshake as min(client MSS, socket
// TCP_MAXSEG), and setsockopt(TCP_MAXSEG) on an established connection only
// changes user_mss, which no longer affects the size of outgoing segments
// (checked on 6.12: listener MSS 92, then setsockopt 1400 - tcpi_snd_mss stays
// at 80). For the same reason the MSS of an established connection cannot be
// lowered temporarily either.
//
// So the ServerHello is written in chunks that fit one segment at client-mss,
// each sent with MSG_EOR, so that the kernel does not merge chunks that wait
// in the socket queue (congestion window, receive window, autocorking) into
// one large segment, and does not collapse them on retransmission either: at
// 100 ms RTT 4 KB written in 80-byte chunks leave in 11-13 segments without it
// and in 52 with it (kernel 6.12, netem).

// Only the beginning of the response is split: the ServerHello record itself
// by default (about 130 bytes, 2 segments at client-mss 92), or the first
// ServerHelloMSSBytes bytes. The rest (ChangeCipherSpec and the 2.5-4.7 KB
// ApplicationData noise) follows in full-size segments. Splitting all of it
// costs 35-60 segments: on a path with 2% loss at least one of them is lost
// in most handshakes, and with an initial window of 10 segments it takes a
// few round trips of slow start just to send the response.
//
// The last small chunk carries MSG_EOR as every other one, so the large rest
// is not appended to it.

// fragmentedWriter writes the ServerHello: the first prefix bytes (see
// splitLength) in chunks of mss, the rest in one go (see writeFragmented). A
// full send buffer is waited for no longer than deadline (the handshake
// deadline).
type fragmentedWriter struct {
	conn     net.Conn
	mss      int
	prefix   int
	deadline time.Time
}

func (w fragmentedWriter) Write(p []byte) (int, error) {
	return writeFragmented(w.conn, p, w.mss, splitLength(p, w.prefix), w.deadline)
}

// splitLength is how many first bytes of the server response are split into
// small segments. prefix > 0 is taken as is (capped by the response length);
// prefix <= 0 means the first TLS record, which is the ServerHello. If data
// does not start with a complete TLS record header, all of it is split.
func splitLength(data []byte, prefix int) int {
	if prefix > 0 {
		return min(prefix, len(data))
	}

	if len(data) < tls.SizeHeader {
		return len(data)
	}

	record := tls.SizeHeader + int(binary.BigEndian.Uint16(data[tls.SizeHeader-2:tls.SizeHeader]))

	return min(record, len(data))
}
