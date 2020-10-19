package config

import (
	"encoding/hex"
	"net"
	"net/url"
	"strconv"
)

type URLs struct {
	TG        string `json:"tg_url"`
	TMe       string `json:"tme_url"`
	TGQRCode  string `json:"tg_qrcode"`
	TMeQRCode string `json:"tme_qrcode"`
}

type IPURLs struct {
	IPv4      URLs   `json:"ipv4"`
	IPv6      URLs   `json:"ipv6"`
	BotSecret string `json:"secret_for_mtproxybot"`
}

func GetURLs() (urls IPURLs) {
	secret := ""

	switch C.SecretMode {
	case SecretModeSimple:
		secret = hex.EncodeToString(C.Secret)
	case SecretModeSecured:
		secret = "dd" + hex.EncodeToString(C.Secret)
	case SecretModeTLS:
		secret = "ee" + hex.EncodeToString(C.Secret) + hex.EncodeToString([]byte(C.CloakHost))
	}

	urls.IPv4 = makeURLs(C.PublicIPv4, secret)
	urls.IPv6 = makeURLs(C.PublicIPv6, secret)
	urls.BotSecret = hex.EncodeToString(C.Secret)

	return urls
}

func makeURLs(addr *net.TCPAddr, secret string) (urls URLs) {
	values := url.Values{}
	values.Set("server", addr.IP.String())
	values.Set("port", strconv.Itoa(addr.Port))
	values.Set("secret", secret)

	urls.TG = makeTGURL(values)
	urls.TMe = makeTMeURL(values)
	urls.TGQRCode = makeQRCodeURL(urls.TG)
	urls.TMeQRCode = makeQRCodeURL(urls.TMe)

	return
}

func makeTGURL(values url.Values) string {
	tgURL := url.URL{
		Scheme:   "tg",
		Host:     "proxy",
		RawQuery: values.Encode(),
	}

	return tgURL.String()
}

func makeTMeURL(values url.Values) string {
	tMeURL := url.URL{
		Scheme:   "https",
		Host:     "t.me",
		Path:     "proxy",
		RawQuery: values.Encode(),
	}

	return tMeURL.String()
}

func makeQRCodeURL(data string) string {
	qr := url.URL{
		Scheme: "https",
		Host:   "api.qrserver.com",
		Path:   "v1/create-qr-code",
	}

	values := url.Values{}
	values.Set("qzone", "4")
	values.Set("format", "svg")
	values.Set("data", data)
	qr.RawQuery = values.Encode()

	return qr.String()
}
