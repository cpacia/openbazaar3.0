package api

import (
	"encoding/json"
	"github.com/cpacia/openbazaar3.0/models"
	"github.com/gorilla/mux"
	peer "github.com/libp2p/go-libp2p-core/peer"
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
		http.Error(w, wrapError(err), http.StatusBadRequest)
		return
	}

	pid, err := peer.Decode(m.PeerID)
	if err != nil {
		http.Error(w, wrapError(err), http.StatusBadRequest)
		return
	}

	if err := g.node.SendChatMessage(pid, m.Message, models.OrderID(m.OrderID), nil); err != nil {
		http.Error(w, wrapError(err), http.StatusInternalServerError)
		return
	}
}

func (g *Gateway) handlePOSTSendGroupChatMessage(w http.ResponseWriter, r *http.Request) {
	type message struct {
		PeerIDs []string `json:"peerIDs"`
		Message string   `json:"message"`
		OrderID string   `json:"orderID"`
	}
	var m message
	if err := json.NewDecoder(r.Body).Decode(&m); err != nil {
		http.Error(w, wrapError(err), http.StatusBadRequest)
		return
	}

	for _, peerID := range m.PeerIDs {
		pid, err := peer.Decode(peerID)
		if err != nil {
			http.Error(w, wrapError(err), http.StatusBadRequest)
			return
		}

		if err := g.node.SendChatMessage(pid, m.Message, models.OrderID(m.OrderID), nil); err != nil {
			http.Error(w, wrapError(err), http.StatusInternalServerError)
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
		http.Error(w, wrapError(err), http.StatusBadRequest)
		return
	}

	pid, err := peer.Decode(m.PeerID)
	if err != nil {
		http.Error(w, wrapError(err), http.StatusBadRequest)
		return
	}

	if err := g.node.SendTypingMessage(pid, models.OrderID(m.OrderID)); err != nil {
		http.Error(w, wrapError(err), http.StatusInternalServerError)
		return
	}
}

func (g *Gateway) handlePOSTSendGroupTypingMessage(w http.ResponseWriter, r *http.Request) {
	type message struct {
		PeerIDs []string `json:"peerIDs"`
		OrderID string   `json:"orderID"`
	}
	var m message
	if err := json.NewDecoder(r.Body).Decode(&m); err != nil {
		http.Error(w, wrapError(err), http.StatusBadRequest)
		return
	}

	for _, peerID := range m.PeerIDs {
		pid, err := peer.Decode(peerID)
		if err != nil {
			http.Error(w, wrapError(err), http.StatusBadRequest)
			return
		}

		if err := g.node.SendTypingMessage(pid, models.OrderID(m.OrderID)); err != nil {
			http.Error(w, wrapError(err), http.StatusInternalServerError)
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
		http.Error(w, wrapError(err), http.StatusBadRequest)
		return
	}

	pid, err := peer.Decode(m.PeerID)
	if err != nil {
		http.Error(w, wrapError(err), http.StatusBadRequest)
		return
	}

	if err := g.node.MarkChatMessagesAsRead(pid, models.OrderID(m.OrderID)); err != nil {
		http.Error(w, wrapError(err), http.StatusInternalServerError)
		return
	}
}

func (g *Gateway) handleGETChatConversations(w http.ResponseWriter, r *http.Request) {
	convos, err := g.node.GetChatConversations()
	if err != nil {
		http.Error(w, wrapError(err), http.StatusInternalServerError)
		return
	}
	if convos == nil {
		convos = []models.ChatConversation{}
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
			http.Error(w, wrapError(err), http.StatusBadRequest)
			return
		}
	}
	pid, err := peer.Decode(peerIDStr)
	if err != nil {
		http.Error(w, wrapError(err), http.StatusBadRequest)
		return
	}
	messages, err := g.node.GetChatMessagesByPeer(pid, limit, offsetID)
	if err != nil {
		http.Error(w, wrapError(err), http.StatusInternalServerError)
		return
	}
	if messages == nil {
		messages = []models.ChatMessage{}
	}
	sanitizedJSONResponse(w, messages)
}

func (g *Gateway) handleGETGroupChatMessages(w http.ResponseWriter, r *http.Request) {
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
			http.Error(w, wrapError(err), http.StatusBadRequest)
			return
		}
	}

	messages, err := g.node.GetChatMessagesByOrderID(models.OrderID(orderID), limit, offsetID)
	if err != nil {
		http.Error(w, wrapError(err), http.StatusInternalServerError)
		return
	}
	if messages == nil {
		messages = []models.ChatMessage{}
	}
	sanitizedJSONResponse(w, messages)
}

func (g *Gateway) handleDELETEChatMessages(w http.ResponseWriter, r *http.Request) {
	messageID := mux.Vars(r)["messageID"]
	err := g.node.DeleteChatMessage(messageID)
	if err != nil {
		http.Error(w, wrapError(err), http.StatusInternalServerError)
		return
	}
}

func (g *Gateway) handleDELETEGroupChatMessages(w http.ResponseWriter, r *http.Request) {
	orderID := mux.Vars(r)["orderID"]
	err := g.node.DeleteGroupChatMessages(models.OrderID(orderID))
	if err != nil {
		http.Error(w, wrapError(err), http.StatusInternalServerError)
		return
	}
}

func (g *Gateway) handleDELETEChatConversation(w http.ResponseWriter, r *http.Request) {
	peerIDStr := mux.Vars(r)["peerID"]

	pid, err := peer.Decode(peerIDStr)
	if err != nil {
		http.Error(w, wrapError(err), http.StatusBadRequest)
		return
	}

	err = g.node.DeleteChatConversation(pid)
	if err != nil {
		http.Error(w, wrapError(err), http.StatusInternalServerError)
		return
	}
}
