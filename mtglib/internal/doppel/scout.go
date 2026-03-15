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

type Scout struct {
	network Network
	urls    []string
}

func (s Scout) Learn(ctx context.Context) ([]time.Duration, error) {
	var durations []time.Duration

	for _, url := range s.urls {
		learned, err := s.learn(ctx, url)
		if err != nil {
			return nil, err
		}

		durations = append(durations, learned...)
	}

	return durations, nil
}

func (s Scout) learn(ctx context.Context, url string) ([]time.Duration, error) {
	client, results := s.makeClient()

	if !strings.HasPrefix(url, "https://") {
		return nil, fmt.Errorf("url %s must be https", url)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := client.Do(req)
	if resp != nil {
		io.Copy(io.Discard, resp.Body) //nolint: errcheck
		resp.Body.Close()              //nolint: errcheck
		client.CloseIdleConnections()
	}

	if err != nil || len(results.data) == 0 {
		return nil, err
	}

	durations := []time.Duration{}
	lastTimestamp := time.Time{}

	for i, v := range results.data {
		if v.recordType != tls.TypeApplicationData {
			continue
		}

		if lastTimestamp.IsZero() {
			if i > 0 {
				lastTimestamp = results.data[i-1].timestamp
			} else {
				lastTimestamp = v.timestamp
			}
		}

		durations = append(durations, v.timestamp.Sub(lastTimestamp))
		lastTimestamp = v.timestamp
	}

	return durations, nil
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
