package network

import "net/http"

type networkHTTPTransport struct {
	userAgent string
	next      http.RoundTripper
}

func (n networkHTTPTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Set("User-Agent", n.userAgent)

	return n.next.RoundTrip(req) //nolint: wrapcheck
}
