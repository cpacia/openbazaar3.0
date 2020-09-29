package wallet

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"github.com/btcsuite/btcd/btcec"
	hd "github.com/btcsuite/btcutil/hdkeychain"
	"github.com/cpacia/openbazaar3.0/events"
	iwallet "github.com/cpacia/wallet-interface"
	"github.com/jarcoal/httpmock"
	"net/http"
	"sort"
	"sync"
	"time"
)

// MockWalletNetwork is a network of mock wallets connected
// together through channels. One mock wallet can send a
// transaction to another one.
type MockWalletNetwork struct {
	wallets []*MockWallet

	outgoing chan iwallet.Transaction
	shutdown chan struct{}

	height uint64

	mtx sync.Mutex
}

// NewMockWalletNetwork creates a network of numWallets mock wallets
// and connects them all together.
func NewMockWalletNetwork(numWallets int) *MockWalletNetwork {
	var wallets []*MockWallet
	outgoing := make(chan iwallet.Transaction)
	for i := 0; i < numWallets; i++ {
		w := NewMockWallet()
		w.outgoing = outgoing
		wallets = append(wallets, w)
	}

	return &MockWalletNetwork{
		wallets:  wallets,
		outgoing: outgoing,
		shutdown: make(chan struct{}),
	}
}

// Start will start the wallet network. This must be called
// before sending any transactions between wallets.
func (n *MockWalletNetwork) Start() {
	for _, w := range n.wallets {
		w.Start()
	}
	go func() {
		for {
			select {
			case cb := <-n.outgoing:
				for _, w := range n.wallets {
					w.incoming <- cb
				}
			case <-n.shutdown:
				return
			}
		}
	}()
}

// Wallets returns a slice of wallets in this network.
func (n *MockWalletNetwork) Wallets() []*MockWallet {
	return n.wallets
}

// GenerateBlock will create a fake block and send it to the
// wallets. All wallets will increment the confirmations on
// their transactions if applicable.
func (n *MockWalletNetwork) GenerateBlock() {
	n.mtx.Lock()
	defer n.mtx.Unlock()

	h := make([]byte, 32)
	rand.Read(h)

	n.height++

	for _, wallet := range n.wallets {
		wallet.block <- iwallet.BlockInfo{
			BlockID:   iwallet.BlockID(hex.EncodeToString(h)),
			Height:    n.height,
			BlockTime: time.Now(),
		}
	}
}

// GenerateToAddress creates new coins out of thin air and sends them to the
// requested address.
func (n *MockWalletNetwork) GenerateToAddress(addr iwallet.Address, amount iwallet.Amount) error {
	n.mtx.Lock()
	defer n.mtx.Unlock()

	txidBytes := make([]byte, 32)
	rand.Read(txidBytes)

	prevHashBytes := make([]byte, 36)
	rand.Read(prevHashBytes)

	prevAddrBytes := make([]byte, 20)
	rand.Read(prevAddrBytes)

	txn := iwallet.Transaction{
		ID: iwallet.TransactionID(hex.EncodeToString(txidBytes)),
		From: []iwallet.SpendInfo{
			{
				ID:      prevHashBytes,
				Amount:  amount,
				Address: iwallet.NewAddress(hex.EncodeToString(prevAddrBytes), iwallet.CtMock),
			},
		},
		To: []iwallet.SpendInfo{
			{
				Address: addr,
				Amount:  amount,
				ID:      append(txidBytes, []byte{0x00, 0x00, 0x00, 0x00}...),
			},
		},
	}

	for _, w := range n.wallets {
		w.incoming <- txn
	}
	return nil
}

var _ = iwallet.Wallet(&MockWallet{})
var _ = iwallet.Escrow(&MockWallet{})
var _ = iwallet.EscrowWithTimeout(&MockWallet{})

// MockWallet is a mock wallet that conforms to the wallet interface. It can
// be hooked up to the MockWalletNetwork to allow transactions between mock
// wallets.
type MockWallet struct {
	mtx sync.RWMutex

	addrs        map[iwallet.Address]bool
	watchedAddrs map[iwallet.Address]struct{}
	transactions map[iwallet.TransactionID]iwallet.Transaction

	utxos map[string]mockUtxo

	blockchainInfo iwallet.BlockInfo

	outgoing chan iwallet.Transaction
	incoming chan iwallet.Transaction
	block    chan iwallet.BlockInfo

	txSubs    []chan iwallet.Transaction
	blockSubs []chan iwallet.BlockInfo

	addrResponse *iwallet.Address

	bus events.Bus

	done chan struct{}
}

