package core

import (
	"bytes"
	"context"
	"github.com/cpacia/openbazaar3.0/database"
	"github.com/cpacia/openbazaar3.0/models"
	"github.com/cpacia/openbazaar3.0/repo"
	"github.com/gogo/protobuf/proto"
	files "github.com/ipfs/go-ipfs-files"
	"github.com/ipfs/go-ipfs/core/coreapi"
	ipnspb "github.com/ipfs/go-ipns/pb"
	fpath "github.com/ipfs/go-path"
	iface "github.com/ipfs/interface-go-ipfs-core"
	"github.com/ipfs/interface-go-ipfs-core/options"
	"github.com/ipfs/interface-go-ipfs-core/path"
	peer "github.com/libp2p/go-libp2p-peer"
	"io/ioutil"
	"os"
	gpath "path"
	"testing"
	"time"
)

func Test_ipfsCat(t *testing.T) {
	network, err := NewMocknet(2)
	if err != nil {
		t.Fatal(err)
	}

	defer network.TearDown()

	api, err := coreapi.NewCoreAPI(network.Nodes()[0].ipfsNode)
	if err != nil {
		t.Fatal(err)
	}

	var (
		testFile     = []byte("test")
		testFilePath = gpath.Join(network.Nodes()[0].repo.DataDir(), "test.bin")
	)

	if err := ioutil.WriteFile(testFilePath, testFile, os.ModePerm); err != nil {
		t.Fatal(err)
	}

	stat, err := os.Lstat(testFilePath)
	if err != nil {
		t.Fatal(err)
	}

	f, err := files.NewSerialFile(testFilePath, false, stat)
	if err != nil {
		t.Fatal(err)
	}

	opts := []options.UnixfsAddOption{
		options.Unixfs.Pin(true),
	}
	pth, err := api.Unixfs().Add(context.Background(), f, opts...)
	if err != nil {
		t.Fatal(err)
	}

	ret, err := network.Nodes()[1].cat(context.Background(), pth)
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(ret, testFile) {
		t.Errorf("Returned wrong file. Expected %s, got %s", string(testFile), string(ret))
	}
}

func Test_ipfsPin(t *testing.T) {
	network, err := NewMocknet(2)
	if err != nil {
		t.Fatal(err)
	}

	defer network.TearDown()

	api, err := coreapi.NewCoreAPI(network.Nodes()[0].ipfsNode)
	if err != nil {
		t.Fatal(err)
	}

	var (
		testFile     = []byte("test")
		testFilePath = gpath.Join(network.Nodes()[0].repo.DataDir(), "test.bin")
	)

	if err := ioutil.WriteFile(testFilePath, testFile, os.ModePerm); err != nil {
		t.Fatal(err)
	}

	stat, err := os.Lstat(testFilePath)
	if err != nil {
		t.Fatal(err)
	}

	f, err := files.NewSerialFile(testFilePath, false, stat)
	if err != nil {
		t.Fatal(err)
	}

	opts := []options.UnixfsAddOption{
		options.Unixfs.Pin(true),
	}
	pth, err := api.Unixfs().Add(context.Background(), f, opts...)
	if err != nil {
		t.Fatal(err)
	}

	err = network.Nodes()[1].pin(context.Background(), pth)
	if err != nil {
		t.Fatal(err)
	}

	has, err := network.Nodes()[1].ipfsNode.Blocks.Blockstore().Has(pth.Cid())
	if err != nil {
		t.Fatal(err)
	}
	if !has {
		t.Error("Cid not stored in node")
	}
}

func Test_ipfsResolve(t *testing.T) {
	network, err := NewMocknet(2)
	if err != nil {
		t.Fatal(err)
	}

	defer network.TearDown()

	pth := fpath.FromString("/ipfs/QmfQkD8pBSBCBxWEwFSu4XaDVSWK6bjnNuaWZjMyQbyDub")
	if err := network.Nodes()[0].ipfsNode.Namesys.Publish(context.Background(), network.Nodes()[0].ipfsNode.PrivateKey, pth); err != nil {
		t.Fatal(err)
	}

	ret, err := network.Nodes()[1].resolve(context.Background(), network.Nodes()[0].Identity(), false)
	if err != nil {
		t.Fatal(err)
	}

	if ret.String() != pth.String() {
		t.Errorf("Returned incorrect value. Expected %s, got %s", pth.String(), ret.String())
	}

	// Disconnect node 0 and try again with cache.
	network.Nodes()[0].ipfsNode.PeerHost.Close()

	ret, err = network.Nodes()[1].resolve(context.Background(), network.Nodes()[0].Identity(), true)
	if err != nil {
		t.Fatal(err)
	}

	if ret.String() != pth.String() {
		t.Errorf("Returned incorrect value. Expected %s, got %s", pth.String(), ret.String())
	}
}

