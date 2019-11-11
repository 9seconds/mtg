package api

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"time"
)

const (
	apiUserAgent   = "mtg"
	apiHTTPTimeout = 30 * time.Second
)

var httpClient = http.Client{
	Timeout: apiHTTPTimeout,
}

func request(url string) (io.ReadCloser, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		panic(err)
	}

	req.Header.Set("Accept", "text/plan")
	req.Header.Set("User-Agent", apiUserAgent)

	resp, err := httpClient.Do(req)
	if err != nil {
		if resp != nil {
			io.Copy(ioutil.Discard, resp.Body) // nolint: errcheck
			resp.Body.Close()
		}

		return nil, fmt.Errorf("cannot perform a request: %w", err)
	}

	return resp.Body, err
}
