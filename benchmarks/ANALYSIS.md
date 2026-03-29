# Benchmark Analysis: Relay Buffer Size & Stack vs Pool

## Setup

- Platform: darwin/arm64, Apple M4, 10 cores
- Date: 2026-03-27
- All benchmarks run with `-count=3` for statistical consistency

## 1. Relay Buffer Size — Impact on Read Calls and Throughput

### Key finding: buffer size has NO measurable impact on throughput or read count

#### Test A: client→telegram (through TLS layer)

| Buffer | Throughput (MB/s) | Underlying Reads | Notes |
|--------|-------------------|------------------|-------|
| 4 KB   | 7,460-7,700       | 322              | |
| 8 KB   | 7,400-7,480       | 322              | |
| 16 KB  | 7,470-7,540       | 322              | |

**Result:** Identical read counts (322). Identical throughput within noise. As expected: tls.Conn.Read() reads from internal readBuf (bytes.Buffer), relay buffer size doesn't propagate to underlying socket.

#### Test B: telegram→client (raw TCP, no TLS)

| Buffer | Throughput (MB/s) | Underlying Reads | Notes |
|--------|-------------------|------------------|-------|
| 4 KB   | 1,946-1,950       | 1,281            | |
| 8 KB   | 1,942-1,946       | 1,281            | |
| 16 KB  | 1,935-1,948       | 1,281            | |

**Result:** Also identical read counts (1,281). Throughput identical.

**Why:** net.Pipe() delivers data synchronously — one Write() maps to exactly one Read(). The relay buffer size determines the *maximum* bytes per Read(), but Read() returns whatever the sender wrote. In real TCP, the kernel determines how much data is available per read(2) call based on:
- TCP receive window
- Nagle algorithm / TCP_NODELAY
- Congestion window
- How much data arrived before the read(2) call

The buffer size only matters when the kernel has MORE data than the buffer can hold. For Telegram traffic over internet (not localhost), individual TCP segments are typically 1.4 KB (MTU). The kernel may batch multiple segments, but rarely >64 KB.

#### Test C: Media download (burst vs MTU)

| Scenario | Buffer | Throughput (MB/s) | Reads |
|----------|--------|-------------------|-------|
| Burst    | 4 KB   | 12,033-12,674     | 1,281 |
| Burst    | 16 KB  | 12,679-12,751     | 1,281 |
| MTU      | 4 KB   | 2,816-2,848       | 7,184 |
| MTU      | 16 KB  | 2,833-2,856       | 7,184 |

**Key finding for MTU test:** Even with 1,460-byte chunks (simulating real TCP), read counts are identical (7,184) for all buffer sizes. This is because each chunk is smaller than even the 4 KB buffer, so buffer size doesn't matter.

Throughput difference between burst and MTU modes (~12 GB/s vs ~2.8 GB/s) comes from overhead of many small writes through net.Pipe(), not from buffer-related syscall counts.

#### Test C: Media upload (through TLS)

| Buffer | Throughput (MB/s) | Underlying Reads |
|--------|-------------------|------------------|
| 4 KB   | 7,630-7,644       | 322              |
| 16 KB  | 7,688-7,823       | 322              |

Same pattern as Test A. TLS layer absorbs the difference.

#### Test D: Small messages (200 bytes × 10,000)

| Direction | Buffer | Throughput (MB/s) | Reads |
|-----------|--------|-------------------|-------|
| tg→client | 4 KB   | 392-396           | 10,001 |
| tg→client | 16 KB  | 400-402           | 10,001 |
| client→tg | 4 KB   | 2,023-2,025       | 64    |
| client→tg | 16 KB  | 2,012-2,028       | 64    |

Small messages: all data is <200 bytes per write, buffer size is irrelevant.

### Conclusion on buffer size

**In practice, relay buffer size does not affect syscall count or throughput.** The argument "4 KB buffer = 4× more syscalls" assumes the kernel always has 16 KB of data ready and the application is the bottleneck. In reality:
1. **client→telegram:** TLS layer has its own readBuf; relay buffer reads from memory
2. **telegram→client:** Data arrives in network-determined chunks (typically ≤MTU); the buffer is almost never the limiting factor
3. **The only scenario where buffer size matters:** sustained high-bandwidth transfer where the kernel accumulates >4 KB between read(2) calls. This is possible for media downloads on fast networks, but the throughput impact is negligible compared to network latency.

