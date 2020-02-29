package api

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

type mockNode struct {
	requestAddressFunc           func(ctx context.Context, to peer.ID, coinType iwallet.CoinType) (iwallet.Address, error)
	sendChatMessageFunc          func(to peer.ID, message string, orderID models.OrderID, done chan<- struct{}) error
	sendTypingMessageFunc        func(to peer.ID, orderID models.OrderID) error
	markChatMessagesAsReadFunc   func(peer peer.ID, orderID models.OrderID) error
	getChatConversationsFunc     func() ([]models.ChatConversation, error)
	getChatMessagesByPeerFunc    func(peer peer.ID, limit int, offsetID string) ([]models.ChatMessage, error)
	getChatMessagesByOrderIDFunc func(orderID models.OrderID, limit int, offsetID string) ([]models.ChatMessage, error)
	deleteChatMessageFunc        func(messageID string) error
	deleteChatConversationFunc   func(peerID peer.ID) error
	deleteGroupChatMessagesFunc  func(orderID models.OrderID) error
	confirmOrderFunc             func(orderID models.OrderID, done chan struct{}) error
	fulfillOrderFunc             func(orderID models.OrderID, fulfillments []models.Fulfillment, done chan struct{}) error
	cancelOrderFunc              func(orderID models.OrderID, done chan struct{}) error
	followNodeFunc               func(peerID peer.ID, done chan<- struct{}) error
	unfollowNodeFunc             func(peerID peer.ID, done chan<- struct{}) error
	getMyFollowersFunc           func() (models.Followers, error)
	getMyFollowingFunc           func() (models.Following, error)
	getFollowersFunc             func(ctx context.Context, peerID peer.ID, useCache bool) (models.Followers, error)
	getFollowingFunc             func(ctx context.Context, peerID peer.ID, useCache bool) (models.Following, error)
	saveListingFunc              func(listing *pb.Listing, done chan<- struct{}) error
	updateAllListingsFunc        func(updateFunc func(l *pb.Listing) (bool, error), done chan<- struct{}) error
	deleteListingFunc            func(slug string, done chan<- struct{}) error
	getMyListingsFunc            func() (models.ListingIndex, error)
	getListingsFunc              func(ctx context.Context, peerID peer.ID, useCache bool) (models.ListingIndex, error)
	getMyListingBySlugFunc       func(slug string) (*pb.SignedListing, error)
	getMyListingByCIDFunc        func(cid cid.Cid) (*pb.SignedListing, error)
	getListingBySlugFunc         func(ctx context.Context, peerID peer.ID, slug string, useCache bool) (*pb.SignedListing, error)
	getListingByCIDFunc          func(ctx context.Context, cid cid.Cid) (*pb.SignedListing, error)
	getImageFunc                 func(ctx context.Context, cid cid.Cid) (io.ReadSeeker, error)
	getAvatarFunc                func(ctx context.Context, peerID peer.ID, size models.ImageSize, useCache bool) (io.ReadSeeker, error)
	getHeaderFunc                func(ctx context.Context, peerID peer.ID, size models.ImageSize, useCache bool) (io.ReadSeeker, error)
	setAvatarImageFunc           func(base64ImageData string, done chan struct{}) (models.ImageHashes, error)
	setHeaderImageFunc           func(base64ImageData string, done chan struct{}) (models.ImageHashes, error)
	setProductImageFunc          func(base64ImageData string, filename string) (models.ImageHashes, error)
	setSelfAsModeratorFunc       func(ctx context.Context, modInfo *models.ModeratorInfo, done chan struct{}) error
	setModeratorsOnListingsFunc  func(mods []peer.ID, done chan struct{}) error
	removeSelfAsModeratorFunc    func(ctx context.Context, done chan<- struct{}) error
	getModeratorsFunc            func(ctx context.Context) []peer.ID
	getModeratorsAsyncFunc       func(ctx context.Context) <-chan peer.ID
	publishFunc                  func(done chan<- struct{})
	usingTestnetFunc             func() bool
	usingTorFunc                 func() bool
	ipfsNodeFunc                 func() *core.IpfsNode
	multiwalletFunc              func() multiwallet.Multiwallet
	identityFunc                 func() peer.ID
	subscribeEventFunc           func(event interface{}) (events.Subscription, error)
	setProfileFunc               func(profile *models.Profile, done chan<- struct{}) error
	getMyProfileFunc             func() (*models.Profile, error)
	getProfileFunc               func(ctx context.Context, peerID peer.ID, useCache bool) (*models.Profile, error)
	purchaseFunc                 func(ctx context.Context, purchase *models.Purchase) (orderID models.OrderID, paymentAddress iwallet.Address, paymentAmount models.CurrencyValue, err error)
	estimateOrderSubtotalFunc    func(ctx context.Context, purchase *models.Purchase) (*models.CurrencyValue, error)
	rejectOrderFunc              func(orderID models.OrderID, reason string, done chan struct{}) error
	refundOrderFunc              func(orderID models.OrderID, done chan struct{}) error
	pingNodeFunc                 func(ctx context.Context, peer peer.ID) error
	getUserPreferencesFunc       func() (*models.UserPreferences, error)
	saveUserPreferencesFunc      func(prefs *models.UserPreferences, done chan struct{}) error
	saveTransactionMetadataFunc  func(metadata *models.TransactionMetadata) error
	getTransactionMetadataFunc   func(txid iwallet.TransactionID) (models.TransactionMetadata, error)
	getExchangeRatesFunc         func() *wallet.ExchangeRateProvider
}

