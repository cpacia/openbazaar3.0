package orders

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/btcsuite/btcd/btcec"
	"github.com/cpacia/openbazaar3.0/database"
	"github.com/cpacia/openbazaar3.0/events"
	"github.com/cpacia/openbazaar3.0/models"
	npb "github.com/cpacia/openbazaar3.0/net/pb"
	"github.com/cpacia/openbazaar3.0/orders/pb"
	"github.com/cpacia/openbazaar3.0/wallet"
	iwallet "github.com/cpacia/wallet-interface"
	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes"
	crypto "github.com/libp2p/go-libp2p-crypto"
	peer "github.com/libp2p/go-libp2p-peer"
	"github.com/multiformats/go-multihash"
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

			if err := order.PutMessage(&resp); err != nil {
				return nil, err
			}
		}
		validationError = true
	}

	var event interface{}
	// TODO: do we want to emit an event in the case of a validation error?
	if !validationError && op.identity != peer {
		event = &events.OrderNotification{
			ID: order.ID.String(),
		}
	}

	if err := order.PutMessage(orderOpen); err != nil {
		return nil, err
	}

	return event, nil
}

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

	if op.identity.Pretty() != order.BuyerID.PeerID { // If we are vendor.
		// Check to make sure we actually have the item for sale.
		for _, listing := range order.Listings {
			myListing, err := dbtx.GetListing(listing.Listing.Slug)
			if err != nil {
				return fmt.Errorf("item %s is not for sale", listing.Listing.Slug)
			}

			// Zero out the inventory on each listing. We will check
			// inventory later.
			for i := range myListing.Listing.Item.Skus {
				myListing.Listing.Item.Skus[i].Quantity = 0
			}
			for i := range listing.Listing.Item.Skus {
				listing.Listing.Item.Skus[i].Quantity = 0
			}

			// We can tell if we have the listing for sale if the serialized bytes match
			// after we've zeroed out the inventory.
			mySer, err := proto.Marshal(myListing.Listing)
			if err != nil {
				return err
			}

			theirSer, err := proto.Marshal(listing.Listing)
			if err != nil {
				return err
			}

			if !bytes.Equal(mySer, theirSer) {
				return fmt.Errorf("item %s is not for sale", listing.Listing.Slug)
			}
		}
	}

	for i, item := range order.Items {
		// Let's check to make sure there is a listing for each
		// item in the order.
		listing, err := extractListing(item.ListingHash, order.Listings)
		if err != nil {
			return fmt.Errorf("listing not found in order for item %s", item.ListingHash)
		}

		// Validate shipping is provided if physical good.
		if listing.Metadata == nil {
			return fmt.Errorf("listing %d metadata is nil", i)
		}
		if listing.Metadata.ContractType == pb.Listing_Metadata_PHYSICAL_GOOD && item.ShippingOption == nil {
			return fmt.Errorf("item %d shipping option is nil", i)
		}

		// Validate pricing currency is not nil
		if listing.Metadata.PricingCurrency == nil {
			return fmt.Errorf("listing %d pricing currency is nil", i)
		}

		// Validate listing item is not nil
		if listing.Item == nil {
			return fmt.Errorf("listing %d item is nil", i)
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
			optionMap[option.Name] = option.Value
		}
		for _, opt := range listing.Item.Options {
			val, ok := optionMap[opt.Name]
			if !ok {
				return fmt.Errorf("item %d option %s not found in listing", i, opt.Name)
			}
			valExists := false
			for _, variant := range opt.Variants {
				if variant.Name == val {
					valExists = true
					break
				}
			}
			if !valExists {
				return fmt.Errorf("item %d variant %s not found in listing option", i, val)
			}
		}
		for x, sku := range listing.Item.Skus {
			if sku.Surcharge == "" {
				return fmt.Errorf("item %d listing sku %d surcharge not set", i, x)
			}
			if ok := validateBigString(sku.Surcharge); !ok {
				return fmt.Errorf("item %d listing sku %d surcharge not valid", i, x)
			}
		}

		// Validate shipping option
		if item.ShippingOption != nil {
			shippingOpts := make(map[string][]*pb.Listing_ShippingOption_Service)
			for _, option := range listing.ShippingOptions {
				shippingOpts[option.Name] = option.Services
			}
			services, ok := shippingOpts[item.ShippingOption.Name]
			if !ok {
				return fmt.Errorf("item %d shipping option %s not found in listing", i, item.ShippingOption.Name)
			}
			serviceExists := false
			for x, service := range services {
				if service.Name == item.ShippingOption.Service {
					serviceExists = true
				}
				if service.Price == "" {
					return fmt.Errorf("item %d shipping service %d price not set", i, x)
				}
				if ok := validateBigString(service.Price); !ok {
					return fmt.Errorf("item %d shipping service %d price not valid", i, x)
				}
			}
			if !serviceExists {
				return fmt.Errorf("item %d shipping service %s not found in listing option", i, item.ShippingOption.Service)
			}
		}

		// Validate coupon discount price
		for _, coupon := range listing.Coupons {
			if coupon.GetPriceDiscount() != "" {
				if ok := validateBigString(coupon.GetPriceDiscount()); !ok {
					return fmt.Errorf("item %d listing coupon price discount not valid", i)
				}
			}
		}

		// Validate listing item price
		if listing.Item.Price == "" {
			return fmt.Errorf("item %d listing price not set", i)
		}
		if ok := validateBigString(listing.Item.Price); !ok {
			return fmt.Errorf("item %d listing price not valid", i)
		}
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
	_, err := models.CurrencyDefinitions.Lookup(order.Payment.Coin)
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
	if order.Payment.Method == pb.OrderOpen_Payment_MODERATED {
		_, err := peer.IDB58Decode(order.Payment.Moderator)
		if err != nil {
			return errors.New("invalid moderator selection")
		}
		if order.Payment.ModeratorKey == nil {
			return errors.New("moderator key not set")
		}
	}

	// Validate rating keys
	for _, key := range order.RatingKeys {
		if _, err := btcec.ParsePubKey(key, btcec.S256()); err != nil {
			return errors.New("invalid rating pubkey")
		}
	}

	// Validate buyer ID
	if order.BuyerID.Pubkeys == nil {
		return errors.New("buyer id pubkeys is nil")
	}
	idPubkey, err := crypto.UnmarshalEd25519PublicKey(order.BuyerID.Pubkeys.Identity)
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

	// Make sure the provided total is the same as what we calculate.
	total, err := CalculateOrderTotal(order, op.erp)
	if err != nil {
		return err
	}
	if total.Cmp(iwallet.NewAmount(order.Payment.Amount)) != 0 {
		return fmt.Errorf("invalid payment amount: expected %s, got %s", total, order.Payment.Amount)
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
			h := sha256.Sum256([]byte(couponCode))
			encoded, err := multihash.Encode(h[:], multihash.SHA2_256)
			if err != nil {
				return orderTotal, err
			}
			couponHash, err := multihash.Cast(encoded)
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

	div := new(big.Float).Quo(rateFloat, big.NewFloat(float64(math.Pow10(int(value.Currency.Divisibility)))))
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
		mh, err := hashListing(sl)
		if err != nil {
			return nil, err
		}
		if mh.B58String() == hash {
			return sl.Listing, nil
		}
	}
	return nil, fmt.Errorf("listing %s not found in order", hash)
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

func hashListing(listing *pb.SignedListing) (multihash.Multihash, error) {
	ser, err := proto.Marshal(listing)
	if err != nil {
		return nil, err
	}
	h := sha256.Sum256(ser)
	encoded, err := multihash.Encode(h[:], multihash.SHA2_256)
	if err != nil {
		return nil, err
	}
	mh, err := multihash.Cast(encoded)
	if err != nil {
		return nil, err
	}
	return mh, nil
}

// validateBigString validates that the string is a base10 big number.
func validateBigString(s string) bool {
	_, ok := new(big.Int).SetString(s, 10)
	return ok
}
