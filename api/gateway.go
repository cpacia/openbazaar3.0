package api

import (
	"github.com/gorilla/mux"
	"github.com/ipfs/go-ipfs/core/corehttp"
	"github.com/op/go-logging"
	"net"
	"net/http"
)

var log = logging.MustGetLogger("api")

type GatewayConfig struct {
	Listener   net.Listener
	NoCors     bool
	AllowedIPs map[string]bool
	Cookie     string
	Username   string
	Password   string
	UseSSL     bool
	SSLCert    string
	SSLKey     string
	PublicOnly bool
}

// Gateway represents an HTTP API gateway
type Gateway struct {
	listener net.Listener
	node     CoreIface
	handler  http.Handler
	config   *GatewayConfig
}

// NewGateway instantiates a new gateway. We multiplex the ob API along with the
// IPFS gateway API.
func NewGateway(node CoreIface, config *GatewayConfig, options ...corehttp.ServeOption) (*Gateway, error) {
	var (
		g = &Gateway{
			node:     node,
			config:   config,
			listener: config.Listener,
		}
		topMux = http.NewServeMux()
	)

	r := g.newV1Router()

	if !config.NoCors {
		r.Use(mux.CORSMethodMiddleware(r))
	}
	r.Use(g.AuthenticationMiddleware)

	topMux.Handle("/v1/ob/", r)
	topMux.Handle("/v1/wallet/", r)

	var (
		err error
		mux = topMux
	)
	for _, option := range options {
		mux, err = option(node.IPFSNode(), config.Listener, mux)
		if err != nil {
			return nil, err
		}
	}
	g.handler = topMux
	return g, nil
}

// Close shutsdown the Gateway listener.
func (g *Gateway) Close() error {
	return g.listener.Close()
}

// Serve begins listening on the configured address.
func (g *Gateway) Serve() error {
	log.Infof("Gateway/API server listening on %s\n", g.listener.Addr())
	var err error
	if g.config.UseSSL {
		err = http.ListenAndServeTLS(g.listener.Addr().String(), g.config.SSLCert, g.config.SSLKey, g.handler)
	} else {
		err = http.Serve(g.listener, g.handler)
	}
	return err
}

func (g *Gateway) newV1Router() *mux.Router {
	r := mux.NewRouter()

	if !g.config.PublicOnly {
		r.HandleFunc("/v1/wallet/address", g.handleGETAddress).Methods("GET")
		r.HandleFunc("/v1/wallet/address/{coinType}", g.handleGETAddress).Methods("GET")
		r.HandleFunc("/v1/wallet/balance", g.handleGETBalance).Methods("GET")
		r.HandleFunc("/v1/wallet/balance/{coinType}", g.handleGETBalance).Methods("GET")
		r.HandleFunc("/v1/ob/profile", g.handlePOSTProfile).Methods("POST")
		r.HandleFunc("/v1/ob/profile", g.handlePUTProfile).Methods("PUT")
		r.HandleFunc("/v1/ob/follow/{peerID}", g.handlePOSTFollow).Methods("POST")
		r.HandleFunc("/v1/ob/unfollow/{peerID}", g.handlePOSTUnFollow).Methods("POST")
		r.HandleFunc("/v1/ob/chatmessage", g.handlePOSTSendChatMessage).Methods("POST")
		r.HandleFunc("/v1/ob/groupchatmessage", g.handlePOSTSendGroupChatMessage).Methods("POST")
		r.HandleFunc("/v1/ob/typingmessage", g.handlePOSTSendTypingMessage).Methods("POST")
		r.HandleFunc("/v1/ob/grouptypingmessage", g.handlePOSTSendGroupTypingMessage).Methods("POST")
		r.HandleFunc("/v1/ob/markchatasread", g.handlePOSTMarkChatMessageAsRead).Methods("POST")
		r.HandleFunc("/v1/ob/chatconversations", g.handleGETChatConversations).Methods("GET")
		r.HandleFunc("/v1/ob/chatmessages/{peerID}", g.handleGETChatMessages).Methods("GET")
		r.HandleFunc("/v1/ob/groupchatmessages/{orderID}", g.handleGETGroupChatMessages).Methods("GET")
		r.HandleFunc("/v1/ob/chatmessage/{messageID}", g.handleDELETEChatMessages).Methods("DELETE")
		r.HandleFunc("/v1/ob/groupchatmessages/{orderID}", g.handleDELETEGroupChatMessages).Methods("DELETE")
		r.HandleFunc("/v1/ob/chatconversation/{peerID}", g.handleDELETEChatConversation).Methods("DELETE")
	}
	r.HandleFunc("/v1/ob/profile/{peerID}", g.handleGETProfile).Methods("GET")
	r.HandleFunc("/v1/ob/profile", g.handleGETProfile).Methods("GET")
	r.HandleFunc("/v1/ob/fetchprofiles", g.handlePOSTFetchProfiles).Methods("POST")
	r.HandleFunc("/v1/ob/followers/{peerID}", g.handleGETFollowers).Methods("GET")
	r.HandleFunc("/v1/ob/followers", g.handleGETFollowers).Methods("GET")
	r.HandleFunc("/v1/ob/following/{peerID}", g.handleGETFollowing).Methods("GET")
	r.HandleFunc("/v1/ob/following", g.handleGETFollowing).Methods("GET")
	return r
}
