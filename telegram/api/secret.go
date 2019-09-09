package api

import (
	"fmt"
	"io/ioutil"
)

const secretURL = "https://core.telegram.org/getProxySecret" // nolint: gas

func Secret() ([]byte, error) {
	resp, err := request(secretURL)
	if err != nil {
		return nil, fmt.Errorf("cannot access telegram server: %w", err)
	}
	defer resp.Close()

	secret, err := ioutil.ReadAll(resp)
	if err != nil {
		return nil, fmt.Errorf("cannot read response: %w", err)
	}

	return secret, nil
}
