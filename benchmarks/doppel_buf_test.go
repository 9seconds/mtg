package benchmarks

import (
	"fmt"
	"runtime"
	"runtime/debug"
	"sync"
	"testing"
	"time"
)

const (
	maxRecordSize = 16384 // tls.MaxRecordSize
	sizeHeader    = 5     // tls.SizeHeader
)

var sink byte

// stackGoroutineRealistic simulates doppel start() with realistic buffer USE.
// The key: merely declaring [16384]byte doesn't grow the stack. Actually
// writing into it (via copy in the write loop) triggers the lazy stack growth
// from 2KB -> 32KB.
func stackGoroutineRealistic(done <-chan struct{}, wg *sync.WaitGroup, payload []byte) {
	// goroutine 1: start() with 16KB stack buffer, actually used
	wg.Add(1)
	go func() {
		defer wg.Done()
		var buf [maxRecordSize]byte
		// Simulate the write path in doppel start():
		//   n, _ := c.p.writeStream.Read(buf[tls.SizeHeader : tls.SizeHeader+size])
		//   tls.WriteRecordInPlace(c.Conn, buf[:], n)
		copy(buf[sizeHeader:], payload)
		<-done
		runtime.KeepAlive(&buf)
	}()

	// goroutine 2: clock tick loop
	wg.Add(1)
	go func() {
		defer wg.Done()
		ticker := time.NewTicker(50 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-done:
				return
			case <-ticker.C:
			}
		}
	}()
}

var bufPool = sync.Pool{
	New: func() any {
		b := make([]byte, maxRecordSize)
		return &b
	},
}

// poolGoroutineRealistic simulates the same pair with pool-based buffer.
func poolGoroutineRealistic(done <-chan struct{}, wg *sync.WaitGroup, payload []byte) {
	// goroutine 1: start() with pooled buffer
	wg.Add(1)
	go func() {
		defer wg.Done()
		bp := bufPool.Get().(*[]byte)
		buf := *bp
		copy(buf[sizeHeader:], payload)
		defer bufPool.Put(bp)
		<-done
		runtime.KeepAlive(&buf)
	}()

	// goroutine 2: clock tick loop
	wg.Add(1)
	go func() {
		defer wg.Done()
		ticker := time.NewTicker(50 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-done:
				return
			case <-ticker.C:
			}
		}
	}()
}

// measureMem forces GC and returns MemStats.
func measureMem() runtime.MemStats {
	runtime.GC()
	runtime.GC()
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	return m
}

// TestDoppelStackGrowthMechanism demonstrates that [16384]byte on the goroutine
// stack only triggers growth when the buffer is ACTUALLY WRITTEN TO (not just
// declared). Go's lazy stack growth means the stack guard page must be hit.
func TestDoppelStackGrowthMechanism(t *testing.T) {
	debug.SetGCPercent(-1)
	defer debug.SetGCPercent(100)

	const N = 2000
	payload := make([]byte, 1400) // typical TLS payload
	for i := range payload {
		payload[i] = byte(i)
	}

	// Phase 1: goroutines that declare [16384]byte but only touch buf[0]
	{
		runtime.GC()
		time.Sleep(50 * time.Millisecond)
		before := measureMem()

		done := make(chan struct{})
		var wg sync.WaitGroup
		for i := 0; i < N; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				var buf [maxRecordSize]byte
				buf[0] = 1
				<-done
				runtime.KeepAlive(&buf)
			}()
		}
		time.Sleep(200 * time.Millisecond)
		after := measureMem()

		stackPerG := (after.StackInuse - before.StackInuse) / N
		t.Logf("DECLARE-ONLY: stack/goroutine = %d bytes (stack not grown)", stackPerG)

		close(done)
		wg.Wait()
	}

	runtime.GC()
	time.Sleep(100 * time.Millisecond)

	// Phase 2: goroutines that actually copy() into the buffer (realistic)
	{
		runtime.GC()
		time.Sleep(50 * time.Millisecond)
		before := measureMem()

		done := make(chan struct{})
		var wg sync.WaitGroup
		for i := 0; i < N; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				var buf [maxRecordSize]byte
				copy(buf[sizeHeader:], payload)
				<-done
				runtime.KeepAlive(&buf)
			}()
		}
		time.Sleep(200 * time.Millisecond)
		after := measureMem()

		stackPerG := (after.StackInuse - before.StackInuse) / N
		t.Logf("COPY-INTO:    stack/goroutine = %d bytes (stack grown to 32KB)", stackPerG)

		close(done)
		wg.Wait()
	}

	runtime.GC()
	time.Sleep(100 * time.Millisecond)

	// Phase 3: pool-based with copy (realistic alternative)
	{
		runtime.GC()
		time.Sleep(50 * time.Millisecond)
		before := measureMem()

		done := make(chan struct{})
		var wg sync.WaitGroup
		for i := 0; i < N; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				bp := bufPool.Get().(*[]byte)
				buf := *bp
				copy(buf[sizeHeader:], payload)
				defer bufPool.Put(bp)
				<-done
				runtime.KeepAlive(&buf)
			}()
		}
		time.Sleep(200 * time.Millisecond)
		after := measureMem()

		stackPerG := (after.StackInuse - before.StackInuse) / N
		heapPerG := (after.HeapInuse - before.HeapInuse) / N
		t.Logf("POOL-BASED:   stack/goroutine = %d bytes, heap/goroutine = %d bytes",
			stackPerG, heapPerG)

		close(done)
		wg.Wait()
	}
}

