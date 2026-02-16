package dc

type dcView struct {
	overrides dcAddrSet
	collected dcAddrSet
}

func (d dcView) getV4(dc int) []Addr {
	addrs := d.overrides.getV4(dc)
	addrs = append(addrs, defaultDCOverridesAddrSet.getV4(dc)...)
	// addrs = append(addrs, d.collected.getV4(dc)...)
	addrs = append(addrs, defaultDCAddrSet.getV4(dc)...)

	return addrs
}

func (d dcView) getV6(dc int) []Addr {
	addrs := d.overrides.getV6(dc)
	addrs = append(addrs, defaultDCOverridesAddrSet.getV6(dc)...)
	// addrs = append(addrs, d.collected.getV6(dc)...)
	addrs = append(addrs, defaultDCAddrSet.getV6(dc)...)

	return addrs
}
