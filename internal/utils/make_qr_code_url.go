package utils

import "net/url"

func MakeQRCodeURL(data string) string {
	values := url.Values{}
	values.Set("qzone", "4")
	values.Set("format", "svg")
	values.Set("data", data)

	rv := url.URL{
		Scheme:   "https",
		Host:     "api.qrserver.com",
		Path:     "/v1/create-qr-code",
		RawQuery: values.Encode(),
	}

	return rv.String()
}
