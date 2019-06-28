package models

import "time"

type Profile struct {
	PeerID           string `json:"peerID"`
	Name             string `json:"name"`
	Location         string `json:"location"`
	About            string `json:"about"`
	ShortDescription string `json:"shortDescription"`

	OfflineMessagingAddress string `json:"offlineMessagingAddress"`

	Nsfw      bool `json:"nsfw"`
	Vendor    bool `json:"vendor"`
	Moderator bool `json:"moderator"`

	ModeratorInfo *ModeratorInfo      `json:"moderatorInfo"`
	ContactInfo   *ProfileContactInfo `json:"contactInfo"`

	Colors ProfileColors `json:"colors"`

	AvatarHashes ProfileImage `json:"avatarHashes"`
	HeaderHashes ProfileImage `json:"headerHashes"`

	Stats *ProfileStats `json:"stats"`

	PublicKey string `json:"publicKey"`

	LastModified time.Time `json:"lastModified"`
}

type ProfileContactInfo struct {
	Website     string          `json:"website"`
	Email       string          `json:"email"`
	PhoneNumber string          `json:"phoneNumber"`
	Social      []SocialAccount `json:"social"`
}

type SocialAccount struct {
	Type     string `json:"type"`
	Username string `json:"username"`
	Proof    string `json:"proof"`
}

type ProfileColors struct {
	Primary       string `json:"primary"`
	Secondary     string `json:"secondary"`
	Text          string `json:"text"`
	Highlight     string `json:"highlight"`
	HighlightText string `json:"highlightText"`
}

type ProfileStats struct {
	FollowerCount  uint32  `json:"followerCount"`
	FollowingCount uint32  `json:"followingCount"`
	ListingCount   uint32  `json:"listingCount"`
	RatingCount    uint32  `json:"ratingCount"`
	PostCount      uint32  `json:"postCount"`
	AverageRating  float32 `json:"averageRating"`
}

type ProfileImage struct {
	Tiny     string `json:"tiny"`
	Small    string `json:"small"`
	Medium   string `json:"medium"`
	Large    string `json:"large"`
	Original string `json:"original"`
}

type ModeratorInfo struct {
	Description        string       `json:"description"`
	TermsAndConditions string       `json:"termsAndConditions"`
	Languages          []string     `json:"languages"`
	AcceptedCurrencies []string     `json:"acceptedCurrencies"`
	Fee                ModeratorFee `json:"fee"`
}

type ModeratorFeeType uint8

const (
	FixedFee ModeratorFeeType = iota
	PercentageFee
	FixedPlusPercentageFee
)

type ModeratorFee struct {
	FixedFee   *CurrencyValue   `json:"fixedFee"`
	Percentage float64          `json:"percentage"`
	FeeType    ModeratorFeeType `json:"feeType"`
}
