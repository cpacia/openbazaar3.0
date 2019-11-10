package api

import (
	"encoding/json"
	"github.com/cpacia/openbazaar3.0/models"
	"github.com/gorilla/mux"
	peer "github.com/libp2p/go-libp2p-peer"
	"net/http"
	"os"
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
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
	} else {
		pid, err := peer.IDB58Decode(peerIDStr)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		profile, err = g.node.GetProfile(r.Context(), pid, useCache)
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
	}
	sanitizedJSONResponse(w, profile)
}

func (g *Gateway) handlePOSTProfile(w http.ResponseWriter, r *http.Request) {
	if _, err := g.node.GetMyProfile(); !os.IsNotExist(err) {
		http.Error(w, "profile exists. use PUT to update.", http.StatusConflict)
		return
	}

	var profile models.Profile
	if err := json.NewDecoder(r.Body).Decode(&profile); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := g.node.SetProfile(&profile, nil); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (g *Gateway) handlePUTProfile(w http.ResponseWriter, r *http.Request) {
	if _, err := g.node.GetMyProfile(); os.IsNotExist(err) {
		http.Error(w, "profile does not exists. use POST to create.", http.StatusConflict)
		return
	}

	var profile models.Profile
	if err := json.NewDecoder(r.Body).Decode(&profile); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := g.node.SetProfile(&profile, nil); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (g *Gateway) handlePOSTFetchProfiles(w http.ResponseWriter, r *http.Request) {
	useCache, _ := strconv.ParseBool(r.URL.Query().Get("usecache"))

	var peerIDs []string
	if err := json.NewDecoder(r.Body).Decode(&peerIDs); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	var (
		profiles     = make([]models.Profile, 0, len(peerIDs))
		responseChan = make(chan models.Profile, 8)
		wg           sync.WaitGroup
	)
	wg.Add(len(peerIDs))
	go func() {
		for _, peerIDStr := range peerIDs {
			pid, err := peer.IDB58Decode(peerIDStr)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			go func(p peer.ID) {
				defer wg.Done()
				profile, err := g.node.GetProfile(r.Context(), p, useCache)
				if err != nil {
					return
				}
				responseChan <- *profile
			}(pid)
		}
		wg.Wait()
		close(responseChan)
	}()
	for profile := range responseChan {
		profiles = append(profiles, profile)
	}

	// TODO: handle async response
	sanitizedJSONResponse(w, profiles)
}
