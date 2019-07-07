package wallet_interface

// TransactionID represents an ID for a transaction made by the wallet
type TransactionID string

// String returns the string representation of the ID.
func (t *TransactionID) String() string {
	return string(*t)
}

// Transaction is a basic record which is used to convey information
// about a transaction to OpenBazaar. It's designed to be generic
// enough to be used by a variety of different coins.
//
// In the case of multisig transactions OpenBazaar will be using the
// To spend info objects in the from field when spending from a
// multisig.
type Transaction struct {
	ID TransactionID

	From []SpendInfo
	To   []SpendInfo

	Height uint64
}

// SpendInfo represents a transaction data element. This could either
// be an input or an outpoint in the Bitcoin context. The ID can
// be used by the wallet to attach metadata needed construct a
// transaction from this data structure. Again in the bitcoin context
// this would be a serialized outpoint (when this represents an input).
type SpendInfo struct {
	ID []byte

	Address   Address
	Amount Amount

	IsRelevant bool
	IsWatched  bool
}

// EscrowSignature represents a signature for an escrow transaction.
type EscrowSignature struct {
	Index     int
	Signature []byte
}
