package api

import (
	"github.com/gorilla/mux"
	"github.com/ipfs/go-ipfs/core/corehttp"
	"github.com/op/go-logging"
	"net"
	"net/http"
)

const AuthCookieName = "OpenBazaar_Auth_Cookie"

var log = logging.MustGetLogger("api")

type GatewayConfig struct {
	Listener   net.Listener
	Cors       string
	AllowedIPs []string
	Cookie     string
	Username   string
	Password   string
	UseSSL     bool
	SSLCert    string
	SSLKey     string
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

	topMux.Handle("v1/ob/", r)
	topMux.Handle("v1/wallet/", r)

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
	// TODO: register handlers here
	return r
}
