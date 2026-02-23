package dc

type dcView struct {
	publicConfigs dcAddrSet
	ownConfigs    dcAddrSet
}

func (d dcView) getV4(dc int) []Addr {
	addrs := d.publicConfigs.getV4(dc)
	addrs = append(addrs, d.ownConfigs.getV4(dc)...)
	addrs = append(addrs, defaultDCAddrSet.getV4(dc)...)

	return addrs
}

func (d dcView) getV6(dc int) []Addr {
	addrs := d.publicConfigs.getV6(dc)
	addrs = append(addrs, d.ownConfigs.getV6(dc)...)
	addrs = append(addrs, defaultDCAddrSet.getV6(dc)...)

	return addrs
}
