package wallet_interface

// FeeLevel represents a user-selected level of fees to
// pay for the transaction. It's up to the wallet to
// determine what these levels will mean for the implementation.
type FeeLevel int

const (
	// FlPriority represents the priority fee.
	FlPriority FeeLevel = iota
	// FlPriority represents the normal fee.
	FlNormal
	// FlEconomic represents the economic fee.
	FlEconomic
)
