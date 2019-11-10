package api

import (
	"bytes"
	"fmt"
	peer "github.com/libp2p/go-libp2p-peer"
	"github.com/libp2p/go-testutil"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
)

type apiTests []apiTest

type apiTest struct {
	name             string
	path             string
	method           string
	body             []byte
	setNodeMethods   func(n *mockNode)
	statusCode       int
	expectedResponse func() ([]byte, error)
}

func runAPITests(t *testing.T, tests apiTests) {
	peerID, err := testutil.RandPeerID()
	if err != nil {
		t.Fatal(err)
	}
	node := &mockNode{
		identityFunc: func() peer.ID {
			return peerID
		},
	}
	gateway := &Gateway{
		node:   node,
		config: &GatewayConfig{},
	}

	ts := httptest.NewServer(gateway.newV1Router())
	defer ts.Close()

	for _, test := range tests {
		test.setNodeMethods(node)
		req, err := http.NewRequest(test.method, fmt.Sprintf("%s%s", ts.URL, test.path), bytes.NewReader(test.body))
		if err != nil {
			t.Fatal(err)
		}
		res, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		if res.StatusCode != test.statusCode {
			t.Errorf("%s. Expected status code %d, got %d", test.name, test.statusCode, res.StatusCode)
			continue
		}
		response, err := ioutil.ReadAll(res.Body)
		res.Body.Close()
		if err != nil {
			log.Fatal(err)
		}
		expected, err := test.expectedResponse()
		if err != nil {
			log.Fatal(err)
		}
		if !bytes.Equal(response, expected) {
			t.Errorf("%s: Expected response %s, got %s", test.name, string(expected), string(response))
			continue
		}
	}
}
