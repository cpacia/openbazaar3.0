package api

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"github.com/cpacia/openbazaar3.0/core/coreiface"
	"github.com/cpacia/openbazaar3.0/models"
	"github.com/gorilla/mux"
	peer "github.com/libp2p/go-libp2p-peer"
	"net/http"
	"strconv"
	"sync"
)

func (g *Gateway) handleGETProfile(w http.ResponseWriter, r *http.Request) {
	peerIDStr := mux.Vars(r)["peerID"]

	useCache, _ := strconv.ParseBool(r.URL.Query().Get("usecache"))

	var (
		profile *models.Profile
		err     error
	)
	if peerIDStr == "" || peerIDStr == g.node.Identity().Pretty() {
		profile, err = g.node.GetMyProfile()
		if errors.Is(err, coreiface.ErrNotFound) {
			http.Error(w, wrapError(err), http.StatusNotFound)
			return
		} else if err != nil {
			http.Error(w, wrapError(err), http.StatusInternalServerError)
			return
		}
	} else {
		pid, err := peer.IDB58Decode(peerIDStr)
		if err != nil {
			http.Error(w, wrapError(err), http.StatusBadRequest)
			return
		}
		profile, err = g.node.GetProfile(r.Context(), pid, useCache)
		if errors.Is(err, coreiface.ErrNotFound) {
			http.Error(w, wrapError(err), http.StatusNotFound)
			return
		} else if err != nil {
			http.Error(w, wrapError(err), http.StatusInternalServerError)
			return
		}
	}
	sanitizedJSONResponse(w, profile)
}

func (g *Gateway) handlePOSTProfile(w http.ResponseWriter, r *http.Request) {
	if _, err := g.node.GetMyProfile(); !errors.Is(err, coreiface.ErrNotFound) {
		http.Error(w, wrapError(errors.New("profile exists. use PUT to update.")), http.StatusConflict)
		return
	}

	var profile models.Profile
	if err := json.NewDecoder(r.Body).Decode(&profile); err != nil {
		http.Error(w, wrapError(err), http.StatusBadRequest)
		return
	}

	if err := g.node.SetProfile(&profile, nil); err != nil {
		if errors.Is(err, coreiface.ErrBadRequest) {
			http.Error(w, wrapError(err), http.StatusBadRequest)
			return
		} else if err != nil {
			http.Error(w, wrapError(err), http.StatusInternalServerError)
			return
		}
	}
	sanitizedJSONResponse(w, struct{}{})
}

func (g *Gateway) handlePUTProfile(w http.ResponseWriter, r *http.Request) {
	if _, err := g.node.GetMyProfile(); errors.Is(err, coreiface.ErrNotFound) {
		http.Error(w, "profile does not exists. use POST to create.", http.StatusConflict)
		return
	}

	var profile models.Profile
	if err := json.NewDecoder(r.Body).Decode(&profile); err != nil {
		http.Error(w, wrapError(err), http.StatusBadRequest)
		return
	}

	if err := g.node.SetProfile(&profile, nil); err != nil {
		if errors.Is(err, coreiface.ErrBadRequest) {
			http.Error(w, wrapError(err), http.StatusBadRequest)
			return
		} else if err != nil {
			http.Error(w, wrapError(err), http.StatusInternalServerError)
			return
		}
	}
	sanitizedJSONResponse(w, struct{}{})
}

func (g *Gateway) handlePOSTFetchProfiles(w http.ResponseWriter, r *http.Request) {
	useCache, _ := strconv.ParseBool(r.URL.Query().Get("usecache"))
	async, _ := strconv.ParseBool(r.URL.Query().Get("async"))

	var peerIDs []string
	if err := json.NewDecoder(r.Body).Decode(&peerIDs); err != nil {
		http.Error(w, wrapError(err), http.StatusBadRequest)
		return
	}

	type profileWithAsyncID struct {
		ID      string         `json:"id"`
		Profile models.Profile `json:"profile"`
	}

	type profileError struct {
		ID     string `json:"id"`
		PeerID string `json:"peerID"`
		Error  string `json:"error"`
	}

	var (
		profiles     = make([]models.Profile, 0, len(peerIDs))
		responseChan = make(chan interface{}, 8)
		wg           sync.WaitGroup
	)

	wg.Add(len(peerIDs))
	go func() {
		for _, peerIDStr := range peerIDs {
			pid, err := peer.IDB58Decode(peerIDStr)
			if err != nil {
				responseChan <- profileError{
					PeerID: peerIDStr,
					Error:  err.Error(),
				}
				wg.Done()
				continue
			}
			go func(p peer.ID) {
				defer wg.Done()
				profile, err := g.node.GetProfile(r.Context(), p, useCache)
				if err != nil {
					responseChan <- profileError{
						PeerID: p.Pretty(),
						Error:  err.Error(),
					}
					return
				}
				responseChan <- profileWithAsyncID{
					Profile: *profile,
				}
			}(pid)
		}
		wg.Wait()
		close(responseChan)
	}()

	if !async {
		for i := range responseChan {
			switch p := i.(type) {
			case profileWithAsyncID:
				profiles = append(profiles, p.Profile)
			}
		}
		sanitizedJSONResponse(w, profiles)
	} else {
		asyncID := r.URL.Query().Get("asyncID")
		if asyncID == "" {
			r := make([]byte, 20)
			rand.Read(r)
			asyncID = hex.EncodeToString(r)
		}
		w.WriteHeader(http.StatusAccepted)
		sanitizedJSONResponse(w, struct {
			ID string `json:"id"`
		}{ID: asyncID})

		go func() {
			for i := range responseChan {
				switch p := i.(type) {
				case profileWithAsyncID:
					p.ID = asyncID
					g.NotifyWebsockets(p)
				case profileError:
					p.ID = asyncID
					g.NotifyWebsockets(p)
				}
			}
		}()
	}
}
