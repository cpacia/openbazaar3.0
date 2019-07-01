package core

import (
	"github.com/cpacia/openbazaar3.0/models"
	"strings"
	"testing"
)

func TestOpenBazaarNode_Profile(t *testing.T) {
	node, err := MockNode()
	if err != nil {
		t.Fatal(err)
	}
	defer node.repo.DestroyRepo()

	name := "Ron Swanson"

	err = node.SetProfile(&models.Profile{
		Name:      name,
		PublicKey: strings.Repeat("s", 66),
	}, nil)
	if err != nil {
		t.Error(err)
	}

	pro, err := node.GetMyProfile()
	if err != nil {
		t.Error(err)
	}

	if pro.Name != name {
		t.Errorf("Returned incorrect profile. Expected name %s go %s", name, pro.Name)
	}
}

func TestOpenBazaarNode_GetProfile(t *testing.T) {
	mocknet, err := NewMocknet(2)
	if err != nil {
		t.Fatal(err)
	}
	defer mocknet.TearDown()

	name := "Ron Swanson"
	done := make(chan struct{})
	err = mocknet.Nodes()[0].SetProfile(&models.Profile{
		Name:      name,
		PublicKey: strings.Repeat("s", 66),
	}, done)
	if err != nil {
		t.Fatal(err)
	}
	<-done

	pro, err := mocknet.Nodes()[1].GetProfile(mocknet.Nodes()[0].Identity(), false)
	if err != nil {
		t.Fatal(err)
	}

	if pro.Name != name {
		t.Errorf("Returned profile with incorrect name. Expected %s, got %s", name, pro.Name)
	}

	// Change name
	name2 := "Peter Griffin"
	done = make(chan struct{})
	err = mocknet.Nodes()[0].SetProfile(&models.Profile{
		Name:      name2,
		PublicKey: strings.Repeat("s", 66),
	}, done)
	if err != nil {
		t.Fatal(err)
	}
	<-done

	// Test fetching from cache
	pro, err = mocknet.Nodes()[1].GetProfile(mocknet.Nodes()[0].Identity(), true)
	if err != nil {
		t.Fatal(err)
	}

	if pro.Name != name {
		t.Errorf("Returned profile with incorrect name. Expected %s, got %s", name, pro.Name)
	}
}

