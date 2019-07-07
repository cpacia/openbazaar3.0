package wallet_interface

// BlockID represents an ID for a block on the network.
type BlockID string

// BlockchainInfo holds the blockchain's best hash and height.
type BlockchainInfo struct {
	BestBlock BlockID
	Height uint64
}