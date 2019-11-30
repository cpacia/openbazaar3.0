package api

import (
	"github.com/gorilla/mux"
	"net/http"
)

type walletBalanceResponse struct {
	Confirmed   string `json:"confirmed"`
	Unconfirmed string `json:"unconfirmed"`
	Height      uint64 `json:"height"`
}

func (g *Gateway) handleGETBalance(w http.ResponseWriter, r *http.Request) {
	coinType := mux.Vars(r)["coinType"]

	if coinType == "" {
		ret := make(map[string]interface{})

		for ct, wallet := range g.node.Multiwallet() {
			unconfirmed, confirmed, err := wallet.Balance()
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			info, err := wallet.BlockchainInfo()
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			ret[ct.CurrencyCode()] = walletBalanceResponse{
				Confirmed:   confirmed.String(),
				Unconfirmed: unconfirmed.String(),
				Height:      info.Height,
			}
		}

		sanitizedJSONResponse(w, ret)
		return
	}

	mw := g.node.Multiwallet()
	wallet, err := mw.WalletForCurrencyCode(coinType)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	unconfirmed, confirmed, err := wallet.Balance()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	info, err := wallet.BlockchainInfo()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	ret := walletBalanceResponse{
		Confirmed:   confirmed.String(),
		Unconfirmed: unconfirmed.String(),
		Height:      info.Height,
	}

	sanitizedJSONResponse(w, ret)
}

type walletAddressResponse struct {
	Address string `json:"Address"`
}

func (g *Gateway) handleGETAddress(w http.ResponseWriter, r *http.Request) {
	coinType := mux.Vars(r)["coinType"]

	if coinType == "" {
		ret := make(map[string]interface{})

		for ct, wallet := range g.node.Multiwallet() {
			address, err := wallet.CurrentAddress()
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			ret[ct.CurrencyCode()] = walletAddressResponse{
				Address: address.String(),
			}
		}

		sanitizedJSONResponse(w, ret)
		return
	}

	mw := g.node.Multiwallet()
	wallet, err := mw.WalletForCurrencyCode(coinType)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	address, err := wallet.CurrentAddress()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	ret := walletAddressResponse{
		Address: address.String(),
	}

	sanitizedJSONResponse(w, ret)
}
