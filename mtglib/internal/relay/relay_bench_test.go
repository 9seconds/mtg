package relay

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/9seconds/mtg/v2/essentials"
	"github.com/9seconds/mtg/v2/mtglib/internal/tls"
)

// mockConn wraps a net.Conn to satisfy essentials.Conn.
type mockConn struct {
	net.Conn
}

func (m mockConn) CloseRead() error  { return nil }
func (m mockConn) CloseWrite() error { return nil }

// countingReader wraps an io.Reader and counts Read calls.
type countingReader struct {
	r     io.Reader
	calls atomic.Int64
}

func (c *countingReader) Read(p []byte) (int, error) {
	c.calls.Add(1)
	return c.r.Read(p)
}

// countingConn wraps essentials.Conn and counts Read calls on the underlying conn.
type countingConn struct {
	essentials.Conn
	readCalls atomic.Int64
}

func (c *countingConn) Read(p []byte) (int, error) {
	c.readCalls.Add(1)
	return c.Conn.Read(p)
}

// makeTLSRecord creates a single TLS application data record with the given payload.
func makeTLSRecord(payload []byte) []byte {
	rec := make([]byte, tls.SizeHeader+len(payload))
	rec[0] = tls.TypeApplicationData
	copy(rec[1:3], tls.TLSVersion[:])
	binary.BigEndian.PutUint16(rec[3:5], uint16(len(payload)))
	copy(rec[5:], payload)
	return rec
}

// makeTLSStream creates a stream of TLS records totaling approximately totalBytes of payload.
func makeTLSStream(totalBytes int, recordPayloadSize int) []byte {
	var buf bytes.Buffer
	payload := make([]byte, recordPayloadSize)
	rand.Read(payload)

	for buf.Len() < totalBytes+tls.SizeHeader {
		remaining := totalBytes - (buf.Len() - (buf.Len()/(recordPayloadSize+tls.SizeHeader))*tls.SizeHeader)
		if remaining <= 0 {
			break
		}
		pSize := recordPayloadSize
		if remaining < pSize {
			pSize = remaining
		}
		rec := makeTLSRecord(payload[:pSize])
		buf.Write(rec)
	}

	return buf.Bytes()
}

// makeXORCipher creates a simple AES-CTR cipher for obfuscation testing.
func makeXORCipher() cipher.Stream {
	key := make([]byte, 32)
	rand.Read(key)
	iv := make([]byte, aes.BlockSize)
	rand.Read(iv)
	block, _ := aes.NewCipher(key)
	return cipher.NewCTR(block, iv)
}

// obfuscatedConn mirrors the obfuscation layer: XOR on read.
type obfuscatedConn struct {
	essentials.Conn
	recvCipher cipher.Stream
}

func (c obfuscatedConn) Read(p []byte) (int, error) {
	n, err := c.Conn.Read(p)
	if err != nil {
		return n, err
	}
	c.recvCipher.XORKeyStream(p[:n], p[:n])
	return n, nil
}

// ============================================================
// Test A: client→telegram direction (through TLS layer)
// Relay buffer reads from tls.Conn.Read() → readBuf (memcpy).
// Buffer size should NOT affect underlying read calls.
// ============================================================

func BenchmarkClientToTelegram_TLSRead(b *testing.B) {
	for _, bufSize := range []int{4096, 8192, 16379} {
		b.Run(fmt.Sprintf("buf=%d", bufSize), func(b *testing.B) {
			// Create TLS stream: full records with max payload
			totalPayload := 10 * 1024 * 1024 // 10 MB
			stream := makeTLSStream(totalPayload, tls.MaxRecordPayloadSize)

			b.ResetTimer()
			b.SetBytes(int64(totalPayload))

			for i := 0; i < b.N; i++ {
				reader := bytes.NewReader(stream)
				counter := &countingReader{r: reader}

				// Simulate: raw tcp → tls.New(read=true)
				serverConn, clientConn := net.Pipe()
				mConn := mockConn{clientConn}
				tlsConn := tls.New(mConn, true, false)

				// Feed data in background
				go func() {
					io.Copy(serverConn, counter)
					serverConn.Close()
				}()

				buf := make([]byte, bufSize)
				io.CopyBuffer(io.Discard, tlsConn, buf)
				clientConn.Close()

				b.ReportMetric(float64(counter.calls.Load()), "underlying_reads")
			}
		})
	}
}

// ============================================================
// Test B: telegram→client direction (raw TCP, no TLS)
// Relay buffer directly determines read(2) size.
// Buffer size DOES affect read calls.
// ============================================================

