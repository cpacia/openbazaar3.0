package wallet

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"github.com/OpenBazaar/wallet-interface"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcutil"
	"github.com/btcsuite/btcutil/hdkeychain"
	"github.com/cpacia/openbazaar3.0/events"
	"math/big"
	"testing"
	"time"
)

func TestMockWallet_Spend(t *testing.T) {
	w := NewMockWallet()

	var (
		testUtxoPrevHash = make([]byte, 32)
	)
	rand.Read(testUtxoPrevHash)

	addr := w.NewAddress(wallet.INTERNAL)

	w.utxos[hex.EncodeToString(testUtxoPrevHash)+"0"] = mockUtxo{
		OutpointHash:  testUtxoPrevHash,
		OutpointIndex: 0,
		Address:       addr,
		Value:         *big.NewInt(10000),
	}

	spendAddr, err := btcutil.NewAddressScriptHash(testUtxoPrevHash, &chaincfg.TestNet3Params)
	if err != nil {
		t.Fatal(err)
	}

	_, err = w.Spend(*big.NewInt(20000), spendAddr, wallet.PRIOIRTY, "", false)
	if err != wallet.ErrorInsuffientFunds {
		t.Error("error is not insufficient funds")
	}

	txid, err := w.Spend(*big.NewInt(9000), spendAddr, wallet.PRIOIRTY, "", false)
	if err != nil {
		t.Fatal(err)
	}

	changeUtxo, ok := w.utxos[txid.String()+"1"]
	if !ok {
		t.Error("wallet missing change utxo")
	}

	if !bytes.Equal(changeUtxo.OutpointHash, txid[:]) {
		t.Errorf("Incorrect change utxo hash returned. Expected %s, got %s", hex.EncodeToString(txid[:]), hex.EncodeToString(changeUtxo.OutpointHash))
	}

	if changeUtxo.OutpointIndex != 1 {
		t.Errorf("Incorrect change utxo index. Expected 1, got %d", changeUtxo.OutpointIndex)
	}

	txn, ok := w.transactions[*txid]
	if !ok {
		t.Error("wallet missing transaction")
	}

	if txn.Txid != txid.String() {
		t.Errorf("Incorrect txn txid. Expected %s, got %s", txid.String(), txn.Txid)
	}

	txns, err := w.Transactions()
	if err != nil {
		t.Fatal(err)
	}

	if len(txns) != 1 {
		t.Errorf("Expected 1 txn, got %d", len(txns))
	}

	if txns[0].Txid != txid.String() {
		t.Errorf("Incorrect txn txid. Expected %s, got %s", txid.String(), txn.Txid)
	}
}

func TestMockWallet_GenerateMultisigScript(t *testing.T) {
	var (
		w1 = NewMockWallet()
		w2 = NewMockWallet()
		w3 = NewMockWallet()
	)

	s1 := make([]byte, 32)
	rand.Read(s1)
	s2 := make([]byte, 32)
	rand.Read(s2)
	s3 := make([]byte, 32)
	rand.Read(s3)

	k1, err := hdkeychain.NewMaster(s1, &chaincfg.TestNet3Params)
	if err != nil {
		t.Fatal(err)
	}
	k2, err := hdkeychain.NewMaster(s1, &chaincfg.TestNet3Params)
	if err != nil {
		t.Fatal(err)
	}
	k3, err := hdkeychain.NewMaster(s1, &chaincfg.TestNet3Params)
	if err != nil {
		t.Fatal(err)
	}

	addr1, rs1, err := w1.GenerateMultisigScript([]hdkeychain.ExtendedKey{*k1, *k2, *k3}, 2, time.Second, nil)
	if err != nil {
		t.Fatal(err)
	}
	addr2, rs2, err := w2.GenerateMultisigScript([]hdkeychain.ExtendedKey{*k1, *k2, *k3}, 2, time.Second, nil)
	if err != nil {
		t.Fatal(err)
	}
	addr3, rs3, err := w3.GenerateMultisigScript([]hdkeychain.ExtendedKey{*k1, *k2, *k3}, 2, time.Second, nil)
	if err != nil {
		t.Fatal(err)
	}

	if addr1.String() != addr2.String() {
		t.Errorf("Addresses are not equal. %s, %s", addr1.String(), addr2.String())
	}

	if addr2.String() != addr3.String() {
		t.Errorf("Addresses are not equal. %s, %s", addr2.String(), addr3.String())
	}

	if !bytes.Equal(rs1, rs2) {
		t.Errorf("Redeem scripts are not equal. %v, %v", rs1, rs2)
	}

	if !bytes.Equal(rs2, rs3) {
		t.Errorf("Redeem scripts are not equal. %v, %v", rs2, rs3)
	}
}

