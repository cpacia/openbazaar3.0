package models

// Followers represents the nodes that are following this node.
type Followers []string

// Count returns the number of followers.
func (f *Followers) Count() int {
	return len(*f)
}

// Following represents the list of nodes this node is following.
type Following []string

// Count returns the number of following.
func (f *Following) Count() int {
	return len(*f)
}

// FollowSequence is a database model which holds the sequence
// number for our outgoing follow and unfollow notifications.
type FollowSequence struct {
	PeerID string `gorm:"primary_key"`
	Num    int
}
