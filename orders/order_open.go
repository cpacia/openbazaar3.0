package orders

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/OpenBazaar/jsonpb"
	"github.com/btcsuite/btcd/btcec"
	"github.com/cpacia/openbazaar3.0/database"
	"github.com/cpacia/openbazaar3.0/events"
	"github.com/cpacia/openbazaar3.0/models"
	npb "github.com/cpacia/openbazaar3.0/net/pb"
	"github.com/cpacia/openbazaar3.0/orders/pb"
	"github.com/cpacia/openbazaar3.0/orders/utils"
	"github.com/cpacia/openbazaar3.0/wallet"
	iwallet "github.com/cpacia/wallet-interface"
	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes"
	crypto "github.com/libp2p/go-libp2p-crypto"
	peer "github.com/libp2p/go-libp2p-peer"
	"math"
	"math/big"
	"strings"
)

func (op *OrderProcessor) handleOrderOpenMessage(dbtx database.Tx, order *models.Order, peer peer.ID, message *npb.OrderMessage) (interface{}, error) {
	orderOpen := new(pb.OrderOpen)
	if err := ptypes.UnmarshalAny(message.Message, orderOpen); err != nil {
		return nil, err
	}

	dup, err := isDuplicate(orderOpen, order.SerializedOrderOpen)
	if err != nil {
		return nil, err
	}
	if order.SerializedOrderOpen != nil && !dup {
		log.Errorf("Duplicate ORDER_OPEN message does not match original for order: %s", order.ID)
		return nil, ErrChangedMessage
	} else if dup {
		return nil, nil
	}

	var validationError bool
	// If the validation fails and we are the vendor, we send a REJECT message back
	// to the buyer. The reject message also gets saved with this order.
	if err := op.validateOrderOpen(dbtx, orderOpen); err != nil {
		log.Errorf("ORDER_OPEN message for order %s from %s failed to validate: %s", order.ID, orderOpen.BuyerID.PeerID, err)
		if op.identity != peer {
			reject := pb.OrderReject{
				Type:   pb.OrderReject_VALIDATION_ERROR,
				Reason: err.Error(),
			}

			rejectAny, err := ptypes.MarshalAny(&reject)
			if err != nil {
				return nil, err
			}

			resp := npb.OrderMessage{
				OrderID:     order.ID.String(),
				MessageType: npb.OrderMessage_ORDER_REJECT,
				Message:     rejectAny,
			}

			payload, err := ptypes.MarshalAny(&resp)
			if err != nil {
				return nil, err
			}

			messageID := make([]byte, 20)
			if _, err := rand.Read(messageID); err != nil {
				return nil, err
			}

			message := npb.Message{
				MessageType: npb.Message_ORDER,
				MessageID:   hex.EncodeToString(messageID),
				Payload:     payload,
			}

			if err := op.messenger.ReliablySendMessage(dbtx, peer, &message, nil); err != nil {
				return nil, err
			}

			if err := order.PutMessage(&reject); err != nil {
				return nil, err
			}
		}
		validationError = true
	}

	var event interface{}
	// TODO: do we want to emit an event in the case of a validation error?
	if !validationError && op.identity != peer {
		event = &events.OrderNotification{
			BuyerHandle: orderOpen.BuyerID.Handle,
			BuyerID:     orderOpen.BuyerID.PeerID,
			ListingType: orderOpen.Listings[0].Listing.Metadata.ContractType.String(),
			OrderID:     order.ID.String(),
			Price: events.ListingPrice{
				Amount:        orderOpen.Payment.Amount,
				CurrencyCode:  orderOpen.Payment.Coin,
				PriceModifier: orderOpen.Listings[0].Listing.Metadata.PriceModifier,
			},
			Slug: orderOpen.Listings[0].Listing.Slug,
			Thumbnail: events.Thumbnail{
				Tiny:  orderOpen.Listings[0].Listing.Item.Images[0].Tiny,
				Small: orderOpen.Listings[0].Listing.Item.Images[0].Small,
			},
			Title: orderOpen.Listings[0].Listing.Slug,
		}
	}

	if err := order.PutMessage(orderOpen); err != nil {
		return nil, err
	}

	return event, nil
}

