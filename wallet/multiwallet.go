package wallet

import (
	"errors"
	iwallet "github.com/cpacia/wallet-interface"
	"strings"
)


var UnsuppertedCoinError = errors.New("multiwallet does not contain an implementation for the given coin")

// TODO: this is a place holder for the new multiwallet
type Multiwallet map[iwallet.CoinType]iwallet.Wallet

func (w *Multiwallet) Start() {
	for _, wallet := range *w {
		wallet.OpenWallet()
	}
}

func (w *Multiwallet) Close() {
	for _, wallet := range *w {
		wallet.CloseWallet()
	}
}

func (w *Multiwallet) WalletForCurrencyCode(currencyCode string) (iwallet.Wallet, error) {
	for cc, wl := range *w {
		if strings.ToUpper(cc.CurrencyCode()) == strings.ToUpper(currencyCode) || strings.ToUpper(cc.CurrencyCode()) == "T"+strings.ToUpper(currencyCode) {
			return wl, nil
		}
	}
	return nil, UnsuppertedCoinError
}
