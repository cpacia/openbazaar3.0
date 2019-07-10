// Code generated by protoc-gen-go. DO NOT EDIT.
// source: orders.proto

package pb

import (
	fmt "fmt"
	proto "github.com/golang/protobuf/proto"
	timestamp "github.com/golang/protobuf/ptypes/timestamp"
	math "math"
)

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// This is a compile-time assertion to ensure that this generated file
// is compatible with the proto package it is being compiled against.
// A compilation error at this line likely means your copy of the
// proto package needs to be updated.
const _ = proto.ProtoPackageIsVersion3 // please upgrade the proto package

type OrderOpen_Payment_Method int32

const (
	OrderOpen_Payment_DIRECT     OrderOpen_Payment_Method = 0
	OrderOpen_Payment_CANCELABLE OrderOpen_Payment_Method = 1
	OrderOpen_Payment_MODERATED  OrderOpen_Payment_Method = 2
)

var OrderOpen_Payment_Method_name = map[int32]string{
	0: "DIRECT",
	1: "CANCELABLE",
	2: "MODERATED",
}

var OrderOpen_Payment_Method_value = map[string]int32{
	"DIRECT":     0,
	"CANCELABLE": 1,
	"MODERATED":  2,
}

func (x OrderOpen_Payment_Method) String() string {
	return proto.EnumName(OrderOpen_Payment_Method_name, int32(x))
}

func (OrderOpen_Payment_Method) EnumDescriptor() ([]byte, []int) {
	return fileDescriptor_e0f5d4cf0fc9e41b, []int{0, 2, 0}
}

type OrderOpen struct {
	Listings             []*Listing           `protobuf:"bytes,1,rep,name=listings,proto3" json:"listings,omitempty"`
	RefundAddress        string               `protobuf:"bytes,2,opt,name=refundAddress,proto3" json:"refundAddress,omitempty"`
	Shipping             *OrderOpen_Shipping  `protobuf:"bytes,3,opt,name=shipping,proto3" json:"shipping,omitempty"`
	BuyerID              *ID                  `protobuf:"bytes,4,opt,name=buyerID,proto3" json:"buyerID,omitempty"`
	Timestamp            *timestamp.Timestamp `protobuf:"bytes,5,opt,name=timestamp,proto3" json:"timestamp,omitempty"`
	Items                []*OrderOpen_Item    `protobuf:"bytes,6,rep,name=items,proto3" json:"items,omitempty"`
	Payment              *OrderOpen_Payment   `protobuf:"bytes,7,opt,name=payment,proto3" json:"payment,omitempty"`
	RatingKeys           [][]byte             `protobuf:"bytes,8,rep,name=ratingKeys,proto3" json:"ratingKeys,omitempty"`
	AlternateContactInfo string               `protobuf:"bytes,9,opt,name=alternateContactInfo,proto3" json:"alternateContactInfo,omitempty"`
	Version              uint32               `protobuf:"varint,10,opt,name=version,proto3" json:"version,omitempty"`
	Signature            []byte               `protobuf:"bytes,11,opt,name=signature,proto3" json:"signature,omitempty"`
	XXX_NoUnkeyedLiteral struct{}             `json:"-"`
	XXX_unrecognized     []byte               `json:"-"`
	XXX_sizecache        int32                `json:"-"`
}

func (m *OrderOpen) Reset()         { *m = OrderOpen{} }
func (m *OrderOpen) String() string { return proto.CompactTextString(m) }
func (*OrderOpen) ProtoMessage()    {}
func (*OrderOpen) Descriptor() ([]byte, []int) {
	return fileDescriptor_e0f5d4cf0fc9e41b, []int{0}
}

func (m *OrderOpen) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_OrderOpen.Unmarshal(m, b)
}
func (m *OrderOpen) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_OrderOpen.Marshal(b, m, deterministic)
}
func (m *OrderOpen) XXX_Merge(src proto.Message) {
	xxx_messageInfo_OrderOpen.Merge(m, src)
}
func (m *OrderOpen) XXX_Size() int {
	return xxx_messageInfo_OrderOpen.Size(m)
}
func (m *OrderOpen) XXX_DiscardUnknown() {
	xxx_messageInfo_OrderOpen.DiscardUnknown(m)
}

var xxx_messageInfo_OrderOpen proto.InternalMessageInfo

func (m *OrderOpen) GetListings() []*Listing {
	if m != nil {
		return m.Listings
	}
	return nil
}

func (m *OrderOpen) GetRefundAddress() string {
	if m != nil {
		return m.RefundAddress
	}
	return ""
}

func (m *OrderOpen) GetShipping() *OrderOpen_Shipping {
	if m != nil {
		return m.Shipping
	}
	return nil
}

func (m *OrderOpen) GetBuyerID() *ID {
	if m != nil {
		return m.BuyerID
	}
	return nil
}

func (m *OrderOpen) GetTimestamp() *timestamp.Timestamp {
	if m != nil {
		return m.Timestamp
	}
	return nil
}

func (m *OrderOpen) GetItems() []*OrderOpen_Item {
	if m != nil {
		return m.Items
	}
	return nil
}

func (m *OrderOpen) GetPayment() *OrderOpen_Payment {
	if m != nil {
		return m.Payment
	}
	return nil
}

func (m *OrderOpen) GetRatingKeys() [][]byte {
	if m != nil {
		return m.RatingKeys
	}
	return nil
}

func (m *OrderOpen) GetAlternateContactInfo() string {
	if m != nil {
		return m.AlternateContactInfo
	}
	return ""
}

func (m *OrderOpen) GetVersion() uint32 {
	if m != nil {
		return m.Version
	}
	return 0
}