// NewMockWallet creates and returns a new mock wallet.
func NewMockWallet() *MockWallet {
	mw := &MockWallet{
		addrs:        make(map[iwallet.Address]bool),
		watchedAddrs: make(map[iwallet.Address]struct{}),
		transactions: make(map[iwallet.TransactionID]iwallet.Transaction),
		utxos:        make(map[string]mockUtxo),
		incoming:     make(chan iwallet.Transaction),
		block:        make(chan iwallet.BlockInfo),
		done:         make(chan struct{}),
	}

	for i := 0; i < 10; i++ {
		b := make([]byte, 20)
		rand.Read(b)
		addr := iwallet.NewAddress(hex.EncodeToString(b), iwallet.CtMock)
		mw.addrs[addr] = false
	}

	return mw
}

// mockUtxo is used for internal accounting.
type mockUtxo struct {
	outpoint []byte
	address  iwallet.Address
	value    iwallet.Amount
	height   uint64
}

// dbTx satisfies the iwallet.Tx interface.
type dbTx struct {
	isClosed bool

	onCommit func() error
}

// Commit will commit the transaction.
func (tx *dbTx) Commit() error {
	if tx.isClosed {
		panic("tx is closed")
	}
	if tx.onCommit != nil {
		if err := tx.onCommit(); err != nil {
			tx.Rollback()
			return err
		}
	}
	tx.isClosed = true
	return nil
}

// Rollback will rollback the transaction.
func (tx *dbTx) Rollback() error {
	if tx.isClosed {
		panic("tx is closed")
	}
	tx.onCommit = nil
	tx.isClosed = true
	return nil
}

// SetEventBus sets an event bus in the mock wallet. This is useful
// for testing integration with the OpenBazaarNode.
func (w *MockWallet) SetEventBus(bus events.Bus) {
	w.bus = bus
}

// SetAddressResponse can be used to set the response for CurrentAddress
// and NewAddress for testing purposes.
func (w *MockWallet) SetAddressResponse(addr iwallet.Address) {
	w.addrResponse = &addr
}

// GenerateTransaction generates a mock transaction paying this wallet
// and returns it.
func (w *MockWallet) GenerateTransaction(amount iwallet.Amount) iwallet.Transaction {
	w.mtx.Lock()
	defer w.mtx.Unlock()

	txidBytes := make([]byte, 32)
	rand.Read(txidBytes)

	prevHashBytes := make([]byte, 36)
	rand.Read(prevHashBytes)

	prevAddrBytes := make([]byte, 20)
	rand.Read(prevAddrBytes)

	b := make([]byte, 20)
	rand.Read(b)
	addr := iwallet.NewAddress(hex.EncodeToString(b), iwallet.CtMock)
	w.addrs[addr] = false

	txn := iwallet.Transaction{
		ID:    iwallet.TransactionID(hex.EncodeToString(txidBytes)),
		Value: amount,
		From: []iwallet.SpendInfo{
			{
				ID:      prevHashBytes,
				Amount:  amount,
				Address: iwallet.NewAddress(hex.EncodeToString(prevAddrBytes), iwallet.CtMock),
			},
		},
		To: []iwallet.SpendInfo{
			{
				Address: addr,
				Amount:  amount,
				ID:      append(txidBytes, []byte{0x00, 0x00, 0x00, 0x00}...),
			},
		},
	}
	return txn
}

// IngestTransaction ingests a transaction into the MockWallet.
func (w *MockWallet) IngestTransaction(txn iwallet.Transaction) {
	w.incoming <- txn
}

