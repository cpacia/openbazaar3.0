package core

import (
	"github.com/cpacia/openbazaar3.0/models/factory"
	"github.com/cpacia/openbazaar3.0/orders/pb"
	"github.com/golang/protobuf/ptypes"
	"github.com/ipfs/go-cid"
	"strings"
	"testing"
	"time"
)

func TestOpenBazaarNode_SaveListing(t *testing.T) {
	node, err := MockNode()
	if err != nil {
		t.Fatal(err)
	}

	listing := factory.NewPhysicalListing("ron-swanson-shirt")

	done := make(chan struct{})
	if err := node.SaveListing(listing, done); err != nil {
		t.Fatal(err)
	}
	<-done

	_, err = node.GetMyListingBySlug("ron-swanson-shirt")
	if err != nil {
		t.Fatal(err)
	}

	index, err := node.GetMyListings()
	if err != nil {
		t.Fatal(err)
	}

	if len(index) != 1 {
		t.Errorf("Returned incorrect number of listings. Expected %d, got %d", 1, len(index))
	}

	c, err := cid.Decode(index[0].Hash)
	if err != nil {
		t.Fatal(err)
	}
	_, err = node.GetMyListingByCID(c)
	if err != nil {
		t.Fatal(err)
	}
}

func TestOpenBazaarNode_DeleteListing(t *testing.T) {
	node, err := MockNode()
	if err != nil {
		t.Fatal(err)
	}

	listing := factory.NewPhysicalListing("ron-swanson-shirt")

	done := make(chan struct{})
	if err := node.SaveListing(listing, done); err != nil {
		t.Fatal(err)
	}
	<-done

	done2 := make(chan struct{})
	if err := node.DeleteListing(listing.Slug, done2); err != nil {
		t.Fatal(err)
	}
	<-done2

	_, err = node.GetMyListingBySlug("ron-swanson-shirt")
	if err == nil {
		t.Fatal(err)
	}

	index, err := node.GetMyListings()
	if err != nil {
		t.Fatal(err)
	}

	if len(index) != 0 {
		t.Errorf("Returned incorrect number of listings. Expected %d, got %d", 0, len(index))
	}
}

func TestOpenBazaarNode_ListingsGet(t *testing.T) {
	network, err := NewMocknet(2)
	if err != nil {
		t.Fatal(err)
	}

	listing := factory.NewPhysicalListing("ron-swanson-shirt")

	done := make(chan struct{})
	if err := network.Nodes()[0].SaveListing(listing, done); err != nil {
		t.Fatal(err)
	}
	<-done

	listing2, err := network.Nodes()[1].GetListingBySlug(network.Nodes()[0].Identity(), listing.Slug, false)
	if err != nil {
		t.Fatal(err)
	}

	if listing2.Slug != listing.Slug {
		t.Errorf("Incorrect slug returned. Expected %s, got %s", listing.Slug, listing2.Slug)
	}

	index, err := network.Nodes()[1].GetListings(network.Nodes()[0].Identity(), false)
	if err != nil {
		t.Fatal(err)
	}

	if len(index) != 1 {
		t.Errorf("Returned incorrect number of listings in index. Expected %d, got %d", 1, len(index))
	}

	c, err := cid.Decode(index[0].Hash)
	if err != nil {
		t.Fatal(err)
	}
	listing2, err = network.Nodes()[1].GetListingByCID(c)
	if err != nil {
		t.Fatal(err)
	}

	if listing2.Slug != listing.Slug {
		t.Errorf("Incorrect slug returned. Expected %s, got %s", listing.Slug, listing2.Slug)
	}
}

