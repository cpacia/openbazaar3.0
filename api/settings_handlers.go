package api

import (
	"encoding/json"
	"errors"
	"github.com/cpacia/openbazaar3.0/core/coreiface"
	"github.com/cpacia/openbazaar3.0/models"
	"github.com/cpacia/openbazaar3.0/version"
	iwallet "github.com/cpacia/wallet-interface"
	"net/http"
)

type nodeConfig struct {
	PeerId  string   `json:"peerID"`
	Testnet bool     `json:"testnet"`
	Tor     bool     `json:"tor"`
	Wallets []string `json:"wallets"`
}

func (g *Gateway) handleGETConfig(w http.ResponseWriter, r *http.Request) {
	ret := nodeConfig{
		PeerId:  g.node.Identity().Pretty(),
		Testnet: g.node.UsingTestnet(),
		Tor:     g.node.UsingTorMode(),
	}

	for currency := range g.node.Multiwallet() {
		ret.Wallets = append(ret.Wallets, currency.CurrencyCode())
	}

	sanitizedJSONResponse(w, &ret)
}

func (g *Gateway) handlePutUserPreferences(w http.ResponseWriter, r *http.Request) {
	var prefs models.UserPreferences

	decoder := json.NewDecoder(r.Body)

	if err := decoder.Decode(&prefs); err != nil {
		http.Error(w, wrapError(err), http.StatusBadRequest)
		return
	}

	err := g.node.SavePreferences(&prefs, nil)
	if errors.Is(err, coreiface.ErrBadRequest) {
		http.Error(w, wrapError(err), http.StatusBadRequest)
		return
	} else if err != nil {
		http.Error(w, wrapError(err), http.StatusInternalServerError)
		return
	}
	sanitizedJSONResponse(w, struct{}{})
}

func (g *Gateway) handleGetUserPreferences(w http.ResponseWriter, r *http.Request) {
	prefs, err := g.node.GetPreferences()
	if err != nil {
		http.Error(w, wrapError(err), http.StatusInternalServerError)
		return
	}
	prefs.UserAgent = version.UserAgent()
	sanitizedJSONResponse(w, prefs)
}

func (g *Gateway) handleGETExchangeRates(w http.ResponseWriter, r *http.Request) {
	rates, err := g.node.ExchangeRates().GetAllRates(iwallet.CtBitcoin, false)
	if err != nil {
		http.Error(w, wrapError(err), http.StatusInternalServerError)
		return
	}
	sanitizedJSONResponse(w, rates)
}
