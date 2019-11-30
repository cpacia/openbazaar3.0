package api

import (
	"github.com/cpacia/multiwallet"
	"github.com/cpacia/openbazaar3.0/wallet"
	iwallet "github.com/cpacia/wallet-interface"
	"net/http"
	"testing"
)

func TestWalletHandlers(t *testing.T) {
	runAPITests(t, apiTests{
		{
			name:   "Get all balances",
			path:   "/v1/wallet/balance",
			method: http.MethodGet,
			setNodeMethods: func(n *mockNode) {
				n.multiwalletFunc = func() multiwallet.Multiwallet {
					mw := multiwallet.Multiwallet{
						"MCK": wallet.NewMockWallet(),
					}
					return mw
				}
			},
			statusCode: http.StatusOK,
			expectedResponse: func() ([]byte, error) {
				ret := map[string]walletBalanceResponse{
					"MCK": {
						Height:      0,
						Unconfirmed: "0",
						Confirmed:   "0",
					},
				}
				return marshalAndSanitizeJSON(ret)
			},
		},
		{
			name:   "Get specific balance",
			path:   "/v1/wallet/balance/mck",
			method: http.MethodGet,
			setNodeMethods: func(n *mockNode) {
				n.multiwalletFunc = func() multiwallet.Multiwallet {
					mw := multiwallet.Multiwallet{
						"MCK": wallet.NewMockWallet(),
					}
					return mw
				}
			},
			statusCode: http.StatusOK,
			expectedResponse: func() ([]byte, error) {
				return marshalAndSanitizeJSON(walletBalanceResponse{
					Height:      0,
					Unconfirmed: "0",
					Confirmed:   "0",
				})
			},
		},
		{
			name:   "Get balance unknown wallet",
			path:   "/v1/wallet/balance/abc",
			method: http.MethodGet,
			setNodeMethods: func(n *mockNode) {
				n.multiwalletFunc = func() multiwallet.Multiwallet {
					mw := multiwallet.Multiwallet{
						"MCK": wallet.NewMockWallet(),
					}
					return mw
				}
			},
			statusCode: http.StatusBadRequest,
			expectedResponse: func() ([]byte, error) {
				return []byte("multiwallet does not contain an implementation for the given coin\n"), nil
			},
		},
		{
			name:   "Get all addresses",
			path:   "/v1/wallet/address",
			method: http.MethodGet,
			setNodeMethods: func(n *mockNode) {
				n.multiwalletFunc = func() multiwallet.Multiwallet {
					w := wallet.NewMockWallet()
					w.SetAddressResponse(iwallet.NewAddress("abc", iwallet.CtMock))

					mw := multiwallet.Multiwallet{
						"MCK": w,
					}
					return mw
				}
			},
			statusCode: http.StatusOK,
			expectedResponse: func() ([]byte, error) {
				ret := map[string]walletAddressResponse{
					"MCK": {
						Address: "abc",
					},
				}
				return marshalAndSanitizeJSON(ret)
			},
		},
		{
			name:   "Get specific adddress",
			path:   "/v1/wallet/address/mck",
			method: http.MethodGet,
			setNodeMethods: func(n *mockNode) {
				n.multiwalletFunc = func() multiwallet.Multiwallet {
					w := wallet.NewMockWallet()
					w.SetAddressResponse(iwallet.NewAddress("abc", iwallet.CtMock))

					mw := multiwallet.Multiwallet{
						"MCK": w,
					}
					return mw
				}
			},
			statusCode: http.StatusOK,
			expectedResponse: func() ([]byte, error) {
				return marshalAndSanitizeJSON(walletAddressResponse{
					Address: "abc",
				})
			},
		},
		{
			name:   "Get address unknown wallet",
			path:   "/v1/wallet/address/abc",
			method: http.MethodGet,
			setNodeMethods: func(n *mockNode) {
				n.multiwalletFunc = func() multiwallet.Multiwallet {
					mw := multiwallet.Multiwallet{
						"MCK": wallet.NewMockWallet(),
					}
					return mw
				}
			},
			statusCode: http.StatusBadRequest,
			expectedResponse: func() ([]byte, error) {
				return []byte("multiwallet does not contain an implementation for the given coin\n"), nil
			},
		},
	})
}
