// +build notor

package net

import (
	"context"
	"errors"
	"crypto/ed25519"
	"github.com/libp2p/go-libp2p"
	"golang.org/x/net/proxy"
)

// SetupTor is used by the `notor` build tag. It will not build the tor C library. We use this primarily for running
// CI tests.
func SetupTor(ctx context.Context, key ed25519.PrivateKey, dataDir string, dualstackMode bool) (string, proxy.Dialer, libp2p.Option, func() error, error) {
	return "", nil, nil, nil, errors.New("openbazaar was built with the notor build tag")
}
