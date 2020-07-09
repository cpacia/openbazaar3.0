package api

import (
	"encoding/json"
	"github.com/gorilla/mux"
	"github.com/ipfs/go-cid"
	"net/http"
	"strconv"
)

func (g *Gateway) handlePOSTPublishChannelMessage(w http.ResponseWriter, r *http.Request) {
	type message struct {
		Message string `json:"message"`
		Topic   string `json:"topic"`
	}
	var m message
	if err := json.NewDecoder(r.Body).Decode(&m); err != nil {
		http.Error(w, wrapError(err), http.StatusBadRequest)
		return
	}

	if err := g.node.PublishChannelMessage(r.Context(), m.Topic, m.Message); err != nil {
		http.Error(w, wrapError(err), http.StatusInternalServerError)
		return
	}
}

func (g *Gateway) handlePOSTOpenChannel(w http.ResponseWriter, r *http.Request) {
	topic := mux.Vars(r)["topic"]

	if err := g.node.OpenChannel(topic); err != nil {
		http.Error(w, wrapError(err), http.StatusInternalServerError)
		return
	}
}

func (g *Gateway) handlePOSTCloseChannel(w http.ResponseWriter, r *http.Request) {
	topic := mux.Vars(r)["topic"]

	if err := g.node.CloseChannel(topic); err != nil {
		http.Error(w, wrapError(err), http.StatusInternalServerError)
		return
	}
}

func (g *Gateway) handleGETListChannels(w http.ResponseWriter, r *http.Request) {
	sanitizedJSONResponse(w, g.node.ListChannels())
}

func (g *Gateway) handleGETChannelMessages(w http.ResponseWriter, r *http.Request) {
	var (
		topic    = mux.Vars(r)["topic"]
		limitStr = r.URL.Query().Get("limit")
		offsetID = r.URL.Query().Get("offsetID")
		limit    = -1
		oid      *cid.Cid
		err      error
	)
	if limitStr != "" {
		limit, err = strconv.Atoi(limitStr)
		if err != nil {
			http.Error(w, wrapError(err), http.StatusBadRequest)
			return
		}
	}
	if offsetID != "" {
		id, err := cid.Decode(offsetID)
		if err != nil {
			http.Error(w, wrapError(err), http.StatusBadRequest)
			return
		}
		oid = &id
	}

	messages, err := g.node.GetChannelMessages(r.Context(), topic, oid, limit)
	if err != nil {
		http.Error(w, wrapError(err), http.StatusInternalServerError)
		return
	}
	sanitizedJSONResponse(w, messages)
}
