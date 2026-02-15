package telegram

type dcAddresses struct {
	v4 map[int][]tgAddr
	v6 map[int][]tgAddr
}

func (a dcAddresses) getV4(dc int) []tgAddr {
	return a.v4[dc]
}

func (a dcAddresses) getV6(dc int) []tgAddr {
	return a.v6[dc]
}

func (a dcAddresses) isValidDC(dc int) bool {
	_, ok := a.v4[dc]
	return ok
}
