package dc

type dcView struct {
	publicConfigs dcAddrSet
}

func (d dcView) getV4(dc int) []Addr {
	var addrs []Addr

	addrs = append(addrs, defaultDCAddrSet.getV4(dc)...)
	addrs = append(addrs, d.publicConfigs.getV4(dc)...)

	return addrs
}

func (d dcView) getV6(dc int) []Addr {
	var addrs []Addr

	addrs = append(addrs, defaultDCAddrSet.getV6(dc)...)
	addrs = append(addrs, d.publicConfigs.getV6(dc)...)

	return addrs
}
