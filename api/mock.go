package api

import (
	"context"
	"github.com/cpacia/openbazaar3.0/events"
	"github.com/cpacia/openbazaar3.0/models"
	"github.com/cpacia/openbazaar3.0/orders/pb"
	iwallet "github.com/cpacia/wallet-interface"
	"github.com/ipfs/go-cid"
	"github.com/ipfs/go-ipfs/core"
	peer "github.com/libp2p/go-libp2p-peer"
)

type mockNode struct {
	requestAddressFunc           func(ctx context.Context, to peer.ID, coinType iwallet.CoinType) (iwallet.Address, error)
	sendChatMesageFunc           func(to peer.ID, message, subject string, done chan<- struct{}) error
	sendTypingMessageFunc        func(to peer.ID, subject string) error
	markChatMessagesAsReadFunc   func(peer peer.ID, subject string) error
	getChatConversationsFunc     func() ([]models.ChatConversation, error)
	getChatMessagesByPeerFunc    func(peer peer.ID) ([]models.ChatMessage, error)
	getChatMessagesBySubjectFunc func(subject string) ([]models.ChatMessage, error)
	confirmOrderFunc             func(orderID models.OrderID, done chan struct{}) error
	followNodeFunc               func(peerID peer.ID, done chan<- struct{}) error
	unfollowNodeFunc             func(peerID peer.ID, done chan<- struct{}) error
	getMyFollowersFunc           func() (models.Followers, error)
	getMyFollowingFunc           func() (models.Following, error)
	getFollowersFunc             func(ctx context.Context, peerID peer.ID, useCache bool) (models.Followers, error)
	getFollowingFunc             func(ctx context.Context, peerID peer.ID, useCache bool) (models.Following, error)
	saveListingFunc              func(listing *pb.Listing, done chan<- struct{}) error
	deleteListingFunc            func(slug string, done chan<- struct{}) error
	getMyListingsFunc            func() (models.ListingIndex, error)
	getListingsFunc              func(ctx context.Context, peerID peer.ID, useCache bool) (models.ListingIndex, error)
	getMyListingsBySlugFunc      func(slug string) (*pb.SignedListing, error)
	getMyListingsByCIDFunc       func(cid cid.Cid) (*pb.SignedListing, error)
	getListingsBySlugFunc        func(ctx context.Context, peerID peer.ID, slug string, useCache bool) (*pb.SignedListing, error)
	getListingsByCIDFunc         func(ctx context.Context, cid cid.Cid) (*pb.SignedListing, error)
	setSelfAsModeratorFunc       func(ctx context.Context, modInfo *models.ModeratorInfo, done chan struct{}) error
	removeSelfAsModeratorFunc    func(ctx context.Context, done chan<- struct{}) error
	getModeratorsFunc            func(ctx context.Context) []peer.ID
	getModeratorsAsyncFunc       func(ctx context.Context) <-chan peer.ID
	publishFunc                  func(done chan<- struct{})
	usingTestnetFunc             func() bool
	ipfsNodeFunc                 func() *core.IpfsNode
	identityFunc                 func() peer.ID
	subscribeEventFunc           func(event interface{}) (events.Subscription, error)
	setProfileFunc               func(profile *models.Profile, done chan<- struct{}) error
	getMyProfileFunc             func() (*models.Profile, error)
	getProfileFunc               func(ctx context.Context, peerID peer.ID, useCache bool) (*models.Profile, error)
	purchaseFunc                 func(ctx context.Context, purchase *models.Purchase) (orderID models.OrderID, paymentAddress iwallet.Address, paymentAmount models.CurrencyValue, err error)
	estimateOrderSubtotalFunc    func(ctx context.Context, purchase *models.Purchase) (*models.CurrencyValue, error)
	rejectOrderFunc              func(orderID models.OrderID, reason string, done chan struct{})
}

