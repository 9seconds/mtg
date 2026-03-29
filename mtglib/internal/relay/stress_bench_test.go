package relay

import (
	"fmt"
	"io"
	"net"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/9seconds/mtg/v2/mtglib/internal/tls"
)

// ============================================================
// Stress test: N concurrent connections, each transferring dataSize bytes.
// Measures total wall-clock time, aggregate throughput, peak memory, GC pauses.
// This is the closest simulation to real proxy load.
// ============================================================

type stressResult struct {
	totalBytes    int64
	wallTime      time.Duration
	gcPauseTotal  time.Duration
	numGC         uint32
	peakStackMB   float64
	peakHeapMB    float64
	peakTotalMB   float64
	throughputMBs float64
}

func runStressTest(b *testing.B, numConns int, dataPerConn int, getBuf func() []byte, putBuf func([]byte)) stressResult {
	b.Helper()

	// Force GC before measuring
	runtime.GC()
	runtime.GC()

	var memBefore runtime.MemStats
	runtime.ReadMemStats(&memBefore)

	var totalTransferred atomic.Int64
	var wg sync.WaitGroup

	start := time.Now()

	// Launch all connections concurrently
	for i := 0; i < numConns; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			serverConn, clientConn := net.Pipe()

			// Writer goroutine: send data
			go func() {
				data := make([]byte, 32*1024) // write in 32KB chunks
				written := 0
				for written < dataPerConn {
					toWrite := len(data)
					if dataPerConn-written < toWrite {
						toWrite = dataPerConn - written
					}
					n, err := serverConn.Write(data[:toWrite])
					written += n
					if err != nil {
						break
					}
				}
				serverConn.Close()
			}()

			// Reader goroutine (the relay pump simulation)
			buf := getBuf()
			n, _ := io.CopyBuffer(io.Discard, clientConn, buf)
			putBuf(buf)
			totalTransferred.Add(n)
			clientConn.Close()
		}()
	}

	wg.Wait()
	elapsed := time.Since(start)

	var memAfter runtime.MemStats
	runtime.ReadMemStats(&memAfter)

	gcPause := time.Duration(memAfter.PauseTotalNs-memBefore.PauseTotalNs) * time.Nanosecond
	numGC := memAfter.NumGC - memBefore.NumGC

	total := totalTransferred.Load()
	throughput := float64(total) / elapsed.Seconds() / (1024 * 1024)

	return stressResult{
		totalBytes:    total,
		wallTime:      elapsed,
		gcPauseTotal:  gcPause,
		numGC:         numGC,
		peakStackMB:   float64(memAfter.StackInuse) / (1024 * 1024),
		peakHeapMB:    float64(memAfter.HeapInuse) / (1024 * 1024),
		peakTotalMB:   float64(memAfter.StackInuse+memAfter.HeapInuse) / (1024 * 1024),
		throughputMBs: throughput,
	}
}

func reportStress(b *testing.B, r stressResult) {
	b.ReportMetric(r.throughputMBs, "MB/s")
	b.ReportMetric(r.peakStackMB, "peak_stack_MB")
	b.ReportMetric(r.peakHeapMB, "peak_heap_MB")
	b.ReportMetric(r.peakTotalMB, "peak_total_MB")
	b.ReportMetric(float64(r.gcPauseTotal.Microseconds()), "gc_pause_us")
	b.ReportMetric(float64(r.numGC), "gc_cycles")
}