---

## 2. Stack vs Pool Memory

### Key finding: stack-allocated 16 KB buffer causes 32 KB per goroutine; pool reduces this by 27-30×

| Approach | N=100 | N=500 | N=1000 | N=2000 |
|----------|-------|-------|--------|--------|
| Stack 16KB | 3.2 MB (32 KB/gor) | 16.4 MB (32 KB/gor) | 32.8 MB (32 KB/gor) | 65.5 MB (32 KB/gor) |
| Pool 16KB  | 0 MB | 0.03-0.1 MB | 0.4-0.8 MB | 2.1-2.4 MB |
| Pool 4KB   | 0 MB | 0-0.1 MB | 0.5-0.7 MB | 2.3-2.5 MB |

**Explanation:**
- Stack variant: Go runtime grows goroutine stack to 32 KB to fit the 16,379-byte array. Confirmed: exactly 32,768 bytes per goroutine consistently.
- Pool variant: Goroutine stack stays at default size (2-8 KB depending on frame). Buffer lives on heap, managed by pool.
- Savings at N=2000 (1000 connections × 2 pumps): **65.5 MB → 2.3 MB = 96.5% reduction in stack memory**

### Pool 16KB vs Pool 4KB

Surprisingly similar! Both have ~2.3 MB stack overhead at N=2000. The difference is in heap:
- Pool 16KB at N=2000: ~16-24 KB heap (pool allocations)
- Pool 4KB at N=2000: ~8-16 KB heap

The heap difference is small because sync.Pool is clever — it reuses buffers and GC cleans idle ones.

### Burst behavior (9seconds' concern about idle pool memory)

| Pool buf size | Idle heap after burst 1 | Active heap during burst 2 |
|---------------|------------------------|---------------------------|
| 4 KB          | 5.6-8.1 MB             | ~8 MB + 2.7 MB stack      |
| 16 KB         | 11.9-13.9 MB           | ~13 MB + 2.7 MB stack     |

**9seconds is partially right:** After a burst of 500 goroutines, pool holds ~6-14 MB of idle heap (depending on buffer size). This is memory that wouldn't exist with stack-allocated buffers (which are freed when goroutines exit).

However:
- This idle memory is released at the next GC cycle (sync.Pool is designed for this)
- During active connections, total memory is still lower: stack(2.7 MB) + heap(8-13 MB) = 10-16 MB vs stack-only 16-32 MB
- The idle overhead is transient; the stack overhead is permanent per goroutine

### Conclusion on stack vs pool

sync.Pool with relay buffers provides **massive stack memory savings** (96.5% at 1000 connections). The trade-off is temporary idle heap memory between connection bursts, but:
1. sync.Pool releases objects at GC
2. Total memory during active connections is still lower
3. Stack memory cannot be reclaimed while goroutine is alive; pool memory can

The 16 KB vs 4 KB pool buffer size makes negligible difference for memory — the savings come from moving the buffer off the stack entirely, not from making it smaller.

---

## 3. CPU Overhead — Stack vs Pool

### Key finding: zero measurable CPU impact from using sync.Pool

| Scenario | stack 16KB | pool 16KB | pool 4KB |
|----------|-----------|-----------|----------|
| Raw relay (10 MB) | 11,018 MB/s | 10,952 MB/s | 11,004 MB/s |
| TLS relay (10 MB) | 9,788 MB/s | 9,633 MB/s | 9,676 MB/s |

All values are within ±2% noise. No statistically significant difference.

### Isolated overhead:
- `sync.Pool.Get() + Put()` = **7.3 ns** per call (one-time per connection, not per read)
- Stack allocation of `[16379]byte` = **0.25 ns**
- Difference: 7 ns per connection. For a transfer lasting ~1,000,000 ns (1 ms), this is **0.0007%** overhead

### Conclusion on CPU

sync.Pool introduces no measurable CPU overhead for relay operations. The ~7 ns per Get/Put is amortized across the entire connection lifetime (millions of ns). Throughput is identical whether using stack-allocated or pool-allocated buffers of any size.
