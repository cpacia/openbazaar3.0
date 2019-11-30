package api

import (
	"context"
	"github.com/cpacia/multiwallet"
	"github.com/cpacia/openbazaar3.0/events"
	"github.com/cpacia/openbazaar3.0/models"
	"github.com/cpacia/openbazaar3.0/orders/pb"
	iwallet "github.com/cpacia/wallet-interface"
	"github.com/ipfs/go-cid"
	"github.com/ipfs/go-ipfs/core"
	peer "github.com/libp2p/go-libp2p-peer"
)

// CoreIface is used to get around a circular import of the Core package.
type CoreIface interface {
	RequestAddress(ctx context.Context, to peer.ID, coinType iwallet.CoinType) (iwallet.Address, error)
	SendChatMessage(to peer.ID, message string, orderID models.OrderID, done chan<- struct{}) error
	SendTypingMessage(to peer.ID, orderID models.OrderID) error
	MarkChatMessagesAsRead(peer peer.ID, orderID models.OrderID) error
	GetChatConversations() ([]models.ChatConversation, error)
	GetChatMessagesByPeer(peer peer.ID, limit int, offsetID string) ([]models.ChatMessage, error)
	GetChatMessagesByOrderID(orderID models.OrderID, limit int, offsetID string) ([]models.ChatMessage, error)
	DeleteChatMessage(messageID string) error
	DeleteChatConversation(peerID peer.ID) error
	DeleteGroupChatMessages(orderID models.OrderID) error
	ConfirmOrder(orderID models.OrderID, done chan struct{}) error
	CancelOrder(orderID models.OrderID, done chan struct{}) error
	FollowNode(peerID peer.ID, done chan<- struct{}) error
	UnfollowNode(peerID peer.ID, done chan<- struct{}) error
	GetMyFollowers() (models.Followers, error)
	GetMyFollowing() (models.Following, error)
	GetFollowers(ctx context.Context, peerID peer.ID, useCache bool) (models.Followers, error)
	GetFollowing(ctx context.Context, peerID peer.ID, useCache bool) (models.Following, error)
	SaveListing(listing *pb.Listing, done chan<- struct{}) error
	DeleteListing(slug string, done chan<- struct{}) error
	GetMyListings() (models.ListingIndex, error)
	GetListings(ctx context.Context, peerID peer.ID, useCache bool) (models.ListingIndex, error)
	GetMyListingBySlug(slug string) (*pb.SignedListing, error)
	GetMyListingByCID(cid cid.Cid) (*pb.SignedListing, error)
	GetListingBySlug(ctx context.Context, peerID peer.ID, slug string, useCache bool) (*pb.SignedListing, error)
	GetListingByCID(ctx context.Context, cid cid.Cid) (*pb.SignedListing, error)
	SetSelfAsModerator(ctx context.Context, modInfo *models.ModeratorInfo, done chan struct{}) error
	RemoveSelfAsModerator(ctx context.Context, done chan<- struct{}) error
	GetModerators(ctx context.Context) []peer.ID
	GetModeratorsAsync(ctx context.Context) <-chan peer.ID
	Publish(done chan<- struct{})
	UsingTestnet() bool
	IPFSNode() *core.IpfsNode
	Multiwallet() multiwallet.Multiwallet
	Identity() peer.ID
	SubscribeEvent(event interface{}) (events.Subscription, error)
	SetProfile(profile *models.Profile, done chan<- struct{}) error
	GetMyProfile() (*models.Profile, error)
	GetProfile(ctx context.Context, peerID peer.ID, useCache bool) (*models.Profile, error)
	PurchaseListing(ctx context.Context, purchase *models.Purchase) (orderID models.OrderID, paymentAddress iwallet.Address, paymentAmount models.CurrencyValue, err error)
	EstimateOrderSubtotal(ctx context.Context, purchase *models.Purchase) (*models.CurrencyValue, error)
	RejectOrder(orderID models.OrderID, reason string, done chan struct{}) error
	RefundOrder(orderID models.OrderID, done chan struct{}) error
	PingNode(ctx context.Context, peer peer.ID) error
	SaveTransactionMetadata(metadata *models.TransactionMetadata) error
	GetTransactionMetadata(txid iwallet.TransactionID) (models.TransactionMetadata, error)
}
