package benchmarks

import (
	"bufio"
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"runtime"
	"testing"
	"time"
	"unsafe"
)

// =========================================================================
// 1. TLS connPayload: bufio.NewReaderSize(conn, 4096) + bytes.Buffer.Grow(4096)
// =========================================================================

// connPayload mirrors tls/conn.go's connPayload struct.
type connPayload struct {
	readBuf      bytes.Buffer
	connBuffered *bufio.Reader
	read         bool
	write        bool
}

func newConnPayload() *connPayload {
	p := &connPayload{
		connBuffered: bufio.NewReaderSize(nil, 4096),
		read:         true,
		write:        true,
	}
	p.readBuf.Grow(4096)
	return p
}

func BenchmarkTLSConnPayload(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = newConnPayload()
	}
}

func TestTLSConnPayloadHeapCost(t *testing.T) {
	const N = 1000
	runtime.GC()
	var m1, m2 runtime.MemStats
	runtime.ReadMemStats(&m1)

	payloads := make([]*connPayload, N)
	for i := 0; i < N; i++ {
		payloads[i] = newConnPayload()
	}

	runtime.ReadMemStats(&m2)
	totalBytes := m2.TotalAlloc - m1.TotalAlloc
	perConn := totalBytes / N

	fmt.Printf("\n=== TLS connPayload heap cost ===\n")
	fmt.Printf("  Struct size (shallow):    %d bytes\n", unsafe.Sizeof(connPayload{}))
	fmt.Printf("  bufio.Reader size:        %d bytes (struct) + 4096 (buf)\n", unsafe.Sizeof(bufio.Reader{}))
	fmt.Printf("  Total alloc for %d conns: %d bytes (%.1f KB)\n", N, totalBytes, float64(totalBytes)/1024)
	fmt.Printf("  Per connection:           %d bytes (%.1f KB)\n", perConn, float64(perConn)/1024)
	fmt.Printf("  At 1000 conns:            %.1f MB\n", float64(perConn)*1000/1024/1024)
	fmt.Printf("  At 2000 conns:            %.1f MB\n", float64(perConn)*2000/1024/1024)

	// Keep alive to prevent GC
	runtime.KeepAlive(payloads)
}

// =========================================================================
// 2. EventTraffic allocations
// =========================================================================

// eventBase mirrors mtglib/events.go
type eventBase struct {
	streamID  string
	timestamp time.Time
}

// EventTraffic mirrors mtglib/events.go
type EventTraffic struct {
	eventBase
	Traffic uint
	IsRead  bool
}

func NewEventTraffic(streamID string, traffic uint, isRead bool) EventTraffic {
	return EventTraffic{
		eventBase: eventBase{
			timestamp: time.Now(),
			streamID:  streamID,
		},
		Traffic: traffic,
		IsRead:  isRead,
	}
}

func BenchmarkEventTraffic(b *testing.B) {
	streamID := "dGVzdC1zdHJlYW0taWQ"
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = NewEventTraffic(streamID, 1024, true)
	}
}

// BenchmarkEventTrafficInterface tests if passing EventTraffic through an
// interface causes heap escape.
func BenchmarkEventTrafficInterface(b *testing.B) {
	streamID := "dGVzdC1zdHJlYW0taWQ"
	b.ReportAllocs()
	var sink interface{}
	for i := 0; i < b.N; i++ {
		sink = NewEventTraffic(streamID, 1024, true)
	}
	runtime.KeepAlive(sink)
}

func TestEventTrafficAllocRate(t *testing.T) {
	streamID := "dGVzdC1zdHJlYW0taWQ"
	const iterations = 100000

	runtime.GC()
	var m1, m2 runtime.MemStats
	runtime.ReadMemStats(&m1)

	var sink interface{}
	for i := 0; i < iterations; i++ {
		// Simulate what connTraffic.Read does: create event and pass to Send
		sink = NewEventTraffic(streamID, 1024, true)
	}

	runtime.ReadMemStats(&m2)
	totalBytes := m2.TotalAlloc - m1.TotalAlloc
	totalAllocs := m2.Mallocs - m1.Mallocs

	fmt.Printf("\n=== EventTraffic allocation rate ===\n")
	fmt.Printf("  Struct size:               %d bytes\n", unsafe.Sizeof(EventTraffic{}))
	fmt.Printf("  eventBase size:            %d bytes\n", unsafe.Sizeof(eventBase{}))
	fmt.Printf("  Total alloc for %d events: %d bytes (%.1f KB)\n", iterations, totalBytes, float64(totalBytes)/1024)
	fmt.Printf("  Per event:                 %d bytes\n", totalBytes/iterations)
	fmt.Printf("  Heap allocs:               %d (%.2f per event)\n", totalAllocs, float64(totalAllocs)/float64(iterations))
	fmt.Printf("  NOTE: Each Read+Write on a connection creates 2 events.\n")
	fmt.Printf("  At 1000 conns * 100 ops/s: %.1f MB/s event alloc\n",
		float64(totalBytes)/float64(iterations)*1000*100*2/1024/1024)
	fmt.Printf("  At 2000 conns * 100 ops/s: %.1f MB/s event alloc\n",
		float64(totalBytes)/float64(iterations)*2000*100*2/1024/1024)

	runtime.KeepAlive(sink)
}