// validateOrderOpen checks all the fields in the order to make sure they are
// properly formatted.
func (op *OrderProcessor) validateOrderOpen(dbtx database.Tx, order *pb.OrderOpen) error {
	if order.Listings == nil {
		return errors.New("listings field is nil")
	}
	if order.Payment == nil {
		return errors.New("payment field is nil")
	}
	if order.Items == nil {
		return errors.New("items field is nil")
	}
	if order.Timestamp == nil {
		return errors.New("timestamp field is nil")
	}
	if order.BuyerID == nil {
		return errors.New("buyer ID field is nil")
	}
	if order.RatingKeys == nil {
		return errors.New("rating keys field is nil")
	}

	wal, err := op.multiwallet.WalletForCurrencyCode(order.Payment.Coin)
	if err != nil {
		return err
	}

	if op.identity.Pretty() != order.BuyerID.PeerID { // If we are vendor.
		// Check to make sure we actually have the item for sale.
		for _, listing := range order.Listings {
			var theirListing pb.SignedListing
			if err := deepCopyListing(&theirListing, listing); err != nil {
				return err
			}

			myListing, err := dbtx.GetListing(theirListing.Listing.Slug)
			if err != nil {
				return fmt.Errorf("item %s is not for sale", theirListing.Listing.Slug)
			}

			// Zero out the inventory on each listing. We will check
			// inventory later.
			for i := range myListing.Listing.Item.Skus {
				myListing.Listing.Item.Skus[i].Quantity = 0
			}
			for i := range theirListing.Listing.Item.Skus {
				theirListing.Listing.Item.Skus[i].Quantity = 0
			}

			// We can tell if we have the listing for sale if the serialized bytes match
			// after we've zeroed out the inventory.
			mySer, err := proto.Marshal(myListing.Listing)
			if err != nil {
				return err
			}

			theirSer, err := proto.Marshal(theirListing.Listing)
			if err != nil {
				return err
			}

			if !bytes.Equal(mySer, theirSer) {
				return fmt.Errorf("item %s is not for sale", listing.Listing.Slug)
			}
		}

		if order.Payment.Method == pb.OrderOpen_Payment_DIRECT {
			has, err := wal.HasKey(iwallet.NewAddress(order.Payment.Address, iwallet.CoinType(order.Payment.Coin)))
			if err != nil {
				return err
			}
			if !has {
				return errors.New("direct payment address not found in wallet")
			}
		}
	}

	for i, item := range order.Items {
		if item == nil {
			return fmt.Errorf("item %d is nil", i)
		}
		// Let's check to make sure there is a listing for each
		// item in the order.
		listing, err := extractListing(item.ListingHash, order.Listings)
		if err != nil {
			return fmt.Errorf("listing not found in order for item %s", item.ListingHash)
		}

		// Validate the rest of the item
		if listing.Metadata.ContractType == pb.Listing_Metadata_CRYPTOCURRENCY && item.PaymentAddress == "" {
			return fmt.Errorf("payment address for cryptocurrency item %d is empty", i)
		}
		if item.Quantity == 0 {
			return fmt.Errorf("item %d quantity is zero", i)
		}

		// Validate selected options
		if len(item.Options) != len(listing.Item.Options) {
			return fmt.Errorf("item %d not all options selected", i)
		}
		optionMap := make(map[string]string)
		for _, option := range item.Options {
			optionMap[strings.ToLower(option.Name)] = strings.ToLower(option.Value)
		}
		for _, opt := range listing.Item.Options {
			val, ok := optionMap[strings.ToLower(opt.Name)]
			if !ok {
				return fmt.Errorf("item %d option %s not found in listing", i, opt.Name)
			}
			valExists := false
			for _, variant := range opt.Variants {
				if strings.ToLower(variant.Name) == val {
					valExists = true
					break
				}
			}
			if !valExists {
				return fmt.Errorf("item %d variant %s not found in listing option", i, val)
			}
		}

		// Validate shipping option
		if item.ShippingOption != nil {
			shippingOpts := make(map[string][]*pb.Listing_ShippingOption_Service)
			for _, option := range listing.ShippingOptions {
				shippingOpts[strings.ToLower(option.Name)] = option.Services
			}
			services, ok := shippingOpts[strings.ToLower(item.ShippingOption.Name)]
			if !ok {
				return fmt.Errorf("item %d shipping option %s not found in listing", i, item.ShippingOption.Name)
			}
			serviceExists := false
			for _, service := range services {
				if strings.ToLower(service.Name) == strings.ToLower(item.ShippingOption.Service) {
					serviceExists = true
				}
			}
			if !serviceExists {
				return fmt.Errorf("item %d shipping service %s not found in listing option", i, item.ShippingOption.Service)
			}
		}
	}

	// Validate buyer ID
	if order.BuyerID.Pubkeys == nil {
		return errors.New("buyer id pubkeys is nil")
	}
	idPubkey, err := crypto.UnmarshalPublicKey(order.BuyerID.Pubkeys.Identity)
	if err != nil {
		return fmt.Errorf("invalid buyer ID pubkey: %s", err)
	}
	pid, err := peer.IDFromPublicKey(idPubkey)
	if err != nil {
		return fmt.Errorf("invalid buyer ID pubkey: %s", err)
	}
	if pid.Pretty() != order.BuyerID.PeerID {
		return errors.New("buyer ID does not match pubkey")
	}
	escrowPubkey, err := btcec.ParsePubKey(order.BuyerID.Pubkeys.Escrow, btcec.S256())
	if err != nil {
		return errors.New("invalid buyer escrow pubkey")
	}
	sig, err := btcec.ParseSignature(order.BuyerID.Sig, btcec.S256())
	if err != nil {
		return errors.New("invalid buyer ID signature")
	}
	idHash := sha256.Sum256([]byte(order.BuyerID.PeerID))
	valid := sig.Verify(idHash[:], escrowPubkey)
	if !valid {
		return errors.New("invalid buyer ID signature")
	}

	// Validate payment
	if order.Payment.Amount == "" {
		return errors.New("payment amount not set")
	}
	if ok := validateBigString(order.Payment.Amount); !ok {
		return errors.New("payment amount not valid")
	}
	if order.Payment.Address == "" {
		return errors.New("order payment address is empty")
	}
	chaincode, err := hex.DecodeString(order.Payment.Chaincode)
	if err != nil {
		return fmt.Errorf("chaincode parse error: %s", err)
	}
	vendorEscrowPubkey, err := btcec.ParsePubKey(order.Listings[0].Listing.VendorID.Pubkeys.Escrow, btcec.S256())
	if err != nil {
		return err
	}
	vendorKey, err := utils.GenerateEscrowPublicKey(vendorEscrowPubkey, chaincode)
	if err != nil {
		return err
	}
	buyerEscrowPubkey, err := btcec.ParsePubKey(order.BuyerID.Pubkeys.Escrow, btcec.S256())
	if err != nil {
		return err
	}
	buyerKey, err := utils.GenerateEscrowPublicKey(buyerEscrowPubkey, chaincode)
	if err != nil {
		return err
	}
	if order.Payment.Method == pb.OrderOpen_Payment_MODERATED {
		_, err := peer.IDB58Decode(order.Payment.Moderator)
		if err != nil {
			return errors.New("invalid moderator selection")
		}
		moderatorEscrowPubkey, err := btcec.ParsePubKey(order.Payment.ModeratorKey, btcec.S256())
		if err != nil {
			return err
		}
		moderatorKey, err := utils.GenerateEscrowPublicKey(moderatorEscrowPubkey, chaincode)
		if err != nil {
			return err
		}
		escrowWallet, ok := wal.(iwallet.Escrow)
		if !ok {
			return errors.New("wallet does not support escorw")
		}
		address, script, err := escrowWallet.CreateMultisigAddress([]btcec.PublicKey{*buyerKey, *vendorKey, *moderatorKey}, 2)
		if err != nil {
			return err
		}
		if order.Payment.Address != address.String() {
			return errors.New("invalid moderated payment address")
		}
		if order.Payment.Script != hex.EncodeToString(script) {
			return errors.New("invalid moderated payment script")
		}
	} else if order.Payment.Method == pb.OrderOpen_Payment_CANCELABLE {
		escrowWallet, ok := wal.(iwallet.Escrow)
		if !ok {
			return errors.New("wallet does not support escorw")
		}
		address, script, err := escrowWallet.CreateMultisigAddress([]btcec.PublicKey{*buyerKey, *vendorKey}, 1)
		if err != nil {
			return err
		}
		if order.Payment.Address != address.String() {
			return errors.New("invalid cancelable payment address")
		}
		if order.Payment.Script != hex.EncodeToString(script) {
			return errors.New("invalid cancelable payment script")
		}
	} else if order.Payment.Method != pb.OrderOpen_Payment_DIRECT {
		return errors.New("invalid payment method")
	}
	_, err = models.CurrencyDefinitions.Lookup(order.Payment.Coin)
	if err != nil {
		return errors.New("unknown payment currency")
	}
	if order.Payment.Method != pb.OrderOpen_Payment_DIRECT {
		if order.Payment.EscrowReleaseFee == "" {
			return errors.New("escrow release fee is empty")
		}
		if ok := validateBigString(order.Payment.EscrowReleaseFee); !ok {
			return errors.New("escrow release fee not valid")
		}
	}

	// Validate rating keys
	for _, key := range order.RatingKeys {
		if _, err := btcec.ParsePubKey(key, btcec.S256()); err != nil {
			return errors.New("invalid rating pubkey")
		}
	}

	return nil
}