// Start is called when the openbazaar-go daemon starts up. At this point in time
// the wallet implementation should start syncing and/or updating balances, but
// not before.
func (w *MockWallet) Start() {
	go func() {
		for {
			select {
			case tx := <-w.incoming:
				w.mtx.Lock()
				txidBytes, err := hex.DecodeString(string(tx.ID))
				if err != nil {
					w.mtx.Unlock()
					return
				}
				var (
					relevant bool
					watched  bool
				)
				total := iwallet.NewAmount(0)
				for i, out := range tx.To {
					if _, ok := w.addrs[out.Address]; ok {
						idx := make([]byte, 4)
						binary.BigEndian.PutUint32(idx, uint32(i))
						op := append(txidBytes, idx...)
						outpoint := hex.EncodeToString(op)
						if _, ok := w.utxos[outpoint]; !ok {
							w.utxos[outpoint] = mockUtxo{
								outpoint: op,
								address:  out.Address,
								value:    out.Amount,
							}
						}
						total = total.Add(out.Amount)
						w.addrs[out.Address] = true
						relevant = true
					}
					if _, ok := w.watchedAddrs[out.Address]; ok {
						watched = true
					}
				}
				for _, in := range tx.From {
					if _, ok := w.addrs[in.Address]; ok {
						delete(w.utxos, hex.EncodeToString(in.ID))
						total = total.Sub(in.Amount)
						relevant = true
					}
					if _, ok := w.watchedAddrs[in.Address]; ok {
						watched = true
					}
				}
				if relevant || watched {
					tx.Value = total
					defaultTime := time.Time{}
					if tx.Timestamp == defaultTime {
						tx.Timestamp = time.Now()
					}
					w.transactions[tx.ID] = tx
					for _, sub := range w.txSubs {
						sub <- tx
					}
					if w.bus != nil {
						w.bus.Emit(&events.TransactionReceived{Transaction: tx})
					}
				}
				w.mtx.Unlock()
			case blockInfo := <-w.block:
				w.mtx.Lock()
				blockInfo.PrevBlock = w.blockchainInfo.BlockID
				w.blockchainInfo = blockInfo
				for txid, txn := range w.transactions {
					if txn.Height == 0 {
						txn.Height = blockInfo.Height
						txn.BlockInfo = &blockInfo
						w.transactions[txid] = txn
					}
				}
				for op, utxo := range w.utxos {
					if utxo.height == 0 {
						utxo.height = blockInfo.Height
						w.utxos[op] = utxo
					}
				}
				if w.bus != nil {
					w.bus.Emit(&events.BlockReceived{CurrencyCode: "MCK", BlockInfo: blockInfo})
				}
				w.mtx.Unlock()
			case <-w.done:
				return
			}
		}
	}()
}

// WalletExists should return whether the wallet exits or has been
// initialized.
func (w *MockWallet) WalletExists() bool {
	return true
}

// CreateWallet should initialize the wallet. This will be called by
// OpenBazaar if WalletExists() returns false.
//
// The xPriv may be used to create a bip44 keychain. The xPriv is
// `cointype` level in the bip44 path. For example in the following
// path the wallet should only derive the paths after `account` as
// m, purpose', and coin_type' are kept private by OpenBazaar so this
// wallet cannot derive keys from other wallets.
//
// m / purpose' / coin_type' / account' / change / address_index
//
// The birthday can be used determine where to sync state from if
// appropriate.
//
// If the wallet does not implement WalletCrypter then pw will be
// nil. Otherwise it should be used to encrypt the private keys.
func (w *MockWallet) CreateWallet(xpriv hd.ExtendedKey, pw []byte, birthday time.Time) error {
	return nil
}

// Open wallet will be called each time on OpenBazaar start. It
// will also be called after CreateWallet().
func (w *MockWallet) OpenWallet() error {
	return nil
}

// CloseWallet will be called when OpenBazaar shuts down.
func (w *MockWallet) CloseWallet() error {
	close(w.done)
	return nil
}

// Begin returns a new database transaction. A transaction must only be used
// once. After Commit() or Rollback() is called the transaction can be discarded.
func (w *MockWallet) Begin() (iwallet.Tx, error) {
	return &dbTx{}, nil
}

// BlockchainInfo returns the best hash and height of the chain.
func (w *MockWallet) BlockchainInfo() (iwallet.BlockInfo, error) {
	w.mtx.RLock()
	defer w.mtx.RUnlock()

	return w.blockchainInfo, nil
}

// CurrentAddress is called when requesting this wallet's receiving
// address. It is customary that the wallet return the first unused
// address and only return a different address after funds have been
// received on the address. This, however, is just a wallet implementation
// detail.
func (w *MockWallet) CurrentAddress() (iwallet.Address, error) {
	w.mtx.Lock()
	defer w.mtx.Unlock()

	if w.addrResponse != nil {
		resp := *w.addrResponse
		w.addrResponse = nil
		return resp, nil
	}

	for addr, used := range w.addrs {
		if !used {
			return addr, nil
		}
	}
	b := make([]byte, 20)
	rand.Read(b)
	addr := iwallet.NewAddress(hex.EncodeToString(b), iwallet.CtMock)
	w.addrs[addr] = false
	return addr, nil
}

// NewAddress should return a new, never before used address. This is called
// by OpenBazaar to get a fresh address for a direct payment order. It
// associates this address with the order and assumes if a payment is received
// by this address that it is for the order. Failure to return a never before
// used address could put the order in a bad state.
//
// Wallets that only use a single address, like Ethereum, should save the
// passed in order ID locally such as to associate payments with orders.
func (w *MockWallet) NewAddress() (iwallet.Address, error) {
	w.mtx.Lock()
	defer w.mtx.Unlock()

	if w.addrResponse != nil {
		resp := *w.addrResponse
		w.addrResponse = nil
		return resp, nil
	}

	b := make([]byte, 20)
	rand.Read(b)
	addr := iwallet.NewAddress(hex.EncodeToString(b), iwallet.CtMock)
	w.addrs[addr] = false
	return addr, nil
}