// BenchmarkStress_ConcurrentRelays runs N concurrent relay pumps with different
// buffer strategies and measures aggregate throughput + memory + GC.
func BenchmarkStress_ConcurrentRelays(b *testing.B) {
	type bufStrategy struct {
		name   string
		getBuf func() []byte
		putBuf func([]byte)
	}

	pool16 := &sync.Pool{New: func() any { buf := make([]byte, tls.MaxRecordPayloadSize); return &buf }}
	pool4 := &sync.Pool{New: func() any { buf := make([]byte, 4096); return &buf }}

	strategies := []bufStrategy{
		{
			name:   "stack_16KB",
			getBuf: func() []byte { buf := make([]byte, tls.MaxRecordPayloadSize); return buf },
			putBuf: func([]byte) {},
		},
		{
			name:   "pool_16KB",
			getBuf: func() []byte { return *pool16.Get().(*[]byte) },
			putBuf: func(b []byte) { pool16.Put(&b) },
		},
		{
			name:   "pool_4KB",
			getBuf: func() []byte { return *pool4.Get().(*[]byte) },
			putBuf: func(b []byte) { pool4.Put(&b) },
		},
	}

	// Test scenarios
	type scenario struct {
		conns       int
		dataPerConn int
		label       string
	}

	scenarios := []scenario{
		{100, 10 * 1024 * 1024, "100conn_10MB"},   // 100 connections × 10 MB = 1 GB total
		{500, 10 * 1024 * 1024, "500conn_10MB"},   // 500 × 10 MB = 5 GB total
		{1000, 10 * 1024 * 1024, "1000conn_10MB"}, // 1000 × 10 MB = 10 GB total
		{2000, 1 * 1024 * 1024, "2000conn_1MB"},   // 2000 × 1 MB = 2 GB (many short conns)
		{500, 50 * 1024 * 1024, "500conn_50MB"},   // 500 × 50 MB = 25 GB (big files)
	}

	for _, sc := range scenarios {
		for _, strat := range strategies {
			name := fmt.Sprintf("%s/%s", sc.label, strat.name)
			getBuf := strat.getBuf
			putBuf := strat.putBuf
			sc := sc

			b.Run(name, func(b *testing.B) {
				for i := 0; i < b.N; i++ {
					r := runStressTest(b, sc.conns, sc.dataPerConn, getBuf, putBuf)
					reportStress(b, r)
				}
			})
		}
	}
}

// BenchmarkStress_PoolContention specifically tests sync.Pool under heavy
// concurrent access — many goroutines doing Get/Put rapidly.
func BenchmarkStress_PoolContention(b *testing.B) {
	pool := &sync.Pool{New: func() any { buf := make([]byte, tls.MaxRecordPayloadSize); return &buf }}

	for _, numWorkers := range []int{100, 500, 1000, 2000} {
		b.Run(fmt.Sprintf("workers=%d", numWorkers), func(b *testing.B) {
			b.RunParallel(func(pb *testing.PB) {
				for pb.Next() {
					bp := pool.Get().(*[]byte)
					// Simulate minimal work with the buffer
					(*bp)[0] = 1
					(*bp)[len(*bp)-1] = 1
					pool.Put(bp)
				}
			})
		})
	}
}

// BenchmarkStress_TinyPackets simulates massive amounts of tiny packets
// (chat messages, typing indicators, status updates, ACKs).
// Each connection sends many small writes — this maximizes per-read overhead.
func BenchmarkStress_TinyPackets(b *testing.B) {
	type bufStrategy struct {
		name   string
		getBuf func() []byte
		putBuf func([]byte)
	}

	pool16 := &sync.Pool{New: func() any { buf := make([]byte, tls.MaxRecordPayloadSize); return &buf }}
	pool4 := &sync.Pool{New: func() any { buf := make([]byte, 4096); return &buf }}

	strategies := []bufStrategy{
		{
			name:   "stack_16KB",
			getBuf: func() []byte { return make([]byte, tls.MaxRecordPayloadSize) },
			putBuf: func([]byte) {},
		},
		{
			name:   "pool_16KB",
			getBuf: func() []byte { return *pool16.Get().(*[]byte) },
			putBuf: func(b []byte) { pool16.Put(&b) },
		},
		{
			name:   "pool_4KB",
			getBuf: func() []byte { return *pool4.Get().(*[]byte) },
			putBuf: func(b []byte) { pool4.Put(&b) },
		},
	}

	type scenario struct {
		conns      int
		pktSize    int
		pktsPerConn int
		label      string
	}

	scenarios := []scenario{
		// Chat-like: 100 connections, 50K tiny packets each (50 bytes = typing indicator / small ACK)
		{100, 50, 50000, "100conn_50B_x50K"},
		// Heavy chat: 500 connections, 10K packets of 200 bytes
		{500, 200, 10000, "500conn_200B_x10K"},
		// Extreme: 1000 connections, 20K packets of 100 bytes each
		{1000, 100, 20000, "1000conn_100B_x20K"},
		// Burst of tiny: 2000 connections, 5K packets of 50 bytes
		{2000, 50, 5000, "2000conn_50B_x5K"},
	}

	for _, sc := range scenarios {
		for _, strat := range strategies {
			name := fmt.Sprintf("%s/%s", sc.label, strat.name)
			getBuf := strat.getBuf
			putBuf := strat.putBuf
			sc := sc

			b.Run(name, func(b *testing.B) {
				totalBytes := int64(sc.conns) * int64(sc.pktSize) * int64(sc.pktsPerConn)
				b.SetBytes(totalBytes)

				for i := 0; i < b.N; i++ {
					runtime.GC()
					var memBefore runtime.MemStats
					runtime.ReadMemStats(&memBefore)

					var totalRead atomic.Int64
					var totalReads atomic.Int64
					var wg sync.WaitGroup

					start := time.Now()

					for c := 0; c < sc.conns; c++ {
						wg.Add(1)
						go func() {
							defer wg.Done()
							serverConn, clientConn := net.Pipe()

							go func() {
								pkt := make([]byte, sc.pktSize)
								for p := 0; p < sc.pktsPerConn; p++ {
									serverConn.Write(pkt)
								}
								serverConn.Close()
							}()

							buf := getBuf()
							var reads int64
							for {
								n, err := clientConn.Read(buf)
								if n > 0 {
									totalRead.Add(int64(n))
									reads++
								}
								if err != nil {
									break
								}
							}
							putBuf(buf)
							totalReads.Add(reads)
							clientConn.Close()
						}()
					}

					wg.Wait()
					elapsed := time.Since(start)

					var memAfter runtime.MemStats
					runtime.ReadMemStats(&memAfter)

					throughput := float64(totalRead.Load()) / elapsed.Seconds() / (1024 * 1024)
					pps := float64(totalReads.Load()) / elapsed.Seconds()

					b.ReportMetric(throughput, "MB/s")
					b.ReportMetric(pps, "packets/s")
					b.ReportMetric(float64(totalReads.Load()), "total_reads")
					b.ReportMetric(float64(memAfter.StackInuse)/(1024*1024), "peak_stack_MB")
					b.ReportMetric(float64(memAfter.HeapInuse)/(1024*1024), "peak_heap_MB")
					b.ReportMetric(float64(memAfter.NumGC-memBefore.NumGC), "gc_cycles")
					b.ReportMetric(float64(memAfter.PauseTotalNs-memBefore.PauseTotalNs)/1000, "gc_pause_us")
				}
			})
		}
	}
}