// CalculateOrderTotal calculates and returns the total for the order with all
// the provided options.
func CalculateOrderTotal(order *pb.OrderOpen, erp *wallet.ExchangeRateProvider) (iwallet.Amount, error) {
	var (
		orderTotal    iwallet.Amount
		physicalGoods = make(map[string]*pb.Listing)
	)

	// Calculate the price of each item
	for _, item := range order.Items {
		// Step one is we just want to get the price, in the payment currency,
		// for the listing.
		var (
			itemTotal    iwallet.Amount
			itemQuantity = item.Quantity
		)

		listing, err := extractListing(item.ListingHash, order.Listings)
		if err != nil {
			return orderTotal, fmt.Errorf("listing not found in contract for item %s", item.ListingHash)
		}

		if listing.Metadata.ContractType == pb.Listing_Metadata_PHYSICAL_GOOD {
			physicalGoods[item.ListingHash] = listing
		}

		pricingCurrency, err := models.CurrencyDefinitions.Lookup(listing.Metadata.PricingCurrency.Code)
		if err != nil {
			return orderTotal, err
		}
		paymentCurrency, err := models.CurrencyDefinitions.Lookup(order.Payment.Coin)
		if err != nil {
			return orderTotal, err
		}

		if listing.Metadata.Format == pb.Listing_Metadata_MARKET_PRICE {
			// To calculate the market price we just use the exchange rate between
			// the two coins. However in this case we use the item quantity being
			// purchased as the amount as we want to find the exchange rate of
			// the given quantity.
			price := models.NewCurrencyValueFromUint(itemQuantity, pricingCurrency)
			itemTotal, err = convertCurrencyAmount(price, paymentCurrency, erp)
			if err != nil {
				return orderTotal, err
			}

			// Now we add or subtract the price modifier.
			f, _ := new(big.Float).SetString(itemTotal.String())
			f.Mul(f, big.NewFloat(float64(listing.Metadata.PriceModifier/100)))
			priceMod, _ := f.Int(nil)
			itemTotal = itemTotal.Add(iwallet.NewAmount(priceMod))

			// Since we already used the quantity to calculate the price we can
			// just set this to 1 to avoid multiplying by the quantity again below.
			itemQuantity = 1
		} else {
			price := models.NewCurrencyValue(listing.Item.Price, pricingCurrency)
			itemTotal, err = convertCurrencyAmount(price, paymentCurrency, erp)
			if err != nil {
				return orderTotal, err
			}
		}

		// Add or subtract any surcharge on the selected sku
		sku, err := getSelectedSku(listing, item.Options)
		if err != nil {
			return orderTotal, err
		}
		surcharge := iwallet.NewAmount(sku.Surcharge)
		surchargeValue := models.NewCurrencyValue(surcharge.String(), pricingCurrency)
		convertedSurcharge, err := convertCurrencyAmount(surchargeValue, paymentCurrency, erp)
		if err != nil {
			return orderTotal, err
		}
		itemTotal.Add(convertedSurcharge)

		// Subtract any coupons
		for _, couponCode := range item.CouponCodes {
			couponHash, err := utils.MultihashSha256([]byte(couponCode))
			if err != nil {
				return orderTotal, err
			}
			for _, vendorCoupon := range listing.Coupons {
				if couponHash.B58String() == vendorCoupon.GetHash() {
					if discount := vendorCoupon.GetPriceDiscount(); discount != "" && iwallet.NewAmount(discount).Cmp(iwallet.NewAmount(0)) > 0 {
						price := models.NewCurrencyValue(discount, pricingCurrency)
						discountAmount, err := convertCurrencyAmount(price, paymentCurrency, erp)
						if err != nil {
							return orderTotal, err
						}
						itemTotal = itemTotal.Sub(discountAmount)
					} else if discount := vendorCoupon.GetPercentDiscount(); discount > 0 {
						f, _ := new(big.Float).SetString(itemTotal.String())
						f.Mul(f, big.NewFloat(float64(-discount/100)))
						discountAmount, _ := f.Int(nil)
						itemTotal = itemTotal.Add(iwallet.NewAmount(discountAmount))
					}
				}
			}
		}
		// Apply tax
		for _, tax := range listing.Taxes {
			for _, taxRegion := range tax.TaxRegions {
				if order.Shipping.Country == taxRegion {
					f, _ := new(big.Float).SetString(itemTotal.String())
					f.Mul(f, big.NewFloat(float64(tax.Percentage/100)))
					govTheft, _ := f.Int(nil)
					itemTotal = itemTotal.Add(iwallet.NewAmount(govTheft))
					break
				}
			}
		}

		// Multiply the item total by the quantity being purchased
		// In the case of a crypto listing, itemQuantity was set to
		// one above so this should have no effect.
		itemTotal = itemTotal.Mul(iwallet.NewAmount(itemQuantity))

		// Finally add the item total to the order total.
		orderTotal = orderTotal.Add(itemTotal)
	}

	// Add in shipping
	shippingTotal, err := calculateShippingTotalForListings(order, physicalGoods, erp)
	if err != nil {
		return orderTotal, err
	}
	orderTotal = orderTotal.Add(shippingTotal)

	return orderTotal, nil
}

