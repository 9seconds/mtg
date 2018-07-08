package telegram

import (
	"bufio"
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

func (t *middleTelegramCaller) Dial(connID string, connOpts *mtproto.ConnectionOpts) (wrappers.StreamReadWriteCloser, error) {
	dc := connOpts.DC
	if dc == 0 {
		dc = 1
	}
	t.dialerMutex.RLock()
	defer t.dialerMutex.RUnlock()

	return t.baseTelegram.dial(dc, connID, connOpts.ConnectionProto)
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

	v4Addresses, err := t.getTelegramAddresses(tgAddrProxyV4)
	if err != nil {
		return errors.Annotate(err, "Cannot get ipv4 addresses")
	}

	v6Addresses, err := t.getTelegramAddresses(tgAddrProxyV6)
	if err != nil {
		return errors.Annotate(err, "Cannot get ipv6 addresses")
	}

	t.dialerMutex.Lock()
	t.proxySecret = secret
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

func (t *middleTelegramCaller) getTelegramAddresses(url string) (map[int16][]string, error) {
	resp, err := t.call(url)
	if err != nil {
		return nil, errors.Annotate(err, "Cannot access telegram server")
	}
	defer resp.Body.Close()

	scanner := bufio.NewScanner(resp.Body)
	data := map[int16][]string{}
	for scanner.Scan() {
		text := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(text, "#") {
			continue
		}

		chunks := middleTelegramProxyConfigSplitter.Split(text, 3)
		if len(chunks) != 3 || chunks[0] != "proxy_for" {
			return nil, errors.Errorf("Incorrect config '%s'", text)
		}
		dcIdx64, err2 := strconv.ParseInt(chunks[1], 10, 16)
		if err2 != nil {
			return nil, errors.Errorf("Incorrect config '%s'", text)
		}
		dcIdx := int16(dcIdx64)

		addr := strings.TrimRight(chunks[2], ";")
		if _, _, err2 = net.SplitHostPort(addr); err != nil {
			return nil, errors.Annotatef(err2, "Incorrect config '%s'", text)
		}

		if addresses, ok := data[dcIdx]; ok {
			data[dcIdx] = append(addresses, addr)
		} else {
			data[dcIdx] = []string{addr}
		}
	}
	err = scanner.Err()
	if err != nil {
		return nil, errors.Annotate(err, "Cannot read response from the telegram")
	}

	return data, nil
}

func (t *middleTelegramCaller) call(url string) (*http.Response, error) {
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Accept", "text/plain")
	req.Header.Set("User-Agent", tgUserAgent)

	return t.httpClient.Do(req)
}
