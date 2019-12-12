package net

import (
	"net"
	"testing"
)

func TestTorControlAddress(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:9051")
	if err != nil {
		t.Fatal(err)
	}

	addr, err := TorControlAddress("")
	if err != nil {
		t.Fatal(err)
	}
	listener.Close()
	if addr != "127.0.0.1:9051" {
		t.Errorf("Expected address %s got %s", "127.0.0.1:9051", addr)
	}

	listener, err = net.Listen("tcp", "127.0.0.1:9151")
	if err != nil {
		t.Fatal(err)
	}

	addr, err = TorControlAddress("")
	if err != nil {
		t.Fatal(err)
	}
	listener.Close()
	if addr != "127.0.0.1:9151" {
		t.Errorf("Expected address %s got %s", "127.0.0.1:9151", addr)
	}

	listener, err = net.Listen("tcp", "127.0.0.1:9000")
	if err != nil {
		t.Fatal(err)
	}

	addr, err = TorControlAddress("127.0.0.1:9000")
	if err != nil {
		t.Fatal(err)
	}
	listener.Close()
	if addr != "127.0.0.1:9000" {
		t.Errorf("Expected address %s got %s", "127.0.0.1:9000", addr)
	}
}

func TestSocks5ProxyAddress(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:9050")
	if err != nil {
		t.Fatal(err)
	}

	addr, err := Socks5ProxyAddress("")
	if err != nil {
		t.Fatal(err)
	}
	listener.Close()
	if addr != "127.0.0.1:9050" {
		t.Errorf("Expected address %s got %s", "127.0.0.1:9050", addr)
	}

	listener, err = net.Listen("tcp", "127.0.0.1:9150")
	if err != nil {
		t.Fatal(err)
	}

	addr, err = Socks5ProxyAddress("")
	if err != nil {
		t.Fatal(err)
	}
	listener.Close()
	if addr != "127.0.0.1:9150" {
		t.Errorf("Expected address %s got %s", "127.0.0.1:9150", addr)
	}

	listener, err = net.Listen("tcp", "127.0.0.1:9000")
	if err != nil {
		t.Fatal(err)
	}

	addr, err = Socks5ProxyAddress("127.0.0.1:9000")
	if err != nil {
		t.Fatal(err)
	}
	listener.Close()
	if addr != "127.0.0.1:9000" {
		t.Errorf("Expected address %s got %s", "127.0.0.1:9000", addr)
	}
}