func calculateShippingTotalForListings(order *pb.OrderOpen, listings map[string]*pb.Listing, erp *wallet.ExchangeRateProvider) (iwallet.Amount, error) {
	type itemShipping struct {
		primary               iwallet.Amount
		secondary             iwallet.Amount
		quantity              uint64
		shippingTaxPercentage float32
		version               uint32
	}
	var (
		is            []itemShipping
		shippingTotal = iwallet.NewAmount(0)
	)

	// First loop through to validate and filter out non-physical items
	for _, item := range order.Items {
		listing, ok := listings[item.ListingHash]
		if !ok {
			continue
		}

		pricingCurrency, err := models.CurrencyDefinitions.Lookup(listing.Metadata.PricingCurrency.Code)
		if err != nil {
			return shippingTotal, err
		}
		paymentCurrency, err := models.CurrencyDefinitions.Lookup(order.Payment.Coin)
		if err != nil {
			return shippingTotal, err
		}

		// Check selected option exists
		shippingOptions := make(map[string]*pb.Listing_ShippingOption)
		for _, so := range listing.ShippingOptions {
			shippingOptions[strings.ToLower(so.Name)] = so
		}
		option, ok := shippingOptions[strings.ToLower(item.ShippingOption.Name)]
		if !ok {
			return shippingTotal, errors.New("shipping option not found in listing")
		}

		if option.Type == pb.Listing_ShippingOption_LOCAL_PICKUP {
			continue
		}

		// Check that this option ships to us
		regions := make(map[pb.CountryCode]bool)
		for _, country := range option.Regions {
			regions[country] = true
		}
		_, shipsToMe := regions[order.Shipping.Country]
		_, shipsToAll := regions[pb.CountryCode_ALL]
		if !shipsToMe && !shipsToAll {
			return shippingTotal, errors.New("listing does ship to selected country")
		}

		// Check service exists
		services := make(map[string]*pb.Listing_ShippingOption_Service)
		for _, shippingService := range option.Services {
			services[strings.ToLower(shippingService.Name)] = shippingService
		}
		service, ok := services[strings.ToLower(item.ShippingOption.Service)]
		if !ok {
			return shippingTotal, errors.New("shipping service not found in listing")
		}

		// Convert to payment currency
		price := models.NewCurrencyValue(service.Price, pricingCurrency)
		primaryTotal, err := convertCurrencyAmount(price, paymentCurrency, erp)
		if err != nil {
			return shippingTotal, err
		}

		// Convert additional item price
		secondaryTotal := iwallet.NewAmount(0)
		if service.AdditionalItemPrice != "" {
			if iwallet.NewAmount(service.AdditionalItemPrice).Cmp(iwallet.NewAmount(0)) > 0 {
				secondaryPrice := models.NewCurrencyValue(service.AdditionalItemPrice, pricingCurrency)
				secondaryTotal, err = convertCurrencyAmount(secondaryPrice, paymentCurrency, erp)
				if err != nil {
					return shippingTotal, err
				}
			}
		}

		// Calculate tax percentage
		var shippingTaxPercentage float32
		for _, tax := range listing.Taxes {
			regions := make(map[pb.CountryCode]bool)
			for _, taxRegion := range tax.TaxRegions {
				regions[taxRegion] = true
			}
			_, ok := regions[order.Shipping.Country]
			if ok && tax.TaxShipping {
				shippingTaxPercentage = tax.Percentage / 100
			}
		}

		is = append(is, itemShipping{
			primary:               primaryTotal,
			secondary:             secondaryTotal,
			quantity:              item.Quantity,
			shippingTaxPercentage: shippingTaxPercentage,
			version:               listing.Metadata.Version,
		})
	}

	// No options to charge shipping on. Return zero.
	if len(is) == 0 {
		return shippingTotal, nil
	}

	// Single item. For the first quantity charge the primary. For all others charge the secondary.
	if len(is) == 1 {
		shippingTotal = shippingTotal.Add(is[0].primary)
		shippingTotal = shippingTotal.Add(calculateShippingTax(is[0].shippingTaxPercentage, is[0].primary))
		if is[0].quantity > 1 {
			shippingTotal = shippingTotal.Add(is[0].secondary.Mul(iwallet.NewAmount(is[0].quantity - 1)))
			shippingTotal = shippingTotal.Add(calculateShippingTax(is[0].shippingTaxPercentage, is[0].secondary.Mul(iwallet.NewAmount(is[0].quantity-1))))
		}
		return shippingTotal, nil
	}

	// Multiple items. We want to charge primary rate for the item with the highest primary
	// rate. All other items and quantities should be charged the secondary rate that corresponds
	// to those items.
	//
	// The way will do this is be first adding in the secondary rates for all items and all quantities.
	// Then subtract off the secondary rate for the item with the highest primary rate, then add on the
	// primary rate.
	highest := iwallet.NewAmount(0)
	var i int
	for x, s := range is {
		if s.primary.Cmp(highest) > 0 {
			highest = s.primary
			i = x
		}
		shippingTotal = shippingTotal.Add(s.secondary.Mul(iwallet.NewAmount(s.quantity)))
		shippingTotal = shippingTotal.Add(calculateShippingTax(s.shippingTaxPercentage, s.secondary.Mul(iwallet.NewAmount(s.quantity))))
	}
	shippingTotal = shippingTotal.Sub(is[i].secondary)
	shippingTotal = shippingTotal.Sub(calculateShippingTax(is[i].shippingTaxPercentage, is[i].secondary))

	shippingTotal = shippingTotal.Add(is[i].primary)
	shippingTotal = shippingTotal.Add(calculateShippingTax(is[i].shippingTaxPercentage, is[i].primary))

	return shippingTotal, nil
}

