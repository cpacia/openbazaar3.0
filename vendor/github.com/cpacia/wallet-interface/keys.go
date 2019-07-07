package wallet_interface

// XPrivKey represents an hierarchical deterministic xPriv key.
type XPrivKey string

// String returns the string representation of the key.
func (x *XPrivKey) String() string {
	return string(*x)
}