func Test_generateListingSlug(t *testing.T) {
	node, err := MockNode()
	if err != nil {
		t.Fatal(err)
	}

	listing := factory.NewPhysicalListing("ron-swanson-shirt")

	done := make(chan struct{})
	if err := node.SaveListing(listing, done); err != nil {
		t.Fatal(err)
	}
	<-done

	tests := []struct {
		title    string
		expected string
	}{
		{
			"test",
			"test",
		},
		{
			"test title",
			"test-title",
		},
		{
			"ron swanson shirt",
			"ron-swanson-shirt1",
		},
		{
			"💩💩💩",
			"and-x1f4a9-and-x1f4a9-and-x1f4a9",
		},
		{
			strings.Repeat("s", 65),
			strings.Repeat("s", 65),
		},
		{
			strings.Repeat("s", 66),
			strings.Repeat("s", 65),
		},
	}

	for _, test := range tests {
		slug, err := node.generateListingSlug(test.title)
		if err != nil {
			t.Fatal(err)
		}
		if slug != test.expected {
			t.Errorf("Expected slug %s, got %s", test.expected, slug)
		}
	}
}

func Test_validateCryptocurrencyListing(t *testing.T) {
	node, err := MockNode()
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct{
		listing *pb.Listing
		transform func(listing *pb.Listing)
		valid bool
	}{
		{
			listing: factory.NewCryptoListing("test-listing"),
			transform: func(listing *pb.Listing){},
			valid: true,
		},
		{
			listing: factory.NewCryptoListing("test-listing"),
			transform: func(listing *pb.Listing){
				listing.Coupons = []*pb.Listing_Coupon {
					{
						Title: "fads",
					},
				}
			},
			valid: false,
		},
		{
			listing: factory.NewCryptoListing("test-listing"),
			transform: func(listing *pb.Listing){
				listing.Item.Options = []*pb.Listing_Item_Option {
					{
						Name: "fasdf",
					},
				}
			},
			valid: false,
		},
		{
			listing: factory.NewCryptoListing("test-listing"),
			transform: func(listing *pb.Listing){
				listing.ShippingOptions = []*pb.Listing_ShippingOption {
					{
						Name: "fasdf",
					},
				}
			},
			valid: false,
		},
		{
			listing: factory.NewCryptoListing("test-listing"),
			transform: func(listing *pb.Listing){
				listing.Item.Condition = "terrible"
			},
			valid: false,
		},
		{
			listing: factory.NewCryptoListing("test-listing"),
			transform: func(listing *pb.Listing){
				listing.Metadata.PricingCurrency.Divisibility = 10
			},
			valid: false,
		},
	}

	for i, test := range tests {
		test.transform(test.listing)
		err := node.validateCryptocurrencyListing(test.listing)
		if test.valid && err != nil {
			t.Errorf("Test %d failed when it should not have: %s", i, err)
		} else if !test.valid && err == nil {
			t.Errorf("Test %d did not fail when it should have", i)
		}
	}
}

func Test_validateMarketPriceListing(t *testing.T) {
	tests := []struct {
		listing   *pb.Listing
		transform func(listing *pb.Listing)
		valid     bool
	}{
		{
			listing:   factory.NewCryptoListing("test-listing"),
			transform: func(listing *pb.Listing) {
				listing.Metadata.Format = pb.Listing_Metadata_MARKET_PRICE
				listing.Item.Price = ""
			},
			valid:     true,
		},
		{
			listing:   factory.NewCryptoListing("test-listing"),
			transform: func(listing *pb.Listing) {
				listing.Metadata.Format = pb.Listing_Metadata_MARKET_PRICE
				listing.Item.Price = ""
				listing.Metadata.PriceModifier = -99.99
			},
			valid:     true,
		},
		{
			listing:   factory.NewCryptoListing("test-listing"),
			transform: func(listing *pb.Listing) {
				listing.Metadata.Format = pb.Listing_Metadata_MARKET_PRICE
				listing.Item.Price = ""
				listing.Metadata.PriceModifier = 1000
			},
			valid:     true,
		},
		{
			listing:   factory.NewCryptoListing("test-listing"),
			transform: func(listing *pb.Listing) {
				listing.Metadata.Format = pb.Listing_Metadata_MARKET_PRICE
				listing.Item.Price = ""
				listing.Metadata.PriceModifier = -100
			},
			valid:     false,
		},
		{
			listing:   factory.NewCryptoListing("test-listing"),
			transform: func(listing *pb.Listing) {
				listing.Metadata.Format = pb.Listing_Metadata_MARKET_PRICE
				listing.Item.Price = ""
				listing.Metadata.PriceModifier = 1001
			},
			valid:     false,
		},
	}

	for i, test := range tests {
		test.transform(test.listing)
		err := validateMarketPriceListing(test.listing)
		if test.valid && err != nil {
			t.Errorf("Test %d failed when it should not have: %s", i, err)
		} else if !test.valid && err == nil {
			t.Errorf("Test %d did not fail when it should have", i)
		}
	}
}

