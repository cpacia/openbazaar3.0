package api

import (
	"github.com/cpacia/openbazaar3.0/models"
	iwallet "github.com/cpacia/wallet-interface"
	"github.com/gorilla/mux"
	"github.com/jinzhu/gorm"
	"net/http"
	"strconv"
	"time"
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
				http.Error(w, wrapError(err), http.StatusInternalServerError)
				return
			}

			info, err := wallet.BlockchainInfo()
			if err != nil {
				http.Error(w, wrapError(err), http.StatusInternalServerError)
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
		http.Error(w, wrapError(err), http.StatusBadRequest)
		return
	}

	unconfirmed, confirmed, err := wallet.Balance()
	if err != nil {
		http.Error(w, wrapError(err), http.StatusInternalServerError)
		return
	}

	info, err := wallet.BlockchainInfo()
	if err != nil {
		http.Error(w, wrapError(err), http.StatusInternalServerError)
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
	Address string `json:"address"`
}

func (g *Gateway) handleGETAddress(w http.ResponseWriter, r *http.Request) {
	coinType := mux.Vars(r)["coinType"]

	if coinType == "" {
		ret := make(map[string]string)

		for ct, wallet := range g.node.Multiwallet() {
			address, err := wallet.CurrentAddress()
			if err != nil {
				http.Error(w, wrapError(err), http.StatusInternalServerError)
				return
			}

			ret[ct.CurrencyCode()] = address.String()
		}

		sanitizedJSONResponse(w, ret)
		return
	}

	mw := g.node.Multiwallet()
	wallet, err := mw.WalletForCurrencyCode(coinType)
	if err != nil {
		http.Error(w, wrapError(err), http.StatusBadRequest)
		return
	}

	address, err := wallet.CurrentAddress()
	if err != nil {
		http.Error(w, wrapError(err), http.StatusInternalServerError)
		return
	}

	ret := walletAddressResponse{
		Address: address.String(),
	}

	sanitizedJSONResponse(w, ret)
}

type walletTransactionResponse struct {
	Txid          string    `json:"txid"`
	Value         string    `json:"value"`
	Address       string    `json:"address"`
	Status        string    `json:"status"`
	Timestamp     time.Time `json:"timestamp"`
	Confirmations uint64    `json:"confirmations"`
	Height        uint64    `json:"height"`
	Memo          string    `json:"memo"`
	OrderID       string    `json:"orderId"`
	Thumbnail     string    `json:"thumbnail"`
}

func (g *Gateway) handleGETTransactions(w http.ResponseWriter, r *http.Request) {
	var (
		coinType = mux.Vars(r)["coinType"]
		limitStr = r.URL.Query().Get("limit")
		offsetID = r.URL.Query().Get("offsetID")
		limit    = -1
		err      error
	)
	if limitStr != "" {
		limit, err = strconv.Atoi(limitStr)
		if err != nil {
			http.Error(w, wrapError(err), http.StatusBadRequest)
			return
		}
	}

	mw := g.node.Multiwallet()
	wallet, err := mw.WalletForCurrencyCode(coinType)
	if err != nil {
		http.Error(w, wrapError(err), http.StatusBadRequest)
		return
	}

	def, err := models.CurrencyDefinitions.Lookup(coinType)
	if err != nil {
		http.Error(w, wrapError(err), http.StatusInternalServerError)
		return
	}

	chainInfo, err := wallet.BlockchainInfo()
	if err != nil {
		http.Error(w, wrapError(err), http.StatusInternalServerError)
		return
	}

	txs, err := wallet.Transactions(limit, iwallet.TransactionID(offsetID))
	if err != nil {
		http.Error(w, wrapError(err), http.StatusInternalServerError)
		return
	}
	confirmedThreshold := uint64(time.Hour / def.BlockInterval)
	ret := make([]walletTransactionResponse, 0, len(txs))
	for _, tx := range txs {
		var (
			confirmations = uint64(0)
			status        string
		)
		if tx.Height > 0 {
			confirmations = (chainInfo.Height - tx.Height) + 1
		}
		if confirmations == 0 {
			status = "UNCONFIRMED"
		} else if confirmations < confirmedThreshold {
			status = "PENDING"
		} else if confirmations >= confirmedThreshold {
			status = "CONFIRMED"
		}
		metadata, err := g.node.GetTransactionMetadata(tx.ID)
		if err != nil && !gorm.IsRecordNotFoundError(err) {
			http.Error(w, wrapError(err), http.StatusInternalServerError)
			return
		}

		ret = append(ret, walletTransactionResponse{
			Txid:          tx.ID.String(),
			Value:         tx.Value.String(),
			Height:        tx.Height,
			Timestamp:     tx.Timestamp,
			Confirmations: confirmations,
			Status:        status,
			Memo:          metadata.Memo,
			Thumbnail:     metadata.Thumbnail,
			OrderID:       metadata.OrderID.String(),
			Address:       metadata.PaymentAddress,
		})
	}

	sanitizedJSONResponse(w, ret)
}

func (g *Gateway) handleGETCurrencies(w http.ResponseWriter, r *http.Request) {
	sanitizedJSONResponse(w, models.CurrencyDefinitions)
}