func (m *OrderOpen) GetSignature() []byte {
	if m != nil {
		return m.Signature
	}
	return nil
}

type OrderOpen_Shipping struct {
	ShipTo               string      `protobuf:"bytes,1,opt,name=shipTo,proto3" json:"shipTo,omitempty"`
	Address              string      `protobuf:"bytes,2,opt,name=address,proto3" json:"address,omitempty"`
	City                 string      `protobuf:"bytes,3,opt,name=city,proto3" json:"city,omitempty"`
	State                string      `protobuf:"bytes,4,opt,name=state,proto3" json:"state,omitempty"`
	PostalCode           string      `protobuf:"bytes,5,opt,name=postalCode,proto3" json:"postalCode,omitempty"`
	Country              CountryCode `protobuf:"varint,6,opt,name=country,proto3,enum=CountryCode" json:"country,omitempty"`
	AddressNotes         string      `protobuf:"bytes,7,opt,name=addressNotes,proto3" json:"addressNotes,omitempty"`
	XXX_NoUnkeyedLiteral struct{}    `json:"-"`
	XXX_unrecognized     []byte      `json:"-"`
	XXX_sizecache        int32       `json:"-"`
}

func (m *OrderOpen_Shipping) Reset()         { *m = OrderOpen_Shipping{} }
func (m *OrderOpen_Shipping) String() string { return proto.CompactTextString(m) }
func (*OrderOpen_Shipping) ProtoMessage()    {}
func (*OrderOpen_Shipping) Descriptor() ([]byte, []int) {
	return fileDescriptor_e0f5d4cf0fc9e41b, []int{0, 0}
}

func (m *OrderOpen_Shipping) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_OrderOpen_Shipping.Unmarshal(m, b)
}
func (m *OrderOpen_Shipping) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_OrderOpen_Shipping.Marshal(b, m, deterministic)
}
func (m *OrderOpen_Shipping) XXX_Merge(src proto.Message) {
	xxx_messageInfo_OrderOpen_Shipping.Merge(m, src)
}
func (m *OrderOpen_Shipping) XXX_Size() int {
	return xxx_messageInfo_OrderOpen_Shipping.Size(m)
}
func (m *OrderOpen_Shipping) XXX_DiscardUnknown() {
	xxx_messageInfo_OrderOpen_Shipping.DiscardUnknown(m)
}

var xxx_messageInfo_OrderOpen_Shipping proto.InternalMessageInfo

func (m *OrderOpen_Shipping) GetShipTo() string {
	if m != nil {
		return m.ShipTo
	}
	return ""
}

func (m *OrderOpen_Shipping) GetAddress() string {
	if m != nil {
		return m.Address
	}
	return ""
}

func (m *OrderOpen_Shipping) GetCity() string {
	if m != nil {
		return m.City
	}
	return ""
}

func (m *OrderOpen_Shipping) GetState() string {
	if m != nil {
		return m.State
	}
	return ""
}

func (m *OrderOpen_Shipping) GetPostalCode() string {
	if m != nil {
		return m.PostalCode
	}
	return ""
}

func (m *OrderOpen_Shipping) GetCountry() CountryCode {
	if m != nil {
		return m.Country
	}
	return CountryCode_NA
}

func (m *OrderOpen_Shipping) GetAddressNotes() string {
	if m != nil {
		return m.AddressNotes
	}
	return ""
}

type OrderOpen_Item struct {
	ListingHash          string                         `protobuf:"bytes,1,opt,name=listingHash,proto3" json:"listingHash,omitempty"`
	Quantity             uint64                         `protobuf:"varint,2,opt,name=quantity,proto3" json:"quantity,omitempty"`
	Options              []*OrderOpen_Item_Option       `protobuf:"bytes,3,rep,name=options,proto3" json:"options,omitempty"`
	ShippingOption       *OrderOpen_Item_ShippingOption `protobuf:"bytes,4,opt,name=shippingOption,proto3" json:"shippingOption,omitempty"`
	Memo                 string                         `protobuf:"bytes,5,opt,name=memo,proto3" json:"memo,omitempty"`
	CouponCodes          []string                       `protobuf:"bytes,6,rep,name=couponCodes,proto3" json:"couponCodes,omitempty"`
	PaymentAddress       string                         `protobuf:"bytes,7,opt,name=paymentAddress,proto3" json:"paymentAddress,omitempty"`
	XXX_NoUnkeyedLiteral struct{}                       `json:"-"`
	XXX_unrecognized     []byte                         `json:"-"`
	XXX_sizecache        int32                          `json:"-"`
}

func (m *OrderOpen_Item) Reset()         { *m = OrderOpen_Item{} }
func (m *OrderOpen_Item) String() string { return proto.CompactTextString(m) }
func (*OrderOpen_Item) ProtoMessage()    {}
func (*OrderOpen_Item) Descriptor() ([]byte, []int) {
	return fileDescriptor_e0f5d4cf0fc9e41b, []int{0, 1}
}

func (m *OrderOpen_Item) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_OrderOpen_Item.Unmarshal(m, b)
}
func (m *OrderOpen_Item) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_OrderOpen_Item.Marshal(b, m, deterministic)
}
func (m *OrderOpen_Item) XXX_Merge(src proto.Message) {
	xxx_messageInfo_OrderOpen_Item.Merge(m, src)
}
func (m *OrderOpen_Item) XXX_Size() int {
	return xxx_messageInfo_OrderOpen_Item.Size(m)
}
func (m *OrderOpen_Item) XXX_DiscardUnknown() {
	xxx_messageInfo_OrderOpen_Item.DiscardUnknown(m)
}

var xxx_messageInfo_OrderOpen_Item proto.InternalMessageInfo

func (m *OrderOpen_Item) GetListingHash() string {
	if m != nil {
		return m.ListingHash
	}
	return ""
}

