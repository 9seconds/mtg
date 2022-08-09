package api

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	apiUserAgent   = "github.com/9seconds/mtg"
	apiHTTPTimeout = 30 * time.Second
)

var httpClient = http.Client{
	Timeout: apiHTTPTimeout,
}

func request(url string) (io.ReadCloser, error) {
	ctx, cancel := context.WithTimeout(context.Background(), apiHTTPTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		panic(err)
	}

	req.Header.Set("Accept", "text/plan")
	req.Header.Set("User-Agent", apiUserAgent)

	resp, err := httpClient.Do(req)
	if err != nil {
		if resp != nil {
			io.Copy(io.Discard, resp.Body) //nolint: errcheck
			resp.Body.Close()
		}

		return nil, fmt.Errorf("cannot perform a request: %w", err)
	}

	return resp.Body, err //nolint: wrapcheck
}
