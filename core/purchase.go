package core

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/btcsuite/btcd/btcec"
	"github.com/btcsuite/btcutil/base58"
	"github.com/cpacia/openbazaar3.0/database"
	"github.com/cpacia/openbazaar3.0/models"
	npb "github.com/cpacia/openbazaar3.0/net/pb"
	"github.com/cpacia/openbazaar3.0/orders"
	"github.com/cpacia/openbazaar3.0/orders/pb"
	iwallet "github.com/cpacia/wallet-interface"
	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes"
	"github.com/ipfs/go-cid"
	"github.com/jinzhu/gorm"
	peer "github.com/libp2p/go-libp2p-peer"
)

const (
	// orderOpenVersion is the current order open version number.
	orderOpenVersion = 1
)

// PurchaseListing attempts to purchase the listing using the provided data in the
// purchase model. It returns the order ID, payment address, payment amount, and
// an error if the purchase failed.
//
// The process here is:
// 1. Build the order using either the DIRECT or MODERATED payment method.
// 2. If DIRECT attempt to send the address request directly to the vendor and wait for a response.
// 3. If no response update the payment method to CANCELABLE and send using the messenger.
// 4. IF MODERATED skip steps 2 and 3 and send the message with the messenger.
func (n *OpenBazaarNode) PurchaseListing(purchase *models.Purchase) (orderID models.OrderID,
	paymentAddress iwallet.Address, paymentAmount models.CurrencyValue, err error) {

	// Create Order object
	orderOpen, err := n.createOrder(purchase)
	if err != nil {
		return
	}

	// Deserialize Vendor ID
	vendorPeerID, err := peer.IDB58Decode(orderOpen.Listings[0].Listing.VendorID.PeerID)
	if err != nil {
		return
	}

	paymentAddress = iwallet.NewAddress(orderOpen.Payment.Address, iwallet.CoinType(normalizeCurrencyCode(orderOpen.Payment.Coin)))
	currency, err := models.CurrencyDefinitions.Lookup(orderOpen.Payment.Coin)
	if err != nil {
		return
	}
	paymentAmount = *models.NewCurrencyValue(orderOpen.Payment.Amount, currency)

	wallet, err := n.multiwallet.WalletForCurrencyCode(orderOpen.Payment.Coin)
	if err != nil {
		return orderID, paymentAddress, paymentAmount, err
	}

	// If this is a direct payment we will first request an address from the vendor.
	// If he is online and responds to our request we will update the payment address
	// in the order with the address he gave us.
	//
	// If the vendor does not respond we will set the payment method to CANCELABLE
	// and use a 1 of 2 multisig address.
	//
	// Moderated orders we don't have to do anything else.
	if orderOpen.Payment.Method == pb.OrderOpen_Payment_DIRECT {
		address, err := n.RequestAddress(vendorPeerID, iwallet.CoinType(normalizeCurrencyCode(orderOpen.Payment.Coin)))
		// Vendor failed to respond to address request so we will change the
		// payment method to CANCELABLE.
		if err != nil {
			escrowWallet, ok := wallet.(iwallet.Escrow)
			if !ok {
				return orderID, paymentAddress, paymentAmount, errors.New("selected payment currency does not support escrow transactions")
			}
			chaincode, err := hex.DecodeString(orderOpen.Payment.Chaincode)
			if err != nil {
				return orderID, paymentAddress, paymentAmount, err
			}

			vendorEscrowPubkey, err := btcec.ParsePubKey(orderOpen.Listings[0].Listing.VendorID.Pubkeys.Escrow, btcec.S256())
			if err != nil {
				return orderID, paymentAddress, paymentAmount, err
			}
			vendorKey, err := generateEscrowPublicKey(vendorEscrowPubkey, chaincode)
			if err != nil {
				return orderID, paymentAddress, paymentAmount, err
			}
			buyerEscrowPubkey := n.escrowMasterKey.PubKey()
			buyerKey, err := generateEscrowPublicKey(buyerEscrowPubkey, chaincode)
			if err != nil {
				return orderID, paymentAddress, paymentAmount, err
			}
			address, script, err := escrowWallet.CreateMultisigAddress([]btcec.PublicKey{*buyerKey, *vendorKey}, 1)
			if err != nil {
				return orderID, paymentAddress, paymentAmount, err
			}

			escrowFee, err := escrowWallet.EstimateEscrowFee(1, iwallet.FlNormal)
			if err != nil {
				return orderID, paymentAddress, paymentAmount, err
			}

			orderOpen.Payment.Method = pb.OrderOpen_Payment_CANCELABLE
			orderOpen.Payment.Address = address.String()
			orderOpen.Payment.AdditionalAddressData = hex.EncodeToString(script)
			orderOpen.Payment.EscrowReleaseFee = escrowFee.String()
		} else {
			if err := wallet.ValidateAddress(paymentAddress); err != nil {
				return orderID, paymentAddress, paymentAmount, fmt.Errorf("vendor provided invalid payment address: %s", err)
			}
			orderOpen.Payment.Address = address.String()
		}
	}

	orderAny, err := ptypes.MarshalAny(orderOpen)
	if err != nil {
		return
	}

	orderIDBytes := make([]byte, 20)
	if _, err := rand.Read(orderIDBytes); err != nil {
		return orderID, paymentAddress, paymentAmount, err
	}

	order := npb.OrderMessage{
		OrderID:     base58.Encode(orderIDBytes),
		MessageType: npb.OrderMessage_ORDER_OPEN,
		Message:     orderAny,
	}

	payload, err := ptypes.MarshalAny(&order)
	if err != nil {
		return
	}

	message := newMessageWithID()
	message.MessageType = npb.Message_ORDER
	message.Payload = payload

	// Process the order, add the watched address to the wallet and send the message.
	err = n.repo.DB().Update(func(tx database.Tx) error {
		if _, err = n.orderProcessor.ProcessMessage(tx, vendorPeerID, &order); err != nil {
			return err
		}

		walletTx, err := wallet.Begin()
		if err != nil {
			return err
		}

		if err := wallet.WatchAddress(walletTx, paymentAddress); err != nil {
			if err := walletTx.Rollback(); err != nil {
				log.Errorf("Wallet rollback error: %s", err)
			}
			return err
		}

		if err := n.messenger.ReliablySendMessage(tx, vendorPeerID, message, nil); err != nil {
			if err := walletTx.Rollback(); err != nil {
				log.Errorf("Wallet rollback error: %s", err)
			}
			return err
		}
		return walletTx.Commit()
	})
	if err != nil {
		return
	}

	return models.OrderID(order.OrderID), paymentAddress, paymentAmount, nil
}