// ValidateAddress validates that the serialization of the address is correct
// for this coin and network. It returns an error if it isn't.
func (w *MockWallet) ValidateAddress(addr iwallet.Address) error {
	if len(addr.String()) != 40 {
		return errors.New("invalid address length")
	}
	if addr.CoinType() != iwallet.CtMock {
		return errors.New("address's cointype is not CtMock")
	}
	return nil
}

// HasKey returns true if the wallet can spend from the given address.
func (w *MockWallet) HasKey(addr iwallet.Address) (bool, error) {
	_, ok := w.addrs[addr]
	return ok, nil
}

func (w *MockWallet) newAddress() (iwallet.Address, error) {
	b := make([]byte, 20)
	rand.Read(b)
	addr := iwallet.NewAddress(hex.EncodeToString(b), iwallet.CtMock)
	w.addrs[addr] = false
	return addr, nil
}

// Balance should return the confirmed and unconfirmed balance for the wallet.
func (w *MockWallet) Balance() (iwallet.Amount, iwallet.Amount, error) {
	w.mtx.RLock()
	defer w.mtx.RUnlock()

	// TODO: this is the lazy way of calculating this. It should probably
	// recursively check if unconfirmed utxos are spends of confirmed parents.
	confirmed, unconfirmed := iwallet.NewAmount(0), iwallet.NewAmount(0)
	for _, utxo := range w.utxos {
		if utxo.height > 0 {
			confirmed = confirmed.Add(utxo.value)
		} else {
			unconfirmed = unconfirmed.Add(utxo.value)
		}
	}
	return confirmed, unconfirmed, nil
}

// IsDust returns whether the amount passed in is considered dust by network. This
// method is called when building payout transactions from the multisig to the various
// participants. If the amount that is supposed to be sent to a given party is below
// the dust threshold, openbazaar-go will not pay that party to avoid building a transaction
// that never confirms.
func (w *MockWallet) IsDust(amount iwallet.Amount) bool {
	return amount.Cmp(iwallet.NewAmount(500)) < 0
}

type txSorter struct {
	transactions []iwallet.Transaction
}

// Len is part of sort.Interface.
func (s *txSorter) Len() int {
	return len(s.transactions)
}

// Swap is part of sort.Interface.
func (s *txSorter) Swap(i, j int) {
	s.transactions[i], s.transactions[j] = s.transactions[j], s.transactions[i]
}

// Less is part of sort.Interface. It is implemented by calling the "by" closure in the sorter.
func (s *txSorter) Less(i, j int) bool {
	return s.transactions[i].Timestamp.Before(s.transactions[j].Timestamp)
}

// Transactions returns a slice of this wallet's transactions. The transactions should
// be sorted last to first and the limit and offset respected. The offsetID means
// 'return transactions starting with the transaction after offsetID in the sorted list'
func (w *MockWallet) Transactions(limit int, offsetID iwallet.TransactionID) ([]iwallet.Transaction, error) {
	w.mtx.RLock()
	defer w.mtx.RUnlock()

	txs := make([]iwallet.Transaction, 0, len(w.transactions))
	for _, tx := range w.transactions {
		matches := false
		for _, from := range tx.From {
			if _, ok := w.addrs[from.Address]; ok {
				matches = true
			}
		}
		for _, to := range tx.To {
			if _, ok := w.addrs[to.Address]; ok {
				matches = true
			}
		}
		if matches {
			txs = append(txs, tx)
		}
	}
	sorted := &txSorter{transactions: txs}

	sort.Sort(sorted)

	if limit < 0 {
		limit = len(sorted.transactions)
	}
	offset := 0
	if offsetID != "" {
		for i, tx := range sorted.transactions {
			if tx.ID == offsetID {
				offset = i + 1
				break
			}
		}
	}

	return sorted.transactions[offset:limit], nil
}

// GetTransaction returns a transaction given it's ID. This is used by OpenBazaar to
// request transactions paid to an order's payment address. This means we expect both
// internal wallet transactions and transactions sending to or from a watched address
// to be returned here.
func (w *MockWallet) GetTransaction(id iwallet.TransactionID) (iwallet.Transaction, error) {
	w.mtx.RLock()
	defer w.mtx.RUnlock()

	tx, ok := w.transactions[id]
	if !ok {
		return tx, errors.New("not found")
	}
	return tx, nil
}

