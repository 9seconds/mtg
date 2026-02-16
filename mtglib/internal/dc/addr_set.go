package dc

import "math/rand/v2"

type dcAddrSet struct {
	v4 map[int][]Addr
	v6 map[int][]Addr
}

func (d dcAddrSet) getV4(dc int) []Addr {
	if d.v4 == nil {
		return nil
	}
	return d.get(d.v4[dc])
}

func (d dcAddrSet) getV6(dc int) []Addr {
	if d.v6 == nil {
		return nil
	}
	return d.get(d.v6[dc])
}

func (d dcAddrSet) get(addrs []Addr) []Addr {
	otherSet := make([]Addr, 0, len(addrs))
	otherSet = append(otherSet, addrs...)

	rand.Shuffle(len(otherSet), func(i, j int) {
		otherSet[i], otherSet[j] = otherSet[j], otherSet[i]
	})

	return otherSet
}
