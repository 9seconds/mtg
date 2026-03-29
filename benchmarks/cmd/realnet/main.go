package main

import (
	"crypto/rand"
	"flag"
	"fmt"
	"io"
	"net"
	"os"
	"runtime"
	"runtime/debug"
	"sync"
	"sync/atomic"
	"time"
)

const (
	maxRecordPayloadSize = 16379
	maxRecordSize        = 16384
)

// --- Buffer strategies ---

type bufStrategy interface {
	Name() string
	Pump(src, dst net.Conn) (int64, error)
}

// Stack-allocated buffer (current mtg code)
type stackStrategy struct{}

func (stackStrategy) Name() string { return "stack" }

func (stackStrategy) Pump(src, dst net.Conn) (int64, error) {
	var buf [maxRecordPayloadSize]byte
	return io.CopyBuffer(dst, src, buf[:])
}

// Pool-allocated buffer
var relayPool = sync.Pool{
	New: func() any {
		b := make([]byte, maxRecordPayloadSize)
		return &b
	},
}

type poolStrategy struct{}

func (poolStrategy) Name() string { return "pool" }

func (poolStrategy) Pump(src, dst net.Conn) (int64, error) {
	bp := relayPool.Get().(*[]byte)
	defer relayPool.Put(bp)
	return io.CopyBuffer(dst, src, *bp)
}

// --- Memory measurement ---

type memSnapshot struct {
	StackInuse uint64
	HeapInuse  uint64
	HeapAlloc  uint64
	NumGC      uint32
	PauseTotalNs uint64
	NumGoroutine int
}

func snapMem() memSnapshot {
	runtime.GC()
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	return memSnapshot{
		StackInuse:   m.StackInuse,
		HeapInuse:    m.HeapInuse,
		HeapAlloc:    m.HeapAlloc,
		NumGC:        m.NumGC,
		PauseTotalNs: m.PauseTotalNs,
		NumGoroutine: runtime.NumGoroutine(),
	}
}

// --- Test harness ---