// EstimateOrderSubtotal estimates the total for the order given the provided
// purchase details. This is only an estimate because it may be based on the
// current exchange rates which may change by the time the order is placed.
func (n *OpenBazaarNode) EstimateOrderSubtotal(purchase *models.Purchase) (*models.CurrencyValue, error) {
	orderOpen, err := n.createOrder(purchase)
	if err != nil {
		return nil, err
	}
	currency, err := models.CurrencyDefinitions.Lookup(orderOpen.Payment.Coin)
	if err != nil {
		return nil, err
	}
	cv := models.NewCurrencyValue(orderOpen.Payment.Amount, currency)
	return cv, nil
}

// createOrder builds and returns an order from the given purchase data. The payment
// method will either be DIRECT or MODERATED depending on which was selected. It
// is expected that whichever function uses this returned order will update the
// payment method to CANCELABLE along with the payment address and additionalAddressData
// if the vendor is not online to respond to the DIRECT payment request.
func (n *OpenBazaarNode) createOrder(purchase *models.Purchase) (*pb.OrderOpen, error) {
	var (
		listings      []*pb.SignedListing
		items         []*pb.OrderOpen_Item
		options       []*pb.OrderOpen_Item_Option
		refundAddress string
	)
	wallet, err := n.multiwallet.WalletForCurrencyCode(purchase.PaymentCoin)
	if err != nil {
		return nil, err
	}

	if purchase.RefundAddress == nil {
		addr, err := wallet.NewAddress()
		if err != nil {
			return nil, err
		}
		refundAddress = addr.String()
	} else {
		refundAddress = *purchase.RefundAddress
	}

	identityPubkey, err := n.ipfsNode.PrivateKey.GetPublic().Bytes()
	if err != nil {
		return nil, err
	}

	profile := &models.Profile{}
	err = n.repo.DB().View(func(tx database.Tx) error {
		profile, err = tx.GetProfile()
		return err
	})
	if err != nil && !gorm.IsRecordNotFoundError(err) {
		return nil, err
	}
	if len(purchase.Items) == 0 {
		return nil, errors.New("no listings selected in purchase")
	}
	addedListings := make(map[string]bool)
	for _, item := range purchase.Items {
		c, err := cid.Decode(item.ListingHash)
		if err != nil {
			return nil, err
		}
		listing, err := n.GetListingByCID(c)
		if err != nil {
			return nil, err
		}
		if err := n.validateListing(listing); err != nil {
			return nil, err
		}
		// If we are purchasing the same listing multiple times but with
		// different options we don't need to include the full listing
		// multiple times. Once is enough.
		if !addedListings[item.ListingHash] {
			listings = append(listings, listing)
			addedListings[item.ListingHash] = true
		}

		for _, option := range item.Options {
			orderOption := &pb.OrderOpen_Item_Option{
				Name:  option.Name,
				Value: option.Value,
			}
			options = append(options, orderOption)
		}
		ser, err := proto.Marshal(listing)
		if err != nil {
			return nil, err
		}
		listingHash, err := multihashSha256(ser)
		if err != nil {
			return nil, err
		}

		orderItem := &pb.OrderOpen_Item{
			ListingHash:    listingHash.B58String(),
			Quantity:       item.Quantity,
			CouponCodes:    item.Coupons,
			Memo:           item.Memo,
			PaymentAddress: item.PaymentAddress,
			ShippingOption: &pb.OrderOpen_Item_ShippingOption{
				Name:    item.Shipping.Name,
				Service: item.Shipping.Service,
			},
			Options: options,
		}
		items = append(items, orderItem)
	}

	order := &pb.OrderOpen{
		Timestamp: ptypes.TimestampNow(),
		BuyerID: &pb.ID{
			PeerID: n.Identity().Pretty(),
			Pubkeys: &pb.ID_Pubkeys{
				Identity: identityPubkey,
				Escrow:   n.escrowMasterKey.PubKey().SerializeCompressed(),
			},
			Handle: profile.Handle,
		},
		AlternateContactInfo: purchase.AlternateContactInfo,
		Listings:             listings,
		Items:                items,
		Shipping: &pb.OrderOpen_Shipping{
			ShipTo:       purchase.ShipTo,
			Address:      purchase.Address,
			City:         purchase.City,
			State:        purchase.State,
			PostalCode:   purchase.PostalCode,
			Country:      pb.CountryCode(pb.CountryCode_value[purchase.CountryCode]),
			AddressNotes: purchase.AddressNotes,
		},
		Version:       orderOpenVersion,
		RefundAddress: refundAddress,
		Payment:       &pb.OrderOpen_Payment{},
	}

	chaincode := make([]byte, 32)
	if _, err := rand.Read(chaincode); err != nil {
		return nil, err
	}

	wallet, err = n.multiwallet.WalletForCurrencyCode(purchase.PaymentCoin)
	if err != nil {
		return nil, err
	}

	escrowWallet, walletSupportsEscrow := wallet.(iwallet.Escrow)
	if !walletSupportsEscrow && purchase.Moderator != "" {
		return nil, errors.New("selected payment currency does not support escrow transactions")
	}

	var (
		paymentMethod = pb.OrderOpen_Payment_DIRECT
		escrowFee     iwallet.Amount
	)
	if purchase.Moderator != "" {
		paymentMethod = pb.OrderOpen_Payment_MODERATED
		escrowFee, err = escrowWallet.EstimateEscrowFee(2, iwallet.FlNormal)
		if err != nil {
			return nil, err
		}
		order.Payment.Moderator = purchase.Moderator
		order.Payment.EscrowReleaseFee = escrowFee.String()

		moderatorPeerID, err := peer.IDHexDecode(purchase.Moderator)
		if err != nil {
			return nil, err
		}

		moderatorProfile, err := n.GetProfile(moderatorPeerID, true)
		if err != nil {
			return nil, err
		}
		moderatorPubkeyBytes, err := hex.DecodeString(moderatorProfile.EscrowPublicKey)
		if err != nil {
			return nil, err
		}
		moderatorEscrowPubkey, err := btcec.ParsePubKey(moderatorPubkeyBytes, btcec.S256())
		if err != nil {
			return nil, err
		}
		moderatorKey, err := generateEscrowPublicKey(moderatorEscrowPubkey, chaincode)
		if err != nil {
			return nil, err
		}

		vendorEscrowPubkey, err := btcec.ParsePubKey(order.Listings[0].Listing.VendorID.Pubkeys.Escrow, btcec.S256())
		if err != nil {
			return nil, err
		}
		vendorKey, err := generateEscrowPublicKey(vendorEscrowPubkey, chaincode)
		if err != nil {
			return nil, err
		}
		buyerEscrowPubkey := n.escrowMasterKey.PubKey()
		buyerKey, err := generateEscrowPublicKey(buyerEscrowPubkey, chaincode)
		if err != nil {
			return nil, err
		}
		address, script, err := escrowWallet.CreateMultisigAddress([]btcec.PublicKey{*buyerKey, *vendorKey, *moderatorKey}, 2)
		if err != nil {
			return nil, err
		}
		order.Payment.Address = address.String()
		order.Payment.AdditionalAddressData = hex.EncodeToString(script)
	}

	order.Payment.Method = paymentMethod
	order.Payment.Chaincode = hex.EncodeToString(chaincode)
	order.Payment.Coin = normalizeCurrencyCode(purchase.PaymentCoin)

	total, err := orders.CalculateOrderTotal(order, n.exchangeRates)
	if err != nil {
		return nil, err
	}
	order.Payment.Amount = total.String()

	ratingKeys, err := generateRatingPublicKeys(n.ratingMasterKey.PubKey(), len(order.Listings), chaincode)
	if err != nil {
		return nil, err
	}
	order.RatingKeys = ratingKeys

	identitySig, err := n.escrowMasterKey.Sign([]byte(n.Identity().Pretty()))
	if err != nil {
		return nil, err
	}
	order.BuyerID.Sig = identitySig.Serialize()
	return order, nil
}
