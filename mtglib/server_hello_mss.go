package mtglib

import (
	"net"
	"time"
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

// fragmentedWriter writes the ServerHello in chunks of mss (see
// writeFragmented). A full send buffer is waited for no longer than deadline
// (the handshake deadline).
type fragmentedWriter struct {
	conn     net.Conn
	mss      int
	deadline time.Time
}

func (w fragmentedWriter) Write(p []byte) (int, error) {
	return writeFragmented(w.conn, p, w.mss, w.deadline)
}
