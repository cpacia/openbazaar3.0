package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/cpacia/openbazaar3.0/core/coreiface"
	"github.com/cpacia/openbazaar3.0/models"
	"github.com/gorilla/mux"
	"github.com/ipfs/go-cid"
	peer "github.com/libp2p/go-libp2p-peer"
	"net/http"
	"strconv"
	"time"
)

func (g *Gateway) handleGETImage(w http.ResponseWriter, r *http.Request) {
	imageIDStr := mux.Vars(r)["imageID"]

	id, cerr := cid.Decode(imageIDStr)
	if cerr != nil {
		http.Error(w, wrapError(fmt.Errorf("invalid image id: %s", cerr.Error())), http.StatusBadRequest)
		return
	}

	reader, err := g.node.GetImage(r.Context(), id)
	if errors.Is(err, coreiface.ErrNotFound) {
		http.Error(w, wrapError(err), http.StatusNotFound)
		return
	} else if err != nil {
		http.Error(w, wrapError(err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Cache-Control", "public, max-age=29030400, immutable")
	w.Header().Del("Content-Type")
	http.ServeContent(w, r, id.String(), time.Now(), reader)
}

func (g *Gateway) handleGETAvatar(w http.ResponseWriter, r *http.Request) {
	peerIDStr := mux.Vars(r)["peerID"]
	sizeStr := mux.Vars(r)["size"]

	pid, cerr := peer.IDB58Decode(peerIDStr)
	if cerr != nil {
		http.Error(w, wrapError(fmt.Errorf("invalid peer id: %s", cerr.Error())), http.StatusBadRequest)
		return
	}

	useCache, _ := strconv.ParseBool(r.URL.Query().Get("usecache"))

	reader, err := g.node.GetAvatar(r.Context(), pid, models.ImageSize(sizeStr), useCache)
	if errors.Is(err, coreiface.ErrNotFound) {
		http.Error(w, wrapError(err), http.StatusNotFound)
		return
	} else if err != nil {
		http.Error(w, wrapError(err), http.StatusInternalServerError)
		return
	}

	http.ServeContent(w, r, peerIDStr, time.Now(), reader)
}

func (g *Gateway) handleGETHeader(w http.ResponseWriter, r *http.Request) {
	peerIDStr := mux.Vars(r)["peerID"]
	sizeStr := mux.Vars(r)["size"]

	pid, cerr := peer.IDB58Decode(peerIDStr)
	if cerr != nil {
		http.Error(w, wrapError(fmt.Errorf("invalid peer id: %s", cerr.Error())), http.StatusBadRequest)
		return
	}

	useCache, _ := strconv.ParseBool(r.URL.Query().Get("usecache"))

	reader, err := g.node.GetHeader(r.Context(), pid, models.ImageSize(sizeStr), useCache)
	if errors.Is(err, coreiface.ErrNotFound) {
		http.Error(w, wrapError(err), http.StatusNotFound)
		return
	} else if err != nil {
		http.Error(w, wrapError(err), http.StatusInternalServerError)
		return
	}

	http.ServeContent(w, r, peerIDStr, time.Now(), reader)
}

func (g *Gateway) handlePOSTAvatar(w http.ResponseWriter, r *http.Request) {
	type ImgData struct {
		Avatar string `json:"avatar"`
	}
	decoder := json.NewDecoder(r.Body)
	data := new(ImgData)
	if err := decoder.Decode(&data); err != nil {
		http.Error(w, wrapError(err), http.StatusBadRequest)
		return
	}

	hashes, err := g.node.SetAvatarImage(data.Avatar, nil)
	if errors.Is(err, coreiface.ErrBadRequest) {
		http.Error(w, wrapError(err), http.StatusBadRequest)
		return
	} else if err != nil {
		http.Error(w, wrapError(err), http.StatusInternalServerError)
		return
	}

	sanitizedJSONResponse(w, hashes)
}

func (g *Gateway) handlePOSTHeader(w http.ResponseWriter, r *http.Request) {
	type ImgData struct {
		Header string `json:"header"`
	}
	decoder := json.NewDecoder(r.Body)
	data := new(ImgData)
	if err := decoder.Decode(&data); err != nil {
		http.Error(w, wrapError(err), http.StatusBadRequest)
		return
	}

	hashes, err := g.node.SetHeaderImage(data.Header, nil)
	if errors.Is(err, coreiface.ErrBadRequest) {
		http.Error(w, wrapError(err), http.StatusBadRequest)
		return
	} else if err != nil {
		http.Error(w, wrapError(err), http.StatusInternalServerError)
		return
	}

	sanitizedJSONResponse(w, hashes)
}

func (g *Gateway) handlePOSTProductImage(w http.ResponseWriter, r *http.Request) {
	type ImgData struct {
		Image    string `json:"image"`
		Filename string `json:"filename"`
	}
	decoder := json.NewDecoder(r.Body)
	data := new(ImgData)
	if err := decoder.Decode(&data); err != nil {
		http.Error(w, wrapError(err), http.StatusBadRequest)
		return
	}

	hashes, err := g.node.SetProductImage(data.Image, data.Filename)
	if errors.Is(err, coreiface.ErrBadRequest) {
		http.Error(w, wrapError(err), http.StatusBadRequest)
		return
	} else if err != nil {
		http.Error(w, wrapError(err), http.StatusInternalServerError)
		return
	}

	sanitizedJSONResponse(w, hashes)
}
