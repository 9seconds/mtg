package config

import (
	"net"
	"net/url"
	"strconv"
)

func getURLs(addr net.IP, port uint16, secret string) (urls URLs) {
	values := url.Values{}
	values.Set("server", addr.String())
	values.Set("port", strconv.Itoa(int(port)))
	values.Set("secret", secret)

	urls.TG = makeTGURL(values)
	urls.TMe = makeTMeURL(values)
	urls.TGQRCode = makeQRCodeURL(urls.TG)
	urls.TMeQRCode = makeQRCodeURL(urls.TG)

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