// GetAddressTransactions returns the transactions sending to or spending from this address.
// Note this will only ever be called for an order's payment address transaction so for the
// purpose of this method the wallet only needs to be able to track transactions paid to a
// wallet address and any watched addresses.
func (w *MockWallet) GetAddressTransactions(addr iwallet.Address) ([]iwallet.Transaction, error) {
	w.mtx.RLock()
	defer w.mtx.RUnlock()

	var txs []iwallet.Transaction
txloop:
	for _, tx := range w.transactions {
		for _, in := range tx.From {
			if in.Address.String() == addr.String() {
				txs = append(txs, tx)
				continue txloop
			}
		}

		for _, out := range tx.To {
			if out.Address.String() == addr.String() {
				txs = append(txs, tx)
				continue txloop
			}
		}
	}
	return txs, nil
}

// EstimateSpendFee should return the anticipated fee to transfer a given amount of coins
// out of the wallet at the provided fee level. Typically this involves building a
// transaction with enough inputs to cover the request amount and calculating the size
// of the transaction. It is OK, if a transaction comes in after this function is called
// that changes the estimated fee as it's only intended to be an estimate.
//
// All amounts should be in the coin's base unit (for example: satoshis).
func (w *MockWallet) EstimateSpendFee(amount iwallet.Amount, feeLevel iwallet.FeeLevel) (iwallet.Amount, error) {
	var fee iwallet.Amount
	switch feeLevel {
	case iwallet.FlEconomic:
		fee = iwallet.NewAmount(250)
	case iwallet.FlNormal:
		fee = iwallet.NewAmount(500)
	case iwallet.FlPriority:
		fee = iwallet.NewAmount(750)
	}
	return fee, nil
}

// Spend is a request to send requested amount to the requested address. The
// fee level is provided by the user. It's up to the implementation to decide
// how best to use the fee level.
//
// The database Tx MUST be respected. When this function is called the wallet
// state changes should be prepped and held in memory. If Rollback() is called
// the state changes should be discarded. Only when Commit() is called should
// the state changes be applied and the transaction broadcasted to the network.
func (w *MockWallet) Spend(tx iwallet.Tx, to iwallet.Address, amt iwallet.Amount, feeLevel iwallet.FeeLevel) (iwallet.TransactionID, error) {
	w.mtx.RLock()
	defer w.mtx.RUnlock()

	// Select fee
	var fee iwallet.Amount
	switch feeLevel {
	case iwallet.FlEconomic:
		fee = iwallet.NewAmount(250)
	case iwallet.FlNormal:
		fee = iwallet.NewAmount(500)
	case iwallet.FlPriority:
		fee = iwallet.NewAmount(750)
	}

	// Keep adding utxos until the total in value is
	// greater than amt + fee
	totalWithFee := amt.Add(fee)
	var (
		totalUtxo iwallet.Amount
		utxos     []mockUtxo
	)
	for _, utxo := range w.utxos {
		utxos = append(utxos, utxo)
		totalUtxo = totalUtxo.Add(utxo.value)

		if totalUtxo.Cmp(totalWithFee) >= 0 {
			break
		}
	}
	if totalUtxo.Cmp(totalWithFee) < 0 {
		return "", errors.New("insufficient funds")
	}

	txidBytes := make([]byte, 32)
	rand.Read(txidBytes)

	txn := iwallet.Transaction{
		ID: iwallet.TransactionID(hex.EncodeToString(txidBytes)),
		To: []iwallet.SpendInfo{
			{
				Address: to,
				Amount:  amt,
				ID:      append(txidBytes, []byte{0x00, 0x00, 0x00, 0x00}...),
			},
		},
	}

	// Maybe add change
	var changeUtxo *mockUtxo
	if totalUtxo.Cmp(totalWithFee) > 0 {
		changeAddr, err := w.newAddress()
		if err != nil {
			return txn.ID, err
		}
		change := iwallet.SpendInfo{
			Address: changeAddr,
			Amount:  totalUtxo.Sub(amt.Add(fee)),
			ID:      append(txidBytes, []byte{0x00, 0x00, 0x00, 0x01}...),
		}
		txn.To = append(txn.To, change)

		changeUtxo = &mockUtxo{
			outpoint: change.ID,
			address:  change.Address,
			value:    change.Amount,
			height:   0,
		}
	}

	var utxosToDelete []string
	for _, utxo := range utxos {
		in := iwallet.SpendInfo{
			ID:      utxo.outpoint,
			Address: utxo.address,
			Amount:  utxo.value,
		}
		txn.From = append(txn.From, in)
		utxosToDelete = append(utxosToDelete, hex.EncodeToString(utxo.outpoint))
	}

	dbTx := tx.(*dbTx)
	dbTx.onCommit = func() error {
		w.mtx.Lock()
		w.transactions[txn.ID] = txn
		for _, utxo := range utxosToDelete {
			delete(w.utxos, utxo)
		}
		if changeUtxo != nil {
			w.utxos[hex.EncodeToString(changeUtxo.outpoint)] = *changeUtxo
			w.addrs[changeUtxo.address] = true
		}
		w.mtx.Unlock()
		if w.outgoing != nil {
			w.outgoing <- txn
		}
		return nil
	}

	return txn.ID, nil
}

