package net

import (
	"errors"
	"github.com/yawning/bulb"
)

// TorControlAddress returns address on which the Tor control is currently
// running. If a config address is passed in it will try to connect on that
// address. Otherwise it will try the default addresses for the Tor
// browser and Tor daemon.
func TorControlAddress(cfgAddr string) (string, error) {
	if cfgAddr != "" {
		conn, err := bulb.Dial("tcp4", cfgAddr)
		if err != nil {
			return "", errors.New("tor control unavailable")
		}
		conn.Close()
		return cfgAddr, nil
	}
	conn, err := bulb.Dial("tcp4", "127.0.0.1:9151")
	if err == nil {
		conn.Close()
		return "127.0.0.1:9151", nil
	}
	conn, err = bulb.Dial("tcp4", "127.0.0.1:9051")
	if err == nil {
		conn.Close()
		return "127.0.0.1:9051", nil
	}
	return "", errors.New("tor control unavailable")
}

// Socks5ProxyAddress returns the address of the Tor socks5 proxy. If
// a config address is pass in it will try that address. Otherwise it
// will to connect to the proxy on the default addresses for the Tor
// browser and Tor daemon.
func Socks5ProxyAddress(cfgAddr string) (string, error) {
	if cfgAddr != "" {
		conn, err := bulb.Dial("tcp4", cfgAddr)
		if err != nil {
			return "", errors.New("tor socks5 proxy unavailable")
		}
		conn.Close()
		return cfgAddr, nil
	}
	conn, err := bulb.Dial("tcp4", "127.0.0.1:9150")
	if err == nil {
		conn.Close()
		return "127.0.0.1:9150", nil
	}
	conn, err = bulb.Dial("tcp4", "127.0.0.1:9050")
	if err == nil {
		conn.Close()
		return "127.0.0.1:9050", nil
	}
	return "", errors.New("tor socks5 proxy unavailable")
}
