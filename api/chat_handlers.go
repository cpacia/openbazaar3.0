package api

import (
	"encoding/json"
	"github.com/gorilla/mux"
	peer "github.com/libp2p/go-libp2p-peer"
	"net/http"
	"strconv"
)

func (g *Gateway) handlePOSTSendChatMessage(w http.ResponseWriter, r *http.Request) {
	type message struct {
		PeerID  string `json:"peerID"`
		Message string `json:"message"`
		OrderID string `json:"orderID"`
	}
	var m message
	if err := json.NewDecoder(r.Body).Decode(&m); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	pid, err := peer.IDB58Decode(m.PeerID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := g.node.SendChatMessage(pid, m.Message, m.OrderID, nil); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (g *Gateway) handlePOSTSendGroupChatMessage(w http.ResponseWriter, r *http.Request) {
	type message struct {
		PeerIDs []string `json:"peerID"`
		Message string   `json:"message"`
		OrderID string   `json:"orderID"`
	}
	var m message
	if err := json.NewDecoder(r.Body).Decode(&m); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	for _, peerID := range m.PeerIDs {
		pid, err := peer.IDB58Decode(peerID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		if err := g.node.SendChatMessage(pid, m.Message, m.OrderID, nil); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
}

func (g *Gateway) handlePOSTSendTypingMessage(w http.ResponseWriter, r *http.Request) {
	type message struct {
		PeerID  string `json:"peerID"`
		OrderID string `json:"orderID"`
	}
	var m message
	if err := json.NewDecoder(r.Body).Decode(&m); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	pid, err := peer.IDB58Decode(m.PeerID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := g.node.SendTypingMessage(pid, m.OrderID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (g *Gateway) handlePOSTSendGroupTypingMessage(w http.ResponseWriter, r *http.Request) {
	type message struct {
		PeerIDs []string `json:"peerID"`
		OrderID string   `json:"orderID"`
	}
	var m message
	if err := json.NewDecoder(r.Body).Decode(&m); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	for _, peerID := range m.PeerIDs {
		pid, err := peer.IDB58Decode(peerID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		if err := g.node.SendTypingMessage(pid, m.OrderID); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
}

func (g *Gateway) handlePOSTMarkChatMessageAsRead(w http.ResponseWriter, r *http.Request) {
	type message struct {
		PeerID  string `json:"peerID"`
		OrderID string `json:"orderID"`
	}
	var m message
	if err := json.NewDecoder(r.Body).Decode(&m); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	pid, err := peer.IDB58Decode(m.PeerID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := g.node.MarkChatMessagesAsRead(pid, m.OrderID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (g *Gateway) handleGETChatConversations(w http.ResponseWriter, r *http.Request) {
	convos, err := g.node.GetChatConversations()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	sanitizedJSONResponse(w, convos)
}

func (g *Gateway) handleGETChatMessages(w http.ResponseWriter, r *http.Request) {
	var (
		peerIDStr = mux.Vars(r)["peerID"]
		limitStr  = r.URL.Query().Get("limit")
		offsetID  = r.URL.Query().Get("offsetID")
		limit     = -1
		err       error
	)
	if limitStr != "" {
		limit, err = strconv.Atoi(limitStr)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
	}
	pid, err := peer.IDB58Decode(peerIDStr)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	messages, err := g.node.GetChatMessagesByPeer(pid, limit, offsetID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	sanitizedJSONResponse(w, messages)
}

func (g *Gateway) handleGETChatConversation(w http.ResponseWriter, r *http.Request) {
	var (
		orderID  = mux.Vars(r)["orderID"]
		limitStr = r.URL.Query().Get("limit")
		offsetID = r.URL.Query().Get("offsetID")
		limit    = -1
		err      error
	)
	if limitStr != "" {
		limit, err = strconv.Atoi(limitStr)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
	}

	messages, err := g.node.GetChatMessagesByOrderID(orderID, limit, offsetID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	sanitizedJSONResponse(w, messages)
}
