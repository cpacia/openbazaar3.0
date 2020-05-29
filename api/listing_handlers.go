package api

import (
	"errors"
	"fmt"
	"github.com/OpenBazaar/jsonpb"
	"github.com/cpacia/openbazaar3.0/core/coreiface"
	"github.com/cpacia/openbazaar3.0/models"
	"github.com/cpacia/openbazaar3.0/orders/pb"
	"github.com/gorilla/mux"
	"github.com/ipfs/go-cid"
	peer "github.com/libp2p/go-libp2p-core/peer"
	"net/http"
	"strconv"
)

func (g *Gateway) handleGETListing(w http.ResponseWriter, r *http.Request) {
	listingIDStr := mux.Vars(r)["listingID"]
	peerIDStr := mux.Vars(r)["peerID"]
	slug := mux.Vars(r)["slug"]

	var (
		listing *pb.SignedListing
		err     error
	)
	if listingIDStr != "" { // Query by CID
		id, cerr := cid.Decode(listingIDStr)
		if cerr != nil {
			http.Error(w, wrapError(fmt.Errorf("invalid listing id: %s", cerr.Error())), http.StatusBadRequest)
			return
		}
		listing, err = g.node.GetListingByCID(r.Context(), id)
		w.Header().Set("Cache-Control", "public, max-age=29030400, immutable")
	} else if peerIDStr != "" && slug != "" { // Query by peerID/slug
		pid, perr := peer.Decode(peerIDStr)
		if perr != nil {
			http.Error(w, wrapError(fmt.Errorf("invalid peer id: %s", perr.Error())), http.StatusBadRequest)
			return
		}
		useCache, _ := strconv.ParseBool(r.URL.Query().Get("usecache"))
		listing, err = g.node.GetListingBySlug(r.Context(), pid, slug, useCache)
	}

	if errors.Is(err, coreiface.ErrNotFound) {
		http.Error(w, wrapError(err), http.StatusNotFound)
		return
	} else if err != nil {
		http.Error(w, wrapError(err), http.StatusInternalServerError)
		return
	}

	sanitizedProtobufResponse(w, listing)
}

func (g *Gateway) handleGETMyListing(w http.ResponseWriter, r *http.Request) {
	slugOrCid := mux.Vars(r)["slugOrCID"]

	var (
		slug    string
		listing *pb.SignedListing
		err     error
	)
	cid, cerr := cid.Decode(slugOrCid)
	if cerr != nil {
		slug = slugOrCid
	}

	if slug != "" {
		listing, err = g.node.GetMyListingBySlug(slug)
	} else {
		listing, err = g.node.GetMyListingByCID(cid)
		w.Header().Set("Cache-Control", "public, max-age=29030400, immutable")
	}

	if errors.Is(err, coreiface.ErrNotFound) {
		http.Error(w, wrapError(err), http.StatusNotFound)
		return
	} else if err != nil {
		http.Error(w, wrapError(err), http.StatusInternalServerError)
		return
	}

	sanitizedProtobufResponse(w, listing)
}

func (g *Gateway) handleGETListingIndex(w http.ResponseWriter, r *http.Request) {
	peerIDStr := mux.Vars(r)["peerID"]
	var (
		index models.ListingIndex
		err   error
	)
	if peerIDStr == "" || peerIDStr == g.node.Identity().Pretty() {
		index, err = g.node.GetMyListings()
	} else {
		pid, perr := peer.Decode(peerIDStr)
		if perr != nil {
			http.Error(w, wrapError(fmt.Errorf("invalid peer id: %s", perr.Error())), http.StatusBadRequest)
			return
		}
		useCache, _ := strconv.ParseBool(r.URL.Query().Get("usecache"))
		index, err = g.node.GetListings(r.Context(), pid, useCache)
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

func (g *Gateway) handlePOSTListing(w http.ResponseWriter, r *http.Request) {
	listing := new(pb.Listing)

	if err := jsonpb.Unmarshal(r.Body, listing); err != nil {
		http.Error(w, wrapError(fmt.Errorf("error unmarshaling listing: %s", err.Error())), http.StatusBadRequest)
		return
	}

	if _, err := g.node.GetMyListingBySlug(listing.Slug); !errors.Is(err, coreiface.ErrNotFound) {
		http.Error(w, wrapError(errors.New("listing exists. use PUT to update.")), http.StatusConflict)
		return
	}

	if err := g.node.SaveListing(listing, nil); err != nil {
		if errors.Is(err, coreiface.ErrBadRequest) {
			http.Error(w, wrapError(err), http.StatusBadRequest)
			return
		} else if err != nil {
			http.Error(w, wrapError(err), http.StatusInternalServerError)
			return
		}
	}
}

func (g *Gateway) handlePUTListing(w http.ResponseWriter, r *http.Request) {
	listing := new(pb.Listing)

	if err := jsonpb.Unmarshal(r.Body, listing); err != nil {
		http.Error(w, wrapError(fmt.Errorf("error unmarshaling listing: %s", err.Error())), http.StatusBadRequest)
	}

	if _, err := g.node.GetMyListingBySlug(listing.Slug); errors.Is(err, coreiface.ErrNotFound) {
		http.Error(w, wrapError(errors.New("listing does not exist. use POST to create.")), http.StatusConflict)
		return
	}

	if err := g.node.SaveListing(listing, nil); err != nil {
		if errors.Is(err, coreiface.ErrBadRequest) {
			http.Error(w, wrapError(err), http.StatusBadRequest)
			return
		} else if err != nil {
			http.Error(w, wrapError(err), http.StatusInternalServerError)
			return
		}
	}
}

func (g *Gateway) handleDELETEListing(w http.ResponseWriter, r *http.Request) {
	slug := mux.Vars(r)["slug"]

	if err := g.node.DeleteListing(slug, nil); err != nil {
		if errors.Is(err, coreiface.ErrNotFound) {
			http.Error(w, wrapError(err), http.StatusNotFound)
			return
		} else if err != nil {
			http.Error(w, wrapError(err), http.StatusInternalServerError)
			return
		}
	}
}
