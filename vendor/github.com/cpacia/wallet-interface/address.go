package wallet_interface

// Address represents a cryptocurrency address used by OpenBazaar.
type Address struct {
	addr string
	typ  CoinType
}

// NewAddress return a new Address.
func NewAddress(addr string, typ CoinType) *Address {
	return &Address{addr, typ}
}

// String returns the address's string representation.
func (a *Address) String() string {
	return a.addr
}

// CoinType returns the addresses type.
func (a *Address) CoinType() CoinType {
	return a.CoinType()
}
