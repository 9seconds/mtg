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
// and each next chunk is written only after the previous one has left the
// socket queue (tcpi_notsent_bytes = 0). Otherwise chunks that hit the
// congestion window or autocorking are merged by the kernel into one large
// segment: at 100 ms RTT 4 KB written in 80-byte chunks without waiting leave
// in 13 segments, with waiting - in 52.

// fragmentedWriter writes the ServerHello in chunks of mss (see
// writeFragmented).
type fragmentedWriter struct {
	conn     net.Conn
	mss      int
	deadline time.Time
}

func (w fragmentedWriter) Write(p []byte) (int, error) {
	return writeFragmented(w.conn, p, w.mss, w.deadline)
}