// =========================================================================
// 3. connRewind buffer (bytes.Buffer for handshake recording)
// =========================================================================

func BenchmarkConnRewindBuffer(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		var buf bytes.Buffer
		// Simulate TLS ClientHello being recorded. Typical ClientHello
		// is 200-600 bytes; we use 512 as a representative size.
		data := make([]byte, 512)
		buf.Write(data)
		_ = buf.Bytes()
	}
}

func TestConnRewindBufferCost(t *testing.T) {
	// Measure bytes.Buffer overhead for various handshake sizes
	sizes := []int{256, 512, 768, 1024, 2048}

	fmt.Printf("\n=== connRewind buffer cost ===\n")
	fmt.Printf("  bytes.Buffer struct size: %d bytes\n", unsafe.Sizeof(bytes.Buffer{}))

	for _, size := range sizes {
		const N = 1000
		runtime.GC()
		var m1, m2 runtime.MemStats
		runtime.ReadMemStats(&m1)

		bufs := make([]bytes.Buffer, N)
		data := make([]byte, size)
		for i := 0; i < N; i++ {
			bufs[i].Write(data)
		}

		runtime.ReadMemStats(&m2)
		totalBytes := m2.TotalAlloc - m1.TotalAlloc
		// Subtract the cost of the data slice and bufs slice themselves
		perConn := totalBytes / N

		fmt.Printf("  Handshake %4d bytes -> buffer alloc per conn: %d bytes\n", size, perConn)
		runtime.KeepAlive(bufs)
	}

	// Estimate at connection scale with typical 512-byte handshake
	const typicalSize = 512
	const N = 1000
	runtime.GC()
	var m1, m2 runtime.MemStats
	runtime.ReadMemStats(&m1)

	bufs := make([]bytes.Buffer, N)
	data := make([]byte, typicalSize)
	for i := 0; i < N; i++ {
		bufs[i].Write(data)
	}

	runtime.ReadMemStats(&m2)
	perConn := (m2.TotalAlloc - m1.TotalAlloc) / N

	fmt.Printf("  At 1000 conns (512B handshake): %.1f MB\n", float64(perConn)*1000/1024/1024)
	fmt.Printf("  At 2000 conns (512B handshake): %.1f MB\n", float64(perConn)*2000/1024/1024)

	runtime.KeepAlive(bufs)
}

// =========================================================================
// 4. streamID generation: make([]byte, 16) + base64 encoding
// =========================================================================

const ConnectionIDBytesLength = 16

func generateStreamIDHeap() string {
	connIDBytes := make([]byte, ConnectionIDBytesLength) // heap alloc
	rand.Read(connIDBytes)                                //nolint: errcheck
	return base64.RawURLEncoding.EncodeToString(connIDBytes) // heap alloc
}

func generateStreamIDStack() string {
	var connIDBytes [ConnectionIDBytesLength]byte // stack
	rand.Read(connIDBytes[:])                     //nolint: errcheck
	return base64.RawURLEncoding.EncodeToString(connIDBytes[:])
}

func BenchmarkStreamIDHeap(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = generateStreamIDHeap()
	}
}

func BenchmarkStreamIDStack(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = generateStreamIDStack()
	}
}

func TestStreamIDAllocCost(t *testing.T) {
	const N = 10000

	// Heap version (current code)
	runtime.GC()
	var m1, m2 runtime.MemStats
	runtime.ReadMemStats(&m1)
	heapIDs := make([]string, N)
	for i := 0; i < N; i++ {
		heapIDs[i] = generateStreamIDHeap()
	}
	runtime.ReadMemStats(&m2)
	heapTotal := m2.TotalAlloc - m1.TotalAlloc
	heapPer := heapTotal / N

	// Stack version (proposed)
	runtime.GC()
	runtime.ReadMemStats(&m1)
	stackIDs := make([]string, N)
	for i := 0; i < N; i++ {
		stackIDs[i] = generateStreamIDStack()
	}
	runtime.ReadMemStats(&m2)
	stackTotal := m2.TotalAlloc - m1.TotalAlloc
	stackPer := stackTotal / N

	fmt.Printf("\n=== streamID generation cost ===\n")
	fmt.Printf("  Heap version (make([]byte,16) + base64):\n")
	fmt.Printf("    Per call:       %d bytes\n", heapPer)
	fmt.Printf("    At 1000 conns:  %.1f KB\n", float64(heapPer)*1000/1024)
	fmt.Printf("    At 2000 conns:  %.1f KB\n", float64(heapPer)*2000/1024)
	fmt.Printf("  Stack version (var buf [16]byte + base64):\n")
	fmt.Printf("    Per call:       %d bytes\n", stackPer)
	fmt.Printf("    At 1000 conns:  %.1f KB\n", float64(stackPer)*1000/1024)
	fmt.Printf("    At 2000 conns:  %.1f KB\n", float64(stackPer)*2000/1024)
	fmt.Printf("  Savings per call: %d bytes (%.0f%%)\n", heapPer-stackPer,
		float64(heapPer-stackPer)/float64(heapPer)*100)

	runtime.KeepAlive(heapIDs)
	runtime.KeepAlive(stackIDs)
}

