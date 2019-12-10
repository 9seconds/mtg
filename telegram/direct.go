package telegram

import "github.com/9seconds/mtg/conntypes"

const (
	directV4DefaultIdx conntypes.DC = 1
	directV6DefaultIdx conntypes.DC = 1
)

var (
	directV4Addresses = map[conntypes.DC][]string{
		0: {"149.154.175.50:443"},
		1: {"149.154.167.51:443"},
		2: {"149.154.175.100:443"},
		3: {"149.154.167.91:443"},
		4: {"149.154.171.5:443"},
	}
	directV6Addresses = map[conntypes.DC][]string{
		0: {"[2001:b28:f23d:f001::a]:443"},
		1: {"[2001:67c:04e8:f002::a]:443"},
		2: {"[2001:b28:f23d:f003::a]:443"},
		3: {"[2001:67c:04e8:f004::a]:443"},
		4: {"[2001:b28:f23f:f005::a]:443"},
	}
)

type directTelegram struct {
	baseTelegram
}

func (d *directTelegram) Dial(dc conntypes.DC,
	protocol conntypes.ConnectionProtocol) (conntypes.StreamReadWriteCloser, error) {
	switch {
	case dc < 0:
		dc = -dc
	case dc == 0:
		dc = conntypes.DCDefaultIdx
	}

	return d.baseTelegram.dial(dc-1, conntypes.ConnectionProtocolAny)
}