func runTest(strat bufStrategy, conns int, dataPerConn int64, reportInterval time.Duration) {
	fmt.Printf("\n=== %s strategy, %d connections, %s per conn ===\n",
		strat.Name(), conns, formatBytes(dataPerConn))

	// Start "telegram" echo servers - one listener, accepts all
	echoLn, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		fmt.Fprintf(os.Stderr, "echo listen: %v\n", err)
		return
	}
	defer echoLn.Close()
	echoAddr := echoLn.Addr().String()

	// Echo server goroutines
	var echoWg sync.WaitGroup
	go func() {
		for {
			c, err := echoLn.Accept()
			if err != nil {
				return
			}
			echoWg.Add(1)
			go func(c net.Conn) {
				defer echoWg.Done()
				defer c.Close()
				io.Copy(c, c) //nolint: errcheck
			}(c)
		}
	}()

	// Start relay listener
	relayLn, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		fmt.Fprintf(os.Stderr, "relay listen: %v\n", err)
		return
	}
	defer relayLn.Close()
	relayAddr := relayLn.Addr().String()

	// Relay server
	var relayWg sync.WaitGroup
	go func() {
		for {
			client, err := relayLn.Accept()
			if err != nil {
				return
			}
			relayWg.Add(1)
			go func(client net.Conn) {
				defer relayWg.Done()
				defer client.Close()

				tg, err := net.Dial("tcp", echoAddr)
				if err != nil {
					return
				}
				defer tg.Close()

				// Bidirectional relay (like mtg relay.Relay)
				done := make(chan struct{})
				go func() {
					defer close(done)
					strat.Pump(client, tg) //nolint: errcheck
					// When one direction is done, close both to unblock the other
					client.Close() //nolint: errcheck
					tg.Close()     //nolint: errcheck
				}()
				strat.Pump(tg, client) //nolint: errcheck
				client.Close() //nolint: errcheck
				tg.Close()     //nolint: errcheck
				<-done
			}(client)
		}
	}()

	// Force GC and take baseline
	debug.SetGCPercent(100)
	runtime.GC()
	runtime.GC()
	time.Sleep(50 * time.Millisecond)
	before := snapMem()

	// Launch clients
	var (
		totalBytes  atomic.Int64
		clientWg    sync.WaitGroup
		startSignal = make(chan struct{})
		peakMem     atomic.Uint64
	)

	// Memory sampler
	samplerDone := make(chan struct{})
	samplerStopped := make(chan struct{})
	go func() {
		defer close(samplerStopped)
		ticker := time.NewTicker(10 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-samplerDone:
				return
			case <-ticker.C:
				var m runtime.MemStats
				runtime.ReadMemStats(&m)
				total := m.StackInuse + m.HeapInuse
				for {
					old := peakMem.Load()
					if total <= old || peakMem.CompareAndSwap(old, total) {
						break
					}
				}
			}
		}
	}()

	for i := 0; i < conns; i++ {
		clientWg.Add(1)
		go func() {
			defer clientWg.Done()
			<-startSignal

			conn, err := net.Dial("tcp", relayAddr)
			if err != nil {
				fmt.Fprintf(os.Stderr, "client dial: %v\n", err)
				return
			}
			defer conn.Close()

			// Write data in chunks, read it back (echo)
			chunk := make([]byte, 4096)
			rand.Read(chunk) //nolint: errcheck
			readBuf := make([]byte, 4096)

			var written int64
			for written < dataPerConn {
				toWrite := int64(len(chunk))
				if written+toWrite > dataPerConn {
					toWrite = dataPerConn - written
				}
				n, err := conn.Write(chunk[:toWrite])
				if err != nil {
					return
				}
				written += int64(n)

				// Read back echo
				remaining := n
				for remaining > 0 {
					rn, err := conn.Read(readBuf)
					if err != nil {
						return
					}
					remaining -= rn
				}
				totalBytes.Add(int64(n * 2)) // write + read
			}
		}()
	}

	start := time.Now()
	close(startSignal)

	// Progress reporter
	reporterDone := make(chan struct{})
	if reportInterval > 0 {
		go func() {
			ticker := time.NewTicker(reportInterval)
			defer ticker.Stop()
			for {
				select {
				case <-reporterDone:
					return
				case <-ticker.C:
					elapsed := time.Since(start)
					bytes := totalBytes.Load()
					fmt.Printf("  [%.1fs] %s transferred, %.1f MB/s\n",
						elapsed.Seconds(), formatBytes(bytes),
						float64(bytes)/elapsed.Seconds()/1024/1024)
				}
			}
		}()
	}

	clientWg.Wait()
	close(reporterDone)
	elapsed := time.Since(start)

	// Stop sampler
	close(samplerDone)
	<-samplerStopped

	after := snapMem()

	// Results
	bytes := totalBytes.Load()
	throughput := float64(bytes) / elapsed.Seconds() / 1024 / 1024

	gcCycles := after.NumGC - before.NumGC
	gcPause := time.Duration(after.PauseTotalNs - before.PauseTotalNs)

	peak := peakMem.Load()
	baseMem := before.StackInuse + before.HeapInuse

	fmt.Printf("\nResults:\n")
	fmt.Printf("  Duration:       %v\n", elapsed.Round(time.Millisecond))
	fmt.Printf("  Total data:     %s\n", formatBytes(bytes))
	fmt.Printf("  Throughput:     %.1f MB/s\n", throughput)
	fmt.Printf("  Peak memory:    %s (baseline %s, delta %s)\n",
		formatBytes(int64(peak)), formatBytes(int64(baseMem)),
		formatBytes(int64(peak)-int64(baseMem)))
	fmt.Printf("  Stack (before): %s → (after): %s\n",
		formatBytes(int64(before.StackInuse)), formatBytes(int64(after.StackInuse)))
	fmt.Printf("  Heap  (before): %s → (after): %s\n",
		formatBytes(int64(before.HeapInuse)), formatBytes(int64(after.HeapInuse)))
	fmt.Printf("  Goroutines:     %d → %d\n", before.NumGoroutine, after.NumGoroutine)
	fmt.Printf("  GC cycles:      %d\n", gcCycles)
	fmt.Printf("  GC total pause: %v\n", gcPause)
	if gcCycles > 0 {
		fmt.Printf("  GC avg pause:   %v\n", gcPause/time.Duration(gcCycles))
	}

	// Cleanup
	relayLn.Close()
	echoLn.Close()
	relayWg.Wait()
	echoWg.Wait()
	runtime.GC()
	time.Sleep(100 * time.Millisecond)
}

func formatBytes(b int64) string {
	switch {
	case b >= 1024*1024*1024:
		return fmt.Sprintf("%.1f GB", float64(b)/1024/1024/1024)
	case b >= 1024*1024:
		return fmt.Sprintf("%.1f MB", float64(b)/1024/1024)
	case b >= 1024:
		return fmt.Sprintf("%.1f KB", float64(b)/1024)
	default:
		return fmt.Sprintf("%d B", b)
	}
}

func main() {
	conns := flag.Int("conns", 500, "number of concurrent connections")
	dataMB := flag.Int("data", 1, "MB of data per connection")
	strategy := flag.String("strategy", "both", "buffer strategy: stack, pool, or both")
	flag.Parse()

	dataPerConn := int64(*dataMB) * 1024 * 1024

	fmt.Printf("Real network relay benchmark\n")
	fmt.Printf("GOMAXPROCS=%d, OS=%s/%s\n", runtime.GOMAXPROCS(0), runtime.GOOS, runtime.GOARCH)
	fmt.Printf("Connections: %d, Data per conn: %s\n\n", *conns, formatBytes(dataPerConn))

	switch *strategy {
	case "stack":
		runTest(stackStrategy{}, *conns, dataPerConn, 2*time.Second)
	case "pool":
		runTest(poolStrategy{}, *conns, dataPerConn, 2*time.Second)
	case "both":
		runTest(stackStrategy{}, *conns, dataPerConn, 2*time.Second)
		fmt.Println("\n" + "============================================================")
		runTest(poolStrategy{}, *conns, dataPerConn, 2*time.Second)
	}
}
