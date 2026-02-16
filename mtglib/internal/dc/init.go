package dc

type preferIP uint8

const (
	preferIPOnlyIPv4 preferIP = iota
	preferIPOnlyIPv6
	preferIPPreferIPv4
	preferIPPreferIPv6
)

const (
	DefaultDC = 2

	defaultAppID   = 123456
	defaultAppHash = ""
)

type Logger interface {
	Info(msg string)
	WarningError(msg string, err error)
}

var (
	// https://github.com/telegramdesktop/tdesktop/blob/master/Telegram/SourceFiles/mtproto/mtproto_dc_options.cpp#L30
	defaultDCAddrSet = dcAddrSet{
		v4: map[int][]Addr{
			1: {
				{Network: "tcp4", Address: "149.154.175.50:443"},
			},
			2: {
				{Network: "tcp4", Address: "149.154.167.51:443"},
				{Network: "tcp4", Address: "95.161.76.100:443"},
			},
			3: {
				{Network: "tcp4", Address: "149.154.175.100:443"},
			},
			4: {
				{Network: "tcp4", Address: "149.154.167.91:443"},
			},
			5: {
				{Network: "tcp4", Address: "149.154.171.5:443"},
			},
		},
		v6: map[int][]Addr{
			1: {
				{Network: "tcp6", Address: "[2001:b28:f23d:f001::a]:443"},
			},
			2: {
				{Network: "tcp6", Address: "[2001:67c:04e8:f002::a]:443"},
			},
			3: {
				{Network: "tcp6", Address: "[2001:b28:f23d:f003::a]:443"},
			},
			4: {
				{Network: "tcp6", Address: "[2001:67c:04e8:f004::a]:443"},
			},
			5: {
				{Network: "tcp6", Address: "[2001:b28:f23f:f005::a]:443"},
			},
		},
	}

	defaultDCOverridesAddrSet = dcAddrSet{
		v4: map[int][]Addr{
			203: {
				{Network: "tcp4", Address: "91.105.192.100:443"},
			},
		},
		v6: map[int][]Addr{
			203: {
				{Network: "tcp6", Address: "[2a0a:f280:0203:000a:5000:0000:0000:0100]:443"},
			},
		},
	}
)