func Test_validatePhysicalListing(t *testing.T) {
	tests := []struct {
		listing   *pb.Listing
		transform func(listing *pb.Listing)
		valid     bool
	}{
		{
			listing:   factory.NewPhysicalListing("test-listing"),
			transform: func(listing *pb.Listing) {},
			valid:     true,
		},
		{
			listing:   factory.NewPhysicalListing("test-listing"),
			transform: func(listing *pb.Listing) {
				listing.Metadata.PricingCurrency = nil
			},
			valid:     false,
		},
		{
			listing:   factory.NewPhysicalListing("test-listing"),
			transform: func(listing *pb.Listing) {
				listing.Metadata.PricingCurrency.Code = ""
			},
			valid:     false,
		},
		{
			listing:   factory.NewPhysicalListing("test-listing"),
			transform: func(listing *pb.Listing) {
				listing.Metadata.PricingCurrency.Name = ""
			},
			valid:     false,
		},
		{
			listing:   factory.NewPhysicalListing("test-listing"),
			transform: func(listing *pb.Listing) {
				listing.Metadata.PricingCurrency.CurrencyType = ""
			},
			valid:     false,
		},
		{
			listing:   factory.NewPhysicalListing("test-listing"),
			transform: func(listing *pb.Listing) {
				listing.Metadata.PricingCurrency.Code = strings.Repeat("s", WordMaxCharacters+1)
			},
			valid:     false,
		},
		{
			listing:   factory.NewPhysicalListing("test-listing"),
			transform: func(listing *pb.Listing) {
				listing.Metadata.PricingCurrency.Name = strings.Repeat("s", WordMaxCharacters+1)
			},
			valid:     false,
		},
		{
			listing:   factory.NewPhysicalListing("test-listing"),
			transform: func(listing *pb.Listing) {
				listing.Metadata.PricingCurrency.CurrencyType = strings.Repeat("s", WordMaxCharacters+1)
			},
			valid:     false,
		},
		{
			listing:   factory.NewPhysicalListing("test-listing"),
			transform: func(listing *pb.Listing) {
				listing.Item.Condition = strings.Repeat("s", SentenceMaxCharacters+1)
			},
			valid:     false,
		},
		{
			listing:   factory.NewPhysicalListing("test-listing"),
			transform: func(listing *pb.Listing) {
				for i:=0; i<MaxListItems+1; i++ {
					listing.Item.Options = append(listing.Item.Options, &pb.Listing_Item_Option{
						Name: "fadsfa",
					})
				}
			},
			valid:     false,
		},
		{
			listing:   factory.NewPhysicalListing("test-listing"),
			transform: func(listing *pb.Listing) {
				listing.ShippingOptions = []*pb.Listing_ShippingOption{}
			},
			valid:     false,
		},
		{
			listing:   factory.NewPhysicalListing("test-listing"),
			transform: func(listing *pb.Listing) {
				listing.ShippingOptions = []*pb.Listing_ShippingOption{}
				for i:=0; i<MaxListItems+1; i++ {
					listing.ShippingOptions = append(listing.ShippingOptions, &pb.Listing_ShippingOption{
						Name: "fadsfa",
					})
				}
			},
			valid:     false,
		},
		{
			listing:   factory.NewPhysicalListing("test-listing"),
			transform: func(listing *pb.Listing) {
				listing.ShippingOptions[0].Name = ""
			},
			valid:     false,
		},
		{
			listing:   factory.NewPhysicalListing("test-listing"),
			transform: func(listing *pb.Listing) {
				listing.ShippingOptions[0].Name = strings.Repeat("s", WordMaxCharacters+1)
				listing.ShippingOptions[0].Regions = []pb.CountryCode {
					pb.CountryCode_UNITED_ARAB_EMIRATES,
				}
			},
			valid:     false,
		},
		{
			listing:   factory.NewPhysicalListing("test-listing"),
			transform: func(listing *pb.Listing) {
				listing.ShippingOptions = []*pb.Listing_ShippingOption{}
				for i:=0; i<2; i++ {
					listing.ShippingOptions = append(listing.ShippingOptions, &pb.Listing_ShippingOption{
						Name: "fadsfa",
						Regions: []pb.CountryCode {
							pb.CountryCode_UNITED_ARAB_EMIRATES,
						},
					})
				}
			},
			valid:     false,
		},
		{
			listing:   factory.NewPhysicalListing("test-listing"),
			transform: func(listing *pb.Listing) {
				listing.ShippingOptions[0].Name = "afsdf"
				listing.ShippingOptions[0].Regions = []pb.CountryCode {
					pb.CountryCode_UNITED_ARAB_EMIRATES,
				}
				listing.ShippingOptions[0].Type = pb.Listing_ShippingOption_FIXED_PRICE + 1
			},
			valid:     false,
		},
		{
			listing:   factory.NewPhysicalListing("test-listing"),
			transform: func(listing *pb.Listing) {
				listing.ShippingOptions[0].Name = "afsdf"
				listing.ShippingOptions[0].Type = pb.Listing_ShippingOption_FIXED_PRICE
				listing.ShippingOptions[0].Regions = []pb.CountryCode{}
			},
			valid:     false,
		},
		{
			listing:   factory.NewPhysicalListing("test-listing"),
			transform: func(listing *pb.Listing) {
				listing.ShippingOptions[0].Name = "afsdf"
				listing.ShippingOptions[0].Type = pb.Listing_ShippingOption_FIXED_PRICE
				listing.ShippingOptions[0].Regions = []pb.CountryCode{
					0,
				}
			},
			valid:     false,
		},
		{
			listing:   factory.NewPhysicalListing("test-listing"),
			transform: func(listing *pb.Listing) {
				listing.ShippingOptions[0].Name = "afsdf"
				listing.ShippingOptions[0].Type = pb.Listing_ShippingOption_FIXED_PRICE
				listing.ShippingOptions[0].Regions = []pb.CountryCode{
					501,
				}
			},
			valid:     false,
		},
		{
			listing:   factory.NewPhysicalListing("test-listing"),
			transform: func(listing *pb.Listing) {
				listing.ShippingOptions[0].Name = "afsdf"
				listing.ShippingOptions[0].Type = pb.Listing_ShippingOption_FIXED_PRICE
				listing.ShippingOptions[0].Regions = []pb.CountryCode{}
				for i:=0; i<MaxCountryCodes+1; i++ {
					listing.ShippingOptions[0].Regions = append(listing.ShippingOptions[0].Regions, pb.CountryCode_AFGHANISTAN)
				}
			},
			valid:     false,
		},
		{
			listing:   factory.NewPhysicalListing("test-listing"),
			transform: func(listing *pb.Listing) {
				listing.ShippingOptions[0].Name = "afsdf"
				listing.ShippingOptions[0].Type = pb.Listing_ShippingOption_FIXED_PRICE
				listing.ShippingOptions[0].Services = []*pb.Listing_ShippingOption_Service{}
			},
			valid:     false,
		},
		{
			listing:   factory.NewPhysicalListing("test-listing"),
			transform: func(listing *pb.Listing) {
				listing.ShippingOptions[0].Name = "afsdf"
				listing.ShippingOptions[0].Type = pb.Listing_ShippingOption_FIXED_PRICE
				listing.ShippingOptions[0].Services = []*pb.Listing_ShippingOption_Service{}
				for i:=0; i<MaxListItems+1; i++ {
					listing.ShippingOptions[0].Services = append(listing.ShippingOptions[0].Services, &pb.Listing_ShippingOption_Service{
						Name: "afdsf",
					})
				}
			},
			valid:     false,
		},
		{
			listing:   factory.NewPhysicalListing("test-listing"),
			transform: func(listing *pb.Listing) {
				listing.ShippingOptions[0].Name = "afsdf"
				listing.ShippingOptions[0].Type = pb.Listing_ShippingOption_FIXED_PRICE
				listing.ShippingOptions[0].Services = []*pb.Listing_ShippingOption_Service{
					{
						Name: "",
					},
				}
			},
			valid:     false,
		},
		{
			listing:   factory.NewPhysicalListing("test-listing"),
			transform: func(listing *pb.Listing) {
				listing.ShippingOptions[0].Name = "afsdf"
				listing.ShippingOptions[0].Type = pb.Listing_ShippingOption_FIXED_PRICE
				listing.ShippingOptions[0].Services = []*pb.Listing_ShippingOption_Service{
					{
						Name: strings.Repeat("s", WordMaxCharacters+1),
					},
				}
			},
			valid:     false,
		},
		{
			listing:   factory.NewPhysicalListing("test-listing"),
			transform: func(listing *pb.Listing) {
				listing.ShippingOptions[0].Name = "afsdf"
				listing.ShippingOptions[0].Type = pb.Listing_ShippingOption_FIXED_PRICE
				listing.ShippingOptions[0].Services = []*pb.Listing_ShippingOption_Service{
					{
						Name: "asdf",
						EstimatedDelivery: "adf",
					},
					{
						Name: "asdf",
					},
				}
			},
			valid:     false,
		},
		{
			listing:   factory.NewPhysicalListing("test-listing"),
			transform: func(listing *pb.Listing) {
				listing.ShippingOptions[0].Name = "afsdf"
				listing.ShippingOptions[0].Type = pb.Listing_ShippingOption_FIXED_PRICE
				listing.ShippingOptions[0].Services = []*pb.Listing_ShippingOption_Service{
					{
						Name: "asdf",
					},
				}
			},
			valid:     false,
		},
		{
			listing:   factory.NewPhysicalListing("test-listing"),
			transform: func(listing *pb.Listing) {
				listing.ShippingOptions[0].Name = "afsdf"
				listing.ShippingOptions[0].Type = pb.Listing_ShippingOption_FIXED_PRICE
				listing.ShippingOptions[0].Services = []*pb.Listing_ShippingOption_Service{
					{
						Name: "asdf",
						EstimatedDelivery: strings.Repeat("s", SentenceMaxCharacters+1),
					},
				}
			},
			valid:     false,
		},
		{
			listing:   factory.NewPhysicalListing("test-listing"),
			transform: func(listing *pb.Listing) {
				listing.ShippingOptions[0].Name = "afsdf"
				listing.ShippingOptions[0].Type = pb.Listing_ShippingOption_FIXED_PRICE
				listing.ShippingOptions[0].Services = []*pb.Listing_ShippingOption_Service{
					{
						Name: "asdf",
						EstimatedDelivery: "asdf",
						Price: strings.Repeat("s", WordMaxCharacters+1),
					},
				}
			},
			valid:     false,
		},
	}

	for i, test := range tests {
		test.transform(test.listing)
		err := validatePhysicalListing(test.listing)
		if test.valid && err != nil {
			t.Errorf("Test %d failed when it should not have: %s", i, err)
		} else if !test.valid && err == nil {
			t.Errorf("Test %d did not fail when it should have", i)
		}
	}
}

