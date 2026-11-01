package cli

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/9seconds/mtg/v2/essentials"
	"github.com/9seconds/mtg/v2/mtglib"
)

const (
	getIPTimeout = 5 * time.Second
)

var getIPServicesPlain = []string{
	"https://ifconfig.co",
	"https://ifconfig.me",
	"https://api.ipify.org",
	"https://ipecho.net/plain",
}

func getIP(ctx context.Context, ntw mtglib.Network, protocol string) (net.IP, error) {
	ctx, cancel := context.WithTimeout(ctx, getIPTimeout)
	defer cancel()

	ctx, cancelCause := context.WithCancelCause(ctx)
	defer cancelCause(nil)

	var ip net.IP

	rvChan := make(chan net.IP)
	errChan := make(chan error)
	errs := []error{}
	wg := &sync.WaitGroup{}
	dialer := ntw.NativeDialer()
	client := ntw.MakeHTTPClient(func(_ context.Context, network, address string) (essentials.Conn, error) {
		conn, err := dialer.DialContext(ctx, protocol, address)
		if err != nil {
			return nil, err
		}
		return essentials.WrapNetConn(conn), err
	})

	for _, url := range getIPServicesPlain {
		wg.Go(func() {
			lErrChan := errChan
			rChan := rvChan

			ip, err := getIPAddressPlain(ctx, client, url)
			if err == nil {
				lErrChan = nil
			} else {
				rChan = nil
			}

			select {
			case <-ctx.Done():
			case lErrChan <- fmt.Errorf("%s: %w", url, err):
			case rChan <- ip:
			}
		})
	}

	wg.Go(func() {
		defer cancelCause(nil)

		for {
			select {
			case <-ctx.Done():
				return
			case foundIP := <-rvChan:
				ip = foundIP
				return
			case err := <-errChan:
				errs = append(errs, err)
				if len(errs) == len(getIPServicesPlain) {
					cancelCause(fmt.Errorf(
						"cannot resolve %s address: %w",
						protocol,
						errors.Join(errs...),
					))
				}
			}
		}
	})

	wg.Wait()

	if ip != nil {
		return ip, nil
	}

	return nil, context.Cause(ctx)
}

func getIPAddressPlain(ctx context.Context, client *http.Client, address string) (net.IP, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, address, nil)
	if err != nil {
		panic(err)
	}

	req.Header.Add("Accept", "text/plain")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	defer func() {
		io.Copy(io.Discard, resp.Body) //nolint: errcheck
		resp.Body.Close()              //nolint: errcheck
	}()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	data = bytes.TrimSpace(data)
	ip := net.ParseIP(string(data))

	if ip == nil {
		return nil, errors.New("cannot parse as IP address")
	}

	return ip, nil
}
