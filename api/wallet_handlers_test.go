package api

import (
	"fmt"
	"github.com/cpacia/multiwallet"
	"github.com/cpacia/openbazaar3.0/events"
	"github.com/cpacia/openbazaar3.0/models"
	"github.com/cpacia/openbazaar3.0/wallet"
	iwallet "github.com/cpacia/wallet-interface"
	"net/http"
	"testing"
	"time"
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
				return []byte(fmt.Sprintf("%s\n", `{"error": "multiwallet does not contain an implementation for the given coin"}`)), nil
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
				ret := map[string]string{
					"MCK": "abc",
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
				return []byte(fmt.Sprintf("%s\n", `{"error": "multiwallet does not contain an implementation for the given coin"}`)), nil
			},
		},
		{
			name:   "Get transactions",
			path:   "/v1/wallet/transactions/mck",
			method: http.MethodGet,
			setNodeMethods: func(n *mockNode) {
				n.multiwalletFunc = func() multiwallet.Multiwallet {
					w := wallet.NewMockWallet()
					w.Start()
					bus := events.NewBus()
					w.SetEventBus(bus)
					sub, _ := bus.Subscribe(&events.TransactionReceived{})
					txn := w.GenerateTransaction(iwallet.NewAmount(100000))
					txn.Timestamp = time.Unix(111111, 0)
					txn.ID = "12345678"
					w.IngestTransaction(txn)
					<-sub.Out()

					mw := multiwallet.Multiwallet{
						"MCK": w,
					}
					return mw
				}
				n.getTransactionMetadataFunc = func(id iwallet.TransactionID) (models.TransactionMetadata, error) {
					return models.TransactionMetadata{
						PaymentAddress: "1234",
						Thumbnail:      "xyz",
						OrderID:        "abc",
						Memo:           "Meeemmmoooooooo",
					}, nil
				}
			},
			statusCode: http.StatusOK,
			expectedResponse: func() ([]byte, error) {
				return marshalAndSanitizeJSON([]walletTransactionResponse{
					{
						Txid:          "12345678",
						Timestamp:     time.Unix(111111, 0),
						Value:         "100000",
						Address:       "1234",
						OrderID:       "abc",
						Memo:          "Meeemmmoooooooo",
						Thumbnail:     "xyz",
						Status:        "UNCONFIRMED",
						Height:        0,
						Confirmations: 0,
					},
				})
			},
		},
		{
			name:   "Get transactions with limit",
			path:   "/v1/wallet/transactions/mck?limit=1",
			method: http.MethodGet,
			setNodeMethods: func(n *mockNode) {
				n.multiwalletFunc = func() multiwallet.Multiwallet {
					w := wallet.NewMockWallet()
					w.Start()
					bus := events.NewBus()
					w.SetEventBus(bus)
					sub, _ := bus.Subscribe(&events.TransactionReceived{})
					txn := w.GenerateTransaction(iwallet.NewAmount(100000))
					txn.Timestamp = time.Unix(111111, 0)
					txn.ID = "12345678"
					w.IngestTransaction(txn)
					<-sub.Out()

					mw := multiwallet.Multiwallet{
						"MCK": w,
					}
					return mw
				}
				n.getTransactionMetadataFunc = func(id iwallet.TransactionID) (models.TransactionMetadata, error) {
					return models.TransactionMetadata{
						PaymentAddress: "1234",
						Thumbnail:      "xyz",
						OrderID:        "abc",
						Memo:           "Meeemmmoooooooo",
					}, nil
				}
			},
			statusCode: http.StatusOK,
			expectedResponse: func() ([]byte, error) {
				return marshalAndSanitizeJSON([]walletTransactionResponse{
					{
						Txid:          "12345678",
						Timestamp:     time.Unix(111111, 0),
						Value:         "100000",
						Address:       "1234",
						OrderID:       "abc",
						Memo:          "Meeemmmoooooooo",
						Thumbnail:     "xyz",
						Status:        "UNCONFIRMED",
						Height:        0,
						Confirmations: 0,
					},
				})
			},
		},
		{
			name:   "Get transactions unknown coin",
			path:   "/v1/wallet/transactions/abc",
			method: http.MethodGet,
			setNodeMethods: func(n *mockNode) {
				n.multiwalletFunc = func() multiwallet.Multiwallet {
					w := wallet.NewMockWallet()
					mw := multiwallet.Multiwallet{
						"MCK": w,
					}
					return mw
				}
			},
			statusCode: http.StatusBadRequest,
			expectedResponse: func() ([]byte, error) {
				return []byte(fmt.Sprintf("%s\n", `{"error": "multiwallet does not contain an implementation for the given coin"}`)), nil
			},
		},
		{
			name:   "Get transactions bad limit",
			path:   "/v1/wallet/transactions/mck?limit=a",
			method: http.MethodGet,
			setNodeMethods: func(n *mockNode) {
				n.multiwalletFunc = func() multiwallet.Multiwallet {
					w := wallet.NewMockWallet()
					mw := multiwallet.Multiwallet{
						"MCK": w,
					}
					return mw
				}
			},
			statusCode: http.StatusBadRequest,
			expectedResponse: func() ([]byte, error) {
				return []byte(fmt.Sprintf("%s\n", `{"error": "strconv.Atoi: parsing "a": invalid syntax"}`)), nil
			},
		},
		{
			name:           "Get wallet currency definitions",
			path:           "/v1/wallet/currencies",
			method:         http.MethodGet,
			setNodeMethods: func(n *mockNode) {},
			statusCode:     http.StatusOK,
			expectedResponse: func() ([]byte, error) {
				return marshalAndSanitizeJSON(models.CurrencyDefinitions)
			},
		},
		{
			name:   "Post spend",
			path:   "/v1/wallet/spend",
			method: http.MethodPost,
			setNodeMethods: func(n *mockNode) {
				n.multiwalletFunc = func() multiwallet.Multiwallet {
					w := wallet.NewMockWallet()
					w.Start()
					bus := events.NewBus()
					w.SetEventBus(bus)
					sub, _ := bus.Subscribe(&events.TransactionReceived{})
					txn := w.GenerateTransaction(iwallet.NewAmount(100000))
					txn.Timestamp = time.Unix(111111, 0)
					txn.ID = "12345678"
					w.IngestTransaction(txn)
					<-sub.Out()

					mw := multiwallet.Multiwallet{
						"MCK": w,
					}
					return mw
				}
				n.saveTransactionMetadataFunc = func(md *models.TransactionMetadata) error {
					return nil
				}
			},
			body:       []byte(`{"coinType":"MCK","address":"de92e54c8a52742be470bdf21f00420828888542","amount":"10000","feeLevel":"economic","memo":"abcd"}`),
			statusCode: http.StatusOK,
			expectedResponse: func() ([]byte, error) {
				return nil, nil
			},
		},
		{
			name:   "Post spend custom fee",
			path:   "/v1/wallet/spend",
			method: http.MethodPost,
			setNodeMethods: func(n *mockNode) {
				n.multiwalletFunc = func() multiwallet.Multiwallet {
					w := wallet.NewMockWallet()
					w.Start()
					bus := events.NewBus()
					w.SetEventBus(bus)
					sub, _ := bus.Subscribe(&events.TransactionReceived{})
					txn := w.GenerateTransaction(iwallet.NewAmount(100000))
					txn.Timestamp = time.Unix(111111, 0)
					txn.ID = "12345678"
					w.IngestTransaction(txn)
					<-sub.Out()

					mw := multiwallet.Multiwallet{
						"MCK": w,
					}
					return mw
				}
				n.saveTransactionMetadataFunc = func(md *models.TransactionMetadata) error {
					return nil
				}
			},
			body:       []byte(`{"coinType":"MCK","address":"de92e54c8a52742be470bdf21f00420828888542","amount":"10000","feeLevel":"10","memo":"abcd"}`),
			statusCode: http.StatusOK,
			expectedResponse: func() ([]byte, error) {
				return nil, nil
			},
		},
		{
			name:   "Post spend unknown coin",
			path:   "/v1/wallet/spend",
			method: http.MethodPost,
			setNodeMethods: func(n *mockNode) {
				n.multiwalletFunc = func() multiwallet.Multiwallet {
					w := wallet.NewMockWallet()
					w.Start()
					bus := events.NewBus()
					w.SetEventBus(bus)
					sub, _ := bus.Subscribe(&events.TransactionReceived{})
					txn := w.GenerateTransaction(iwallet.NewAmount(100000))
					txn.Timestamp = time.Unix(111111, 0)
					txn.ID = "12345678"
					w.IngestTransaction(txn)
					<-sub.Out()

					mw := multiwallet.Multiwallet{
						"MCK": w,
					}
					return mw
				}
				n.saveTransactionMetadataFunc = func(md *models.TransactionMetadata) error {
					return nil
				}
			},
			body:       []byte(`{"coinType":"BTC","address":"de92e54c8a52742be470bdf21f00420828888542","amount":"10000","feeLevel":"10","memo":"abcd"}`),
			statusCode: http.StatusBadRequest,
			expectedResponse: func() ([]byte, error) {
				return []byte(fmt.Sprintf("%s\n", `{"error": "multiwallet does not contain an implementation for the given coin"}`)), nil
			},
		},
		{
			name:   "Post spend invalid JSON",
			path:   "/v1/wallet/spend",
			method: http.MethodPost,
			setNodeMethods: func(n *mockNode) {
				n.multiwalletFunc = func() multiwallet.Multiwallet {
					w := wallet.NewMockWallet()
					w.Start()
					bus := events.NewBus()
					w.SetEventBus(bus)
					sub, _ := bus.Subscribe(&events.TransactionReceived{})
					txn := w.GenerateTransaction(iwallet.NewAmount(100000))
					txn.Timestamp = time.Unix(111111, 0)
					txn.ID = "12345678"
					w.IngestTransaction(txn)
					<-sub.Out()

					mw := multiwallet.Multiwallet{
						"MCK": w,
					}
					return mw
				}
				n.saveTransactionMetadataFunc = func(md *models.TransactionMetadata) error {
					return nil
				}
			},
			body:       []byte(`"coinType":"BTC","address":"de92e54c8a52742be470bdf21f00420828888542","amount":"10000","feeLevel":"10","memo":"abcd"}`),
			statusCode: http.StatusBadRequest,
			expectedResponse: func() ([]byte, error) {
				return []byte(fmt.Sprintf("%s\n", `{"error": "json: cannot unmarshal string into Go value of type api.Spend"}`)), nil
			},
		},
		{
			name:   "Post spend zero amount",
			path:   "/v1/wallet/spend",
			method: http.MethodPost,
			setNodeMethods: func(n *mockNode) {
				n.multiwalletFunc = func() multiwallet.Multiwallet {
					w := wallet.NewMockWallet()
					w.Start()
					bus := events.NewBus()
					w.SetEventBus(bus)
					sub, _ := bus.Subscribe(&events.TransactionReceived{})
					txn := w.GenerateTransaction(iwallet.NewAmount(100000))
					txn.Timestamp = time.Unix(111111, 0)
					txn.ID = "12345678"
					w.IngestTransaction(txn)
					<-sub.Out()

					mw := multiwallet.Multiwallet{
						"MCK": w,
					}
					return mw
				}
				n.saveTransactionMetadataFunc = func(md *models.TransactionMetadata) error {
					return nil
				}
			},
			body:       []byte(`{"coinType":"MCK","address":"de92e54c8a52742be470bdf21f00420828888542","amount":"0","feeLevel":"10","memo":"abcd"}`),
			statusCode: http.StatusBadRequest,
			expectedResponse: func() ([]byte, error) {
				return []byte(fmt.Sprintf("%s\n", `{"error": "cannot send zero amount"}`)), nil
			},
		},
		{
			name:   "Post spend invalid custom fee",
			path:   "/v1/wallet/spend",
			method: http.MethodPost,
			setNodeMethods: func(n *mockNode) {
				n.multiwalletFunc = func() multiwallet.Multiwallet {
					w := wallet.NewMockWallet()
					w.Start()
					bus := events.NewBus()
					w.SetEventBus(bus)
					sub, _ := bus.Subscribe(&events.TransactionReceived{})
					txn := w.GenerateTransaction(iwallet.NewAmount(100000))
					txn.Timestamp = time.Unix(111111, 0)
					txn.ID = "12345678"
					w.IngestTransaction(txn)
					<-sub.Out()

					mw := multiwallet.Multiwallet{
						"MCK": w,
					}
					return mw
				}
				n.saveTransactionMetadataFunc = func(md *models.TransactionMetadata) error {
					return nil
				}
			},
			body:       []byte(`{"coinType":"MCK","address":"de92e54c8a52742be470bdf21f00420828888542","amount":"10000","feeLevel":"asdfadsf","memo":"abcd"}`),
			statusCode: http.StatusBadRequest,
			expectedResponse: func() ([]byte, error) {
				return []byte(fmt.Sprintf("%s\n", `{"error": "invalid custom fee"}`)), nil
			},
		},
	})
}