func Test_validateListing(t *testing.T) {
	node, err := MockNode()
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		listing   *pb.SignedListing
		transform func(sl *pb.SignedListing)
		valid     bool
	}{
		{
			listing:   factory.NewSignedListing(),
			transform: func(sl *pb.SignedListing) {},
			valid:     true,
		},
		{
			listing:   factory.NewSignedListing(),
			transform: func(sl *pb.SignedListing) {
				sl.Listing.Slug = ""
			},
			valid:     false,
		},
		{
			listing:   factory.NewSignedListing(),
			transform: func(sl *pb.SignedListing) {
				sl.Listing.Slug = strings.Repeat("s", SentenceMaxCharacters+1)
			},
			valid:     false,
		},
		{
			listing:   factory.NewSignedListing(),
			transform: func(sl *pb.SignedListing) {
				sl.Listing.Slug = " "
			},
			valid:     false,
		},
		{
			listing:   factory.NewSignedListing(),
			transform: func(sl *pb.SignedListing) {
				sl.Listing.Slug = "/"
			},
			valid:     false,
		},
		{
			listing:   factory.NewSignedListing(),
			transform: func(sl *pb.SignedListing) {
				sl.Listing.Metadata = nil
			},
			valid:     false,
		},
		{
			listing:   factory.NewSignedListing(),
			transform: func(sl *pb.SignedListing) {
				sl.Listing.Metadata.ContractType = pb.Listing_Metadata_CRYPTOCURRENCY+1
			},
			valid:     false,
		},
		{
			listing:   factory.NewSignedListing(),
			transform: func(sl *pb.SignedListing) {
				sl.Listing.Metadata.Format = pb.Listing_Metadata_MARKET_PRICE+1
			},
			valid:     false,
		},
		{
			listing:   factory.NewSignedListing(),
			transform: func(sl *pb.SignedListing) {
				sl.Listing.Metadata.Expiry = nil
			},
			valid:     false,
		},
		{
			listing:   factory.NewSignedListing(),
			transform: func(sl *pb.SignedListing) {
				ts, _ := ptypes.TimestampProto(time.Time{})
				sl.Listing.Metadata.Expiry = ts
			},
			valid:     false,
		},
		{
			listing:   factory.NewSignedListing(),
			transform: func(sl *pb.SignedListing) {
				sl.Listing.Metadata.Language = strings.Repeat("s", WordMaxCharacters+1)
			},
			valid:     false,
		},
		{
			listing:   factory.NewSignedListing(),
			transform: func(sl *pb.SignedListing) {
				node.testnet = false
				sl.Listing.Metadata.EscrowTimeoutHours = 1
			},
			valid:     false,
		},
		{
			listing:   factory.NewSignedListing(),
			transform: func(sl *pb.SignedListing) {
				sl.Listing.Metadata.AcceptedCurrencies = []string{}
			},
			valid:     false,
		},
		{
			listing:   factory.NewSignedListing(),
			transform: func(sl *pb.SignedListing) {
				sl.Listing.Metadata.AcceptedCurrencies = []string{}
				for i:=0; i<MaxListItems+1; i++ {
					sl.Listing.Metadata.AcceptedCurrencies = append(sl.Listing.Metadata.AcceptedCurrencies, "abc")
				}
			},
			valid:     false,
		},
		{
			listing:   factory.NewSignedListing(),
			transform: func(sl *pb.SignedListing) {
				sl.Listing.Metadata.AcceptedCurrencies = []string{
					strings.Repeat("s", WordMaxCharacters+1),
				}
			},
			valid:     false,
		},
	}

	for i, test := range tests {
		test.transform(test.listing)
		err := node.validateListing(test.listing)
		if test.valid && err != nil {
			t.Errorf("Test %d failed when it should not have: %s", i, err)
		} else if !test.valid && err == nil {
			t.Errorf("Test %d did not fail when it should have", i)
		}
	}
}