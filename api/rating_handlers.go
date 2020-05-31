package api

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/OpenBazaar/jsonpb"
	"github.com/cpacia/openbazaar3.0/core/coreiface"
	"github.com/cpacia/openbazaar3.0/models"
	"github.com/gorilla/mux"
	"github.com/ipfs/go-cid"
	"github.com/libp2p/go-libp2p-core/peer"
	"net/http"
	"strconv"
	"sync"
)

func (g *Gateway) handleGETRatingIndex(w http.ResponseWriter, r *http.Request) {
	peerIDStr := mux.Vars(r)["peerID"]
	var (
		index models.RatingIndex
		err   error
	)
	if peerIDStr == "" || peerIDStr == g.node.Identity().Pretty() {
		index, err = g.node.GetMyRatings()
	} else {
		pid, perr := peer.Decode(peerIDStr)
		if perr != nil {
			http.Error(w, wrapError(fmt.Errorf("invalid peer id: %s", perr.Error())), http.StatusBadRequest)
			return
		}
		useCache, _ := strconv.ParseBool(r.URL.Query().Get("usecache"))
		index, err = g.node.GetRatings(r.Context(), pid, useCache)
	}

	if errors.Is(err, coreiface.ErrNotFound) {
		http.Error(w, wrapError(err), http.StatusNotFound)
		return
	} else if err != nil {
		http.Error(w, wrapError(err), http.StatusInternalServerError)
		return
	}

	sanitizedJSONResponse(w, index)
}

func (g *Gateway) handleGETRating(w http.ResponseWriter, r *http.Request) {
	ratingIDStr := mux.Vars(r)["ratingID"]

	id, cerr := cid.Decode(ratingIDStr)
	if cerr != nil {
		http.Error(w, wrapError(fmt.Errorf("invalid rating id: %s", cerr.Error())), http.StatusBadRequest)
		return
	}
	rating, err := g.node.GetRating(r.Context(), id)

	if errors.Is(err, coreiface.ErrNotFound) {
		http.Error(w, wrapError(err), http.StatusNotFound)
		return
	} else if err != nil {
		http.Error(w, wrapError(err), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Cache-Control", "public, max-age=29030400, immutable")

	sanitizedProtobufResponse(w, rating)
}

func (g *Gateway) handleGETRatings(w http.ResponseWriter, r *http.Request) {
	peerIDStr := mux.Vars(r)["peerID"]
	slug := mux.Vars(r)["slug"]
	useCache, _ := strconv.ParseBool(r.URL.Query().Get("usecache"))

	pid, err := peer.Decode(peerIDStr)
	if err != nil {
		http.Error(w, wrapError(fmt.Errorf("invalid peer id: %s", err.Error())), http.StatusBadRequest)
		return
	}
	var index models.RatingIndex
	if pid == g.node.Identity() {
		index, err = g.node.GetMyRatings()
	} else {
		index, err = g.node.GetRatings(r.Context(), pid, useCache)
	}
	if errors.Is(err, coreiface.ErrNotFound) {
		sanitizedJSONResponse(w, []string{})
		return
	} else if err != nil {
		http.Error(w, wrapError(err), http.StatusInternalServerError)
		return
	}
	ratings, err := index.GetRatingCIDs(slug)
	if err != nil {
		http.Error(w, wrapError(err), http.StatusInternalServerError)
		return
	}
	ret := make([]string, len(ratings))
	for i, r := range ratings {
		ret[i] = r.String()
	}

	sanitizedJSONResponse(w, ret)
}

func (g *Gateway) handlePOSTFetchRatings(w http.ResponseWriter, r *http.Request) {
	async, _ := strconv.ParseBool(r.URL.Query().Get("async"))

	var ratingIDs []string
	if err := json.NewDecoder(r.Body).Decode(&ratingIDs); err != nil {
		http.Error(w, wrapError(err), http.StatusBadRequest)
		return
	}

	type ratingWithAsyncID struct {
		ID     string          `json:"id"`
		Rating json.RawMessage `json:"rating"`
	}

	type ratingError struct {
		ID       string `json:"id"`
		RatingID string `json:"ratingID"`
		Error    string `json:"error"`
	}

	var (
		ratings      = make([]json.RawMessage, 0, len(ratingIDs))
		responseChan = make(chan interface{}, 8)
		wg           sync.WaitGroup
		marshaler    = jsonpb.Marshaler{Indent: "    "}
	)

	wg.Add(len(ratingIDs))
	go func() {
		for _, ratingID := range ratingIDs {
			rid, err := cid.Decode(ratingID)
			if err != nil {
				responseChan <- ratingError{
					RatingID: ratingID,
					Error:    err.Error(),
				}
				wg.Done()
				continue
			}
			go func(id cid.Cid) {
				defer wg.Done()
				rating, err := g.node.GetRating(r.Context(), id)
				if err != nil {
					responseChan <- ratingError{
						RatingID: id.String(),
						Error:    err.Error(),
					}
					return
				}
				ratingJSON, err := marshaler.MarshalToString(rating)
				if err != nil {
					responseChan <- ratingError{
						RatingID: id.String(),
						Error:    err.Error(),
					}
					return
				}
				responseChan <- ratingWithAsyncID{
					Rating: []byte(ratingJSON),
				}
			}(rid)
		}
		wg.Wait()
		close(responseChan)
	}()

	if !async {
		for i := range responseChan {
			switch p := i.(type) {
			case ratingWithAsyncID:
				ratings = append(ratings, p.Rating)
			}
		}
		sanitizedJSONResponse(w, ratings)
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
				case ratingWithAsyncID:
					p.ID = asyncID
					g.NotifyWebsockets(p)
				case ratingError:
					p.ID = asyncID
					g.NotifyWebsockets(p)
				}
			}
		}()
	}
}
