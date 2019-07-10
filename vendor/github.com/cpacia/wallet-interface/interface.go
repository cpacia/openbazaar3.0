package wallet_interface

import (
	"github.com/btcsuite/btcd/btcec"
	hd "github.com/btcsuite/btcutil/hdkeychain"
	"time"
)

// Tx represents a database transaction used for atomic updates. It is expected that
// wallets implementing the full interface will respect the transaction and only
// commit the particular change on Commit() and will roll back the database change
// on Rollback(). In the case of a cryptocurrency transaction this would imply that
// the transaction not be broadcasted until Commit() and Rollback() will prevent
// broadcast and restore the prior wallet state.
type Tx interface {
	// Commit commits all changes that have been made to wallet state.
	// Depending on the backend implementation this could be to a cache that
	// is periodically synced to persistent storage or directly to persistent
	// storage.  In any case, all transactions which are started after the commit
	// finishes will include all changes made by this transaction.
	Commit() error

	// Rollback undoes all changes that have been made to the wallet state.
	Rollback() error
}

type WalletLoader interface {
	// WalletExists should return whether the wallet exits or has been
	// initialized.
	WalletExists() bool

	// CreateWallet should initialize the wallet. This will be called by
	// OpenBazaar if WalletExists() returns false.
	//
	// The xPriv may be used to create a bip44 keychain. The xPriv is
	// `cointype` level in the bip44 path. For example in the following
	// path the wallet should only derive the paths after `cointype` as
	// m and purpose' are kept private by OpenBazaar so this wallet cannot
	// derive keys from other wallets.
	//
	// m / purpose' / coin_type' / account' / change / address_index
	//
	// The birthday can be used determine where to sync state from if
	// appropriate.
	//
	// If the wallet does not implement WalletCrypter then pw will be
	// nil. Otherwise it should be used to encrypt the private keys.
	CreateWallet(xpriv hd.ExtendedKey, pw []byte, birthday time.Time) error

	// Open wallet will be called each time on OpenBazaar start. It
	// will also be called after CreateWallet().
	OpenWallet() error

	// CloseWallet will be called when OpenBazaar shuts down.
	CloseWallet() error
}

type Wallet interface {
	// WalletLoader must be implemented by this interface.
	WalletLoader

	// Begin returns a new database transaction. A transaction must only be used
	// once. After Commit() or Rollback() is called the transaction can be discarded.
	Begin() (Tx, error)

	// BlockchainInfo returns the best hash and height of the chain.
	BlockchainInfo() (BlockchainInfo, error)

	// CurrentAddress is called when requesting this wallet's receiving
	// address. It is customary that the wallet return the first unused
	// address and only return a different address after funds have been
	// received on the address. This, however, is just a wallet implementation
	// detail.
	CurrentAddress() (Address, error)

	// NewAddress should return a new, never before used address. This is called
	// by OpenBazaar to get a fresh address for a direct payment order. It
	// associates this address with the order and assumes if a payment is received
	// by this address that it is for the order. Failure to return a never before
	// used address could put the order in a bad state.
	NewAddress() (Address, error)

	// Balance should return the confirmed and unconfirmed balance for the wallet.
	Balance() (unconfirmed Amount, confirmed Amount, err error)

	// IsDust returns whether the amount passed in is considered dust by network. This
	// method is called when building payout transactions from the multisig to the various
	// participants. If the amount that is supposed to be sent to a given party is below
	// the dust threshold, openbazaar-go will not pay that party to avoid building a transaction
	// that never confirms.
	IsDust(amount Amount) bool

	// Transactions returns a slice of this wallet's transactions.
	Transactions() ([]Transaction, error)

	// GetTransaction returns a transaction given it's ID.
	GetTransaction(id TransactionID) (Transaction, error)

	// EstimateSpendFee should return the anticipated fee to transfer a given amount of coins
	// out of the wallet at the provided fee level. Typically this involves building a
	// transaction with enough inputs to cover the request amount and calculating the size
	// of the transaction. It is OK, if a transaction comes in after this function is called
	// that changes the estimated fee as it's only intended to be an estimate.
	//
	// All amounts should be in the coin's base unit (for example: satoshis).
	EstimateSpendFee(amount Amount, feeLevel FeeLevel) (Amount, error)

	// Spend is a request to send requested amount to the requested address. The
	// fee level is provided by the user. It's up to the implementation to decide
	// how best to use the fee level.
	//
	// The database Tx MUST be respected. When this function is called the wallet
	// state changes should be prepped and held in memory. If Rollback() is called
	// the state changes should be discarded. Only when Commit() is called should
	// the state changes be applied and the transaction broadcasted to the network.
	Spend(dbtx Tx, to Address, amt Amount, feeLevel FeeLevel) (TransactionID, error)

	// SweepWallet should sweep the full balance of the wallet to the requested
	// address. It is expected for most coins that the fee will be subtracted
	// from the amount sent rather than added to it.
	SweepWallet(dbtx Tx, to Address, level FeeLevel) (TransactionID, error)

	// SubscribeTransactions returns a chan over which the wallet is expected
	// to push both transactions relevant for this wallet as well as transactions
	// sending to or spending from a watched address.
	SubscribeTransactions() chan<- Transaction

	// SubscribeBlocks returns a chan over which the wallet is expected
	// to push info about new blocks when they arrive.
	SubscribeBlocks() chan<- BlockchainInfo
}