func (m *mockNode) RequestAddress(ctx context.Context, to peer.ID, coinType iwallet.CoinType) (iwallet.Address, error) {
	return m.requestAddressFunc(ctx, to, coinType)
}
func (m *mockNode) SendChatMessage(to peer.ID, message, subject string, done chan<- struct{}) error {
	return m.sendChatMesageFunc(to, message, subject, done)
}
func (m *mockNode) SendTypingMessage(to peer.ID, subject string) error {
	return m.sendTypingMessageFunc(to, subject)
}
func (m *mockNode) MarkChatMessagesAsRead(peer peer.ID, subject string) error {
	return m.markChatMessagesAsReadFunc(peer, subject)
}
func (m *mockNode) GetChatConversations() ([]models.ChatConversation, error) {
	return m.getChatConversationsFunc()
}
func (m *mockNode) GetChatMessagesByPeer(peer peer.ID) ([]models.ChatMessage, error) {
	return m.GetChatMessagesByPeer(peer)
}
func (m *mockNode) GetChatMessagesBySubject(subject string) ([]models.ChatMessage, error) {
	return m.getChatMessagesBySubjectFunc(subject)
}
func (m *mockNode) ConfirmOrder(orderID models.OrderID, done chan struct{}) error {
	return m.confirmOrderFunc(orderID, done)
}
func (m *mockNode) FollowNode(peerID peer.ID, done chan<- struct{}) error {
	return m.followNodeFunc(peerID, done)
}
func (m *mockNode) UnfollowNode(peerID peer.ID, done chan<- struct{}) error {
	return m.unfollowNodeFunc(peerID, done)
}
func (m *mockNode) GetMyFollowers() (models.Followers, error) {
	return m.getMyFollowersFunc()
}
func (m *mockNode) GetMyFollowing() (models.Following, error) {
	return m.getMyFollowingFunc()
}
func (m *mockNode) GetFollowers(ctx context.Context, peerID peer.ID, useCache bool) (models.Followers, error) {
	return m.getFollowersFunc(ctx, peerID, useCache)
}
func (m *mockNode) GetFollowing(ctx context.Context, peerID peer.ID, useCache bool) (models.Following, error) {
	return m.getFollowingFunc(ctx, peerID, useCache)
}
func (m *mockNode) SaveListing(listing *pb.Listing, done chan<- struct{}) error {
	return m.saveListingFunc(listing, done)
}
func (m *mockNode) DeleteListing(slug string, done chan<- struct{}) error {
	return m.deleteListingFunc(slug, done)
}
func (m *mockNode) GetMyListings() (models.ListingIndex, error) {
	return m.getMyListingsFunc()
}
func (m *mockNode) GetListings(ctx context.Context, peerID peer.ID, useCache bool) (models.ListingIndex, error) {
	return m.getListingsFunc(ctx, peerID, useCache)
}
func (m *mockNode) GetMyListingBySlug(slug string) (*pb.SignedListing, error) {
	return m.getMyListingsBySlugFunc(slug)
}
func (m *mockNode) GetMyListingByCID(cid cid.Cid) (*pb.SignedListing, error) {
	return m.getMyListingsByCIDFunc(cid)
}
func (m *mockNode) GetListingBySlug(ctx context.Context, peerID peer.ID, slug string, useCache bool) (*pb.SignedListing, error) {
	return m.getListingsBySlugFunc(ctx, peerID, slug, useCache)
}
func (m *mockNode) GetListingByCID(ctx context.Context, cid cid.Cid) (*pb.SignedListing, error) {
	return m.getListingsByCIDFunc(ctx, cid)
}
func (m *mockNode) SetSelfAsModerator(ctx context.Context, modInfo *models.ModeratorInfo, done chan struct{}) error {
	return m.setSelfAsModeratorFunc(ctx, modInfo, done)
}
func (m *mockNode) RemoveSelfAsModerator(ctx context.Context, done chan<- struct{}) error {
	return m.removeSelfAsModeratorFunc(ctx, done)
}
func (m *mockNode) GetModerators(ctx context.Context) []peer.ID {
	return m.getModeratorsFunc(ctx)
}
func (m *mockNode) GetModeratorsAsync(ctx context.Context) <-chan peer.ID {
	return m.getModeratorsAsyncFunc(ctx)
}
func (m *mockNode) Publish(done chan<- struct{}) {
	m.publishFunc(done)
}
func (m *mockNode) UsingTestnet() bool {
	return m.usingTestnetFunc()
}
func (m *mockNode) IPFSNode() *core.IpfsNode {
	return m.ipfsNodeFunc()
}
func (m *mockNode) Identity() peer.ID {
	return m.identityFunc()
}
func (m *mockNode) SubscribeEvent(event interface{}) (events.Subscription, error) {
	return m.subscribeEventFunc(event)
}
func (m *mockNode) SetProfile(profile *models.Profile, done chan<- struct{}) error {
	return m.setProfileFunc(profile, done)
}
func (m *mockNode) GetMyProfile() (*models.Profile, error) {
	return m.getMyProfileFunc()
}
func (m *mockNode) GetProfile(ctx context.Context, peerID peer.ID, useCache bool) (*models.Profile, error) {
	return m.getProfileFunc(ctx, peerID, useCache)
}
func (m *mockNode) PurchaseListing(ctx context.Context, purchase *models.Purchase) (orderID models.OrderID, paymentAddress iwallet.Address, paymentAmount models.CurrencyValue, err error) {
	return m.purchaseFunc(ctx, purchase)
}
func (m *mockNode) EstimateOrderSubtotal(ctx context.Context, purchase *models.Purchase) (*models.CurrencyValue, error) {
	return m.estimateOrderSubtotalFunc(ctx, purchase)
}
func (m *mockNode) RejectOrder(orderID models.OrderID, reason string, done chan struct{}) error {
	return m.RejectOrder(orderID, reason, done)
}
