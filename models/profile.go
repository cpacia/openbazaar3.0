package models

import "time"

// Profile is a user profile that is saved in the public data directory.
type Profile struct {
	PeerID           string `json:"peerID"`
	Name             string `json:"name"`
	Handle           string `json:"handle"`
	Location         string `json:"location"`
	About            string `json:"about"`
	ShortDescription string `json:"shortDescription"`

	Nsfw      bool `json:"nsfw"`
	Vendor    bool `json:"vendor"`
	Moderator bool `json:"moderator"`

	ModeratorInfo *ModeratorInfo      `json:"moderatorInfo,omitempty"`
	ContactInfo   *ProfileContactInfo `json:"contactInfo,omitempty"`

	Colors ProfileColors `json:"colors"`

	AvatarHashes ProfileImage `json:"avatarHashes"`
	HeaderHashes ProfileImage `json:"headerHashes"`

	Stats *ProfileStats `json:"stats,omitempty"`

	EscrowPublicKey string `json:"publicKey"`

	OfflineInboxes []string `json:"offlineInboxes"`

	LastModified time.Time `json:"lastModified"`
}

// ProfileContactInfo is the user contact info.
type ProfileContactInfo struct {
	Website     string          `json:"website"`
	Email       string          `json:"email"`
	PhoneNumber string          `json:"phoneNumber"`
	Social      []SocialAccount `json:"social"`
}

// SocialAccount allows the user to list their social media accounts.
// The proof field should be a URL to a tweet or post saying something
// like: "My OpenBazaar ID is xxxxx".
// The proof is not automatically validated. The user will have to
// manually click the links.
type SocialAccount struct {
	Type     string `json:"type"`
	Username string `json:"username"`
	Proof    string `json:"proof"`
}

// ProfileColors allows the user to set their profile colors.
type ProfileColors struct {
	Primary       string `json:"primary"`
	Secondary     string `json:"secondary"`
	Text          string `json:"text"`
	Highlight     string `json:"highlight"`
	HighlightText string `json:"highlightText"`
}

// ProfileStats holds stats about the user. This should
// not be user editable.
type ProfileStats struct {
	FollowerCount  uint32  `json:"followerCount"`
	FollowingCount uint32  `json:"followingCount"`
	ListingCount   uint32  `json:"listingCount"`
	RatingCount    uint32  `json:"ratingCount"`
	PostCount      uint32  `json:"postCount"`
	AverageRating  float32 `json:"averageRating"`
}

// ProfileImage holds image hashes.
type ProfileImage struct {
	Tiny     string `json:"tiny"`
	Small    string `json:"small"`
	Medium   string `json:"medium"`
	Large    string `json:"large"`
	Original string `json:"original"`
}

// ModeratorInfo is set only if the user is a moderator.
// It contains information about their moderation terms.
// This is included in the profile so we don't need to
// do a separate IPNS query to get the moderator info.
type ModeratorInfo struct {
	Description        string       `json:"description"`
	TermsAndConditions string       `json:"termsAndConditions"`
	Languages          []string     `json:"languages"`
	AcceptedCurrencies []string     `json:"acceptedCurrencies"`
	Fee                ModeratorFee `json:"fee"`
}

// ModeratorFeeType denotes the type of fee structure.
type ModeratorFeeType uint8

const (
	FixedFee ModeratorFeeType = iota
	PercentageFee
	FixedPlusPercentageFee
)

// ModeratorFee holds the moderator fee information.
type ModeratorFee struct {
	FixedFee   *CurrencyValue   `json:"fixedFee"`
	Percentage float64          `json:"percentage"`
	FeeType    ModeratorFeeType `json:"feeType"`
}
