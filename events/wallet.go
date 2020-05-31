package events

import iwallet "github.com/cpacia/wallet-interface"

// TransactionReceived is an event that fires whenever a transaction
// relevant to a wallet is received.
type TransactionReceived struct {
	iwallet.Transaction
	CurrencyCode string
}

// SpendFromPaymentAddress is an event that fires whenever funds leave
// the payment address.
type SpendFromPaymentAddress struct {
	iwallet.Transaction
	CurrencyCode string
}

// BlockReceived is an event that fires when a new block is
// received by a wallet.
type BlockReceived struct {
	iwallet.BlockInfo
	CurrencyCode string
}

type WalletInfo struct {
	ConfirmedBalance   iwallet.Amount `json:"confirmed"`
	UnconfirmedBalance iwallet.Amount `json:"unconfirmed"`
	ChainHeight        uint64         `json:"height"`
}

type WalletUpdate map[string]WalletInfo
