// Relay server — the process we measure.
// Accepts TCP connections, connects to echo backend, relays bidirectionally.
// Exposes /metrics HTTP endpoint for monitoring.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

const (
	bufSize16K = 16379 // tls.MaxRecordPayloadSize
	bufSize4K  = 4096
)

// --- Buffer strategies ---

var pool16K = sync.Pool{New: func() any { b := make([]byte, bufSize16K); return &b }}
var pool4K = sync.Pool{New: func() any { b := make([]byte, bufSize4K); return &b }}

type strategy int

const (
	stratStack16K strategy = iota
	stratPool16K
	stratPool4K
)

func (s strategy) String() string {
	switch s {
	case stratStack16K:
		return "stack-16k"
	case stratPool16K:
		return "pool-16k"
	case stratPool4K:
		return "pool-4k"
	}
	return "unknown"
}

func parseStrategy(s string) strategy {
	switch s {
	case "stack-16k", "stack":
		return stratStack16K
	case "pool-16k", "pool":
		return stratPool16K
	case "pool-4k":
		return stratPool4K
	default:
		fmt.Fprintf(os.Stderr, "unknown strategy: %s (use stack-16k, pool-16k, pool-4k)\n", s)
		os.Exit(1)
		return 0
	}
}

// pump copies src→dst using the given strategy. Returns bytes copied.
func pump(strat strategy, dst, src net.Conn) int64 {
	var n int64
	var err error
	switch strat {
	case stratStack16K:
		var buf [bufSize16K]byte
		n, err = io.CopyBuffer(dst, src, buf[:])
	case stratPool16K:
		bp := pool16K.Get().(*[]byte)
		n, err = io.CopyBuffer(dst, src, *bp)
		pool16K.Put(bp)
	case stratPool4K:
		bp := pool4K.Get().(*[]byte)
		n, err = io.CopyBuffer(dst, src, *bp)
		pool4K.Put(bp)
	}
	_ = err
	return n
}

// --- Metrics ---

type metrics struct {
	ActiveConns  atomic.Int64
	TotalConns   atomic.Int64
	TotalBytes   atomic.Int64
	FailedConns  atomic.Int64
}

var m metrics

type metricsSnapshot struct {
	Strategy     string  `json:"strategy"`
	Uptime       string  `json:"uptime"`
	ActiveConns  int64   `json:"active_conns"`
	TotalConns   int64   `json:"total_conns"`
	TotalBytes   int64   `json:"total_bytes"`
	FailedConns  int64   `json:"failed_conns"`
	Goroutines   int     `json:"goroutines"`
	RSSKB        int64   `json:"rss_kb"`
	VmRSSKB      int64   `json:"vm_rss_kb"`
	StackInuse   uint64  `json:"stack_inuse_bytes"`
	HeapInuse    uint64  `json:"heap_inuse_bytes"`
	HeapAlloc    uint64  `json:"heap_alloc_bytes"`
	HeapSys      uint64  `json:"heap_sys_bytes"`
	StackSys     uint64  `json:"stack_sys_bytes"`
	Sys          uint64  `json:"sys_bytes"`
	NumGC        uint32  `json:"num_gc"`
	GCPauseTotalUs int64 `json:"gc_pause_total_us"`
	GOMAXPROCS   int     `json:"gomaxprocs"`
}

func readRSSKB() int64 {
	data, err := os.ReadFile("/proc/self/status")
	if err != nil {
		return -1
	}
	for _, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(line, "VmRSS:") {
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				v, _ := strconv.ParseInt(fields[1], 10, 64)
				return v
			}
		}
	}
	return -1
}

func getMetrics(strat strategy, startTime time.Time) metricsSnapshot {
	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)

	return metricsSnapshot{
		Strategy:       strat.String(),
		Uptime:         time.Since(startTime).Round(time.Second).String(),
		ActiveConns:    m.ActiveConns.Load(),
		TotalConns:     m.TotalConns.Load(),
		TotalBytes:     m.TotalBytes.Load(),
		FailedConns:    m.FailedConns.Load(),
		Goroutines:     runtime.NumGoroutine(),
		RSSKB:          readRSSKB(),
		VmRSSKB:        readRSSKB(),
		StackInuse:     ms.StackInuse,
		HeapInuse:      ms.HeapInuse,
		HeapAlloc:      ms.HeapAlloc,
		HeapSys:        ms.HeapSys,
		StackSys:       ms.StackSys,
		Sys:            ms.Sys,
		NumGC:          ms.NumGC,
		GCPauseTotalUs: int64(ms.PauseTotalNs / 1000),
		GOMAXPROCS:     runtime.GOMAXPROCS(0),
	}
}

