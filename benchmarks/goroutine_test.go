package benchmarks

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"testing"
	"time"
)

// stableGoroutineCount returns the current goroutine count after forcing GC
// and giving the runtime a moment to settle.
func stableGoroutineCount() int {
	runtime.GC()
	runtime.Gosched()
	return runtime.NumGoroutine()
}

// memUsage returns StackInuse + HeapAlloc after GC, which gives a stable
// measurement of memory actually consumed by goroutines and their data.
func memUsage() uint64 {
	runtime.GC()
	runtime.GC() // two passes for more stability
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	return m.StackInuse + m.HeapAlloc
}

// -------------------------------------------------------
// 1. Memory cost of idle goroutines (blocked on channel)
// -------------------------------------------------------

func TestIdleGoroutineMemory(t *testing.T) {
	for _, n := range []int{1000, 2000, 5000, 10000} {
		t.Run(fmt.Sprintf("N=%d", n), func(t *testing.T) {
			blocker := make(chan struct{})
			var wg sync.WaitGroup

			// Let runtime settle before measuring
			runtime.GC()
			time.Sleep(10 * time.Millisecond)

			before := memUsage()
			goroutinesBefore := runtime.NumGoroutine()

			wg.Add(n)
			for i := 0; i < n; i++ {
				go func() {
					wg.Done()
					<-blocker
				}()
			}
			wg.Wait() // all goroutines are alive and blocked

			after := memUsage()
			goroutinesAfter := runtime.NumGoroutine()

			spawned := goroutinesAfter - goroutinesBefore
			totalBytes := int64(after) - int64(before)
			perGoroutine := float64(totalBytes) / float64(spawned)

			t.Logf("Spawned %d goroutines (idle, blocked on channel)", spawned)
			t.Logf("Total memory delta: %d bytes (%.2f KiB)", totalBytes, float64(totalBytes)/1024)
			t.Logf("Per goroutine: %.0f bytes (%.2f KiB)", perGoroutine, perGoroutine/1024)

			close(blocker)
			runtime.Gosched()
		})
	}
}

// -------------------------------------------------------
// 2. Memory cost of goroutines with grown stacks
// -------------------------------------------------------

//go:noinline
func growStack(depth int, blocker chan struct{}) {
	var buf [1024]byte // 1 KiB per frame
	_ = buf
	if depth > 0 {
		growStack(depth-1, blocker)
		return
	}
	<-blocker
}

func TestGrownStackGoroutineMemory(t *testing.T) {
	for _, n := range []int{1000, 2000, 5000} {
		t.Run(fmt.Sprintf("N=%d", n), func(t *testing.T) {
			blocker := make(chan struct{})
			ready := make(chan struct{})

			runtime.GC()
			time.Sleep(10 * time.Millisecond)
			before := memUsage()

			for i := 0; i < n; i++ {
				go func() {
					ready <- struct{}{}
					growStack(8, blocker) // ~8 KiB of stack frames
				}()
				<-ready
			}

			after := memUsage()
			totalBytes := int64(after) - int64(before)
			perGoroutine := float64(totalBytes) / float64(n)

			t.Logf("Spawned %d goroutines with grown stacks (~8 KiB frames)", n)
			t.Logf("Total memory delta: %d bytes (%.2f KiB)", totalBytes, float64(totalBytes)/1024)
			t.Logf("Per goroutine: %.0f bytes (%.2f KiB)", perGoroutine, perGoroutine/1024)

			close(blocker)
			runtime.Gosched()
		})
	}
}

// -------------------------------------------------------
// 3. Verify context.AfterFunc does NOT spawn goroutines
//    until context is cancelled
// -------------------------------------------------------

func TestAfterFuncNoGoroutineUntilCancel(t *testing.T) {
	const N = 1000

	goroutinesBefore := stableGoroutineCount()

	ctxs := make([]context.Context, N)
	cancels := make([]context.CancelFunc, N)
	stops := make([]func() bool, N)

	for i := 0; i < N; i++ {
		ctxs[i], cancels[i] = context.WithCancel(context.Background())
		stops[i] = context.AfterFunc(ctxs[i], func() {
			// noop callback
		})
	}

	goroutinesAfter := stableGoroutineCount()
	delta := goroutinesAfter - goroutinesBefore

	t.Logf("Registered %d AfterFunc callbacks", N)
	t.Logf("Goroutine delta BEFORE cancel: %d (should be 0 or near 0)", delta)

	if delta > 5 {
		t.Errorf("Expected ~0 extra goroutines before cancel, got %d", delta)
	}

	// Now cancel all contexts and check goroutines spike momentarily
	for i := 0; i < N; i++ {
		cancels[i]()
	}
	runtime.Gosched()
	goroutinesPostCancel := runtime.NumGoroutine()
	t.Logf("Goroutines right after cancelling %d contexts: %d (baseline was %d)",
		N, goroutinesPostCancel, goroutinesBefore)

	// Cleanup
	_ = stops
}

// -------------------------------------------------------
// 4. Memory comparison: N goroutines vs N AfterFunc
// -------------------------------------------------------