func (m *OrderOpen_Item) GetQuantity() uint64 {
	if m != nil {
		return m.Quantity
	}
	return 0
}

func (m *OrderOpen_Item) GetOptions() []*OrderOpen_Item_Option {
	if m != nil {
		return m.Options
	}
	return nil
}

func (m *OrderOpen_Item) GetShippingOption() *OrderOpen_Item_ShippingOption {
	if m != nil {
		return m.ShippingOption
	}
	return nil
}

func (m *OrderOpen_Item) GetMemo() string {
	if m != nil {
		return m.Memo
	}
	return ""
}

func (m *OrderOpen_Item) GetCouponCodes() []string {
	if m != nil {
		return m.CouponCodes
	}
	return nil
}

func (m *OrderOpen_Item) GetPaymentAddress() string {
	if m != nil {
		return m.PaymentAddress
	}
	return ""
}

type OrderOpen_Item_Option struct {
	Name                 string   `protobuf:"bytes,1,opt,name=name,proto3" json:"name,omitempty"`
	Value                string   `protobuf:"bytes,2,opt,name=value,proto3" json:"value,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *OrderOpen_Item_Option) Reset()         { *m = OrderOpen_Item_Option{} }
func (m *OrderOpen_Item_Option) String() string { return proto.CompactTextString(m) }
func (*OrderOpen_Item_Option) ProtoMessage()    {}
func (*OrderOpen_Item_Option) Descriptor() ([]byte, []int) {
	return fileDescriptor_e0f5d4cf0fc9e41b, []int{0, 1, 0}
}

func (m *OrderOpen_Item_Option) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_OrderOpen_Item_Option.Unmarshal(m, b)
}
func (m *OrderOpen_Item_Option) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_OrderOpen_Item_Option.Marshal(b, m, deterministic)
}
func (m *OrderOpen_Item_Option) XXX_Merge(src proto.Message) {
	xxx_messageInfo_OrderOpen_Item_Option.Merge(m, src)
}
func (m *OrderOpen_Item_Option) XXX_Size() int {
	return xxx_messageInfo_OrderOpen_Item_Option.Size(m)
}
func (m *OrderOpen_Item_Option) XXX_DiscardUnknown() {
	xxx_messageInfo_OrderOpen_Item_Option.DiscardUnknown(m)
}

var xxx_messageInfo_OrderOpen_Item_Option proto.InternalMessageInfo

func (m *OrderOpen_Item_Option) GetName() string {
	if m != nil {
		return m.Name
	}
	return ""
}

func (m *OrderOpen_Item_Option) GetValue() string {
	if m != nil {
		return m.Value
	}
	return ""
}

type OrderOpen_Item_ShippingOption struct {
	Name                 string   `protobuf:"bytes,1,opt,name=name,proto3" json:"name,omitempty"`
	Service              string   `protobuf:"bytes,2,opt,name=service,proto3" json:"service,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *OrderOpen_Item_ShippingOption) Reset()         { *m = OrderOpen_Item_ShippingOption{} }
func (m *OrderOpen_Item_ShippingOption) String() string { return proto.CompactTextString(m) }
func (*OrderOpen_Item_ShippingOption) ProtoMessage()    {}
func (*OrderOpen_Item_ShippingOption) Descriptor() ([]byte, []int) {
	return fileDescriptor_e0f5d4cf0fc9e41b, []int{0, 1, 1}
}

func (m *OrderOpen_Item_ShippingOption) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_OrderOpen_Item_ShippingOption.Unmarshal(m, b)
}
func (m *OrderOpen_Item_ShippingOption) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_OrderOpen_Item_ShippingOption.Marshal(b, m, deterministic)
}
func (m *OrderOpen_Item_ShippingOption) XXX_Merge(src proto.Message) {
	xxx_messageInfo_OrderOpen_Item_ShippingOption.Merge(m, src)
}
func (m *OrderOpen_Item_ShippingOption) XXX_Size() int {
	return xxx_messageInfo_OrderOpen_Item_ShippingOption.Size(m)
}
func (m *OrderOpen_Item_ShippingOption) XXX_DiscardUnknown() {
	xxx_messageInfo_OrderOpen_Item_ShippingOption.DiscardUnknown(m)
}

var xxx_messageInfo_OrderOpen_Item_ShippingOption proto.InternalMessageInfo

func (m *OrderOpen_Item_ShippingOption) GetName() string {
	if m != nil {
		return m.Name
	}
	return ""
}

func (m *OrderOpen_Item_ShippingOption) GetService() string {
	if m != nil {
		return m.Service
	}
	return ""
}

type OrderOpen_Payment struct {
	Method                OrderOpen_Payment_Method `protobuf:"varint,1,opt,name=method,proto3,enum=OrderOpen_Payment_Method" json:"method,omitempty"`
	Moderator             string                   `protobuf:"bytes,2,opt,name=moderator,proto3" json:"moderator,omitempty"`
	Amount                string                   `protobuf:"bytes,3,opt,name=amount,proto3" json:"amount,omitempty"`
	Chaincode             string                   `protobuf:"bytes,4,opt,name=chaincode,proto3" json:"chaincode,omitempty"`
	Address               string                   `protobuf:"bytes,5,opt,name=address,proto3" json:"address,omitempty"`
	AdditionalAddressData string                   `protobuf:"bytes,6,opt,name=additionalAddressData,proto3" json:"additionalAddressData,omitempty"`
	ModeratorKey          []byte                   `protobuf:"bytes,7,opt,name=moderatorKey,proto3" json:"moderatorKey,omitempty"`
	Coin                  string                   `protobuf:"bytes,8,opt,name=coin,proto3" json:"coin,omitempty"`
	EscrowReleaseFee      string                   `protobuf:"bytes,9,opt,name=escrowReleaseFee,proto3" json:"escrowReleaseFee,omitempty"`
	XXX_NoUnkeyedLiteral  struct{}                 `json:"-"`
	XXX_unrecognized      []byte                   `json:"-"`
	XXX_sizecache         int32                    `json:"-"`
}