// Escrow is functions related to the OpenBazaar escrow system. This interface should
// be implemented but it's technically optional as some coins like Monero have a
// hard time implementing escrow. If it's not implemented then this coin will not
// be selectable for either escrow payments or offline payments.
type Escrow interface {
	// WatchAddress is used by the escrow system to tell the wallet to listen
	// on the escrow address. It's expected that payments into and spends from
	// this address will be pushed back to OpenBazaar.
	//
	// Note a database transaction is used here. Same rules of Commit() and
	// Rollback() apply.
	WatchAddress(dbtx Tx, addr Address) error

	// EstimateEscrowFee estimates the fee to release the funds from escrow.
	// this assumes only one input. If there are more inputs OpenBazaar will
	// will add 50% of the returned fee for each additional input. This is a
	// crude fee calculating but it simplifies things quite a bit.
	EstimateEscrowFee(threshold int, level FeeLevel) (Amount, error)

	// CreateMultisigAddress creates a new threshold multisig address using the
	// provided pubkeys and the threshold. The multisig address is returned along
	// with a byte slice. The byte slice will typically be the redeem script for
	// the address (in Bitcoin related coins). The slice will be saved in OpenBazaar
	// with the order and passed back into the wallet when signing the transaction.
	// In practice this does not need to be a redeem script so long as the wallet
	// knows how to sign the transaction when it sees it.
	//
	// This function should be deterministic as both buyer and vendor will be passing
	// in the same set of keys and expecting to get back the same address and redeem
	// script. If this is not the case the vendor will reject the order.
	//
	// Note that this is normally a 2 of 3 escrow in the normal case, however OpenBazaar
	// also uses 1 of 2 multisigs as a form of a "cancelable" address when sending to
	// a node that is offline. This allows the sender to cancel the payment if the vendor
	// never comes back online.
	CreateMultisigAddress(keys []btcec.PublicKey, threshold int) (Address, []byte, error)

	// SignMultisigTransaction should use the provided key to create a signature for
	// the multisig transaction. Since this a threshold signature this function will
	// separately by each party signing this transaction. The resulting signatures
	// will be shared between the relevant parties and one of them will aggregate
	// the signatures into a transaction for broadcast.
	//
	// For coins like bitcoin you may need to return one signature *per input* which is
	// why a slice of signatures is returned.
	SignMultisigTransaction(txn Transaction, key *btcec.PrivateKey, redeemScript []byte) ([]EscrowSignature, error)

	// BuildAndSend should used the passed in signatures to build the transaction.
	// Note the signatures are a slice of slices. This is because coins like Bitcoin
	// may require one signature *per input*. In this case the outer slice is the
	// signatures from the different key holders and the inner slice is the keys
	// per input.
	//
	// Note a database transaction is used here. Same rules of Commit() and
	// Rollback() apply.
	BuildAndSend(dbtx Tx, txn Transaction, signatures [][]EscrowSignature, redeemScript []byte) error
}

// EscrowWithTimeout is an optional interface to be implemented by wallets whos coins
// are capable of supporting time based release of funds from escrow.
type EscrowWithTimeout interface {

	// CreateMultisigWithTimeout is the same as CreateMultisigAddress but it adds
	// an additional timeout to the address. The address should have two ways to
	// release the funds:
	//  - m of n signatures are provided (or)
	//  - timeout has passed and a signature for timeoutKey is provided.
	CreateMultisigWithTimeout(keys []btcec.PublicKey, threshold int, timeout time.Duration, timeoutKey btcec.PublicKey) (Address, []byte, error)
}

// WalletCrypter is an optional interface that the wallet may implement to allow
// for encrypting private keys. If this is implemented OpenBazaar will call these
// functions as specified below.
type WalletCrypter interface {
	// SetPassphase is called after creating the wallet. It gives the wallet
	// the opportunity to set up encryption of the private keys.
	SetPassphase(pw []byte) error

	// ChangePassphrase is called in response to user action requesting the
	// passphrase be changed. It is expected that this will return an error
	// if the old password is incorrect.
	ChangePassphrase(old, new []byte) error

	// RemovePassphrase is called in response to user action requesting the
	// passphrase be removed. It is expected that this will return an error
	// if the old password is incorrect.
	RemovePassphrase(pw []byte) error

	// Unlock is called just prior to calling Spend(). The wallet should
	// decrypt the private key and hold the decrypted key in memory for
	// the provided duration after which it should be purged from memory.
	// If the provided password is incorrect it should error.
	Unlock(pw []byte, howLong time.Duration) error
}