func TestMockWallet_Multisign(t *testing.T) {
	var (
		w1 = NewMockWallet()
		w2 = NewMockWallet()
	)

	s1 := make([]byte, 32)
	rand.Read(s1)
	s2 := make([]byte, 32)
	rand.Read(s2)
	s3 := make([]byte, 32)
	rand.Read(s3)

	k1, err := hdkeychain.NewMaster(s1, &chaincfg.TestNet3Params)
	if err != nil {
		t.Fatal(err)
	}
	k2, err := hdkeychain.NewMaster(s1, &chaincfg.TestNet3Params)
	if err != nil {
		t.Fatal(err)
	}
	k3, err := hdkeychain.NewMaster(s1, &chaincfg.TestNet3Params)
	if err != nil {
		t.Fatal(err)
	}

	addr, redeemScript, err := w1.GenerateMultisigScript([]hdkeychain.ExtendedKey{*k1, *k2, *k3}, 2, time.Second, nil)
	if err != nil {
		t.Fatal(err)
	}

	outpointHash := make([]byte, 32)
	rand.Read(outpointHash)

	ins := []wallet.TransactionInput{
		{
			LinkedAddress: addr,
			Value:         10000,
			OutpointIndex: 0,
			OutpointHash:  outpointHash,
		},
	}

	spendAddrHash := make([]byte, 32)
	rand.Read(spendAddrHash)
	spendAddr, err := btcutil.NewAddressScriptHash(spendAddrHash, &chaincfg.TestNet3Params)
	if err != nil {
		t.Fatal(err)
	}

	outs := []wallet.TransactionOutput{
		{
			Address: spendAddr,
			Value:   9750,
			Index:   0,
		},
	}

	sig1, err := w1.CreateMultisigSignature(ins, outs, k1, redeemScript, *big.NewInt(0))
	if err != nil {
		t.Fatal(err)
	}
	sig2, err := w2.CreateMultisigSignature(ins, outs, k2, redeemScript, *big.NewInt(0))
	if err != nil {
		t.Fatal(err)
	}

	_, err = w1.Multisign(ins, outs, sig1, sig2, redeemScript, *big.NewInt(0), true)
	if err != nil {
		t.Fatal(err)
	}

	txs, err := w1.Transactions()
	if err != nil {
		t.Fatal(err)
	}
	if len(txs) != 1 {
		t.Error("Failed to record transaction")
	}

	if txs[0].WatchOnly != true {
		t.Error("Transaction not marked watch only")
	}
}

func TestMockWallet_SweepAddress(t *testing.T) {
	w1 := NewMockWallet()

	s1 := make([]byte, 32)
	rand.Read(s1)

	k1, err := hdkeychain.NewMaster(s1, &chaincfg.TestNet3Params)
	if err != nil {
		t.Fatal(err)
	}

	redeemScript := make([]byte, 32)
	rand.Read(redeemScript)

	outpointHash := make([]byte, 32)
	rand.Read(outpointHash)

	inAddrHash := make([]byte, 32)
	rand.Read(inAddrHash)
	inAddr, err := btcutil.NewAddressScriptHash(inAddrHash, &chaincfg.TestNet3Params)
	if err != nil {
		t.Fatal(err)
	}

	ins := []wallet.TransactionInput{
		{
			LinkedAddress: inAddr,
			Value:         10000,
			OutpointIndex: 0,
			OutpointHash:  outpointHash,
		},
	}

	spendAddrHash := make([]byte, 32)
	rand.Read(spendAddrHash)
	scriptHashAddr, err := btcutil.NewAddressScriptHash(spendAddrHash, &chaincfg.TestNet3Params)
	if err != nil {
		t.Fatal(err)
	}

	spendAddr := btcutil.Address(scriptHashAddr)

	_, err = w1.SweepAddress(ins, &spendAddr, k1, &redeemScript, wallet.PRIOIRTY)

	txs, err := w1.Transactions()
	if err != nil {
		t.Fatal(err)
	}
	if len(txs) != 1 {
		t.Error("Failed to record transaction")
	}

	if txs[0].WatchOnly != true {
		t.Error("Transaction not marked watch only")
	}
}