// TestDoppelCombinedOverhead measures the memory of the full doppel Conn pair
// (start goroutine + clock goroutine) at various concurrency levels.
// Uses realistic buffer usage pattern that triggers stack growth.
func TestDoppelCombinedOverhead(t *testing.T) {
	payload := make([]byte, 1400)
	for i := range payload {
		payload[i] = byte(i)
	}

	for _, n := range []int{500, 1000, 2000} {
		t.Run(fmt.Sprintf("N=%d", n), func(t *testing.T) {
			debug.SetGCPercent(-1)
			defer debug.SetGCPercent(100)

			// Stack-allocated approach (current code pattern)
			var stackTotal uint64
			{
				runtime.GC()
				time.Sleep(50 * time.Millisecond)
				before := measureMem()

				done := make(chan struct{})
				var wg sync.WaitGroup
				for i := 0; i < n; i++ {
					stackGoroutineRealistic(done, &wg, payload)
				}
				time.Sleep(200 * time.Millisecond)
				after := measureMem()

				stackMem := after.StackInuse - before.StackInuse
				heapMem := after.HeapInuse - before.HeapInuse
				stackTotal = stackMem + heapMem

				t.Logf("STACK: %d conns (2 goroutines each = %d goroutines)", n, n*2)
				t.Logf("  StackInuse: %d KB (%d bytes/conn)", stackMem/1024, stackMem/uint64(n))
				t.Logf("  HeapInuse:  %d KB (%d bytes/conn)", heapMem/1024, heapMem/uint64(n))
				t.Logf("  Total:      %d KB (%.1f MB)", (stackMem+heapMem)/1024,
					float64(stackMem+heapMem)/(1024*1024))

				close(done)
				wg.Wait()
			}

			runtime.GC()
			time.Sleep(100 * time.Millisecond)

			// Pool-based approach
			{
				runtime.GC()
				time.Sleep(50 * time.Millisecond)
				before := measureMem()

				done := make(chan struct{})
				var wg sync.WaitGroup
				for i := 0; i < n; i++ {
					poolGoroutineRealistic(done, &wg, payload)
				}
				time.Sleep(200 * time.Millisecond)
				after := measureMem()

				stackMem := after.StackInuse - before.StackInuse
				heapMem := after.HeapInuse - before.HeapInuse
				poolTotal := stackMem + heapMem

				t.Logf("POOL:  %d conns (2 goroutines each = %d goroutines)", n, n*2)
				t.Logf("  StackInuse: %d KB (%d bytes/conn)", stackMem/1024, stackMem/uint64(n))
				t.Logf("  HeapInuse:  %d KB (%d bytes/conn)", heapMem/1024, heapMem/uint64(n))
				t.Logf("  Total:      %d KB (%.1f MB)", (stackMem+heapMem)/1024,
					float64(stackMem+heapMem)/(1024*1024))

				savings := int64(stackTotal) - int64(poolTotal)
				t.Logf("SAVINGS: %d KB total (%d bytes/conn), %.0f%% reduction",
					savings/1024, savings/int64(n),
					float64(savings)/float64(stackTotal)*100)

				close(done)
				wg.Wait()
			}
		})
	}
}

// BenchmarkDoppelBufStack benchmarks goroutine pair lifecycle with stack buffer.
func BenchmarkDoppelBufStack(b *testing.B) {
	payload := make([]byte, 1400)
	for b.Loop() {
		done := make(chan struct{})
		var wg sync.WaitGroup
		stackGoroutineRealistic(done, &wg, payload)
		close(done)
		wg.Wait()
	}
}

// BenchmarkDoppelBufPool benchmarks goroutine pair lifecycle with pool buffer.
func BenchmarkDoppelBufPool(b *testing.B) {
	payload := make([]byte, 1400)
	for b.Loop() {
		done := make(chan struct{})
		var wg sync.WaitGroup
		poolGoroutineRealistic(done, &wg, payload)
		close(done)
		wg.Wait()
	}
}

// BenchmarkDoppelThroughputStack simulates write throughput with stack buffer.
func BenchmarkDoppelThroughputStack(b *testing.B) {
	payload := make([]byte, 1400)
	for i := range payload {
		payload[i] = byte(i)
	}
	b.SetBytes(int64(len(payload)))

	for b.Loop() {
		var buf [maxRecordSize]byte
		copy(buf[sizeHeader:], payload)
		sink = buf[sizeHeader]
	}
}

// BenchmarkDoppelThroughputPool simulates write throughput with pooled buffer.
func BenchmarkDoppelThroughputPool(b *testing.B) {
	payload := make([]byte, 1400)
	for i := range payload {
		payload[i] = byte(i)
	}
	b.SetBytes(int64(len(payload)))

	for b.Loop() {
		bp := bufPool.Get().(*[]byte)
		buf := *bp
		copy(buf[sizeHeader:], payload)
		sink = buf[sizeHeader]
		bufPool.Put(bp)
	}
}
