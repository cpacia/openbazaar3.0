package coreiface

import (
	"context"
	"github.com/cpacia/multiwallet"
	"github.com/cpacia/openbazaar3.0/events"
	"github.com/cpacia/openbazaar3.0/models"
	"github.com/cpacia/openbazaar3.0/orders/pb"
	"github.com/cpacia/openbazaar3.0/wallet"
	iwallet "github.com/cpacia/wallet-interface"
	"github.com/ipfs/go-cid"
	"github.com/ipfs/go-ipfs/core"
	peer "github.com/libp2p/go-libp2p-peer"
	"io"
)

// CoreIface enumerates the interface of the OpenBazaarNode object in the Core package.
// We primarily use this to get around circular imports though it should server as the API
// contract for the Core package.
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
	FulfillOrder(orderID models.OrderID, fulfillments []models.Fulfillment, done chan struct{}) error
	CancelOrder(orderID models.OrderID, done chan struct{}) error
	FollowNode(peerID peer.ID, done chan<- struct{}) error
	UnfollowNode(peerID peer.ID, done chan<- struct{}) error
	GetMyFollowers() (models.Followers, error)
	GetMyFollowing() (models.Following, error)
	GetFollowers(ctx context.Context, peerID peer.ID, useCache bool) (models.Followers, error)
	GetFollowing(ctx context.Context, peerID peer.ID, useCache bool) (models.Following, error)
	SaveListing(listing *pb.Listing, done chan<- struct{}) error
	UpdateAllListings(updateFunc func(l *pb.Listing) (bool, error), done chan<- struct{}) error
	DeleteListing(slug string, done chan<- struct{}) error
	GetMyListings() (models.ListingIndex, error)
	GetListings(ctx context.Context, peerID peer.ID, useCache bool) (models.ListingIndex, error)
	GetMyListingBySlug(slug string) (*pb.SignedListing, error)
	GetMyListingByCID(cid cid.Cid) (*pb.SignedListing, error)
	GetListingBySlug(ctx context.Context, peerID peer.ID, slug string, useCache bool) (*pb.SignedListing, error)
	GetListingByCID(ctx context.Context, cid cid.Cid) (*pb.SignedListing, error)
	GetImage(ctx context.Context, cid cid.Cid) (io.ReadSeeker, error)
	GetAvatar(ctx context.Context, peerID peer.ID, size models.ImageSize, useCache bool) (io.ReadSeeker, error)
	GetHeader(ctx context.Context, peerID peer.ID, size models.ImageSize, useCache bool) (io.ReadSeeker, error)
	SetAvatarImage(base64ImageData string, done chan struct{}) (models.ImageHashes, error)
	SetHeaderImage(base64ImageData string, done chan struct{}) (models.ImageHashes, error)
	SetProductImage(base64ImageData string, filename string) (models.ImageHashes, error)
	SetSelfAsModerator(ctx context.Context, modInfo *models.ModeratorInfo, done chan struct{}) error
	RemoveSelfAsModerator(ctx context.Context, done chan<- struct{}) error
	GetModerators(ctx context.Context) []peer.ID
	GetModeratorsAsync(ctx context.Context) <-chan peer.ID
	SetModeratorsOnListings(mods []peer.ID, done chan struct{}) error
	GetPreferences() (*models.UserPreferences, error)
	SavePreferences(prefs *models.UserPreferences, done chan struct{}) error
	Publish(done chan<- struct{})
	UsingTestnet() bool
	UsingTorMode() bool
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
	ExchangeRates() *wallet.ExchangeRateProvider
}