// SweepWallet should sweep the full balance of the wallet to the requested
// address. It is expected for most coins that the fee will be subtracted
// from the amount sent rather than added to it.
func (w *MockWallet) SweepWallet(tx iwallet.Tx, to iwallet.Address, feeLevel iwallet.FeeLevel) (iwallet.TransactionID, error) {
	w.mtx.RLock()
	defer w.mtx.RUnlock()

	// Select fee
	var fee iwallet.Amount
	switch feeLevel {
	case iwallet.FlEconomic:
		fee = iwallet.NewAmount(250)
	case iwallet.FlNormal:
		fee = iwallet.NewAmount(500)
	case iwallet.FlPriority:
		fee = iwallet.NewAmount(750)
	}

	var (
		totalUtxo iwallet.Amount
		utxos     []mockUtxo
	)
	for _, utxo := range w.utxos {
		utxos = append(utxos, utxo)
		totalUtxo = totalUtxo.Add(utxo.value)
	}

	txidBytes := make([]byte, 32)
	rand.Read(txidBytes)

	txn := iwallet.Transaction{
		ID: iwallet.TransactionID(hex.EncodeToString(txidBytes)),
		To: []iwallet.SpendInfo{
			{
				Address: to,
				Amount:  totalUtxo.Sub(fee),
				ID:      append(txidBytes, []byte{0x00, 0x00, 0x00, 0x00}...),
			},
		},
	}

	var utxosToDelete []string
	for _, utxo := range utxos {
		in := iwallet.SpendInfo{
			ID:      utxo.outpoint,
			Address: utxo.address,
			Amount:  utxo.value,
		}
		txn.From = append(txn.From, in)
		utxosToDelete = append(utxosToDelete, hex.EncodeToString(utxo.outpoint))
	}

	dbTx := tx.(*dbTx)
	dbTx.onCommit = func() error {
		w.mtx.Lock()
		w.transactions[txn.ID] = txn
		for _, utxo := range utxosToDelete {
			delete(w.utxos, utxo)
		}
		w.mtx.Unlock()
		if w.outgoing != nil {
			w.outgoing <- txn
		}
		return nil
	}

	return txn.ID, nil
}

// SubscribeTransactions returns a chan over which the wallet is expected
// to push both transactions relevant for this wallet as well as transactions
// sending to or spending from a watched address.
func (w *MockWallet) SubscribeTransactions() <-chan iwallet.Transaction {
	w.mtx.Lock()
	defer w.mtx.Unlock()

	ch := make(chan iwallet.Transaction)
	w.txSubs = append(w.txSubs, ch)
	return ch
}

// SubscribeBlocks returns a chan over which the wallet is expected
// to push info about new blocks when they arrive.
func (w *MockWallet) SubscribeBlocks() <-chan iwallet.BlockInfo {
	ch := make(chan iwallet.BlockInfo)
	w.blockSubs = append(w.blockSubs, ch)
	return ch
}

// WatchAddress is used by the escrow system to tell the wallet to listen
// on the escrow address. It's expected that payments into and spends from
// this address will be pushed back to OpenBazaar.
//
// Note a database transaction is used here. Same rules of Commit() and
// Rollback() apply.
func (w *MockWallet) WatchAddress(tx iwallet.Tx, addrs ...iwallet.Address) error {
	dbtx := tx.(*dbTx)
	dbtx.onCommit = func() error {
		w.mtx.Lock()
		defer w.mtx.Unlock()
		for _, addr := range addrs {
			_, ok := w.addrs[addr]
			if !ok {
				w.watchedAddrs[addr] = struct{}{}
			}
		}
		return nil
	}
	return nil
}