func BenchmarkTelegramToClient_RawRead(b *testing.B) {
	for _, bufSize := range []int{4096, 8192, 16379} {
		b.Run(fmt.Sprintf("buf=%d", bufSize), func(b *testing.B) {
			totalPayload := 10 * 1024 * 1024 // 10 MB

			b.ResetTimer()
			b.SetBytes(int64(totalPayload))

			for i := 0; i < b.N; i++ {
				serverConn, clientConn := net.Pipe()
				mConn := mockConn{clientConn}

				cipherStream := makeXORCipher()
				obfConn := obfuscatedConn{Conn: mConn, recvCipher: cipherStream}

				// Wrap in counting at the raw conn level
				cc := &countingConn{Conn: mConn}
				obfConnCounted := obfuscatedConn{Conn: cc, recvCipher: cipherStream}

				_ = obfConn // unused, use counted version

				// Feed data
				data := make([]byte, totalPayload)
				rand.Read(data)

				go func() {
					// Encrypt before sending (to match obfuscation XOR)
					sendCipher := makeXORCipher()
					sendCipher.XORKeyStream(data, data)
					serverConn.Write(data)
					serverConn.Close()
				}()

				buf := make([]byte, bufSize)
				io.CopyBuffer(io.Discard, obfConnCounted, buf)
				clientConn.Close()

				b.ReportMetric(float64(cc.readCalls.Load()), "underlying_reads")
			}
		})
	}
}

// ============================================================
// Test C: Media/file streaming (10 MB burst and realistic MTU)
// ============================================================

// BenchmarkMediaDownload_Burst simulates downloading media from Telegram.
// telegram→client direction, data available in large chunks.
func BenchmarkMediaDownload_Burst(b *testing.B) {
	for _, bufSize := range []int{4096, 8192, 16379} {
		b.Run(fmt.Sprintf("buf=%d", bufSize), func(b *testing.B) {
			totalPayload := 10 * 1024 * 1024
			data := make([]byte, totalPayload)
			rand.Read(data)

			b.ResetTimer()
			b.SetBytes(int64(totalPayload))

			for i := 0; i < b.N; i++ {
				serverConn, clientConn := net.Pipe()
				cc := &countingConn{Conn: mockConn{clientConn}}

				go func() {
					serverConn.Write(data)
					serverConn.Close()
				}()

				buf := make([]byte, bufSize)
				io.CopyBuffer(io.Discard, cc, buf)
				clientConn.Close()

				b.ReportMetric(float64(cc.readCalls.Load()), "underlying_reads")
			}
		})
	}
}

// BenchmarkMediaDownload_MTU simulates realistic TCP behavior where data arrives
// in MTU-sized chunks (~1460 bytes per segment).
func BenchmarkMediaDownload_MTU(b *testing.B) {
	for _, bufSize := range []int{4096, 8192, 16379} {
		b.Run(fmt.Sprintf("buf=%d", bufSize), func(b *testing.B) {
			totalPayload := 10 * 1024 * 1024
			mtuSize := 1460

			b.ResetTimer()
			b.SetBytes(int64(totalPayload))

			for i := 0; i < b.N; i++ {
				serverConn, clientConn := net.Pipe()
				cc := &countingConn{Conn: mockConn{clientConn}}

				go func() {
					data := make([]byte, mtuSize)
					rand.Read(data)
					written := 0
					for written < totalPayload {
						toWrite := mtuSize
						if totalPayload-written < toWrite {
							toWrite = totalPayload - written
						}
						serverConn.Write(data[:toWrite])
						written += toWrite
					}
					serverConn.Close()
				}()

				buf := make([]byte, bufSize)
				io.CopyBuffer(io.Discard, cc, buf)
				clientConn.Close()

				b.ReportMetric(float64(cc.readCalls.Load()), "underlying_reads")
			}
		})
	}
}

// BenchmarkMediaUpload_TLS simulates uploading media through the TLS layer
// (client→telegram direction). Buffer size should not matter.
func BenchmarkMediaUpload_TLS(b *testing.B) {
	for _, bufSize := range []int{4096, 8192, 16379} {
		b.Run(fmt.Sprintf("buf=%d", bufSize), func(b *testing.B) {
			totalPayload := 10 * 1024 * 1024
			stream := makeTLSStream(totalPayload, tls.MaxRecordPayloadSize)

			b.ResetTimer()
			b.SetBytes(int64(totalPayload))

			for i := 0; i < b.N; i++ {
				reader := bytes.NewReader(stream)
				counter := &countingReader{r: reader}

				serverConn, clientConn := net.Pipe()
				mConn := mockConn{clientConn}
				tlsConn := tls.New(mConn, true, false)

				go func() {
					io.Copy(serverConn, counter)
					serverConn.Close()
				}()

				buf := make([]byte, bufSize)
				io.CopyBuffer(io.Discard, tlsConn, buf)
				clientConn.Close()

				b.ReportMetric(float64(counter.calls.Load()), "underlying_reads")
			}
		})
	}
}