func (m *OrderOpen_Payment) Reset()         { *m = OrderOpen_Payment{} }
func (m *OrderOpen_Payment) String() string { return proto.CompactTextString(m) }
func (*OrderOpen_Payment) ProtoMessage()    {}
func (*OrderOpen_Payment) Descriptor() ([]byte, []int) {
	return fileDescriptor_e0f5d4cf0fc9e41b, []int{0, 2}
}

func (m *OrderOpen_Payment) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_OrderOpen_Payment.Unmarshal(m, b)
}
func (m *OrderOpen_Payment) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_OrderOpen_Payment.Marshal(b, m, deterministic)
}
func (m *OrderOpen_Payment) XXX_Merge(src proto.Message) {
	xxx_messageInfo_OrderOpen_Payment.Merge(m, src)
}
func (m *OrderOpen_Payment) XXX_Size() int {
	return xxx_messageInfo_OrderOpen_Payment.Size(m)
}
func (m *OrderOpen_Payment) XXX_DiscardUnknown() {
	xxx_messageInfo_OrderOpen_Payment.DiscardUnknown(m)
}

var xxx_messageInfo_OrderOpen_Payment proto.InternalMessageInfo

func (m *OrderOpen_Payment) GetMethod() OrderOpen_Payment_Method {
	if m != nil {
		return m.Method
	}
	return OrderOpen_Payment_DIRECT
}

func (m *OrderOpen_Payment) GetModerator() string {
	if m != nil {
		return m.Moderator
	}
	return ""
}

func (m *OrderOpen_Payment) GetAmount() string {
	if m != nil {
		return m.Amount
	}
	return ""
}

func (m *OrderOpen_Payment) GetChaincode() string {
	if m != nil {
		return m.Chaincode
	}
	return ""
}

func (m *OrderOpen_Payment) GetAddress() string {
	if m != nil {
		return m.Address
	}
	return ""
}

func (m *OrderOpen_Payment) GetAdditionalAddressData() string {
	if m != nil {
		return m.AdditionalAddressData
	}
	return ""
}

func (m *OrderOpen_Payment) GetModeratorKey() []byte {
	if m != nil {
		return m.ModeratorKey
	}
	return nil
}

func (m *OrderOpen_Payment) GetCoin() string {
	if m != nil {
		return m.Coin
	}
	return ""
}

func (m *OrderOpen_Payment) GetEscrowReleaseFee() string {
	if m != nil {
		return m.EscrowReleaseFee
	}
	return ""
}

type OrderReject struct {
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *OrderReject) Reset()         { *m = OrderReject{} }
func (m *OrderReject) String() string { return proto.CompactTextString(m) }
func (*OrderReject) ProtoMessage()    {}
func (*OrderReject) Descriptor() ([]byte, []int) {
	return fileDescriptor_e0f5d4cf0fc9e41b, []int{1}
}

func (m *OrderReject) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_OrderReject.Unmarshal(m, b)
}
func (m *OrderReject) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_OrderReject.Marshal(b, m, deterministic)
}
func (m *OrderReject) XXX_Merge(src proto.Message) {
	xxx_messageInfo_OrderReject.Merge(m, src)
}
func (m *OrderReject) XXX_Size() int {
	return xxx_messageInfo_OrderReject.Size(m)
}
func (m *OrderReject) XXX_DiscardUnknown() {
	xxx_messageInfo_OrderReject.DiscardUnknown(m)
}

var xxx_messageInfo_OrderReject proto.InternalMessageInfo

type OrderCancel struct {
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *OrderCancel) Reset()         { *m = OrderCancel{} }
func (m *OrderCancel) String() string { return proto.CompactTextString(m) }
func (*OrderCancel) ProtoMessage()    {}
func (*OrderCancel) Descriptor() ([]byte, []int) {
	return fileDescriptor_e0f5d4cf0fc9e41b, []int{2}
}

func (m *OrderCancel) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_OrderCancel.Unmarshal(m, b)
}
func (m *OrderCancel) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_OrderCancel.Marshal(b, m, deterministic)
}
func (m *OrderCancel) XXX_Merge(src proto.Message) {
	xxx_messageInfo_OrderCancel.Merge(m, src)
}
func (m *OrderCancel) XXX_Size() int {
	return xxx_messageInfo_OrderCancel.Size(m)
}
func (m *OrderCancel) XXX_DiscardUnknown() {
	xxx_messageInfo_OrderCancel.DiscardUnknown(m)
}

var xxx_messageInfo_OrderCancel proto.InternalMessageInfo

type OrderConfirmation struct {
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *OrderConfirmation) Reset()         { *m = OrderConfirmation{} }
func (m *OrderConfirmation) String() string { return proto.CompactTextString(m) }
func (*OrderConfirmation) ProtoMessage()    {}
func (*OrderConfirmation) Descriptor() ([]byte, []int) {
	return fileDescriptor_e0f5d4cf0fc9e41b, []int{3}
}

func (m *OrderConfirmation) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_OrderConfirmation.Unmarshal(m, b)
}
func (m *OrderConfirmation) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_OrderConfirmation.Marshal(b, m, deterministic)
}
func (m *OrderConfirmation) XXX_Merge(src proto.Message) {
	xxx_messageInfo_OrderConfirmation.Merge(m, src)
}
func (m *OrderConfirmation) XXX_Size() int {
	return xxx_messageInfo_OrderConfirmation.Size(m)
}
func (m *OrderConfirmation) XXX_DiscardUnknown() {
	xxx_messageInfo_OrderConfirmation.DiscardUnknown(m)
}