// EstimateEscrowFee estimates the fee to release the funds from escrow.
// this assumes only one input. If there are more inputs OpenBazaar will
// will add 50% of the returned fee for each additional input. This is a
// crude fee calculating but it simplifies things quite a bit.
func (w *MockWallet) EstimateEscrowFee(threshold int, feeLevel iwallet.FeeLevel) (iwallet.Amount, error) {
	var (
		fee                   iwallet.Amount
		feePerAdditionalInput iwallet.Amount
	)
	switch feeLevel {
	case iwallet.FlEconomic:
		fee = iwallet.NewAmount(250)
		feePerAdditionalInput = iwallet.NewAmount(100)
	case iwallet.FlNormal:
		fee = iwallet.NewAmount(500)
		feePerAdditionalInput = iwallet.NewAmount(200)
	case iwallet.FlPriority:
		fee = iwallet.NewAmount(750)
		feePerAdditionalInput = iwallet.NewAmount(300)
	}
	for i := 0; i < threshold; i++ {
		fee = fee.Add(feePerAdditionalInput)
	}
	return fee, nil
}

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
func (w *MockWallet) CreateMultisigAddress(keys []btcec.PublicKey, threshold int) (iwallet.Address, []byte, error) {
	var redeemScript []byte
	for _, key := range keys {
		redeemScript = append(redeemScript, key.SerializeCompressed()...)
	}
	t := make([]byte, 4)
	binary.BigEndian.PutUint32(t, uint32(threshold))
	redeemScript = append(redeemScript, t...)

	h := sha256.Sum256(redeemScript)
	addr := iwallet.NewAddress(hex.EncodeToString(h[:]), iwallet.CtMock)
	return addr, redeemScript, nil
}

// CreateMultisigWithTimeout is the same as CreateMultisigAddress but it adds
// an additional timeout to the address. The address should have two ways to
// release the funds:
//  - m of n signatures are provided (or)
//  - timeout has passed and a signature for timeoutKey is provided.
func (w *MockWallet) CreateMultisigWithTimeout(keys []btcec.PublicKey, threshold int, timeout time.Duration, timeoutKey btcec.PublicKey) (iwallet.Address, []byte, error) {
	var redeemScript []byte
	for _, key := range keys {
		redeemScript = append(redeemScript, key.SerializeCompressed()...)
	}
	t := make([]byte, 4)
	binary.BigEndian.PutUint32(t, uint32(threshold))
	redeemScript = append(redeemScript, t...)
	redeemScript = append(redeemScript, timeoutKey.SerializeCompressed()...)
	u := time.Now().Add(timeout).Unix()
	expiry := make([]byte, 8)
	binary.BigEndian.PutUint64(expiry, uint64(u))
	redeemScript = append(redeemScript, expiry...)

	h := sha256.Sum256(redeemScript)
	addr := iwallet.NewAddress(hex.EncodeToString(h[:]), iwallet.CtMock)
	return addr, redeemScript, nil
}

// CanReleaseFunds returns whether the wallet can release the funds from escrow. This MUST
// return false if the wallet is encrypted or if there is insufficient coins in the wallet
// to pay the transaction fee/gas. This method should not actually move any funds.
func (w *MockWallet) CanReleaseFunds(txn iwallet.Transaction, signatures [][]iwallet.EscrowSignature, redeemScript []byte) (bool, error) {
	return true, nil
}

// ReleaseFundsAfterTimeout will release funds from the escrow. The signature will
// be created using the timeoutKey.
func (w *MockWallet) ReleaseFundsAfterTimeout(tx iwallet.Tx, txn iwallet.Transaction, timeoutKey btcec.PrivateKey, redeemScript []byte) (iwallet.TransactionID, error) {
	w.mtx.RLock()
	defer w.mtx.RUnlock()

	dbtx := tx.(*dbTx)

	txidBytes := make([]byte, 32)
	rand.Read(txidBytes)
	txn.ID = iwallet.TransactionID(hex.EncodeToString(txidBytes))

	expiry := binary.BigEndian.Uint64(redeemScript[len(redeemScript)-8:])
	ts := time.Unix(int64(expiry), 0)

	if ts.After(time.Now()) {
		return txn.ID, errors.New("timeout has not yet passed")
	}

	var utxosToAdd []mockUtxo
	for i, out := range txn.To {
		if _, ok := w.addrs[out.Address]; ok {
			idx := make([]byte, 4)
			binary.BigEndian.PutUint32(idx, uint32(i))
			utxosToAdd = append(utxosToAdd, mockUtxo{
				address:  out.Address,
				value:    out.Amount,
				outpoint: append(txidBytes, idx...),
			})
		}
	}

	dbtx.onCommit = func() error {
		w.mtx.Lock()

		for _, utxo := range utxosToAdd {
			w.utxos[hex.EncodeToString(utxo.outpoint)] = utxo
			w.addrs[utxo.address] = true
		}

		w.transactions[txn.ID] = txn
		w.mtx.Unlock()

		if w.outgoing != nil {
			w.outgoing <- txn
		}

		for _, sub := range w.txSubs {
			sub <- txn
		}
		return nil
	}

	return txn.ID, nil
}

