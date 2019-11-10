package core

import (
	"context"
	"errors"
	"github.com/cpacia/openbazaar3.0/events"
	"github.com/cpacia/openbazaar3.0/net/pb"
	iwallet "github.com/cpacia/wallet-interface"
	"github.com/golang/protobuf/ptypes"
	peer "github.com/libp2p/go-libp2p-peer"
	"time"
)

const (
	// addressRequestTimeout is the amount of time to wait for a response
	// to a ADDRESS_REQUEST message.
	addressRequestTimeout = time.Second * 3
)

// ErrNoResponse response a failed address request due to the remote peer not responding.
var ErrNoResponse = errors.New("no response to address request from peer")

// RequestAddress requests a fresh payment address from the remote peer for in the given coin.
// This is sent as an online message online and will not retry if no response is returned.
func (n *OpenBazaarNode) RequestAddress(ctx context.Context, to peer.ID, coinType iwallet.CoinType) (iwallet.Address, error) {
	addrReq := pb.AddressRequestMessage{
		Coin: coinType.CurrencyCode(),
	}

	payload, err := ptypes.MarshalAny(&addrReq)
	if err != nil {
		return iwallet.Address{}, err
	}

	message := newMessageWithID()
	message.MessageType = pb.Message_ADDRESS_REQUEST
	message.Payload = payload

	sub, err := n.eventBus.Subscribe(&events.AddressRequestResponseNotification{})
	if err != nil {
		return iwallet.Address{}, err
	}
	defer sub.Close()

	ctx, cancel := context.WithTimeout(ctx, addressRequestTimeout)
	defer cancel()

	go n.networkService.SendMessage(ctx, to, message)

	for {
		select {
		case resp := <-sub.Out():
			addrReqResp := resp.(*events.AddressRequestResponseNotification)

			// We only care about responses from our peer and for our coin type. If we receive anything else
			// we'll just continue.
			if addrReqResp.PeerID != to.Pretty() || normalizeCurrencyCode(addrReqResp.Coin) != coinType.CurrencyCode() {
				continue
			}

			return iwallet.NewAddress(addrReqResp.Address, coinType), nil
		case <-time.After(addressRequestTimeout):
			return iwallet.Address{}, ErrNoResponse
		case <-ctx.Done():
			return iwallet.Address{}, ErrNoResponse
		}
	}
}

// handleAddressRequest is the handler for the ADDRESS_REQUEST message. It responds to
// request with an ADDRESS_RESPONSE message using an online message. Unknown coin types
// are ignored.
func (n *OpenBazaarNode) handleAddressRequest(from peer.ID, message *pb.Message) error {
	if message.MessageType != pb.Message_ADDRESS_REQUEST {
		return errors.New("message is not type ADDRESS_REQUEST")
	}

	req := new(pb.AddressRequestMessage)
	if err := ptypes.UnmarshalAny(message.Payload, req); err != nil {
		return err
	}

	wallet, err := n.multiwallet.WalletForCurrencyCode(req.Coin)
	if err != nil {
		return err
	}

	addr, err := wallet.NewAddress()
	if err != nil {
		return err
	}

	addrResp := pb.AddressResponseMessage{
		Address: addr.String(),
		Coin:    addr.CoinType().CurrencyCode(),
	}

	payload, err := ptypes.MarshalAny(&addrResp)
	if err != nil {
		return err
	}

	resp := newMessageWithID()
	resp.MessageType = pb.Message_ADDRESS_RESPONSE
	resp.Payload = payload

	return n.networkService.SendMessage(context.Background(), from, resp)
}

// handleAddressResponse is the handler for the ADDRESS_RESPONSE message. It pushes
// the response to the event bus for any listening subscribers.
func (n *OpenBazaarNode) handleAddressResponse(from peer.ID, message *pb.Message) error {
	if message.MessageType != pb.Message_ADDRESS_RESPONSE {
		return errors.New("message is not type ADDRESS_RESPONSE")
	}

	resp := new(pb.AddressResponseMessage)
	if err := ptypes.UnmarshalAny(message.Payload, resp); err != nil {
		return err
	}

	n.eventBus.Emit(&events.AddressRequestResponseNotification{
		PeerID:  from.Pretty(),
		Address: resp.Address,
		Coin:    resp.Coin,
	})
	return nil
}