var xxx_messageInfo_OrderConfirmation proto.InternalMessageInfo

type OrderFulfillment struct {
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *OrderFulfillment) Reset()         { *m = OrderFulfillment{} }
func (m *OrderFulfillment) String() string { return proto.CompactTextString(m) }
func (*OrderFulfillment) ProtoMessage()    {}
func (*OrderFulfillment) Descriptor() ([]byte, []int) {
	return fileDescriptor_e0f5d4cf0fc9e41b, []int{4}
}

func (m *OrderFulfillment) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_OrderFulfillment.Unmarshal(m, b)
}
func (m *OrderFulfillment) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_OrderFulfillment.Marshal(b, m, deterministic)
}
func (m *OrderFulfillment) XXX_Merge(src proto.Message) {
	xxx_messageInfo_OrderFulfillment.Merge(m, src)
}
func (m *OrderFulfillment) XXX_Size() int {
	return xxx_messageInfo_OrderFulfillment.Size(m)
}
func (m *OrderFulfillment) XXX_DiscardUnknown() {
	xxx_messageInfo_OrderFulfillment.DiscardUnknown(m)
}

var xxx_messageInfo_OrderFulfillment proto.InternalMessageInfo

type OrderComplete struct {
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *OrderComplete) Reset()         { *m = OrderComplete{} }
func (m *OrderComplete) String() string { return proto.CompactTextString(m) }
func (*OrderComplete) ProtoMessage()    {}
func (*OrderComplete) Descriptor() ([]byte, []int) {
	return fileDescriptor_e0f5d4cf0fc9e41b, []int{5}
}

func (m *OrderComplete) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_OrderComplete.Unmarshal(m, b)
}
func (m *OrderComplete) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_OrderComplete.Marshal(b, m, deterministic)
}
func (m *OrderComplete) XXX_Merge(src proto.Message) {
	xxx_messageInfo_OrderComplete.Merge(m, src)
}
func (m *OrderComplete) XXX_Size() int {
	return xxx_messageInfo_OrderComplete.Size(m)
}
func (m *OrderComplete) XXX_DiscardUnknown() {
	xxx_messageInfo_OrderComplete.DiscardUnknown(m)
}

var xxx_messageInfo_OrderComplete proto.InternalMessageInfo

type DisputeOpen struct {
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *DisputeOpen) Reset()         { *m = DisputeOpen{} }
func (m *DisputeOpen) String() string { return proto.CompactTextString(m) }
func (*DisputeOpen) ProtoMessage()    {}
func (*DisputeOpen) Descriptor() ([]byte, []int) {
	return fileDescriptor_e0f5d4cf0fc9e41b, []int{6}
}

func (m *DisputeOpen) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_DisputeOpen.Unmarshal(m, b)
}
func (m *DisputeOpen) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_DisputeOpen.Marshal(b, m, deterministic)
}
func (m *DisputeOpen) XXX_Merge(src proto.Message) {
	xxx_messageInfo_DisputeOpen.Merge(m, src)
}
func (m *DisputeOpen) XXX_Size() int {
	return xxx_messageInfo_DisputeOpen.Size(m)
}
func (m *DisputeOpen) XXX_DiscardUnknown() {
	xxx_messageInfo_DisputeOpen.DiscardUnknown(m)
}

var xxx_messageInfo_DisputeOpen proto.InternalMessageInfo

type DisputeUpdate struct {
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *DisputeUpdate) Reset()         { *m = DisputeUpdate{} }
func (m *DisputeUpdate) String() string { return proto.CompactTextString(m) }
func (*DisputeUpdate) ProtoMessage()    {}
func (*DisputeUpdate) Descriptor() ([]byte, []int) {
	return fileDescriptor_e0f5d4cf0fc9e41b, []int{7}
}

func (m *DisputeUpdate) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_DisputeUpdate.Unmarshal(m, b)
}
func (m *DisputeUpdate) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_DisputeUpdate.Marshal(b, m, deterministic)
}
func (m *DisputeUpdate) XXX_Merge(src proto.Message) {
	xxx_messageInfo_DisputeUpdate.Merge(m, src)
}
func (m *DisputeUpdate) XXX_Size() int {
	return xxx_messageInfo_DisputeUpdate.Size(m)
}
func (m *DisputeUpdate) XXX_DiscardUnknown() {
	xxx_messageInfo_DisputeUpdate.DiscardUnknown(m)
}

var xxx_messageInfo_DisputeUpdate proto.InternalMessageInfo

type DisputeClose struct {
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *DisputeClose) Reset()         { *m = DisputeClose{} }
func (m *DisputeClose) String() string { return proto.CompactTextString(m) }
func (*DisputeClose) ProtoMessage()    {}
func (*DisputeClose) Descriptor() ([]byte, []int) {
	return fileDescriptor_e0f5d4cf0fc9e41b, []int{8}
}

func (m *DisputeClose) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_DisputeClose.Unmarshal(m, b)
}
func (m *DisputeClose) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_DisputeClose.Marshal(b, m, deterministic)
}
func (m *DisputeClose) XXX_Merge(src proto.Message) {
	xxx_messageInfo_DisputeClose.Merge(m, src)
}
func (m *DisputeClose) XXX_Size() int {
	return xxx_messageInfo_DisputeClose.Size(m)
}
func (m *DisputeClose) XXX_DiscardUnknown() {
	xxx_messageInfo_DisputeClose.DiscardUnknown(m)
}

var xxx_messageInfo_DisputeClose proto.InternalMessageInfo

