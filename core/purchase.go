package core

import (
	"crypto/rand"
	"encoding/hex"
	"github.com/cpacia/openbazaar3.0/database"
	"github.com/cpacia/openbazaar3.0/models"
	"github.com/cpacia/openbazaar3.0/orders/pb"
	iwallet "github.com/cpacia/wallet-interface"
	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes"
	"github.com/ipfs/go-cid"
	"github.com/ipfs/interface-go-ipfs-core/path"
	"github.com/jinzhu/gorm"
)

const (
	// orderOpenVersion is the current order open version number.
	orderOpenVersion = 1
)

// PurchaseListing attempts to purchase the listing using the provided data in the
// purchase model. It returns the order ID, payment address, payment amount, and
// an error if the purchase failed.
func (n *OpenBazaarNode) PurchaseListing(purchase *models.Purchase) (orderID models.OrderID,
	paymentAddress iwallet.Address, paymentAmount iwallet.Amount, err error) {

	// Create Order object
	/*order, err := n.createOrder(purchase)
	if err != nil {
		return
	}*/

	return
}

func (n *OpenBazaarNode) createOrder(purchase *models.Purchase) (*pb.OrderOpen, error) {
	var (
		listings      []*pb.Listing
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
	secp256k1Pubkey, err := n.masterPrivKey.ECPubKey()
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
	for _, item := range purchase.Items {
		c, err := cid.Decode(item.ListingHash)
		if err != nil {
			return nil, err
		}
		listingBytes, err := n.cat(path.IpfsPath(c))
		if err != nil {
			return nil, err
		}
		listing := new(pb.Listing)
		if err := proto.Unmarshal(listingBytes, listing); err != nil {
			return nil, err
		}
		listings = append(listings, listing)

		listingHash, err := multihashSha256(listingBytes)
		if err != nil {
			return nil, err
		}

		for _, option := range item.Options {
			orderOption := &pb.OrderOpen_Item_Option{
				Name:  option.Name,
				Value: option.Value,
			}
			options = append(options, orderOption)
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

	chaincode := make([]byte, 32)
	rand.Read(chaincode)

	order := &pb.OrderOpen{
		Timestamp: ptypes.TimestampNow(),
		BuyerID: &pb.ID{
			PeerID: n.Identity().Pretty(),
			Pubkeys: &pb.ID_Pubkeys{
				Identity:  identityPubkey,
				Secp256K1: secp256k1Pubkey.SerializeCompressed(),
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
		Payment: &pb.OrderOpen_Payment{
			Moderator: purchase.Moderator,
			Chaincode: hex.EncodeToString(chaincode),
			Coin:      purchase.PaymentCoin,
		},
	}

	ratingKeys, err := generateRatingPublicKeys(n.ratingMasterKey, len(order.Listings), chaincode)
	if err != nil {
		return nil, err
	}
	order.RatingKeys = ratingKeys

	privKey, err := n.masterPrivKey.ECPrivKey()
	if err != nil {
		return nil, err
	}
	identitySig, err := privKey.Sign([]byte(n.Identity().Pretty()))
	if err != nil {
		return nil, err
	}
	order.BuyerID.Sig = identitySig.Serialize()
	return order, nil
}