func Test_ipfsCache(t *testing.T) {
	r, err := repo.MockRepo()
	if err != nil {
		t.Fatal()
	}

	defer r.DestroyRepo()

	p, err := peer.IDB58Decode("QmfQkD8pBSBCBxWEwFSu4XaDVSWK6bjnNuaWZjMyQbyDub")
	if err != nil {
		t.Fatal(err)
	}
	pth := path.New("/ipfs/Qmd9hFFuueFrSR7YwUuAfirXXJ7ANZAMc5sx4HFxn7mPkc")
	err = r.DB().Update(func(tx database.Tx) error {
		return putToDatastoreCache(tx, p, pth)
	})
	if err != nil {
		t.Fatal(err)
	}

	var ret path.Path
	err = r.DB().View(func(tx database.Tx) error {
		ret, err = getFromDatastore(tx, p)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}

	if ret.String() != pth.String() {
		t.Errorf("Database returned incorrect cached value. Expected %s, got %s", pth.String(), ret.String())
	}
}

func Test_ipfsFetchGraph(t *testing.T) {
	mocknet, err := NewMocknet(2)
	if err != nil {
		t.Fatal(err)
	}

	defer mocknet.TearDown()

	done := make(chan struct{})
	if err := mocknet.Nodes()[0].SetProfile(&models.Profile{Name: "Ron Paul"}, done); err != nil {
		t.Fatal(err)
	}
	select {
	case <-done:
	case <-time.After(time.Second * 10):
		t.Fatal("Timeout waiting on channel")
	}

	graph, err := mocknet.Nodes()[0].fetchGraph(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	if len(graph) != 4 && len(graph) != 5 {
		t.Errorf("Expected 4 or 5 elements in the graph. Got %d", len(graph))
	}
}

func Test_ipfsCid(t *testing.T) {
	node, err := MockNode()
	if err != nil {
		t.Fatal(err)
	}
	defer node.DestroyNode()
	cid, err := node.cid([]byte("hola"))
	if err != nil {
		t.Fatal(err)
	}
	expected := "QmZNhQu5jyvfNtZrp7Xvu7iqhuF4occHLcxZKj7S3F8a3D"
	if cid.String() != expected {
		t.Errorf("Returned incorrect cid. Expected %s, got %s", expected, cid.String())
	}
}

func Test_publishIPNSRecordToPubsub(t *testing.T){
	mocknet, err := NewMocknet(2)
	if err != nil {
		t.Fatal(err)
	}

	defer mocknet.TearDown()

	api, err := coreapi.NewCoreAPI(mocknet.Nodes()[0].ipfsNode)
	if err != nil {
		t.Fatal(err)
	}

	sub, err := api.PubSub().Subscribe(context.Background(), ipnsPubsubTopic)
	if err != nil {
		t.Fatal(err)
	}

	done := make(chan struct{})
	if err := mocknet.Nodes()[1].SetProfile(&models.Profile{Name: "Ron Paul"}, done); err != nil {
		t.Fatal(err)
	}

	select {
	case <-done:
	case <-time.After(time.Second*10):
		t.Fatal("timed out")
	}

	ch := make(chan iface.PubSubMessage)
	go func() {
		message, err := sub.Next(context.Background())
		if err != nil {
			t.Fatal(err)
		}
		ch <- message
	}()

	var message iface.PubSubMessage
	select {
	case message = <-ch:
	case <-time.After(time.Second*10):
		t.Fatal("timed out")
	}

	if message.From() != mocknet.Nodes()[1].Identity() {
		t.Errorf("Expected identity %s, got %s", mocknet.Nodes()[1].Identity(), message.From())
	}

	rec := new(ipnspb.IpnsEntry)
	if err := proto.Unmarshal(message.Data(), rec); err != nil {
		t.Fatal(err)
	}
}