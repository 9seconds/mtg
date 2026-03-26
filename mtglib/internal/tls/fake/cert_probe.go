package fake

import (
	"crypto/tls"
	"encoding/binary"
	"fmt"
	"net"
	"sync"
	"time"
)

const (
	probeDialTimeout      = 10 * time.Second
	probeHandshakeTimeout = 10 * time.Second
	defaultProbeCount     = 15

	tlsTypeChangeCipherSpec = 0x14
	tlsTypeApplicationData  = 0x17
)

// CertProbeResult holds the measured encrypted handshake size.
type CertProbeResult struct {
	Mean   int
	Jitter int
}

// ProbeCertSize connects to hostname:port via TLS multiple times and measures
// the total ApplicationData payload bytes sent by the server during the
// handshake (between ChangeCipherSpec and the first application-level data).
// This corresponds to EncryptedExtensions + Certificate + CertificateVerify +
// Finished in TLS 1.3, which is what the FakeTLS noise must mimic.
func ProbeCertSize(hostname string, port int, count int) (CertProbeResult, error) {
	if count <= 0 {
		count = defaultProbeCount
	}

	addr := net.JoinHostPort(hostname, fmt.Sprintf("%d", port))
	sizes := make([]int, 0, count)

	for i := 0; i < count; i++ {
		size, err := probeSingle(addr, hostname)
		if err != nil {
			if len(sizes) > 0 {
				break // use what we have
			}

			return CertProbeResult{}, fmt.Errorf("probe %d failed: %w", i, err)
		}

		sizes = append(sizes, size)
	}

	if len(sizes) == 0 {
		return CertProbeResult{}, fmt.Errorf("no successful probes")
	}

	// Calculate mean and jitter (max deviation from mean).
	sum := 0
	for _, s := range sizes {
		sum += s
	}

	mean := sum / len(sizes)

	maxDev := 0
	for _, s := range sizes {
		d := s - mean
		if d < 0 {
			d = -d
		}

		if d > maxDev {
			maxDev = d
		}
	}

	// Ensure minimum jitter of 100 bytes for variability.
	if maxDev < 100 {
		maxDev = 100
	}

	return CertProbeResult{Mean: mean, Jitter: maxDev}, nil
}

// probeSingle does one TLS handshake and measures ApplicationData bytes
// received during the handshake.
func probeSingle(addr, hostname string) (int, error) {
	rawConn, err := net.DialTimeout("tcp", addr, probeDialTimeout)
	if err != nil {
		return 0, err
	}
	defer rawConn.Close()

	capture := &recordCapture{conn: rawConn}

	tlsConn := tls.Client(capture, &tls.Config{
		ServerName: hostname,
		MinVersion: tls.VersionTLS12,
	})
	tlsConn.SetDeadline(time.Now().Add(probeHandshakeTimeout)) //nolint: errcheck

	if err := tlsConn.Handshake(); err != nil {
		return 0, err
	}

	tlsConn.Close() //nolint: errcheck

	return capture.appDataBytes, nil
}

// recordCapture wraps a net.Conn and parses the raw TLS record stream to
// measure ApplicationData payload sizes sent by the server during handshake.
// It tracks record boundaries by maintaining a state machine over Read calls.
type recordCapture struct {
	conn         net.Conn
	mu           sync.Mutex
	appDataBytes int
	seenCCS      bool
	done         bool

	// Record boundary tracking for the read side.
	readRemaining int // bytes left in current record payload
	readHeaderBuf [5]byte
	readHeaderPos int
}

func (rc *recordCapture) Read(p []byte) (int, error) {
	n, err := rc.conn.Read(p)
	if n > 0 && !rc.done {
		rc.mu.Lock()
		rc.parseReadBytes(p[:n])
		rc.mu.Unlock()
	}

	return n, err
}

func (rc *recordCapture) parseReadBytes(data []byte) {
	for len(data) > 0 {
		if rc.readRemaining > 0 {
			// Consuming payload of current record.
			consume := rc.readRemaining
			if consume > len(data) {
				consume = len(data)
			}

			rc.readRemaining -= consume
			data = data[consume:]

			continue
		}

		// Accumulate header bytes (5 bytes per record).
		need := 5 - rc.readHeaderPos
		if need > len(data) {
			need = len(data)
		}

		copy(rc.readHeaderBuf[rc.readHeaderPos:], data[:need])
		rc.readHeaderPos += need
		data = data[need:]

		if rc.readHeaderPos < 5 {
			return // incomplete header
		}

		// Full header available.
		recordType := rc.readHeaderBuf[0]
		payloadLen := int(binary.BigEndian.Uint16(rc.readHeaderBuf[3:5]))
		rc.readHeaderPos = 0
		rc.readRemaining = payloadLen

		if recordType == tlsTypeChangeCipherSpec {
			rc.seenCCS = true
		} else if recordType == tlsTypeApplicationData && rc.seenCCS {
			rc.appDataBytes += payloadLen
		}
	}
}

func (rc *recordCapture) Write(p []byte) (int, error) {
	// After client writes post-CCS data, server handshake records are done.
	if rc.seenCCS && rc.appDataBytes > 0 {
		rc.done = true
	}

	return rc.conn.Write(p)
}

func (rc *recordCapture) Close() error                       { return rc.conn.Close() }
func (rc *recordCapture) LocalAddr() net.Addr                { return rc.conn.LocalAddr() }
func (rc *recordCapture) RemoteAddr() net.Addr               { return rc.conn.RemoteAddr() }
func (rc *recordCapture) SetDeadline(t time.Time) error      { return rc.conn.SetDeadline(t) }
func (rc *recordCapture) SetReadDeadline(t time.Time) error  { return rc.conn.SetReadDeadline(t) }
func (rc *recordCapture) SetWriteDeadline(t time.Time) error { return rc.conn.SetWriteDeadline(t) }

// Ensure recordCapture implements net.Conn.
var _ net.Conn = (*recordCapture)(nil)
