package api

import (
	"bufio"
	"fmt"
	"net"
	"regexp"
	"strconv"
	"strings"

	"github.com/9seconds/mtg/conntypes"
)

const (
	addressesURLV4 = "https://core.telegram.org/getProxyConfig"   // nolint: gas
	addressesURLV6 = "https://core.telegram.org/getProxyConfigV6" // nolint: gas
)

var addressesProxyForSplitter = regexp.MustCompile(`\s+`)

func AddressesV4() (map[conntypes.DC][]string, conntypes.DC, error) {
	return getAddresses(addressesURLV4)
}

func AddressesV6() (map[conntypes.DC][]string, conntypes.DC, error) {
	return getAddresses(addressesURLV6)
}

func getAddresses(url string) (map[conntypes.DC][]string, conntypes.DC, error) {
	resp, err := request(url)
	if err != nil {
		return nil, 0, fmt.Errorf("cannot get http response: %w", err)
	}

	defer resp.Close()

	scanner := bufio.NewScanner(resp)
	data := map[conntypes.DC][]string{}
	defaultDC := conntypes.DCDefaultIdx

	for scanner.Scan() {
		text := strings.TrimSpace(scanner.Text())

		switch {
		case strings.HasPrefix(text, "#"):
			continue
		case strings.HasPrefix(text, "proxy_for"):
			addr, idx, err := addressesParseProxyFor(text)
			if err != nil {
				return nil, 0, fmt.Errorf("cannot parse 'proxy_for' section: %w", err)
			}

			if addresses, ok := data[idx]; ok {
				data[idx] = append(addresses, addr)
			} else {
				data[idx] = []string{addr}
			}
		case strings.HasPrefix(text, "default"):
			idx, err := addressesParseDefault(text)
			if err != nil {
				return nil, 0, fmt.Errorf("cannot parse 'default' section: %w", err)
			}

			defaultDC = idx
		}
	}

	err = scanner.Err()
	if err != nil {
		return nil, 0, fmt.Errorf("cannot parse http response: %w", err)
	}

	return data, defaultDC, nil
}

func addressesParseProxyFor(text string) (string, conntypes.DC, error) {
	chunks := addressesProxyForSplitter.Split(text, 3)
	if len(chunks) != 3 || chunks[0] != "proxy_for" {
		return "", 0, fmt.Errorf("incorrect config %s", text)
	}

	dc, err := strconv.ParseInt(chunks[1], 10, 16)
	if err != nil {
		return "", 0, fmt.Errorf("incorrect config '%s': %w", text, err)
	}

	addr := strings.TrimRight(chunks[2], ";")
	if _, _, err = net.SplitHostPort(addr); err != nil {
		return "", 0, fmt.Errorf("incorrect config '%s': %w", text, err)
	}

	return addr, conntypes.DC(dc), nil
}

func addressesParseDefault(text string) (conntypes.DC, error) {
	chunks := addressesProxyForSplitter.Split(text, 2)
	if len(chunks) != 2 || chunks[0] != "default" {
		return 0, fmt.Errorf("incorrect config '%s'", text)
	}

	dcString := strings.TrimRight(chunks[1], ";")

	dc, err := strconv.ParseInt(dcString, 10, 16)
	if err != nil {
		return 0, fmt.Errorf("incorrect config '%s': %w", text, err)
	}

	return conntypes.DC(dc), nil
}
