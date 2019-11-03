package events

import iwallet "github.com/cpacia/wallet-interface"

// TransactionReceived is an event that fires whenever a transaction
// relevant to a wallet is received.
type TransactionReceived struct {
	iwallet.Transaction
}

// BlockReceived is an event that fires when a new block is
// received by a wallet.
type BlockReceived struct {
	iwallet.BlockInfo
	CurrencyCode string
}
