package hub

import "github.com/9seconds/mtg/protocol"

type connectionHubRequest struct {
	req          *protocol.TelegramRequest
	responseChan chan<- *connection
}
