package network

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/url"

	"github.com/9seconds/mtg/v2/essentials"
	"github.com/txthinking/socks5"
)

type socks5Dialer struct {
	Dialer

	username     []byte
	password     []byte
	proxyAddress string
}

func (s socks5Dialer) Dial(network, address string) (essentials.Conn, error) {
	return s.DialContext(context.Background(), network, address)
}

func (s socks5Dialer) DialContext(ctx context.Context, network, address string) (essentials.Conn, error) {
	switch network {
	case "tcp", "tcp4", "tcp6":
	default:
		return nil, fmt.Errorf("%s network type is not supported", network)
	}

	conn, err := s.Dialer.DialContext(ctx, network, s.proxyAddress)
	if err != nil {
		return nil, fmt.Errorf("cannot dial to the proxy: %w", err)
	}

	if err := s.handshake(conn); err != nil {
		conn.Close()

		return nil, fmt.Errorf("cannot perform a handshake: %w", err)
	}

	if err := s.connect(conn, address); err != nil {
		conn.Close()

		return nil, fmt.Errorf("cannot connect to a destination host %s: %w", address, err)
	}

	return conn, nil
}

func (s socks5Dialer) handshake(conn io.ReadWriter) error {
	authMethod := socks5.MethodUsernamePassword
	if len(s.username)+len(s.password) == 0 {
		authMethod = socks5.MethodNone
	}

	if err := s.handshakeNegotiation(conn, authMethod); err != nil {
		return fmt.Errorf("cannot perform negotiation: %w", err)
	}

	if authMethod == socks5.MethodNone {
		return nil
	}

	if err := s.handshakeAuth(conn); err != nil {
		return fmt.Errorf("cannot authenticate: %w", err)
	}

	return nil
}

func (s socks5Dialer) handshakeNegotiation(conn io.ReadWriter, authMethod byte) error {
	request := socks5.NewNegotiationRequest([]byte{authMethod})
	if _, err := request.WriteTo(conn); err != nil {
		return fmt.Errorf("cannot send request: %w", err)
	}

	response, err := socks5.NewNegotiationReplyFrom(conn)
	if err != nil {
		return fmt.Errorf("cannot read response: %w", err)
	}

	if response.Method != authMethod {
		return fmt.Errorf("%v is unsupported auth method", authMethod)
	}

	return nil
}

func (s socks5Dialer) handshakeAuth(conn io.ReadWriter) error {
	request := socks5.NewUserPassNegotiationRequest(s.username, s.password)

	if _, err := request.WriteTo(conn); err != nil {
		return fmt.Errorf("cannot send a request: %w", err)
	}

	response, err := socks5.NewUserPassNegotiationReplyFrom(conn)
	if err != nil {
		return fmt.Errorf("cannot read a response: %w", err)
	}

	if response.Status != socks5.UserPassStatusSuccess {
		return fmt.Errorf("authenticate has failed: %v", response.Status)
	}

	return nil
}

func (s socks5Dialer) connect(conn io.ReadWriter, address string) error {
	addrType, host, port, err := socks5.ParseAddress(address)
	if err != nil {
		return fmt.Errorf("cannot parse address: %w", err)
	}

	if addrType == socks5.ATYPDomain {
		host = host[1:]
	}

	request := socks5.NewRequest(socks5.CmdConnect, addrType, host, port)

	if _, err := request.WriteTo(conn); err != nil {
		return fmt.Errorf("cannot send a request: %w", err)
	}

	response, err := socks5.NewReplyFrom(conn)
	if err != nil {
		return fmt.Errorf("cannot read a response: %w", err)
	}

	if response.Rep != socks5.RepSuccess {
		return fmt.Errorf("unsuccessful request: %v", response.Rep)
	}

	return nil
}

// NewSocks5Dialer build a new dialer from a given one (so, in theory you can
// chain here). Proxy parameters are passed with URI in a form of:
//
//	socks5://[user:[password]]@host:port
func NewSocks5Dialer(baseDialer Dialer, proxyURL *url.URL) (Dialer, error) {
	if _, _, err := net.SplitHostPort(proxyURL.Host); err != nil {
		return nil, fmt.Errorf("incorrect url %s", proxyURL.Redacted())
	}

	dialer := socks5Dialer{
		Dialer:       baseDialer,
		proxyAddress: proxyURL.Host,
	}

	if proxyURL.User != nil {
		password, isSet := proxyURL.User.Password()
		if isSet {
			dialer.username = []byte(proxyURL.User.Username())
			dialer.password = []byte(password)
		}
	}

	return dialer, nil
}
