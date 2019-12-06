package models

import (
	"encoding/json"
	peer "github.com/libp2p/go-libp2p-peer"
)

// UserPreferences are set by the client and persisted in the database.
type UserPreferences struct {
	PaymentDataInQR    bool            `json:"paymentDataInQR"`
	ShowNotifications  bool            `json:"showNotifications"`
	ShowNsfw           bool            `json:"showNsfw"`
	ShippingAddresses  json.RawMessage `json:"shippingAddresses"`
	LocalCurrency      string          `json:"localCurrency"`
	Country            string          `json:"country"`
	TermsAndConditions string          `json:"termsAndConditions"`
	RefundPolicy       string          `json:"refundPolicy"`
	Blocked            json.RawMessage `json:"blockedNodes"`
	StoreModerators    json.RawMessage `json:"storeModerators"`
	MisPaymentBuffer   float32         `json:"mispaymentBuffer"`
	AutoConfirm        bool            `json:"autoConfirm"`
	EmailNotifications string          `json:"emailNotifications"`
	PrefCurrencies     json.RawMessage `json:"preferredCurrencies"`
}

type shippingAddress struct {
	Name           string `json:"name"`
	Company        string `json:"company"`
	AddressLineOne string `json:"addressLineOne"`
	AddressLineTwo string `json:"addressLineTwo"`
	City           string `json:"city"`
	State          string `json:"state"`
	Country        string `json:"country"`
	PostalCode     string `json:"postalCode"`
	AddressNotes   string `json:"addressNotes"`
}

type prefsJSON struct {
	PaymentDataInQR     bool              `json:"paymentDataInQR"`
	ShowNotifications   bool              `json:"showNotifications"`
	ShowNsfw            bool              `json:"showNsfw"`
	ShippingAddresses   []shippingAddress `json:"shippingAddresses"`
	LocalCurrency       string            `json:"localCurrency"`
	Country             string            `json:"country"`
	TermsAndConditions  string            `json:"termsAndConditions"`
	RefundPolicy        string            `json:"refundPolicy"`
	BlockedNodes        []string          `json:"blockedNodes"`
	StoreModerators     []string          `json:"storeModerators"`
	MisPaymentBuffer    float32           `json:"mispaymentBuffer"`
	AutoConfirm         bool              `json:"autoConfirm"`
	EmailNotifications  string            `json:"emailNotifications"`
	PreferredCurrencies []string          `json:"preferredCurrencies"`
}

// BlockedNodes returns the blocked peer IDs.
func (prefs *UserPreferences) BlockedNodes() ([]peer.ID, error) {
	var peerIDStrs []string
	if prefs.Blocked != nil {
		if err := json.Unmarshal(prefs.Blocked, &peerIDStrs); err != nil {
			return nil, err
		}
	}
	ret := make([]peer.ID, len(peerIDStrs))
	for _, s := range peerIDStrs {
		pid, err := peer.IDB58Decode(s)
		if err != nil {
			return nil, err
		}
		ret = append(ret, pid)
	}
	return ret, nil
}

// PreferredCurrencies returns the preferred currencies for the node.
func (prefs *UserPreferences) PreferredCurrencies() ([]string, error) {
	var prefCurrencies []string
	if prefs.PrefCurrencies != nil {
		if err := json.Unmarshal(prefs.PrefCurrencies, &prefCurrencies); err != nil {
			return nil, err
		}
	}
	return prefCurrencies, nil
}

// UnmarshalJSON unmarshals the JSON object into a UserPreferences object.
func (prefs *UserPreferences) UnmarshalJSON(b []byte) error {
	var c0 prefsJSON

	err := json.Unmarshal(b, &c0)
	if err == nil {
		shippingAddrs, err := json.Marshal(c0.ShippingAddresses)
		if err != nil {
			return err
		}
		blockedNodes, err := json.Marshal(c0.BlockedNodes)
		if err != nil {
			return err
		}
		storeModerators, err := json.Marshal(c0.StoreModerators)
		if err != nil {
			return err
		}
		preferredCurrencies, err := json.Marshal(c0.PreferredCurrencies)
		if err != nil {
			return err
		}

		prefs.PaymentDataInQR = c0.PaymentDataInQR
		prefs.ShowNotifications = c0.ShowNotifications
		prefs.ShowNsfw = c0.ShowNsfw
		prefs.ShippingAddresses = shippingAddrs
		prefs.LocalCurrency = c0.LocalCurrency
		prefs.Country = c0.Country
		prefs.TermsAndConditions = c0.TermsAndConditions
		prefs.RefundPolicy = c0.RefundPolicy
		prefs.Blocked = blockedNodes
		prefs.StoreModerators = storeModerators
		prefs.MisPaymentBuffer = c0.MisPaymentBuffer
		prefs.AutoConfirm = c0.AutoConfirm
		prefs.EmailNotifications = c0.EmailNotifications
		prefs.PrefCurrencies = preferredCurrencies
	}

	return err
}
