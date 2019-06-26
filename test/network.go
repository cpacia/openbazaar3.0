package test

import (
	"context"
	"errors"
	"fmt"
	"github.com/cpacia/openbazaar3.0/core"
	"github.com/cpacia/openbazaar3.0/repo"
	"math/rand"
	"os"
	"path"
	"sync"
	"time"
)

const (
	tempDirectoryName = "openbazaar-test"
)

var (
	testSetupOnce sync.Once
	portMap       map[int]bool
	portMapMtx    sync.Mutex
)

func init() {
	testSetupOnce.Do(func() {
		portMap = make(map[int]bool)
		rand.Seed(time.Now().Unix())
	})
}

// TestNetwork holds a list of nodes that are connected to each other
// locally. It also tracks the ports used by this network so it can
// release them to be reused on teardown.
type TestNetwork struct {
	Nodes []*core.OpenBazaarNode
	ports []int
}

// TearDown will destroy all test nodes and repos.
func (tn *TestNetwork) TearDown() {
	for _, node := range tn.Nodes {
		node.DestroyNode()
	}
	for _, port := range tn.ports {
		portMapMtx.Lock()
		delete(portMap, port)
		portMapMtx.Unlock()
	}
}

// NewTestNetwork spins up a network of nodes and connects them all
// together locally.
func NewTestNetwork(numNodes int) (*TestNetwork, error) {
	if numNodes < 2 || numNodes > 10000 {
		return nil, errors.New("numNodes must be between 2 and 10000")
	}

	nodes := make([]*core.OpenBazaarNode, 0, numNodes)
	ports := make([]int, 0, numNodes*2)

	for i := 0; i < numNodes; i++ {
		swarmPort := unusedPort()
		gatewayPort := unusedPort()
		ports = append(ports, swarmPort, gatewayPort)

		var bootstrapAddrs []string
		for _, node := range nodes {
			ipfsCfg, err := node.IPFSNode().Repo.Config()
			if err != nil {
				return nil, err
			}
			bootstrapAddrs = append(bootstrapAddrs, ipfsCfg.Addresses.Swarm[0]+"/ipfs/"+node.Identity().Pretty())
			if len(bootstrapAddrs) >= 10 {
				break
			}
		}

		config := &repo.Config{
			LogLevel:      "debug",
			DataDir:       path.Join(os.TempDir(), fmt.Sprintf("%s-%d", tempDirectoryName, swarmPort)),
			SwarmAddrs:    []string{fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", swarmPort)},
			GatewayAddr:   fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", gatewayPort),
			BoostrapAddrs: bootstrapAddrs,
			Testnet:       true,
		}

		node, err := core.NewNode(context.Background(), config)
		if err != nil {
			return nil, err
		}

		nodes = append(nodes, node)
	}

	return &TestNetwork{nodes, ports}, nil
}

// unusedPort returns a suitable port for testing that is not currently
// used by another test.
func unusedPort() int {
	portMapMtx.Lock()
	portMapMtx.Unlock()
	for {
		port := rand.Intn(65535)
		port++
		if !portMap[port] {
			portMap[port] = true
			return port
		}
	}
}
