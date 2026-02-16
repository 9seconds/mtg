package dc

type Addr struct {
	Network string
	Address string
}

func (d Addr) String() string {
	return d.Address
}