type Refund struct {
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *Refund) Reset()         { *m = Refund{} }
func (m *Refund) String() string { return proto.CompactTextString(m) }
func (*Refund) ProtoMessage()    {}
func (*Refund) Descriptor() ([]byte, []int) {
	return fileDescriptor_e0f5d4cf0fc9e41b, []int{9}
}

func (m *Refund) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_Refund.Unmarshal(m, b)
}
func (m *Refund) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_Refund.Marshal(b, m, deterministic)
}
func (m *Refund) XXX_Merge(src proto.Message) {
	xxx_messageInfo_Refund.Merge(m, src)
}
func (m *Refund) XXX_Size() int {
	return xxx_messageInfo_Refund.Size(m)
}
func (m *Refund) XXX_DiscardUnknown() {
	xxx_messageInfo_Refund.DiscardUnknown(m)
}

var xxx_messageInfo_Refund proto.InternalMessageInfo

type PaymentSent struct {
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *PaymentSent) Reset()         { *m = PaymentSent{} }
func (m *PaymentSent) String() string { return proto.CompactTextString(m) }
func (*PaymentSent) ProtoMessage()    {}
func (*PaymentSent) Descriptor() ([]byte, []int) {
	return fileDescriptor_e0f5d4cf0fc9e41b, []int{10}
}

func (m *PaymentSent) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_PaymentSent.Unmarshal(m, b)
}
func (m *PaymentSent) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_PaymentSent.Marshal(b, m, deterministic)
}
func (m *PaymentSent) XXX_Merge(src proto.Message) {
	xxx_messageInfo_PaymentSent.Merge(m, src)
}
func (m *PaymentSent) XXX_Size() int {
	return xxx_messageInfo_PaymentSent.Size(m)
}
func (m *PaymentSent) XXX_DiscardUnknown() {
	xxx_messageInfo_PaymentSent.DiscardUnknown(m)
}

var xxx_messageInfo_PaymentSent proto.InternalMessageInfo

type PaymentFinalized struct {
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *PaymentFinalized) Reset()         { *m = PaymentFinalized{} }
func (m *PaymentFinalized) String() string { return proto.CompactTextString(m) }
func (*PaymentFinalized) ProtoMessage()    {}
func (*PaymentFinalized) Descriptor() ([]byte, []int) {
	return fileDescriptor_e0f5d4cf0fc9e41b, []int{11}
}

func (m *PaymentFinalized) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_PaymentFinalized.Unmarshal(m, b)
}
func (m *PaymentFinalized) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_PaymentFinalized.Marshal(b, m, deterministic)
}
func (m *PaymentFinalized) XXX_Merge(src proto.Message) {
	xxx_messageInfo_PaymentFinalized.Merge(m, src)
}
func (m *PaymentFinalized) XXX_Size() int {
	return xxx_messageInfo_PaymentFinalized.Size(m)
}
func (m *PaymentFinalized) XXX_DiscardUnknown() {
	xxx_messageInfo_PaymentFinalized.DiscardUnknown(m)
}

var xxx_messageInfo_PaymentFinalized proto.InternalMessageInfo

func init() {
	proto.RegisterEnum("OrderOpen_Payment_Method", OrderOpen_Payment_Method_name, OrderOpen_Payment_Method_value)
	proto.RegisterType((*OrderOpen)(nil), "OrderOpen")
	proto.RegisterType((*OrderOpen_Shipping)(nil), "OrderOpen.Shipping")
	proto.RegisterType((*OrderOpen_Item)(nil), "OrderOpen.Item")
	proto.RegisterType((*OrderOpen_Item_Option)(nil), "OrderOpen.Item.Option")
	proto.RegisterType((*OrderOpen_Item_ShippingOption)(nil), "OrderOpen.Item.ShippingOption")
	proto.RegisterType((*OrderOpen_Payment)(nil), "OrderOpen.Payment")
	proto.RegisterType((*OrderReject)(nil), "OrderReject")
	proto.RegisterType((*OrderCancel)(nil), "OrderCancel")
	proto.RegisterType((*OrderConfirmation)(nil), "OrderConfirmation")
	proto.RegisterType((*OrderFulfillment)(nil), "OrderFulfillment")
	proto.RegisterType((*OrderComplete)(nil), "OrderComplete")
	proto.RegisterType((*DisputeOpen)(nil), "DisputeOpen")
	proto.RegisterType((*DisputeUpdate)(nil), "DisputeUpdate")
	proto.RegisterType((*DisputeClose)(nil), "DisputeClose")
	proto.RegisterType((*Refund)(nil), "Refund")
	proto.RegisterType((*PaymentSent)(nil), "PaymentSent")
	proto.RegisterType((*PaymentFinalized)(nil), "PaymentFinalized")
}

func init() { proto.RegisterFile("orders.proto", fileDescriptor_e0f5d4cf0fc9e41b) }