// SignMultisigTransaction should use the provided key to create a signature for
// the multisig transaction. Since this a threshold signature this function will
// separately by each party signing this transaction. The resulting signatures
// will be shared between the relevant parties and one of them will aggregate
// the signatures into a transaction for broadcast.
//
// For coins like bitcoin you may need to return one signature *per input* which is
// why a slice of signatures is returned.
func (w *MockWallet) SignMultisigTransaction(txn iwallet.Transaction, key btcec.PrivateKey, redeemScript []byte) ([]iwallet.EscrowSignature, error) {
	var sigs []iwallet.EscrowSignature
	for i := range txn.From {
		sigBytes := make([]byte, 64)
		rand.Read(sigBytes)

		sigs = append(sigs, iwallet.EscrowSignature{
			Index:     i,
			Signature: sigBytes,
		})
	}
	return sigs, nil
}

// BuildAndSend should used the passed in signatures to build the transaction.
// Note the signatures are a slice of slices. This is because coins like Bitcoin
// may require one signature *per input*. In this case the outer slice is the
// signatures from the different key holders and the inner slice is the keys
// per input.
//
// Note a database transaction is used here. Same rules of Commit() and
// Rollback() apply.
func (w *MockWallet) BuildAndSend(tx iwallet.Tx, txn iwallet.Transaction, signatures [][]iwallet.EscrowSignature, redeemScript []byte) (iwallet.TransactionID, error) {
	w.mtx.RLock()
	defer w.mtx.RUnlock()

	dbtx := tx.(*dbTx)

	txidBytes := make([]byte, 32)
	rand.Read(txidBytes)
	txn.ID = iwallet.TransactionID(hex.EncodeToString(txidBytes))

	for i, in := range txn.From {
		outpointIndex := binary.BigEndian.Uint32(in.ID[32:])
		prev, ok := w.transactions[iwallet.TransactionID(hex.EncodeToString(in.ID[:32]))]
		if ok {
			txn.From[i].Address = prev.To[outpointIndex].Address
			txn.From[i].Amount = prev.To[outpointIndex].Amount
		}
	}

	var utxosToAdd []mockUtxo
	for i, out := range txn.To {
		if _, ok := w.addrs[out.Address]; ok {
			idx := make([]byte, 4)
			binary.BigEndian.PutUint32(idx, uint32(i))
			utxosToAdd = append(utxosToAdd, mockUtxo{
				address:  out.Address,
				value:    out.Amount,
				outpoint: append(txidBytes, idx...),
			})
		}
	}

	dbtx.onCommit = func() error {
		w.mtx.Lock()

		for _, utxo := range utxosToAdd {
			w.utxos[hex.EncodeToString(utxo.outpoint)] = utxo
			w.addrs[utxo.address] = true
		}

		w.transactions[txn.ID] = txn
		w.mtx.Unlock()

		if w.outgoing != nil {
			w.outgoing <- txn
		}

		for _, sub := range w.txSubs {
			sub <- txn
		}
		return nil
	}

	return txn.ID, nil
}

// NewMockExchangeRates returns a mock exchange rate provider that returns
// mock data.
func NewMockExchangeRates() (*ExchangeRateProvider, error) {
	mockedHTTPClient := http.Client{}
	httpmock.ActivateNonDefault(&mockedHTTPClient)

	httpmock.RegisterResponder(http.MethodGet, "https://testrates.com/api",
		func(req *http.Request) (*http.Response, error) {
			return httpmock.NewJsonResponse(http.StatusInternalServerError, MockExchangeRateResponse)
		},
	)

	provider := NewExchangeRateProvider([]string{"https://testrates.com/api"})
	obAPI, ok := provider.providers[0].(*openBazaarAPI)
	if !ok {
		return nil, errors.New("type assertion failure provider 0 is not openBazaarAPI")
	}
	obAPI.client = &mockedHTTPClient

	return provider, nil
}

// MockExchangeRateResponse is a mock response from an exchange rate API.
var MockExchangeRateResponse = map[string]apiRate{
	"BTC": {Last: 1},
	"BCH": {Last: 31.588915},
	"LTC": {Last: 112.163246},
	"ZEC": {Last: 128.77109},
	"ETH": {Last: 42.35316},
	"USD": {Last: 12863.08},
	"EUR": {Last: 11444.58},
	"JPY": {Last: 1398311.17},
	"CNY": {Last: 88439.82},
	"MCK": {Last: 500},
}
