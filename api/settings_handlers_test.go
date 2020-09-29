package api

import (
	"fmt"
	"github.com/cpacia/multiwallet"
	"github.com/cpacia/openbazaar3.0/core/coreiface"
	"github.com/cpacia/openbazaar3.0/models"
	"github.com/cpacia/openbazaar3.0/version"
	"github.com/cpacia/openbazaar3.0/wallet"
	iwallet "github.com/cpacia/wallet-interface"
	peer "github.com/libp2p/go-libp2p-core/peer"
	"net/http"
	"testing"
)

func TestSettingsHandlers(t *testing.T) {
	runAPITests(t, apiTests{
		{
			name:   "Get config",
			path:   "/v1/ob/config",
			method: http.MethodGet,
			setNodeMethods: func(n *mockNode) {
				n.identityFunc = func() peer.ID {
					p, _ := peer.Decode("12D3KooWBfmETW1ZbkdZbKKPpE3jpjyQ5WBXoDF8y9oE8vMQPKLi")
					return p
				}
				n.usingTestnetFunc = func() bool {
					return true
				}
				n.usingTorFunc = func() bool {
					return true
				}
				n.multiwalletFunc = func() multiwallet.Multiwallet {
					m := make(multiwallet.Multiwallet)
					m[iwallet.CtBitcoin] = nil
					return m
				}
			},
			statusCode: http.StatusOK,
			expectedResponse: func() ([]byte, error) {
				n := nodeConfig{
					PeerID:  "12D3KooWBfmETW1ZbkdZbKKPpE3jpjyQ5WBXoDF8y9oE8vMQPKLi",
					Testnet: true,
					Tor:     true,
					Wallets: []string{"BTC"},
				}
				return marshalAndSanitizeJSON(&n)
			},
		},
		{
			name:   "Put user preferences",
			path:   "/v1/ob/preferences",
			method: http.MethodPut,
			setNodeMethods: func(n *mockNode) {
				n.saveUserPreferencesFunc = func(prefs *models.UserPreferences, done chan struct{}) error {
					return nil
				}
			},
			body:       []byte(`{"RefundPolicy": "asdf"}`),
			statusCode: http.StatusOK,
			expectedResponse: func() ([]byte, error) {
				return []byte(`{}`), nil
			},
		},
		{
			name:   "Put user preferences bad request",
			path:   "/v1/ob/preferences",
			method: http.MethodPut,
			setNodeMethods: func(n *mockNode) {
				n.saveUserPreferencesFunc = func(prefs *models.UserPreferences, done chan struct{}) error {
					return coreiface.ErrBadRequest
				}
			},
			body:       []byte(`{"RefundPolicy": "asdf"}`),
			statusCode: http.StatusBadRequest,
			expectedResponse: func() ([]byte, error) {
				return []byte(fmt.Sprintf("%s\n", `{"error": "bad request"}`)), nil
			},
		},
		{
			name:   "Get user preferences",
			path:   "/v1/ob/preferences",
			method: http.MethodGet,
			setNodeMethods: func(n *mockNode) {
				n.getUserPreferencesFunc = func() (*models.UserPreferences, error) {
					return &models.UserPreferences{
						RefundPolicy: "asdf",
					}, nil
				}
			},
			statusCode: http.StatusOK,
			expectedResponse: func() ([]byte, error) {
				return marshalAndSanitizeJSON(&models.UserPreferences{RefundPolicy: "asdf", UserAgent: version.UserAgent()})
			},
		},
		{
			name:   "Get exchange rates",
			path:   "/v1/ob/exchangerates",
			method: http.MethodGet,
			setNodeMethods: func(n *mockNode) {
				n.getExchangeRatesFunc = func() *wallet.ExchangeRateProvider {
					erp, _ := wallet.NewMockExchangeRates()
					return erp
				}
			},
			statusCode: http.StatusOK,
			expectedResponse: func() ([]byte, error) {
				erp, err := wallet.NewMockExchangeRates()
				if err != nil {
					return nil, err
				}
				rates, err := erp.GetAllRates(iwallet.CtBitcoin, true)
				if err != nil {
					return nil, err
				}
				return marshalAndSanitizeJSON(rates)
			},
		},
	})
}