func TestOpenBazaarNode_validateProfile(t *testing.T) {
	tests := []struct {
		name    string
		profile *models.Profile
		valid   bool
	}{
		{
			name: "Valid name",
			profile: &models.Profile{
				Name:      "Ron Swanson",
				PublicKey: strings.Repeat("s", 66),
			},
			valid: true,
		},
		{
			name: "Name zero len",
			profile: &models.Profile{
				Name:      "",
				PublicKey: strings.Repeat("s", 66),
			},
			valid: false,
		},
		{
			name: "Name > max len",
			profile: &models.Profile{
				Name:      strings.Repeat("r", WordMaxCharacters+1),
				PublicKey: strings.Repeat("s", 66),
			},
			valid: false,
		},
		{
			name: "Name at max len",
			profile: &models.Profile{
				Name:      strings.Repeat("r", WordMaxCharacters),
				PublicKey: strings.Repeat("s", 66),
			},
			valid: true,
		},
		{
			name: "Location at max len",
			profile: &models.Profile{
				Name:      "Ron Swanson",
				Location:  strings.Repeat("r", WordMaxCharacters),
				PublicKey: strings.Repeat("s", 66),
			},
			valid: true,
		},
		{
			name: "Location > max len",
			profile: &models.Profile{
				Name:      "Ron Swanson",
				Location:  strings.Repeat("r", WordMaxCharacters+1),
				PublicKey: strings.Repeat("s", 66),
			},
			valid: false,
		},
		{
			name: "About at max len",
			profile: &models.Profile{
				Name:      "Ron Swanson",
				About:     strings.Repeat("r", AboutMaxCharacters),
				PublicKey: strings.Repeat("s", 66),
			},
			valid: true,
		},
		{
			name: "About > max len",
			profile: &models.Profile{
				Name:      "Ron Swanson",
				About:     strings.Repeat("r", AboutMaxCharacters+1),
				PublicKey: strings.Repeat("s", 66),
			},
			valid: false,
		},
		{
			name: "Short description at max len",
			profile: &models.Profile{
				Name:             "Ron Swanson",
				ShortDescription: strings.Repeat("r", ShortDescriptionLength),
				PublicKey:        strings.Repeat("s", 66),
			},
			valid: true,
		},
		{
			name: "Short description > max len",
			profile: &models.Profile{
				Name:             "Ron Swanson",
				ShortDescription: strings.Repeat("r", ShortDescriptionLength+1),
				PublicKey:        strings.Repeat("s", 66),
			},
			valid: false,
		},
		{
			name: "Public key correct len",
			profile: &models.Profile{
				Name:      "Ron Swanson",
				PublicKey: strings.Repeat("r", 66),
			},
			valid: true,
		},
		{
			name: "Public key != len",
			profile: &models.Profile{
				Name:      "Ron Swanson",
				PublicKey: strings.Repeat("r", 67),
			},
			valid: false,
		},
		{
			name: "Contact info:website correct len",
			profile: &models.Profile{
				Name:      "Ron Swanson",
				PublicKey: strings.Repeat("r", 66),
				ContactInfo: &models.ProfileContactInfo{
					Website: strings.Repeat("s", URLMaxCharacters),
				},
			},
			valid: true,
		},
		{
			name: "Contact info:website > max len",
			profile: &models.Profile{
				Name:      "Ron Swanson",
				PublicKey: strings.Repeat("r", 66),
				ContactInfo: &models.ProfileContactInfo{
					Website: strings.Repeat("s", URLMaxCharacters+1),
				},
			},
			valid: false,
		},
		{
			name: "Contact info:email correct len",
			profile: &models.Profile{
				Name:      "Ron Swanson",
				PublicKey: strings.Repeat("r", 66),
				ContactInfo: &models.ProfileContactInfo{
					Email: strings.Repeat("s", SentenceMaxCharacters),
				},
			},
			valid: true,
		},
		{
			name: "Contact info:email > max len",
			profile: &models.Profile{
				Name:      "Ron Swanson",
				PublicKey: strings.Repeat("r", 66),
				ContactInfo: &models.ProfileContactInfo{
					Email: strings.Repeat("s", SentenceMaxCharacters+1),
				},
			},
			valid: false,
		},
		{
			name: "Contact info:phone number correct len",
			profile: &models.Profile{
				Name:      "Ron Swanson",
				PublicKey: strings.Repeat("r", 66),
				ContactInfo: &models.ProfileContactInfo{
					PhoneNumber: strings.Repeat("s", WordMaxCharacters),
				},
			},
			valid: true,
		},
		{
			name: "Contact info:phone number > max len",
			profile: &models.Profile{
				Name:      "Ron Swanson",
				PublicKey: strings.Repeat("r", 66),
				ContactInfo: &models.ProfileContactInfo{
					PhoneNumber: strings.Repeat("s", WordMaxCharacters+1),
				},
			},
			valid: false,
		},
		{
			name: "Contact info:social list correct len",
			profile: &models.Profile{
				Name:      "Ron Swanson",
				PublicKey: strings.Repeat("r", 66),
				ContactInfo: &models.ProfileContactInfo{
					Social: make([]models.SocialAccount, MaxListItems),
				},
			},
			valid: true,
		},
		{
			name: "Contact info:social list > max len",
			profile: &models.Profile{
				Name:      "Ron Swanson",
				PublicKey: strings.Repeat("r", 66),
				ContactInfo: &models.ProfileContactInfo{
					Social: make([]models.SocialAccount, MaxListItems+1),
				},
			},
			valid: false,
		},
		{
			name: "Contact info:social:username correct len",
			profile: &models.Profile{
				Name:      "Ron Swanson",
				PublicKey: strings.Repeat("r", 66),
				ContactInfo: &models.ProfileContactInfo{
					Social: []models.SocialAccount{
						{
							Username: strings.Repeat("s", WordMaxCharacters),
						},
					},
				},
			},
			valid: true,
		},
		{
			name: "Contact info:social:username > max len",
			profile: &models.Profile{
				Name:      "Ron Swanson",
				PublicKey: strings.Repeat("r", 66),
				ContactInfo: &models.ProfileContactInfo{
					Social: []models.SocialAccount{
						{
							Username: strings.Repeat("s", WordMaxCharacters+1),
						},
					},
				},
			},
			valid: false,
		},
		{
			name: "Contact info:social:type correct len",
			profile: &models.Profile{
				Name:      "Ron Swanson",
				PublicKey: strings.Repeat("r", 66),
				ContactInfo: &models.ProfileContactInfo{
					Social: []models.SocialAccount{
						{
							Type: strings.Repeat("s", WordMaxCharacters),
						},
					},
				},
			},
			valid: true,
		},
		{
			name: "Contact info:social:type > max len",
			profile: &models.Profile{
				Name:      "Ron Swanson",
				PublicKey: strings.Repeat("r", 66),
				ContactInfo: &models.ProfileContactInfo{
					Social: []models.SocialAccount{
						{
							Type: strings.Repeat("s", WordMaxCharacters+1),
						},
					},
				},
			},
			valid: false,
		},
		{
			name: "Contact info:social:proof correct len",
			profile: &models.Profile{
				Name:      "Ron Swanson",
				PublicKey: strings.Repeat("r", 66),
				ContactInfo: &models.ProfileContactInfo{
					Social: []models.SocialAccount{
						{
							Proof: strings.Repeat("s", URLMaxCharacters),
						},
					},
				},
			},
			valid: true,
		},
		{
			name: "Contact info:social:proof > max len",
			profile: &models.Profile{
				Name:      "Ron Swanson",
				PublicKey: strings.Repeat("r", 66),
				ContactInfo: &models.ProfileContactInfo{
					Social: []models.SocialAccount{
						{
							Proof: strings.Repeat("s", URLMaxCharacters+1),
						},
					},
				},
			},
			valid: false,
		},
		{
			name: "Mod info:description correct len",
			profile: &models.Profile{
				Name:      "Ron Swanson",
				PublicKey: strings.Repeat("r", 66),
				ModeratorInfo: &models.ModeratorInfo{
					Description: strings.Repeat("s", AboutMaxCharacters),
				},
			},
			valid: true,
		},
		{
			name: "Mod info:description > max len",
			profile: &models.Profile{
				Name:      "Ron Swanson",
				PublicKey: strings.Repeat("r", 66),
				ModeratorInfo: &models.ModeratorInfo{
					Description: strings.Repeat("s", AboutMaxCharacters+1),
				},
			},
			valid: false,
		},
		{
			name: "Mod info:terms and conditions correct len",
			profile: &models.Profile{
				Name:      "Ron Swanson",
				PublicKey: strings.Repeat("r", 66),
				ModeratorInfo: &models.ModeratorInfo{
					TermsAndConditions: strings.Repeat("s", PolicyMaxCharacters),
				},
			},
			valid: true,
		},
		{
			name: "Mod info:terms and conditions > max len",
			profile: &models.Profile{
				Name:      "Ron Swanson",
				PublicKey: strings.Repeat("r", 66),
				ModeratorInfo: &models.ModeratorInfo{
					TermsAndConditions: strings.Repeat("s", PolicyMaxCharacters+1),
				},
			},
			valid: false,
		},
		{
			name: "Mod info:language list correct len",
			profile: &models.Profile{
				Name:      "Ron Swanson",
				PublicKey: strings.Repeat("r", 66),
				ModeratorInfo: &models.ModeratorInfo{
					Languages: make([]string, MaxListItems),
				},
			},
			valid: true,
		},
		{
			name: "Mod info:languages list > max len",
			profile: &models.Profile{
				Name:      "Ron Swanson",
				PublicKey: strings.Repeat("r", 66),
				ModeratorInfo: &models.ModeratorInfo{
					Languages: make([]string, MaxListItems+1),
				},
			},
			valid: false,
		},
		{
			name: "Mod info:language correct len",
			profile: &models.Profile{
				Name:      "Ron Swanson",
				PublicKey: strings.Repeat("r", 66),
				ModeratorInfo: &models.ModeratorInfo{
					Languages: []string{
						strings.Repeat("s", WordMaxCharacters),
					},
				},
			},
			valid: true,
		},
		{
			name: "Mod info:languages > max len",
			profile: &models.Profile{
				Name:      "Ron Swanson",
				PublicKey: strings.Repeat("r", 66),
				ModeratorInfo: &models.ModeratorInfo{
					Languages: []string{
						strings.Repeat("s", WordMaxCharacters+1),
					},
				},
			},
			valid: false,
		},
		{
			name: "Mod info:fixed fee:currency code correct len",
			profile: &models.Profile{
				Name:      "Ron Swanson",
				PublicKey: strings.Repeat("r", 66),
				ModeratorInfo: &models.ModeratorInfo{
					Fee: models.ModeratorFee{
						FixedFee: &models.CurrencyValue{
							CurrencyCode: strings.Repeat("s", WordMaxCharacters),
						},
					},
				},
			},
			valid: true,
		},
		{
			name: "Mod info:fixed fee: currency code > max len",
			profile: &models.Profile{
				Name:      "Ron Swanson",
				PublicKey: strings.Repeat("r", 66),
				ModeratorInfo: &models.ModeratorInfo{
					Fee: models.ModeratorFee{
						FixedFee: &models.CurrencyValue{
							CurrencyCode: strings.Repeat("s", WordMaxCharacters+1),
						},
					},
				},
			},
			valid: false,
		},
		{
			name: "Valid avatar hashes",
			profile: &models.Profile{
				Name:      "Ron Swanson",
				PublicKey: strings.Repeat("r", 66),
				AvatarHashes: models.ProfileImage{
					Large:    "QmfQkD8pBSBCBxWEwFSu4XaDVSWK6bjnNuaWZjMyQbyDub",
					Medium:   "QmfQkD8pBSBCBxWEwFSu4XaDVSWK6bjnNuaWZjMyQbyDub",
					Original: "QmfQkD8pBSBCBxWEwFSu4XaDVSWK6bjnNuaWZjMyQbyDub",
					Small:    "QmfQkD8pBSBCBxWEwFSu4XaDVSWK6bjnNuaWZjMyQbyDub",
					Tiny:     "QmfQkD8pBSBCBxWEwFSu4XaDVSWK6bjnNuaWZjMyQbyDub",
				},
			},
			valid: true,
		},
		{
			name: "Invalid avatar hashes",
			profile: &models.Profile{
				Name:      "Ron Swanson",
				PublicKey: strings.Repeat("r", 66),
				AvatarHashes: models.ProfileImage{
					Large:    "xxx",
					Medium:   "xxx",
					Original: "xxx",
					Small:    "xxx",
					Tiny:     "xxx",
				},
			},
			valid: false,
		},
		{
			name: "Valid header hashes",
			profile: &models.Profile{
				Name:      "Ron Swanson",
				PublicKey: strings.Repeat("r", 66),
				HeaderHashes: models.ProfileImage{
					Large:    "QmfQkD8pBSBCBxWEwFSu4XaDVSWK6bjnNuaWZjMyQbyDub",
					Medium:   "QmfQkD8pBSBCBxWEwFSu4XaDVSWK6bjnNuaWZjMyQbyDub",
					Original: "QmfQkD8pBSBCBxWEwFSu4XaDVSWK6bjnNuaWZjMyQbyDub",
					Small:    "QmfQkD8pBSBCBxWEwFSu4XaDVSWK6bjnNuaWZjMyQbyDub",
					Tiny:     "QmfQkD8pBSBCBxWEwFSu4XaDVSWK6bjnNuaWZjMyQbyDub",
				},
			},
			valid: true,
		},
		{
			name: "Invalid header hashes",
			profile: &models.Profile{
				Name:      "Ron Swanson",
				PublicKey: strings.Repeat("r", 66),
				HeaderHashes: models.ProfileImage{
					Large:    "xxx",
					Medium:   "xxx",
					Original: "xxx",
					Small:    "xxx",
					Tiny:     "xxx",
				},
			},
			valid: false,
		},
		{
			name: "Average rating correct range",
			profile: &models.Profile{
				Name:      "Ron Swanson",
				PublicKey: strings.Repeat("r", 66),
				Stats: &models.ProfileStats{
					AverageRating: 5,
				},
			},
			valid: true,
		},
		{
			name: "Average rating incorrect range",
			profile: &models.Profile{
				Name:      "Ron Swanson",
				PublicKey: strings.Repeat("r", 66),
				Stats: &models.ProfileStats{
					AverageRating: 6,
				},
			},
			valid: false,
		},
	}

	for _, test := range tests {
		err := validateProfile(test.profile)
		if test.valid && err != nil {
			t.Errorf("Test %s: returned unexpected error: %s", test.name, err)
		} else if !test.valid && err == nil {
			t.Errorf("Test %s: did not return error", test.name)
		}
	}
}