// BenchmarkStress_GCPressure measures how GC behaves under load.
// Stack-allocated buffers don't create GC work; pool buffers do.
// This tests whether pool-induced GC pressure hurts throughput.
func BenchmarkStress_GCPressure(b *testing.B) {
	numConns := 500
	dataPerConn := 10 * 1024 * 1024

	pool16 := &sync.Pool{New: func() any { buf := make([]byte, tls.MaxRecordPayloadSize); return &buf }}

	b.Run("stack_16KB", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			runtime.GC()
			var memBefore runtime.MemStats
			runtime.ReadMemStats(&memBefore)

			r := runStressTest(b, numConns, dataPerConn, func() []byte {
				buf := make([]byte, tls.MaxRecordPayloadSize)
				return buf
			}, func([]byte) {})

			var memAfter runtime.MemStats
			runtime.ReadMemStats(&memAfter)

			b.ReportMetric(r.throughputMBs, "MB/s")
			b.ReportMetric(float64(memAfter.NumGC-memBefore.NumGC), "gc_cycles")
			b.ReportMetric(float64(memAfter.PauseTotalNs-memBefore.PauseTotalNs)/1000, "gc_pause_us")
			b.ReportMetric(float64(memAfter.StackInuse)/(1024*1024), "final_stack_MB")
			b.ReportMetric(float64(memAfter.HeapInuse)/(1024*1024), "final_heap_MB")
		}
	})

	b.Run("pool_16KB", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			runtime.GC()
			var memBefore runtime.MemStats
			runtime.ReadMemStats(&memBefore)

			r := runStressTest(b, numConns, dataPerConn, func() []byte {
				return *pool16.Get().(*[]byte)
			}, func(buf []byte) {
				pool16.Put(&buf)
			})

			var memAfter runtime.MemStats
			runtime.ReadMemStats(&memAfter)

			b.ReportMetric(r.throughputMBs, "MB/s")
			b.ReportMetric(float64(memAfter.NumGC-memBefore.NumGC), "gc_cycles")
			b.ReportMetric(float64(memAfter.PauseTotalNs-memBefore.PauseTotalNs)/1000, "gc_pause_us")
			b.ReportMetric(float64(memAfter.StackInuse)/(1024*1024), "final_stack_MB")
			b.ReportMetric(float64(memAfter.HeapInuse)/(1024*1024), "final_heap_MB")
		}
	})
}
