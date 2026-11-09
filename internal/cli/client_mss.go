package cli

import (
	"github.com/9seconds/mtg/v2/internal/config"
	"github.com/9seconds/mtg/v2/mtglib"
	"github.com/9seconds/mtg/v2/network"
)

// clientMSSPlan maps [network] client-mss / client-mss-bulk /
// client-mss-bytes onto two mechanisms:
//   - listenerMSS is TCP_MAXSEG of the listening socket, it applies to the
//     whole connection;
//   - serverHelloMSS splits the beginning of the server response in user
//     space: the first serverHelloBytes bytes, or the ServerHello record if it
//     is 0.
//
// client-mss and bulk > 0: the socket gets bulk, the beginning of the
// response is split by client-mss. client-mss and bulk = 0: the whole
// connection runs at client-mss (like iptables TCPMSS). client-mss = 0:
// nothing. Outside Linux the options are ignored with a warning.
func clientMSSPlan(conf *config.Config, logger mtglib.Logger) (serverHelloMSS, serverHelloBytes, listenerMSS int) {
	handshake, bulk := conf.GetClientMSS()
	if handshake == 0 {
		return 0, 0, 0
	}

	if !network.ListenerMSSSupported {
		logger.Warning("network.client-mss is supported only on Linux; ignored")

		return 0, 0, 0
	}

	if bulk == 0 {
		serverHelloMSS, listenerMSS = 0, int(handshake)
	} else {
		serverHelloMSS, listenerMSS = int(handshake), int(bulk)
		serverHelloBytes = int(conf.GetClientMSSBytes())
	}

	logger.BindInt("server_hello_mss", serverHelloMSS).
		BindInt("server_hello_mss_bytes", serverHelloBytes).
		BindInt("listener_mss", listenerMSS).
		Info("client MSS shaping is enabled")

	return serverHelloMSS, serverHelloBytes, listenerMSS
}