// calculateShippingTax is a helper function to calculate the tax given the shipping rate and tax rate.
func calculateShippingTax(shippingTaxPercentage float32, shippingRate iwallet.Amount) iwallet.Amount {
	f, _ := new(big.Float).SetString(shippingRate.String())
	f.Mul(f, big.NewFloat(float64(shippingTaxPercentage)))
	governmentTheft, _ := f.Int(nil)
	return iwallet.NewAmount(governmentTheft)
}

// convertCurrencyAmount converts the value of one currency into another using the exchange rate.
func convertCurrencyAmount(value *models.CurrencyValue, paymentCurrency *models.Currency, erp *wallet.ExchangeRateProvider) (iwallet.Amount, error) {
	// If both currency types are the same then just return the value.
	if value.Currency.Equal(paymentCurrency) {
		return value.Amount, nil
	}

	if paymentCurrency.CurrencyType != models.CurrencyTypeCrypto {
		return value.Amount, errors.New("payment currency is not type crypto")
	}

	rate, err := erp.GetRate(paymentCurrency.Code, value.Currency.Code, true)
	if err != nil {
		return value.Amount, err
	}

	rateFloat, ok := new(big.Float).SetString(rate.String())
	if !ok {
		return value.Amount, errors.New("error converting exchange rate to float")
	}

	div := new(big.Float).Quo(rateFloat, big.NewFloat(math.Pow10(int(value.Currency.Divisibility))))
	div.Quo(big.NewFloat(1), div)

	v, _ := div.Float64()

	converted, err := value.ConvertTo(paymentCurrency, v)
	if err != nil {
		return value.Amount, err
	}
	return converted.Amount, nil
}