// ============================================================
// Test D: Small messages (chat traffic)
// ============================================================

func BenchmarkSmallMessages_TelegramToClient(b *testing.B) {
	for _, bufSize := range []int{4096, 8192, 16379} {
		b.Run(fmt.Sprintf("buf=%d", bufSize), func(b *testing.B) {
			// 10000 messages of 200 bytes each = 2 MB
			msgSize := 200
			numMsgs := 10000
			totalPayload := msgSize * numMsgs

			b.ResetTimer()
			b.SetBytes(int64(totalPayload))

			for i := 0; i < b.N; i++ {
				serverConn, clientConn := net.Pipe()
				cc := &countingConn{Conn: mockConn{clientConn}}

				go func() {
					msg := make([]byte, msgSize)
					rand.Read(msg)
					for j := 0; j < numMsgs; j++ {
						serverConn.Write(msg)
					}
					serverConn.Close()
				}()

				buf := make([]byte, bufSize)
				io.CopyBuffer(io.Discard, cc, buf)
				clientConn.Close()

				b.ReportMetric(float64(cc.readCalls.Load()), "underlying_reads")
			}
		})
	}
}

func BenchmarkSmallMessages_ClientToTelegram(b *testing.B) {
	for _, bufSize := range []int{4096, 8192, 16379} {
		b.Run(fmt.Sprintf("buf=%d", bufSize), func(b *testing.B) {
			msgSize := 200
			numMsgs := 10000
			totalPayload := msgSize * numMsgs

			// Wrap small messages in TLS records
			var streamBuf bytes.Buffer
			msg := make([]byte, msgSize)
			rand.Read(msg)
			for j := 0; j < numMsgs; j++ {
				streamBuf.Write(makeTLSRecord(msg))
			}
			stream := streamBuf.Bytes()

			b.ResetTimer()
			b.SetBytes(int64(totalPayload))

			for i := 0; i < b.N; i++ {
				reader := bytes.NewReader(stream)
				counter := &countingReader{r: reader}

				serverConn, clientConn := net.Pipe()
				mConn := mockConn{clientConn}
				tlsConn := tls.New(mConn, true, false)

				go func() {
					io.Copy(serverConn, counter)
					serverConn.Close()
				}()

				buf := make([]byte, bufSize)
				io.CopyBuffer(io.Discard, tlsConn, buf)
				clientConn.Close()

				b.ReportMetric(float64(counter.calls.Load()), "underlying_reads")
			}
		})
	}
}

// ============================================================
// CPU overhead benchmarks: stack vs pool allocation
// ============================================================

// BenchmarkCPU_StackVsPool_Relay measures the CPU overhead of using sync.Pool
// vs stack-allocated buffers in a realistic relay scenario.
// This is the core question: does Pool.Get/Put add measurable CPU cost?
func BenchmarkCPU_StackVsPool_Relay(b *testing.B) {
	totalPayload := 10 * 1024 * 1024 // 10 MB

	b.Run("stack_16KB", func(b *testing.B) {
		b.SetBytes(int64(totalPayload))
		for i := 0; i < b.N; i++ {
			serverConn, clientConn := net.Pipe()
			go func() {
				data := make([]byte, totalPayload)
				serverConn.Write(data)
				serverConn.Close()
			}()
			var buf [tls.MaxRecordPayloadSize]byte
			io.CopyBuffer(io.Discard, clientConn, buf[:])
			clientConn.Close()
		}
	})

	pool16 := &sync.Pool{New: func() any { b := make([]byte, tls.MaxRecordPayloadSize); return &b }}

	b.Run("pool_16KB", func(b *testing.B) {
		b.SetBytes(int64(totalPayload))
		for i := 0; i < b.N; i++ {
			serverConn, clientConn := net.Pipe()
			go func() {
				data := make([]byte, totalPayload)
				serverConn.Write(data)
				serverConn.Close()
			}()
			bp := pool16.Get().(*[]byte)
			io.CopyBuffer(io.Discard, clientConn, *bp)
			pool16.Put(bp)
			clientConn.Close()
		}
	})

	pool4 := &sync.Pool{New: func() any { b := make([]byte, 4096); return &b }}

	b.Run("pool_4KB", func(b *testing.B) {
		b.SetBytes(int64(totalPayload))
		for i := 0; i < b.N; i++ {
			serverConn, clientConn := net.Pipe()
			go func() {
				data := make([]byte, totalPayload)
				serverConn.Write(data)
				serverConn.Close()
			}()
			bp := pool4.Get().(*[]byte)
			io.CopyBuffer(io.Discard, clientConn, *bp)
			pool4.Put(bp)
			clientConn.Close()
		}
	})
}

