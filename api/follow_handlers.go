package api

import (
	"errors"
	"github.com/cpacia/openbazaar3.0/core/coreiface"
	"github.com/cpacia/openbazaar3.0/models"
	"github.com/gorilla/mux"
	peer "github.com/libp2p/go-libp2p-core/peer"
	"net/http"
	"strconv"
)

func (g *Gateway) handleGETFollowers(w http.ResponseWriter, r *http.Request) {
	peerIDStr := mux.Vars(r)["peerID"]

	useCache, _ := strconv.ParseBool(r.URL.Query().Get("usecache"))

	var (
		followers models.Followers
		err       error
	)
	if peerIDStr == "" || peerIDStr == g.node.Identity().Pretty() {
		followers, err = g.node.GetMyFollowers()
		if errors.Is(err, coreiface.ErrNotFound) {
			http.Error(w, wrapError(err), http.StatusNotFound)
			return
		} else if err != nil {
			http.Error(w, wrapError(err), http.StatusInternalServerError)
			return
		}
	} else {
		pid, err := peer.Decode(peerIDStr)
		if err != nil {
			http.Error(w, wrapError(err), http.StatusBadRequest)
			return
		}
		followers, err = g.node.GetFollowers(r.Context(), pid, useCache)
		if errors.Is(err, coreiface.ErrNotFound) {
			http.Error(w, wrapError(err), http.StatusNotFound)
			return
		} else if err != nil {
			http.Error(w, wrapError(err), http.StatusInternalServerError)
			return
		}
	}
	if followers == nil {
		followers = models.Followers{}
	}
	sanitizedJSONResponse(w, followers)
}

func (g *Gateway) handleGETFollowing(w http.ResponseWriter, r *http.Request) {
	peerIDStr := mux.Vars(r)["peerID"]

	useCache, _ := strconv.ParseBool(r.URL.Query().Get("usecache"))

	var (
		following models.Following
		err       error
	)
	if peerIDStr == "" || peerIDStr == g.node.Identity().Pretty() {
		following, err = g.node.GetMyFollowing()
		if errors.Is(err, coreiface.ErrNotFound) {
			http.Error(w, wrapError(err), http.StatusNotFound)
			return
		} else if err != nil {
			http.Error(w, wrapError(err), http.StatusInternalServerError)
			return
		}
	} else {
		pid, err := peer.Decode(peerIDStr)
		if err != nil {
			http.Error(w, wrapError(err), http.StatusBadRequest)
			return
		}
		following, err = g.node.GetFollowing(r.Context(), pid, useCache)
		if errors.Is(err, coreiface.ErrNotFound) {
			http.Error(w, wrapError(err), http.StatusNotFound)
			return
		} else if err != nil {
			http.Error(w, wrapError(err), http.StatusInternalServerError)
			return
		}
	}
	if following == nil {
		following = models.Following{}
	}
	sanitizedJSONResponse(w, following)
}

func (g *Gateway) handlePOSTFollow(w http.ResponseWriter, r *http.Request) {
	peerIDStr := mux.Vars(r)["peerID"]
	pid, err := peer.Decode(peerIDStr)
	if err != nil {
		http.Error(w, wrapError(err), http.StatusBadRequest)
		return
	}
	err = g.node.FollowNode(pid, nil)
	if errors.Is(err, coreiface.ErrBadRequest) {
		http.Error(w, wrapError(err), http.StatusBadRequest)
		return
	} else if err != nil {
		http.Error(w, wrapError(err), http.StatusInternalServerError)
		return
	}
}

func (g *Gateway) handlePOSTUnFollow(w http.ResponseWriter, r *http.Request) {
	peerIDStr := mux.Vars(r)["peerID"]
	pid, err := peer.Decode(peerIDStr)
	if err != nil {
		http.Error(w, wrapError(err), http.StatusBadRequest)
		return
	}
	err = g.node.UnfollowNode(pid, nil)
	if errors.Is(err, coreiface.ErrBadRequest) {
		http.Error(w, wrapError(err), http.StatusBadRequest)
		return
	} else if err != nil {
		http.Error(w, wrapError(err), http.StatusInternalServerError)
		return
	}
}
