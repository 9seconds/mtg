package telegram

import "math/rand"

type addressPool struct {
	v4 [][]tgAddr
	v6 [][]tgAddr
}

func (a addressPool) getV4(dc int) []tgAddr {
	return a.get(a.v4, dc-1)
}

func (a addressPool) getV6(dc int) []tgAddr {
	return a.get(a.v6, dc-1)
}

func (a addressPool) get(addresses [][]tgAddr, dc int) []tgAddr {
	if dc < 0 || dc >= len(addresses) {
		return nil
	}

	rv := make([]tgAddr, len(addresses[dc]))
	copy(rv, addresses[dc])

	if len(rv) > 1 {
		rand.Shuffle(len(rv), func(i, j int) {
			rv[i], rv[j] = rv[j], rv[i]
		})
	}

	return rv
}