func TestMemoryGoroutinesVsAfterFunc(t *testing.T) {
	const N = 5000

	// --- Goroutines ---
	blocker := make(chan struct{})
	var wg sync.WaitGroup

	runtime.GC()
	time.Sleep(10 * time.Millisecond)
	beforeG := memUsage()

	wg.Add(N)
	for i := 0; i < N; i++ {
		ctx, cancel := context.WithCancel(context.Background())
		_ = cancel
		go func() {
			wg.Done()
			<-ctx.Done()
		}()
	}
	wg.Wait()
	afterG := memUsage()
	goroutineMemory := int64(afterG) - int64(beforeG)

	close(blocker)
	runtime.Gosched()
	time.Sleep(10 * time.Millisecond)

	// --- AfterFunc ---
	runtime.GC()
	time.Sleep(10 * time.Millisecond)
	beforeAF := memUsage()

	cancels := make([]context.CancelFunc, N)
	for i := 0; i < N; i++ {
		var cancel context.CancelFunc
		var ctx context.Context
		ctx, cancel = context.WithCancel(context.Background())
		cancels[i] = cancel
		context.AfterFunc(ctx, func() {})
	}
	afterAF := memUsage()
	afterFuncMemory := int64(afterAF) - int64(beforeAF)

	t.Logf("N = %d", N)
	t.Logf("Goroutine approach:    %d bytes total, %.0f bytes/each", goroutineMemory, float64(goroutineMemory)/N)
	t.Logf("AfterFunc approach:    %d bytes total, %.0f bytes/each", afterFuncMemory, float64(afterFuncMemory)/N)
	if goroutineMemory > 0 {
		t.Logf("Memory ratio (goroutine/AfterFunc): %.1fx", float64(goroutineMemory)/float64(afterFuncMemory))
	}

	// Cleanup
	for _, c := range cancels {
		c()
	}
}

// -------------------------------------------------------
// 5. Benchmark: idle goroutine vs context.AfterFunc
// -------------------------------------------------------

func BenchmarkIdleGoroutine(b *testing.B) {
	for i := 0; i < b.N; i++ {
		ctx, cancel := context.WithCancel(context.Background())
		done := make(chan struct{})
		go func() {
			<-ctx.Done()
			close(done)
		}()
		cancel()
		<-done
	}
}

func BenchmarkAfterFunc(b *testing.B) {
	for i := 0; i < b.N; i++ {
		ctx, cancel := context.WithCancel(context.Background())
		done := make(chan struct{})
		context.AfterFunc(ctx, func() {
			close(done)
		})
		cancel()
		<-done
	}
}

// -------------------------------------------------------
// 6. Projection: savings from replacing proxy.go:68-71
//    and relay.go:19-23 with context.AfterFunc
// -------------------------------------------------------

func TestProjectedSavings(t *testing.T) {
	// Measure per-goroutine cost with large sample
	const sampleSize = 5000
	blocker := make(chan struct{})
	var wg sync.WaitGroup

	runtime.GC()
	time.Sleep(10 * time.Millisecond)
	before := memUsage()

	wg.Add(sampleSize)
	for i := 0; i < sampleSize; i++ {
		go func() {
			wg.Done()
			<-blocker
		}()
	}
	wg.Wait()
	after := memUsage()
	close(blocker)

	perGoroutine := float64(int64(after)-int64(before)) / float64(sampleSize)

	t.Logf("=== Goroutine Audit per Connection ===")
	t.Logf("1. proxy.go:68-71     ctx.Done() -> Close()        [REPLACEABLE with AfterFunc]")
	t.Logf("2. relay.go:19-23     ctx.Done() -> close conns    [REPLACEABLE with AfterFunc]")
	t.Logf("3. relay.go:27-31     pump (client->telegram)      [NOT replaceable, does I/O]")
	t.Logf("4. doppel/conn.go:108 clock.Start()                [NOT replaceable, timer loop]")
	t.Logf("5. doppel/conn.go:111 start() write loop           [NOT replaceable, I/O loop]")
	t.Logf("")
	t.Logf("Total goroutines per connection: 5 (+ ServeConn from ants pool)")
	t.Logf("Replaceable with AfterFunc: 2")
	t.Logf("")
	t.Logf("Measured per-goroutine overhead: %.0f bytes (%.2f KiB)", perGoroutine, perGoroutine/1024)
	t.Logf("")

	for _, conns := range []int{1000, 2000} {
		saved := 2 * conns // 2 goroutines saved per connection
		savedBytes := float64(saved) * perGoroutine
		t.Logf("At %d connections:", conns)
		t.Logf("  Goroutines saved: %d", saved)
		t.Logf("  Memory saved: %.2f MiB", savedBytes/1024/1024)
		t.Logf("  Remaining goroutines: %d (3 per conn)", 3*conns)
	}

	t.Logf("")
	t.Logf("Note: domain fronting path also spawns relay goroutines,")
	t.Logf("but it's an alternative to the telegram relay, not additive.")
}
