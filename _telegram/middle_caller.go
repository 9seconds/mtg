package telegram

import (
	"bufio"
	"context"
	"io/ioutil"
	"net"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/juju/errors"
	"go.uber.org/zap"

	"github.com/9seconds/mtg/mtproto"
	"github.com/9seconds/mtg/wrappers"
)

const (
	middleTelegramAutoUpdateInterval = 6 * time.Hour
	middleTelegramHTTPClientTimeout  = 30 * time.Second

	tgAddrProxySecret = "https://core.telegram.org/getProxySecret"   // nolint: gas
	tgAddrProxyV4     = "https://core.telegram.org/getProxyConfig"   // nolint: gas
	tgAddrProxyV6     = "https://core.telegram.org/getProxyConfigV6" // nolint: gas
	tgUserAgent       = "mtg"
)

var middleTelegramProxyConfigSplitter = regexp.MustCompile(`\s+`)

type middleTelegramCaller struct {
	baseTelegram

	proxySecret []byte
	dialerMutex *sync.RWMutex
	httpClient  *http.Client
}

func (t *middleTelegramCaller) Dial(ctx context.Context, cancel context.CancelFunc, connID string,
	connOpts *mtproto.ConnectionOpts) (wrappers.StreamReadWriteCloser, error) {
	dc := connOpts.DC
	if dc == 0 {
		dc = 1
	}
	t.dialerMutex.RLock()
	defer t.dialerMutex.RUnlock()

	return t.baseTelegram.dial(ctx, cancel, dc, connID, connOpts.ConnectionProto)
}

func (t *middleTelegramCaller) autoUpdate() {
	for range time.Tick(middleTelegramAutoUpdateInterval) {
		if err := t.update(); err != nil {
			zap.S().Warnw("Cannot update from Telegram", "error", err)
		}
	}
}

func (t *middleTelegramCaller) update() error {
	secret, err := t.getTelegramProxySecret()
	if err != nil {
		return errors.Annotate(err, "Cannot get proxy secret")
	}

	v4Addresses, v4DefaultIdx, err := t.getTelegramAddresses(tgAddrProxyV4)
	if err != nil {
		return errors.Annotate(err, "Cannot get ipv4 addresses")
	}

	v6Addresses, v6DefaultIdx, err := t.getTelegramAddresses(tgAddrProxyV6)
	if err != nil {
		return errors.Annotate(err, "Cannot get ipv6 addresses")
	}

	t.dialerMutex.Lock()
	t.proxySecret = secret
	t.v4DefaultIdx = v4DefaultIdx
	t.v6DefaultIdx = v6DefaultIdx
	t.v4Addresses = v4Addresses
	t.v6Addresses = v6Addresses
	t.dialerMutex.Unlock()

	zap.S().Infow("Telegram middle proxy data has been updated")

	return nil
}

func (t *middleTelegramCaller) getTelegramProxySecret() ([]byte, error) {
	resp, err := t.call(tgAddrProxySecret)
	if err != nil {
		return nil, errors.Annotate(err, "Cannot access telegram server")
	}
	defer resp.Body.Close() // nolint: errcheck

	secret, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Annotate(err, "Cannot read response")
	}

	return secret, nil
}

func (t *middleTelegramCaller) getTelegramAddresses(url string) (map[int16][]string, int16, error) { // nolint: gocyclo
	resp, err := t.call(url)
	if err != nil {
		return nil, 0, errors.Annotate(err, "Cannot access telegram server")
	}
	defer resp.Body.Close() // nolint: errcheck

	scanner := bufio.NewScanner(resp.Body)
	data := map[int16][]string{}

	var defaultIdx int16 = 1
	for scanner.Scan() {
		text := strings.TrimSpace(scanner.Text())
		switch {
		case strings.HasPrefix(text, "#"):
			continue
		case strings.HasPrefix(text, "proxy_for"):
			addr, idx, err2 := t.parseProxyFor(text)
			if err2 != nil {
				return nil, 0, errors.Annotate(err2, "Cannot parse 'proxy_for' section")
			}
			if addresses, ok := data[idx]; ok {
				data[idx] = append(addresses, addr)
			} else {
				data[idx] = []string{addr}
			}
		case strings.HasPrefix(text, "default"):
			idx, err2 := t.parseDefault(text)
			if err2 != nil {
				return nil, 0, errors.Annotate(err2, "Cannot parse 'default' section")
			}
			defaultIdx = idx
		default:
			return nil, 0, errors.Errorf("Unknown config string '%s'", text)
		}
	}

	err = scanner.Err()
	if err != nil {
		return nil, 0, errors.Annotate(err, "Cannot read response from the telegram")
	}

	return data, defaultIdx, nil
}

func (t *middleTelegramCaller) parseProxyFor(text string) (string, int16, error) {
	chunks := middleTelegramProxyConfigSplitter.Split(text, 3)
	if len(chunks) != 3 || chunks[0] != "proxy_for" {
		return "", 0, errors.Errorf("Incorrect config '%s'", text)
	}

	dcIdx, err := strconv.ParseInt(chunks[1], 10, 16)
	if err != nil {
		return "", 0, errors.Annotatef(err, "Incorrect config '%s'", text)
	}

	addr := strings.TrimRight(chunks[2], ";")
	if _, _, err = net.SplitHostPort(addr); err != nil {
		return "", 0, errors.Annotatef(err, "Incorrect config '%s'", text)
	}

	return addr, int16(dcIdx), nil
}

func (t *middleTelegramCaller) parseDefault(text string) (int16, error) {
	chunks := middleTelegramProxyConfigSplitter.Split(text, 2)
	if len(chunks) != 2 || chunks[0] != "default" {
		return 0, errors.Errorf("Incorrect config '%s'", text)
	}

	dcIdxString := strings.TrimRight(chunks[1], ";")
	dcIdx, err := strconv.ParseInt(dcIdxString, 10, 16)
	if err != nil {
		return 0, errors.Annotatef(err, "Incorrect config '%s'", text)
	}

	return int16(dcIdx), nil
}

func (t *middleTelegramCaller) call(url string) (*http.Response, error) {
	req, _ := http.NewRequest("GET", url, nil) // nolint: gosec
	req.Header.Set("Accept", "text/plain")
	req.Header.Set("User-Agent", tgUserAgent)

	return t.httpClient.Do(req)
}