// BenchmarkCPU_PoolGetPut measures the raw overhead of sync.Pool.Get/Put
// operations (without any I/O), to isolate pool machinery cost.
func BenchmarkCPU_PoolGetPut(b *testing.B) {
	pool := &sync.Pool{New: func() any { buf := make([]byte, tls.MaxRecordPayloadSize); return &buf }}

	// Warm up the pool
	items := make([]*[]byte, 100)
	for i := range items {
		items[i] = pool.Get().(*[]byte)
	}
	for _, item := range items {
		pool.Put(item)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		bp := pool.Get().(*[]byte)
		pool.Put(bp)
	}
}

// BenchmarkCPU_StackAlloc measures the cost of stack-allocating the buffer.
func BenchmarkCPU_StackAlloc(b *testing.B) {
	for i := 0; i < b.N; i++ {
		var buf [tls.MaxRecordPayloadSize]byte
		sinkByte = buf[0]
		sinkByte = buf[len(buf)-1]
	}
}

// BenchmarkCPU_TLSRelay_StackVsPool measures CPU for the full TLS path
// (client→telegram direction) with stack vs pool buffers.
func BenchmarkCPU_TLSRelay_StackVsPool(b *testing.B) {
	totalPayload := 10 * 1024 * 1024
	stream := makeTLSStream(totalPayload, tls.MaxRecordPayloadSize)

	b.Run("stack_16KB", func(b *testing.B) {
		b.SetBytes(int64(totalPayload))
		for i := 0; i < b.N; i++ {
			reader := bytes.NewReader(stream)
			serverConn, clientConn := net.Pipe()
			tlsConn := tls.New(mockConn{clientConn}, true, false)
			go func() {
				io.Copy(serverConn, reader)
				serverConn.Close()
			}()
			var buf [tls.MaxRecordPayloadSize]byte
			io.CopyBuffer(io.Discard, tlsConn, buf[:])
			clientConn.Close()
		}
	})

	pool16 := &sync.Pool{New: func() any { b := make([]byte, tls.MaxRecordPayloadSize); return &b }}

	b.Run("pool_16KB", func(b *testing.B) {
		b.SetBytes(int64(totalPayload))
		for i := 0; i < b.N; i++ {
			reader := bytes.NewReader(stream)
			serverConn, clientConn := net.Pipe()
			tlsConn := tls.New(mockConn{clientConn}, true, false)
			go func() {
				io.Copy(serverConn, reader)
				serverConn.Close()
			}()
			bp := pool16.Get().(*[]byte)
			io.CopyBuffer(io.Discard, tlsConn, *bp)
			pool16.Put(bp)
			clientConn.Close()
		}
	})

	pool4 := &sync.Pool{New: func() any { b := make([]byte, 4096); return &b }}

	b.Run("pool_4KB", func(b *testing.B) {
		b.SetBytes(int64(totalPayload))
		for i := 0; i < b.N; i++ {
			reader := bytes.NewReader(stream)
			serverConn, clientConn := net.Pipe()
			tlsConn := tls.New(mockConn{clientConn}, true, false)
			go func() {
				io.Copy(serverConn, reader)
				serverConn.Close()
			}()
			bp := pool4.Get().(*[]byte)
			io.CopyBuffer(io.Discard, tlsConn, *bp)
			pool4.Put(bp)
			clientConn.Close()
		}
	})
}

// ============================================================
// Concurrent memory measurement helpers for stack_bench_test.go
// ============================================================

var sinkByte byte // prevent compiler optimization

// blockingRead simulates a long-lived relay pump with stack buffer.
func blockingReadStack(wg *sync.WaitGroup, ready chan struct{}, stop chan struct{}) {
	defer wg.Done()
	var buf [tls.MaxRecordPayloadSize]byte
	sinkByte = buf[0] // ensure buf is used
	ready <- struct{}{}
	<-stop
	sinkByte = buf[len(buf)-1]
}

// blockingReadPool simulates relay pump with pooled buffer.
func blockingReadPool(wg *sync.WaitGroup, ready chan struct{}, stop chan struct{}, pool *sync.Pool) {
	defer wg.Done()
	bp := pool.Get().(*[]byte)
	defer pool.Put(bp)
	sinkByte = (*bp)[0]
	ready <- struct{}{}
	<-stop
	sinkByte = (*bp)[len(*bp)-1]
}