// --- Connection handler ---

func handleConn(strat strategy, echoAddr string, conn net.Conn) {
	defer conn.Close()
	m.ActiveConns.Add(1)
	m.TotalConns.Add(1)
	defer m.ActiveConns.Add(-1)

	backend, err := net.DialTimeout("tcp", echoAddr, 10*time.Second)
	if err != nil {
		m.FailedConns.Add(1)
		return
	}
	defer backend.Close()

	done := make(chan struct{})
	go func() {
		defer close(done)
		n := pump(strat, backend, conn)
		m.TotalBytes.Add(n)
		conn.Close()
		backend.Close()
	}()

	n := pump(strat, conn, backend)
	m.TotalBytes.Add(n)
	conn.Close()
	backend.Close()
	<-done
}

// --- Metrics logger (writes to file every second) ---

func metricsLogger(ctx context.Context, strat strategy, startTime time.Time, logPath string) {
	f, err := os.Create(logPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "cannot create metrics log: %v\n", err)
		return
	}
	defer f.Close()

	// CSV header
	fmt.Fprintf(f, "time_s,active_conns,total_conns,total_bytes_mb,rss_kb,stack_inuse_kb,heap_inuse_kb,heap_alloc_kb,sys_kb,goroutines,num_gc,gc_pause_us,failed_conns\n")

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			snap := getMetrics(strat, startTime)
			elapsed := time.Since(startTime).Seconds()
			fmt.Fprintf(f, "%.0f,%d,%d,%.1f,%d,%d,%d,%d,%d,%d,%d,%d,%d\n",
				elapsed,
				snap.ActiveConns,
				snap.TotalConns,
				float64(snap.TotalBytes)/1024/1024,
				snap.RSSKB,
				snap.StackInuse/1024,
				snap.HeapInuse/1024,
				snap.HeapAlloc/1024,
				snap.Sys/1024,
				snap.Goroutines,
				snap.NumGC,
				snap.GCPauseTotalUs,
				snap.FailedConns,
			)
			f.Sync()
		}
	}
}

func main() {
	addr := flag.String("addr", "0.0.0.0:19998", "relay listen address")
	echoAddr := flag.String("echo", "72.56.22.248:19999", "echo server address")
	stratName := flag.String("strategy", "stack-16k", "buffer strategy: stack-16k, pool-16k, pool-4k")
	metricsAddr := flag.String("metrics", "0.0.0.0:19997", "HTTP metrics address")
	metricsLog := flag.String("metrics-log", "", "path to CSV metrics log file (optional)")
	flag.Parse()

	strat := parseStrategy(*stratName)
	startTime := time.Now()

	fmt.Printf("relay server: strategy=%s, listen=%s, echo=%s, metrics=%s\n",
		strat, *addr, *echoAddr, *metricsAddr)

	// HTTP metrics endpoint
	http.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {
		snap := getMetrics(strat, startTime)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(snap)
	})
	http.HandleFunc("/gc", func(w http.ResponseWriter, r *http.Request) {
		runtime.GC()
		fmt.Fprintf(w, "GC triggered\n")
	})
	http.HandleFunc("/reset", func(w http.ResponseWriter, r *http.Request) {
		m.TotalConns.Store(0)
		m.TotalBytes.Store(0)
		m.FailedConns.Store(0)
		fmt.Fprintf(w, "counters reset\n")
	})
	go http.ListenAndServe(*metricsAddr, nil)

	// Metrics logger
	if *metricsLog != "" {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		go metricsLogger(ctx, strat, startTime, *metricsLog)
	}

	// TCP listener
	ln, err := net.Listen("tcp", *addr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "listen: %v\n", err)
		os.Exit(1)
	}

	for {
		conn, err := ln.Accept()
		if err != nil {
			fmt.Fprintf(os.Stderr, "accept: %v\n", err)
			continue
		}
		go handleConn(strat, *echoAddr, conn)
	}
}
