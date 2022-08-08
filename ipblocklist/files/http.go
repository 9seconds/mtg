package files

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

type httpFile struct {
	http *http.Client
	url  string
}

func (h httpFile) Open(ctx context.Context) (io.ReadCloser, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, h.url, nil)
	if err != nil {
		panic(err)
	}

	response, err := h.http.Do(request)
	if err != nil {
		if response != nil {
			io.Copy(io.Discard, response.Body) //nolint: errcheck
			response.Body.Close()
		}

		return nil, fmt.Errorf("cannot get url %s: %w", h.url, err)
	}

	if response.StatusCode >= http.StatusBadRequest {
		return nil, fmt.Errorf("unexpected status code %d", response.StatusCode)
	}

	return response.Body, nil
}

func (h httpFile) String() string {
	return h.url
}

// NewHTTP returns a file abstraction for HTTP/HTTPS endpoint. You also need to
// provide a valid instance of [http.Client] to access it.
func NewHTTP(client *http.Client, endpoint string) (File, error) {
	if client == nil {
		return nil, ErrBadHTTPClient
	}

	parsed, err := url.Parse(endpoint)
	if err != nil {
		return nil, fmt.Errorf("incorrect url %s: %w", endpoint, err)
	}

	switch parsed.Scheme {
	case "http", "https":
	default:
		return nil, fmt.Errorf("unsupported url %s", endpoint)
	}

	return httpFile{
		http: client,
		url:  endpoint,
	}, nil
}