// extractListing will return the listing with the given hash from the provided
// slice of listings if it exists.
func extractListing(hash string, listings []*pb.SignedListing) (*pb.Listing, error) {
	for _, sl := range listings {
		mh, err := utils.HashListing(sl)
		if err != nil {
			return nil, err
		}
		if mh.B58String() == hash {
			return sl.Listing, nil
		}
	}
	return nil, fmt.Errorf("listing %s not found in order", hash)
}

func deepCopyListing(dest *pb.SignedListing, src *pb.SignedListing) error {
	m := jsonpb.Marshaler{
		EnumsAsInts:  false,
		EmitDefaults: true,
		Indent:       "",
		OrigName:     false,
	}
	out, err := m.MarshalToString(src)
	if err != nil {
		return err
	}
	return jsonpb.UnmarshalString(out, dest)
}

// getSelectedSku returns the SKU from the listing which matches the provided options.
func getSelectedSku(listing *pb.Listing, options []*pb.OrderOpen_Item_Option) (*pb.Listing_Item_Sku, error) {
	if len(listing.Item.Options) == 0 {
		return &pb.Listing_Item_Sku{Surcharge: "0"}, nil
	}
	opts := make(map[string]string)
	for _, option := range options {
		opts[strings.ToLower(option.Name)] = strings.ToLower(option.Value)
	}
	for _, sku := range listing.Item.Skus {
		matches := true
		for _, sel := range sku.Selections {
			if opts[strings.ToLower(sel.Option)] != strings.ToLower(sel.Variant) {
				matches = false
			}
		}
		if matches {
			return sku, nil
		}
	}
	return nil, errors.New("selected sku not found in listing")
}

// validateBigString validates that the string is a base10 big number.
func validateBigString(s string) bool {
	_, ok := new(big.Int).SetString(s, 10)
	return ok
}