var fileDescriptor_e0f5d4cf0fc9e41b = []byte{
	// 868 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x6c, 0x54, 0x5d, 0x6f, 0xe3, 0x44,
	0x14, 0xc5, 0x4d, 0xea, 0xc4, 0x37, 0x1f, 0x1b, 0x66, 0x97, 0x95, 0x89, 0x60, 0x89, 0xac, 0x65,
	0x15, 0x21, 0xe4, 0x42, 0x96, 0x07, 0x9e, 0x90, 0xba, 0x49, 0x2a, 0xa2, 0xed, 0x6e, 0xd1, 0xb4,
	0xbc, 0xf0, 0x36, 0xb5, 0x6f, 0xd2, 0x41, 0xf6, 0x8c, 0xf1, 0x8c, 0x8b, 0xc2, 0xaf, 0xe0, 0x57,
	0xf0, 0x83, 0xf8, 0x37, 0xbc, 0xa1, 0x19, 0x8f, 0xf3, 0xd1, 0xf6, 0xcd, 0xe7, 0xdc, 0x73, 0x3d,
	0xbe, 0xe7, 0x9e, 0x31, 0xf4, 0x65, 0x99, 0x62, 0xa9, 0xe2, 0xa2, 0x94, 0x5a, 0x8e, 0xbf, 0xda,
	0x48, 0xb9, 0xc9, 0xf0, 0xcc, 0xa2, 0xdb, 0x6a, 0x7d, 0xa6, 0x79, 0x8e, 0x4a, 0xb3, 0xbc, 0x70,
	0x02, 0x92, 0xc8, 0x4a, 0xe8, 0x72, 0x9b, 0xc8, 0x14, 0x9b, 0xa6, 0x41, 0xc6, 0x95, 0xe6, 0x62,
	0xe3, 0x60, 0x3f, 0x91, 0x79, 0x2e, 0x45, 0x8d, 0xa2, 0x7f, 0x7a, 0x10, 0x5c, 0x99, 0x23, 0xae,
	0x0a, 0x14, 0xe4, 0x35, 0x74, 0x9d, 0x58, 0x85, 0xde, 0xa4, 0x35, 0xed, 0xcd, 0xba, 0xf1, 0x65,
	0x4d, 0xd0, 0x5d, 0x85, 0xbc, 0x86, 0x41, 0x89, 0xeb, 0x4a, 0xa4, 0xe7, 0x69, 0x5a, 0xa2, 0x52,
	0xe1, 0xc9, 0xc4, 0x9b, 0x06, 0xf4, 0x98, 0x24, 0x67, 0xd0, 0x55, 0x77, 0xbc, 0x28, 0xb8, 0xd8,
	0x84, 0xad, 0x89, 0x37, 0xed, 0xcd, 0x9e, 0xc7, 0xbb, 0x93, 0xe2, 0x6b, 0x57, 0xa2, 0x3b, 0x11,
	0xf9, 0x12, 0x3a, 0xb7, 0xd5, 0x16, 0xcb, 0xd5, 0x22, 0x6c, 0x5b, 0x7d, 0x2b, 0x5e, 0x2d, 0x68,
	0xc3, 0x91, 0x1f, 0x21, 0xd8, 0x4d, 0x1b, 0x9e, 0x5a, 0xc1, 0x38, 0xae, 0xfd, 0x88, 0x1b, 0x3f,
	0xe2, 0x9b, 0x46, 0x41, 0xf7, 0x62, 0xf2, 0x35, 0x9c, 0x72, 0x8d, 0xb9, 0x0a, 0x7d, 0x3b, 0xd2,
	0xb3, 0x83, 0xcf, 0x58, 0x69, 0xcc, 0x69, 0x5d, 0x25, 0xdf, 0x42, 0xa7, 0x60, 0xdb, 0x1c, 0x85,
	0x0e, 0x3b, 0xf6, 0xf5, 0xe4, 0x40, 0xf8, 0x4b, 0x5d, 0xa1, 0x8d, 0x84, 0xbc, 0x02, 0x28, 0x99,
	0xf1, 0xe3, 0x3d, 0x6e, 0x55, 0xd8, 0x9d, 0xb4, 0xa6, 0x7d, 0x7a, 0xc0, 0x90, 0x19, 0xbc, 0x60,
	0x99, 0xc6, 0x52, 0x30, 0x8d, 0x73, 0x29, 0x34, 0x4b, 0xf4, 0x4a, 0xac, 0x65, 0x18, 0x58, 0xaf,
	0x9e, 0xac, 0x91, 0x10, 0x3a, 0xf7, 0x58, 0x2a, 0x2e, 0x45, 0x08, 0x13, 0x6f, 0x3a, 0xa0, 0x0d,
	0x24, 0x5f, 0x40, 0xa0, 0xf8, 0x46, 0x30, 0x5d, 0x95, 0x18, 0xf6, 0x26, 0xde, 0xb4, 0x4f, 0xf7,
	0xc4, 0xf8, 0x5f, 0x0f, 0xba, 0x8d, 0xa1, 0xe4, 0x25, 0xf8, 0xc6, 0xd2, 0x1b, 0x19, 0x7a, 0xf6,
	0x28, 0x87, 0xcc, 0xcb, 0xd9, 0xd1, 0xbe, 0x1a, 0x48, 0x08, 0xb4, 0x13, 0xae, 0xb7, 0x76, 0x4b,
	0x01, 0xb5, 0xcf, 0xe4, 0x05, 0x9c, 0x2a, 0xcd, 0x34, 0xda, 0x55, 0x04, 0xb4, 0x06, 0x66, 0xe8,
	0x42, 0x2a, 0xcd, 0xb2, 0xb9, 0x4c, 0xd1, 0x2e, 0x21, 0xa0, 0x07, 0x0c, 0x79, 0x03, 0x1d, 0x17,
	0xc0, 0xd0, 0x9f, 0x78, 0xd3, 0xe1, 0xac, 0x1f, 0xcf, 0x6b, 0x6c, 0xca, 0xb4, 0x29, 0x92, 0x08,
	0xfa, 0xee, 0xf0, 0x8f, 0x52, 0xa3, 0xb2, 0x7e, 0x07, 0xf4, 0x88, 0x1b, 0xff, 0xdd, 0x82, 0xb6,
	0x59, 0x0f, 0x99, 0x40, 0xcf, 0x45, 0xef, 0x67, 0xa6, 0xee, 0xdc, 0x54, 0x87, 0x14, 0x19, 0x43,
	0xf7, 0x8f, 0x8a, 0x09, 0x6d, 0x86, 0x30, 0xb3, 0xb5, 0xe9, 0x0e, 0x93, 0xef, 0xa0, 0x23, 0x0b,
	0xcd, 0xa5, 0x50, 0x61, 0xcb, 0xae, 0xff, 0xe5, 0x83, 0xf5, 0xc7, 0x57, 0xb6, 0x4c, 0x1b, 0x19,
	0xb9, 0x80, 0x61, 0x93, 0xc9, 0xba, 0xe4, 0xe2, 0xf8, 0xea, 0x61, 0xe3, 0xf5, 0x91, 0x8a, 0x3e,
	0xe8, 0x32, 0xb6, 0xe6, 0x98, 0x4b, 0x67, 0x93, 0x7d, 0x36, 0xb3, 0x24, 0xb2, 0x2a, 0xa4, 0x30,
	0x7e, 0xd4, 0x81, 0x0c, 0xe8, 0x21, 0x45, 0xde, 0xc0, 0xd0, 0x45, 0xac, 0xb9, 0x5d, 0xb5, 0x39,
	0x0f, 0xd8, 0xf1, 0x0c, 0xfc, 0xfd, 0x39, 0x82, 0xe5, 0xe8, 0x8c, 0xb1, 0xcf, 0x66, 0x7d, 0xf7,
	0x2c, 0xab, 0xd0, 0xad, 0xba, 0x06, 0xe3, 0x9f, 0x60, 0x78, 0xfd, 0xe8, 0x1b, 0x1f, 0xf5, 0x86,
	0xd0, 0x51, 0x58, 0xde, 0xf3, 0xa4, 0xe9, 0x6e, 0xe0, 0xf8, 0xbf, 0x13, 0xe8, 0xb8, 0x8b, 0x40,
	0xbe, 0x07, 0x3f, 0x47, 0x7d, 0x27, 0x53, 0xdb, 0x3b, 0x9c, 0x7d, 0xfe, 0xf8, 0xb2, 0xc4, 0x1f,
	0xac, 0x80, 0x3a, 0xa1, 0x09, 0x71, 0x2e, 0x53, 0x2c, 0x99, 0x96, 0xa5, 0x7b, 0xf5, 0x9e, 0x30,
	0xb9, 0x65, 0xb9, 0xc9, 0x87, 0xcb, 0xa1, 0x43, 0xa6, 0x2b, 0xb9, 0x63, 0x5c, 0x98, 0x5f, 0x9a,
	0x4b, 0xe3, 0x9e, 0x38, 0x4c, 0xf5, 0xe9, 0x71, 0xaa, 0x7f, 0x80, 0xcf, 0x58, 0x9a, 0x72, 0x33,
	0x26, 0xcb, 0x9c, 0x6b, 0x0b, 0xa6, 0x99, 0x4d, 0x66, 0x40, 0x9f, 0x2e, 0x9a, 0x64, 0xee, 0x3e,
	0xe9, 0x3d, 0x6e, 0xad, 0xf9, 0x7d, 0x7a, 0xc4, 0xd9, 0xfb, 0x22, 0xb9, 0x08, 0xbb, 0xee, 0xbe,
	0x48, 0x2e, 0xc8, 0x37, 0x30, 0x42, 0x95, 0x94, 0xf2, 0x4f, 0x8a, 0x19, 0x32, 0x85, 0x17, 0x88,
	0xee, 0xaa, 0x3f, 0xe2, 0xa3, 0xb7, 0xe0, 0xd7, 0xce, 0x10, 0x00, 0x7f, 0xb1, 0xa2, 0xcb, 0xf9,
	0xcd, 0xe8, 0x13, 0x32, 0x04, 0x98, 0x9f, 0x7f, 0x9c, 0x2f, 0x2f, 0xcf, 0xdf, 0x5d, 0x2e, 0x47,
	0x1e, 0x19, 0x40, 0xf0, 0xe1, 0x6a, 0xb1, 0xa4, 0xe7, 0x37, 0xcb, 0xc5, 0xe8, 0x24, 0x1a, 0x40,
	0xcf, 0x1a, 0x4c, 0xf1, 0x77, 0x4c, 0xf4, 0x0e, 0xce, 0x99, 0x48, 0x30, 0x8b, 0x9e, 0xc3, 0xa7,
	0x35, 0x94, 0x62, 0xcd, 0xcb, 0x9c, 0x99, 0xc1, 0x22, 0x02, 0x23, 0x4b, 0x5e, 0x54, 0xd9, 0x9a,
	0x67, 0x99, 0x59, 0x49, 0xf4, 0x0c, 0x06, 0x4e, 0x98, 0x17, 0x19, 0x6a, 0x34, 0x2f, 0x5a, 0x70,
	0x55, 0x54, 0x1a, 0xcd, 0xea, 0x4c, 0xdd, 0xc1, 0x5f, 0x8b, 0x94, 0x69, 0x8c, 0x86, 0xd0, 0x77,
	0xc4, 0x3c, 0x93, 0x0a, 0xa3, 0x2e, 0xf8, 0xd4, 0xfe, 0xe7, 0x4d, 0xa7, 0x5b, 0xf4, 0xb5, 0x79,
	0x33, 0x81, 0x91, 0x83, 0x17, 0x5c, 0xb0, 0x8c, 0xff, 0x85, 0xe9, 0xbb, 0xf6, 0x6f, 0x27, 0xc5,
	0xed, 0xad, 0x6f, 0x7f, 0xcf, 0x6f, 0xff, 0x0f, 0x00, 0x00, 0xff, 0xff, 0x9a, 0xb4, 0x34, 0xf2,
	0xcc, 0x06, 0x00, 0x00,
}