// =========================================================================
// Combined summary
// =========================================================================

func TestCombinedSummary(t *testing.T) {
	const N = 1000

	// 1. TLS connPayload
	runtime.GC()
	var m1, m2 runtime.MemStats
	runtime.ReadMemStats(&m1)
	payloads := make([]*connPayload, N)
	for i := 0; i < N; i++ {
		payloads[i] = newConnPayload()
	}
	runtime.ReadMemStats(&m2)
	tlsPerConn := (m2.TotalAlloc - m1.TotalAlloc) / N

	// 2. connRewind (512 byte handshake)
	runtime.GC()
	runtime.ReadMemStats(&m1)
	bufs := make([]bytes.Buffer, N)
	data := make([]byte, 512)
	for i := 0; i < N; i++ {
		bufs[i].Write(data)
	}
	runtime.ReadMemStats(&m2)
	rewindPerConn := (m2.TotalAlloc - m1.TotalAlloc) / N

	// 3. streamID (heap)
	runtime.GC()
	runtime.ReadMemStats(&m1)
	ids := make([]string, N)
	for i := 0; i < N; i++ {
		ids[i] = generateStreamIDHeap()
	}
	runtime.ReadMemStats(&m2)
	streamIDPerConn := (m2.TotalAlloc - m1.TotalAlloc) / N

	// 4. EventTraffic per op (interface escape)
	runtime.GC()
	runtime.ReadMemStats(&m1)
	var sink interface{}
	for i := 0; i < N; i++ {
		sink = NewEventTraffic("test", 1024, true)
	}
	runtime.ReadMemStats(&m2)
	eventPer := (m2.TotalAlloc - m1.TotalAlloc) / N

	totalPerConn := tlsPerConn + rewindPerConn + streamIDPerConn

	fmt.Printf("\n")
	fmt.Printf("╔══════════════════════════════════════════════════════════╗\n")
	fmt.Printf("║          PER-CONNECTION ALLOCATION SUMMARY              ║\n")
	fmt.Printf("╠══════════════════════════════════════════════════════════╣\n")
	fmt.Printf("║ Component              │ Per Conn  │ 1000     │ 2000    ║\n")
	fmt.Printf("╠════════════════════════╪═══════════╪══════════╪═════════╣\n")
	fmt.Printf("║ TLS connPayload        │ %5d B   │ %5.1f MB │ %5.1f MB║\n",
		tlsPerConn, float64(tlsPerConn)*1000/1024/1024, float64(tlsPerConn)*2000/1024/1024)
	fmt.Printf("║ connRewind (512B hs)   │ %5d B   │ %5.1f MB │ %5.1f MB║\n",
		rewindPerConn, float64(rewindPerConn)*1000/1024/1024, float64(rewindPerConn)*2000/1024/1024)
	fmt.Printf("║ streamID generation    │ %5d B   │ %5.1f KB │ %5.1f KB║\n",
		streamIDPerConn, float64(streamIDPerConn)*1000/1024, float64(streamIDPerConn)*2000/1024)
	fmt.Printf("╠════════════════════════╪═══════════╪══════════╪═════════╣\n")
	fmt.Printf("║ TOTAL (one-time/conn)  │ %5d B   │ %5.1f MB │ %5.1f MB║\n",
		totalPerConn, float64(totalPerConn)*1000/1024/1024, float64(totalPerConn)*2000/1024/1024)
	fmt.Printf("╠════════════════════════╪═══════════╪══════════╪═════════╣\n")
	fmt.Printf("║ EventTraffic (per op)  │ %5d B   │  ongoing │ ongoing ║\n", eventPer)
	fmt.Printf("║   (rate at 100 ops/s)  │           │ %5.1f MB/s         ║\n",
		float64(eventPer)*1000*100*2/1024/1024)
	fmt.Printf("╚══════════════════════════════════════════════════════════╝\n")

	runtime.KeepAlive(payloads)
	runtime.KeepAlive(bufs)
	runtime.KeepAlive(ids)
	runtime.KeepAlive(sink)
}
