package relay

import (
	"fmt"
	"runtime"
	"sync"
	"testing"

	"github.com/dolonet/mtg-multi/mtglib/internal/tls"
)

// BenchmarkStackVsPool measures memory consumption when N goroutines hold
// either a stack-allocated buffer or a pool-allocated buffer.
// Each goroutine simulates one pump direction of a relay connection.
// Real connections have 2 pumps each, so N goroutines ≈ N/2 connections.

func BenchmarkStackMemory(b *testing.B) {
	for _, numGoroutines := range []int{100, 500, 1000, 2000} {
		b.Run(fmt.Sprintf("goroutines=%d", numGoroutines), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				var memBefore, memAfter runtime.MemStats

				runtime.GC()
				runtime.ReadMemStats(&memBefore)

				var wg sync.WaitGroup
				ready := make(chan struct{}, numGoroutines)
				stop := make(chan struct{})

				wg.Add(numGoroutines)
				for j := 0; j < numGoroutines; j++ {
					go blockingReadStack(&wg, ready, stop)
				}

				// Wait for all goroutines to be ready (holding their buffers)
				for j := 0; j < numGoroutines; j++ {
					<-ready
				}

				runtime.ReadMemStats(&memAfter)

				stackDelta := memAfter.StackInuse - memBefore.StackInuse
				heapDelta := memAfter.HeapInuse - memBefore.HeapInuse
				totalDelta := stackDelta + heapDelta

				b.ReportMetric(float64(stackDelta), "stack_bytes")
				b.ReportMetric(float64(heapDelta), "heap_bytes")
				b.ReportMetric(float64(totalDelta), "total_bytes")
				b.ReportMetric(float64(stackDelta)/float64(numGoroutines), "stack_per_goroutine")

				close(stop)
				wg.Wait()
			}
		})
	}
}

func BenchmarkPoolMemory_16KB(b *testing.B) {
	benchmarkPoolMemory(b, tls.MaxRecordPayloadSize)
}

func BenchmarkPoolMemory_4KB(b *testing.B) {
	benchmarkPoolMemory(b, 4096)
}

func benchmarkPoolMemory(b *testing.B, poolBufSize int) {
	b.Helper()

	pool := &sync.Pool{
		New: func() any {
			buf := make([]byte, poolBufSize)
			return &buf
		},
	}

	for _, numGoroutines := range []int{100, 500, 1000, 2000} {
		b.Run(fmt.Sprintf("goroutines=%d", numGoroutines), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				var memBefore, memAfter runtime.MemStats

				// Ensure pool is empty
				runtime.GC()
				runtime.ReadMemStats(&memBefore)

				var wg sync.WaitGroup
				ready := make(chan struct{}, numGoroutines)
				stop := make(chan struct{})

				wg.Add(numGoroutines)
				for j := 0; j < numGoroutines; j++ {
					go blockingReadPool(&wg, ready, stop, pool)
				}

				for j := 0; j < numGoroutines; j++ {
					<-ready
				}

				runtime.ReadMemStats(&memAfter)

				stackDelta := memAfter.StackInuse - memBefore.StackInuse
				heapDelta := memAfter.HeapInuse - memBefore.HeapInuse
				totalDelta := stackDelta + heapDelta

				b.ReportMetric(float64(stackDelta), "stack_bytes")
				b.ReportMetric(float64(heapDelta), "heap_bytes")
				b.ReportMetric(float64(totalDelta), "total_bytes")
				b.ReportMetric(float64(stackDelta)/float64(numGoroutines), "stack_per_goroutine")

				close(stop)
				wg.Wait()
			}
		})
	}
}

// BenchmarkPoolMemory_Burst tests the scenario 9seconds described:
// connections come in bursts, pool holds unused buffers between bursts.
func BenchmarkPoolMemory_Burst(b *testing.B) {
	for _, poolBufSize := range []int{4096, 16379} {
		b.Run(fmt.Sprintf("poolBuf=%d", poolBufSize), func(b *testing.B) {
			pool := &sync.Pool{
				New: func() any {
					buf := make([]byte, poolBufSize)
					return &buf
				},
			}

			for i := 0; i < b.N; i++ {
				// Burst 1: 500 goroutines
				var wg sync.WaitGroup
				ready := make(chan struct{}, 500)
				stop := make(chan struct{})

				wg.Add(500)
				for j := 0; j < 500; j++ {
					go blockingReadPool(&wg, ready, stop, pool)
				}
				for j := 0; j < 500; j++ {
					<-ready
				}
				close(stop)
				wg.Wait()

				// Between bursts: measure idle pool memory
				var memAfterBurst runtime.MemStats
				runtime.ReadMemStats(&memAfterBurst)

				// Burst 2: 500 goroutines again (pool should reuse)
				ready2 := make(chan struct{}, 500)
				stop2 := make(chan struct{})

				wg.Add(500)
				for j := 0; j < 500; j++ {
					go blockingReadPool(&wg, ready2, stop2, pool)
				}
				for j := 0; j < 500; j++ {
					<-ready2
				}

				var memDuringBurst2 runtime.MemStats
				runtime.ReadMemStats(&memDuringBurst2)

				b.ReportMetric(float64(memAfterBurst.HeapInuse), "idle_heap_bytes")
				b.ReportMetric(float64(memDuringBurst2.HeapInuse), "burst2_heap_bytes")
				b.ReportMetric(float64(memDuringBurst2.StackInuse), "burst2_stack_bytes")

				close(stop2)
				wg.Wait()
			}
		})
	}
}
