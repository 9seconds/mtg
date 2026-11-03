// Package dcprobe verifies that a TCP endpoint is a real Telegram DC by
// performing the unauthenticated first step of the MTProto handshake
// (req_pq_multi -> resPQ) on top of mtg's existing obfuscated2 transport.
//
// No auth_key is generated; no long-lived state is introduced. Two TL
// messages, one round-trip. A generic listener cannot fake the reply
// because it must echo back our random nonce in resPQ.
//
// References:
//   - https://core.telegram.org/mtproto/auth_key      (handshake step 1)
//   - https://core.telegram.org/schema/mtproto        (TL schema)
//   - https://core.telegram.org/mtproto/mtproto-transports#padded-intermediate
package dcprobe

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"time"

	"github.com/9seconds/mtg/v2/essentials"
	"github.com/9seconds/mtg/v2/mtglib/obfuscation"
)

// MTProto wire constants (https://core.telegram.org/schema/mtproto).
//
//	req_pq_multi#be7e8ef1 nonce:int128 = ResPQ;
//	resPQ#05162463 nonce:int128 server_nonce:int128 pq:string
//	    server_public_key_fingerprints:Vector<long> = ResPQ;
const (
	ctorReqPQMulti uint32 = 0xbe7e8ef1
	ctorResPQ      uint32 = 0x05162463

	// Minimum legal resPQ frame: 20-byte unencrypted-message envelope +
	// 4-byte ctor + 16-byte nonce echo. Anything below cannot be a resPQ.
	minResPQFrame = 20 + 4 + 16
	// Upper bound: real resPQ replies are ~84 bytes (envelope + ~64-byte
	// payload). 256 is comfortable headroom; anything beyond is hostile or
	// not Telegram.
	maxResPQFrame = 256
)

