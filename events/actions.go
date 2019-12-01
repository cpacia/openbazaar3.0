package events

import "time"

type Follow struct {
	Notification
	PeerID string `json:"peerID"`
}

type Unfollow struct {
	Notification
	PeerID string `json:"peerID"`
}

type ModeratorAdd struct {
	Notification
	PeerID string `json:"peerID"`
}

type ModeratorRemove struct {
	Notification
	PeerID string `json:"peerID"`
}

type ChatMessage struct {
	MessageID string    `json:"messageID"`
	PeerID    string    `json:"peerID"`
	OrderID   string    `json:"orderID"`
	Timestamp time.Time `json:"timestamp"`
	Read      bool      `json:"read"`
	Outgoing  bool      `json:"outgoing"`
	Message   string    `json:"message"`
}

type ChatRead struct {
	MessageID string `json:"messageID"`
	PeerID    string `json:"peerID"`
	OrderID   string `json:"orderID"`
}

type ChatTyping struct {
	PeerID  string `json:"peerID"`
	OrderID string `json:"orderID"`
}

type IncomingTransaction struct {
	Wallet        string    `json:"wallet"`
	Txid          string    `json:"txid"`
	Value         int64     `json:"value"`
	Address       string    `json:"address"`
	Status        string    `json:"status"`
	Memo          string    `json:"memo"`
	Timestamp     time.Time `json:"timestamp"`
	Confirmations int32     `json:"confirmations"`
	OrderID       string    `json:"orderId"`
	Thumbnail     string    `json:"thumbnail"`
	Height        int32     `json:"height"`
	CanBumpFee    bool      `json:"canBumpFee"`
}

type AddressRequestResponse struct {
	PeerID  string `json:"peerID"`
	Address string `json:"address"`
	Coin    string `json:"coin"`
}