func (m *mockNode) RequestAddress(ctx context.Context, to peer.ID, coinType iwallet.CoinType) (iwallet.Address, error) {
	return m.requestAddressFunc(ctx, to, coinType)
}
func (m *mockNode) SendChatMessage(to peer.ID, message string, orderID models.OrderID, done chan<- struct{}) error {
	return m.sendChatMessageFunc(to, message, orderID, done)
}
func (m *mockNode) SendTypingMessage(to peer.ID, orderID models.OrderID) error {
	return m.sendTypingMessageFunc(to, orderID)
}
func (m *mockNode) MarkChatMessagesAsRead(peer peer.ID, orderID models.OrderID) error {
	return m.markChatMessagesAsReadFunc(peer, orderID)
}
func (m *mockNode) GetChatConversations() ([]models.ChatConversation, error) {
	return m.getChatConversationsFunc()
}
func (m *mockNode) GetChatMessagesByPeer(peer peer.ID, limit int, offsetID string) ([]models.ChatMessage, error) {
	return m.getChatMessagesByPeerFunc(peer, limit, offsetID)
}
func (m *mockNode) GetChatMessagesByOrderID(orderID models.OrderID, limit int, offsetID string) ([]models.ChatMessage, error) {
	return m.getChatMessagesByOrderIDFunc(orderID, limit, offsetID)
}
func (m *mockNode) DeleteChatMessage(messageID string) error {
	return m.deleteChatMessageFunc(messageID)
}
func (m *mockNode) DeleteChatConversation(peerID peer.ID) error {
	return m.deleteChatConversationFunc(peerID)
}
func (m *mockNode) DeleteGroupChatMessages(orderID models.OrderID) error {
	return m.deleteGroupChatMessagesFunc(orderID)
}
func (m *mockNode) ConfirmOrder(orderID models.OrderID, done chan struct{}) error {
	return m.confirmOrderFunc(orderID, done)
}
func (m *mockNode) FulfillOrder(orderID models.OrderID, fulfillments []models.Fulfillment, done chan struct{}) error {
	return m.fulfillOrderFunc(orderID, fulfillments, done)
}
func (m *mockNode) CancelOrder(orderID models.OrderID, done chan struct{}) error {
	return m.cancelOrderFunc(orderID, done)
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
func (m *mockNode) UpdateAllListings(updateFunc func(l *pb.Listing) (bool, error), done chan<- struct{}) error {
	return m.updateAllListingsFunc(updateFunc, done)
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
	return m.getMyListingBySlugFunc(slug)
}
func (m *mockNode) GetMyListingByCID(cid cid.Cid) (*pb.SignedListing, error) {
	return m.getMyListingByCIDFunc(cid)
}
func (m *mockNode) GetListingBySlug(ctx context.Context, peerID peer.ID, slug string, useCache bool) (*pb.SignedListing, error) {
	return m.getListingBySlugFunc(ctx, peerID, slug, useCache)
}
func (m *mockNode) GetListingByCID(ctx context.Context, cid cid.Cid) (*pb.SignedListing, error) {
	return m.getListingByCIDFunc(ctx, cid)
}
func (m *mockNode) GetImage(ctx context.Context, cid cid.Cid) (io.ReadSeeker, error) {
	return m.getImageFunc(ctx, cid)
}
func (m *mockNode) GetAvatar(ctx context.Context, peerID peer.ID, size models.ImageSize, useCache bool) (io.ReadSeeker, error) {
	return m.getAvatarFunc(ctx, peerID, size, useCache)
}
func (m *mockNode) GetHeader(ctx context.Context, peerID peer.ID, size models.ImageSize, useCache bool) (io.ReadSeeker, error) {
	return m.getHeaderFunc(ctx, peerID, size, useCache)
}
func (m *mockNode) SetAvatarImage(base64ImageData string, done chan struct{}) (models.ImageHashes, error) {
	return m.setAvatarImageFunc(base64ImageData, done)
}
func (m *mockNode) SetHeaderImage(base64ImageData string, done chan struct{}) (models.ImageHashes, error) {
	return m.setHeaderImageFunc(base64ImageData, done)
}
func (m *mockNode) SetProductImage(base64ImageData string, filename string) (models.ImageHashes, error) {
	return m.setProductImageFunc(base64ImageData, filename)
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
func (m *mockNode) SetModeratorsOnListings(mods []peer.ID, done chan struct{}) error {
	return m.setModeratorsOnListingsFunc(mods, done)
}
func (m *mockNode) Publish(done chan<- struct{}) {
	m.publishFunc(done)
}
func (m *mockNode) UsingTestnet() bool {
	return m.usingTestnetFunc()
}
func (m *mockNode) UsingTorMode() bool {
	return m.usingTorFunc()
}
func (m *mockNode) IPFSNode() *core.IpfsNode {
	return m.ipfsNodeFunc()
}
func (m *mockNode) Multiwallet() multiwallet.Multiwallet {
	return m.multiwalletFunc()
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
	return m.rejectOrderFunc(orderID, reason, done)
}
func (m *mockNode) RefundOrder(orderID models.OrderID, done chan struct{}) error {
	return m.refundOrderFunc(orderID, done)
}
func (m *mockNode) PingNode(ctx context.Context, peer peer.ID) error {
	return m.pingNodeFunc(ctx, peer)
}
func (m *mockNode) SaveTransactionMetadata(metadata *models.TransactionMetadata) error {
	return m.saveTransactionMetadataFunc(metadata)
}
func (m *mockNode) GetTransactionMetadata(txid iwallet.TransactionID) (models.TransactionMetadata, error) {
	return m.getTransactionMetadataFunc(txid)
}
func (m *mockNode) SavePreferences(prefs *models.UserPreferences, done chan struct{}) error {
	return m.saveUserPreferencesFunc(prefs, done)
}
func (m *mockNode) GetPreferences() (*models.UserPreferences, error) {
	return m.getUserPreferencesFunc()
}
func (m *mockNode) ExchangeRates() *wallet.ExchangeRateProvider {
	return m.getExchangeRatesFunc()
}