// Probe sends req_pq_multi over an obfuscated2 + padded-intermediate transport
// and verifies that the peer replies with a matching resPQ.
//
// conn must be a freshly opened reliable byte stream (typically TCP) to a
// Telegram DC, but a SOCKS/proxy-wrapped net.Conn works just as well — Probe
// adapts whatever it gets to the half-close interface mtg's obfuscator
// requires. Probe does NOT close conn — the caller does. dc is the DC number
// (1..5) that gets baked into the obfuscated2 handshake frame.
//
// The returned duration is the round-trip from "first byte sent after the
// obfs handshake" to "resPQ frame fully read".
func Probe(ctx context.Context, conn net.Conn, dc int) (time.Duration, error) {
	if deadline, ok := ctx.Deadline(); ok {
		_ = conn.SetDeadline(deadline)
		defer func() { _ = conn.SetDeadline(time.Time{}) }()
	}

	// Honour ctx cancellation as well as its deadline: a parent ctx that is
	// canceled (without an earlier deadline expiring) would otherwise let
	// Probe block on an in-flight Read until the deadline. Forcing the
	// deadline to "now" makes the next syscall return an i/o timeout error
	// that Probe wraps and surfaces.
	stop := context.AfterFunc(ctx, func() {
		_ = conn.SetDeadline(time.Now())
	})
	defer stop()

	// 1. obfuscated2 handshake. Empty Secret = no MTProxy secret mixing,
	// which is how mtg itself talks to a DC (see mtglib/proxy.go).
	obfsConn, err := obfuscation.Obfuscator{}.SendHandshake(adaptConn(conn), dc)
	if err != nil {
		return 0, fmt.Errorf("obfuscated2 handshake: %w", err)
	}

	// 2. build req_pq_multi TL payload: 4-byte LE constructor + 16-byte nonce.
	var nonce [16]byte
	if _, err := rand.Read(nonce[:]); err != nil {
		return 0, fmt.Errorf("read nonce: %w", err)
	}
	tlBody := make([]byte, 4+16)
	binary.LittleEndian.PutUint32(tlBody[:4], ctorReqPQMulti)
	copy(tlBody[4:], nonce[:])

	// 3. wrap in an MTProto unencrypted message envelope (per
	// https://core.telegram.org/mtproto/description#unencrypted-message):
	//   auth_key_id:long(=0) | message_id:long | message_data_length:int | message_data:bytes
	// Without this envelope the DC silently drops the connection.
	msg := make([]byte, 8+8+4+len(tlBody))
	// auth_key_id = 0 (already zeroed by make)
	binary.LittleEndian.PutUint64(msg[8:16], generateMessageID())
	binary.LittleEndian.PutUint32(msg[16:20], uint32(len(tlBody)))
	copy(msg[20:], tlBody)

	// 4. wrap in a padded-intermediate frame: length(LE) + msg.
	// Padding is allowed [0..15] but not required when len(msg) % 4 == 0.
	frame := make([]byte, 4+len(msg))
	binary.LittleEndian.PutUint32(frame[:4], uint32(len(msg)))
	copy(frame[4:], msg)

	start := time.Now()
	if _, err := obfsConn.Write(frame); err != nil {
		return 0, fmt.Errorf("write req_pq_multi: %w", err)
	}

	// 5. read padded-intermediate reply: length, then that many bytes.
	// The reply is itself an MTProto unencrypted message (same envelope as
	// what we sent), so we must skip 20 bytes to get to the resPQ TL.
	var lenBuf [4]byte
	if _, err := io.ReadFull(obfsConn, lenBuf[:]); err != nil {
		return 0, fmt.Errorf("read frame length: %w", err)
	}
	respLen := binary.LittleEndian.Uint32(lenBuf[:])
	if respLen < minResPQFrame {
		return 0, fmt.Errorf("%w: resPQ frame too short (%d bytes)", ErrNotTelegram, respLen)
	}
	if respLen > maxResPQFrame {
		return 0, fmt.Errorf("%w: resPQ frame too large (%d bytes, max %d)", ErrNotTelegram, respLen, maxResPQFrame)
	}
	resp := make([]byte, respLen)
	if _, err := io.ReadFull(obfsConn, resp); err != nil {
		return 0, fmt.Errorf("read resPQ frame: %w", err)
	}
	rtt := time.Since(start)

	// 6. unwrap the MTProto envelope: skip auth_key_id(8) + message_id(8) +
	// message_data_length(4) = 20 bytes.
	tlResp := resp[20:]

	// 7. verify constructor and nonce echo. We deliberately do not parse
	// server_nonce, pq, or fingerprints — they are not needed to prove
	// the peer can speak MTProto.
	if got := binary.LittleEndian.Uint32(tlResp[:4]); got != ctorResPQ {
		return rtt, fmt.Errorf("%w: got constructor 0x%08x, want resPQ 0x%08x", ErrNotTelegram, got, ctorResPQ)
	}
	if !bytes.Equal(tlResp[4:4+16], nonce[:]) {
		return rtt, fmt.Errorf("%w: nonce echo mismatch", ErrNotTelegram)
	}

	return rtt, nil
}

// generateMessageID returns an MTProto message_id roughly synchronized with
// server time, with the lower 2 bits cleared (client-to-server requests).
// See https://core.telegram.org/mtproto/description#message-identifier-msg-id.
func generateMessageID() uint64 {
	nano := uint64(time.Now().UnixNano())
	sec := nano / 1_000_000_000
	nsInSec := nano % 1_000_000_000
	subsec := (nsInSec << 32) / 1_000_000_000
	id := (sec << 32) | subsec
	return id &^ 3
}

// ErrNotTelegram is returned (wrapped) when the peer's reply is not a
// well-formed resPQ matching our nonce. Use errors.Is to distinguish
// "the TCP connection was OK but the peer is not a Telegram DC" from
// transport errors.
var ErrNotTelegram = errors.New("peer did not respond with a matching resPQ")

// adaptConn returns conn as essentials.Conn if it already satisfies the
// interface (typically *net.TCPConn), otherwise wraps it with no-op
// CloseRead/CloseWrite. mtg's obfuscator only ever calls Read/Write/Close,
// so the no-ops are safe.
func adaptConn(conn net.Conn) essentials.Conn {
	if ec, ok := conn.(essentials.Conn); ok {
		return ec
	}
	return halfCloseShim{Conn: conn}
}

type halfCloseShim struct{ net.Conn }

func (halfCloseShim) CloseRead() error  { return nil }
func (halfCloseShim) CloseWrite() error { return nil }
