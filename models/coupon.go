package models

// Coupon is coupon for a listing with the given slug.
// The hash is a multihash of the code. You can think
// of the code as a password needed to use the coupon.
type Coupon struct {
	Slug string
	Code string
	Hash string
}