func TestMockWalletNetwork(t *testing.T) {
	network := NewMockWalletNetwork(3)
	network.Start()

	for _, wallet := range network.Wallets() {
		wallet.SetEventBus(events.NewBus())
	}

	sub, err := network.Wallets()[0].bus.Subscribe(&events.TransactionReceived{})
	if err != nil {
		t.Fatal(err)
	}

	addr := network.Wallets()[0].NewAddress(wallet.EXTERNAL)

	// Generate coins and send to wallet 0.
	if err := network.GenerateToAddress(addr, *big.NewInt(10000)); err != nil {
		t.Fatal(err)
	}

	<-sub.Out()

	if len(network.Wallets()[0].utxos) != 1 {
		t.Error("Failed to record new utxo")
	}

	if len(network.Wallets()[0].transactions) != 1 {
		t.Error("Failed to record new txn")
	}

	confirmed, unconfirmed := network.Wallets()[0].Balance()
	if confirmed.Cmp(big.NewInt(0)) != 0 {
		t.Error("Confirmed balance is not zero")
	}

	if unconfirmed.Cmp(big.NewInt(10000)) != 0 {
		t.Errorf("Incorrect unconfirmed balance. Expected %d, got %d", 10000, unconfirmed.Int64())
	}

	if len(network.Wallets()[1].utxos) != 0 {
		t.Error("Incorrectly recorded utxo")
	}

	if len(network.Wallets()[1].transactions) != 0 {
		t.Error("Incorrectly recorded txn")
	}

	if len(network.Wallets()[2].utxos) != 0 {
		t.Error("Incorrectly recorded utxo")
	}

	if len(network.Wallets()[2].transactions) != 0 {
		t.Error("Incorrectly recorded txn")
	}

	blockSub, err := network.Wallets()[0].bus.Subscribe(&events.BlockReceived{})
	if err != nil {
		t.Fatal(err)
	}

	// Generate block and check the transaction confirms.
	network.GenerateBlock()

	<-blockSub.Out()

	confirmed, unconfirmed = network.Wallets()[0].Balance()
	if confirmed.Cmp(big.NewInt(10000)) != 0 {
		t.Errorf("Incorrect confirmed balance. Expected %d, got %d", 10000, confirmed.Int64())
	}

	if unconfirmed.Cmp(big.NewInt(0)) != 0 {
		t.Error("Unconfirmed balance is not zero")
	}

	sub2, err := network.Wallets()[2].bus.Subscribe(&events.TransactionReceived{})
	if err != nil {
		t.Fatal(err)
	}

	// Wallet 0 send coins to wallet 2.
	addr2 := network.Wallets()[2].CurrentAddress(wallet.EXTERNAL)

	if _, err := network.Wallets()[0].Spend(*big.NewInt(9000), addr2, wallet.PRIOIRTY, "", false); err != nil {
		t.Fatal(err)
	}

	<-sub2.Out()

	if len(network.Wallets()[2].utxos) != 1 {
		t.Error("Failed to record new utxo")
	}

	if len(network.Wallets()[2].transactions) != 1 {
		t.Error("Failed to record new txn")
	}

	confirmed, unconfirmed = network.Wallets()[2].Balance()
	if confirmed.Cmp(big.NewInt(0)) != 0 {
		t.Error("Confirmed balance is not zero")
	}

	if unconfirmed.Cmp(big.NewInt(9000)) != 0 {
		t.Errorf("Incorrect unconfirmed balance. Expected %d, got %d", 9000, unconfirmed.Int64())
	}

	if len(network.Wallets()[0].utxos) != 1 {
		t.Error("Failed to record new utxo")
	}

	if len(network.Wallets()[0].transactions) != 2 {
		t.Error("Failed to record new txn")
	}

	confirmed, unconfirmed = network.Wallets()[0].Balance()
	if confirmed.Cmp(big.NewInt(0)) != 0 {
		t.Error("Confirmed balance is not zero")
	}

	if unconfirmed.Cmp(big.NewInt(750)) != 0 {
		t.Errorf("Incorrect unconfirmed balance. Expected %d, got %d", 750, unconfirmed.Int64())
	}
}
