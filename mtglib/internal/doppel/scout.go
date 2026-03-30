package doppel

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/9seconds/mtg/v2/essentials"
	"github.com/9seconds/mtg/v2/mtglib/internal/tls"
)

// ScoutResult holds measurements from a single scout HTTP request.
type ScoutResult struct {
	Durations []time.Duration
	CertSize  int // total ApplicationData bytes during TLS handshake; 0 if unknown
}

type Scout struct {
	network Network
	urls    []string
}

func (s Scout) Learn(ctx context.Context) (ScoutResult, error) {
	var combined ScoutResult

	for _, url := range s.urls {
		learned, err := s.learn(ctx, url)
		if err != nil {
			return ScoutResult{}, err
		}

		combined.Durations = append(combined.Durations, learned.Durations...)

		if learned.CertSize > 0 && combined.CertSize == 0 {
			combined.CertSize = learned.CertSize
		}
	}

	return combined, nil
}

func (s Scout) learn(ctx context.Context, url string) (ScoutResult, error) {
	client, results := s.makeClient()

	if !strings.HasPrefix(url, "https://") {
		return ScoutResult{}, fmt.Errorf("url %s must be https", url)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return ScoutResult{}, err
	}

	resp, err := client.Do(req)
	if resp != nil {
		io.Copy(io.Discard, resp.Body) //nolint: errcheck
		resp.Body.Close()              //nolint: errcheck
		client.CloseIdleConnections()
	}

	data, writeIndex := results.Snapshot()

	if err != nil || len(data) == 0 {
		return ScoutResult{}, err
	}

	var result ScoutResult

	// Compute inter-record durations (existing logic).
	lastTimestamp := time.Time{}

	for i, v := range data {
		if v.recordType != tls.TypeApplicationData {
			continue
		}

		if lastTimestamp.IsZero() {
			if i > 0 {
				lastTimestamp = data[i-1].timestamp
			} else {
				lastTimestamp = v.timestamp
			}
		}

		result.Durations = append(result.Durations, v.timestamp.Sub(lastTimestamp))
		lastTimestamp = v.timestamp
	}

	// Compute cert size: sum of ApplicationData payload between CCS and
	// the first client Write (which marks the end of server handshake).
	seenCCS := false
	boundary := writeIndex
	if boundary < 0 {
		boundary = len(data)
	}

	for i, v := range data {
		if i >= boundary {
			break
		}

		if v.recordType == tls.TypeChangeCipherSpec {
			seenCCS = true
			continue
		}

		if seenCCS && v.recordType == tls.TypeApplicationData {
			result.CertSize += v.payloadLen
		}
	}

	return result, nil
}

func (s Scout) makeClient() (*http.Client, *ScoutConnCollected) {
	dialer := s.network.NativeDialer()
	collected := NewScoutConnCollected()
	client := s.network.MakeHTTPClient(func(
		ctx context.Context,
		network string,
		address string,
	) (essentials.Conn, error) {
		conn, err := dialer.DialContext(ctx, network, address)
		if err != nil {
			return nil, err
		}

		return NewScoutConn(essentials.WrapNetConn(conn), collected), nil
	})

	return client, collected
}

func NewScout(network Network, urls []string) Scout {
	return Scout{
		network: network,
		urls:    urls,
	}
}
