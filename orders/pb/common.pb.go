// Code generated by protoc-gen-go. DO NOT EDIT.
// source: common.proto

package pb

import (
	fmt "fmt"
	proto "github.com/golang/protobuf/proto"
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

type Currency struct {
	Code                 string   `protobuf:"bytes,1,opt,name=code,proto3" json:"code,omitempty"`
	Divisibility         uint32   `protobuf:"varint,2,opt,name=divisibility,proto3" json:"divisibility,omitempty"`
	Name                 string   `protobuf:"bytes,3,opt,name=name,proto3" json:"name,omitempty"`
	CurrencyType         string   `protobuf:"bytes,4,opt,name=currencyType,proto3" json:"currencyType,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *Currency) Reset()         { *m = Currency{} }
func (m *Currency) String() string { return proto.CompactTextString(m) }
func (*Currency) ProtoMessage()    {}
func (*Currency) Descriptor() ([]byte, []int) {
	return fileDescriptor_555bd8c177793206, []int{0}
}

func (m *Currency) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_Currency.Unmarshal(m, b)
}
func (m *Currency) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_Currency.Marshal(b, m, deterministic)
}
func (m *Currency) XXX_Merge(src proto.Message) {
	xxx_messageInfo_Currency.Merge(m, src)
}
func (m *Currency) XXX_Size() int {
	return xxx_messageInfo_Currency.Size(m)
}
func (m *Currency) XXX_DiscardUnknown() {
	xxx_messageInfo_Currency.DiscardUnknown(m)
}

var xxx_messageInfo_Currency proto.InternalMessageInfo

func (m *Currency) GetCode() string {
	if m != nil {
		return m.Code
	}
	return ""
}

func (m *Currency) GetDivisibility() uint32 {
	if m != nil {
		return m.Divisibility
	}
	return 0
}

func (m *Currency) GetName() string {
	if m != nil {
		return m.Name
	}
	return ""
}

func (m *Currency) GetCurrencyType() string {
	if m != nil {
		return m.CurrencyType
	}
	return ""
}

type CurrencyValue struct {
	Currency             *Currency `protobuf:"bytes,1,opt,name=currency,proto3" json:"currency,omitempty"`
	Amount               string    `protobuf:"bytes,2,opt,name=amount,proto3" json:"amount,omitempty"`
	XXX_NoUnkeyedLiteral struct{}  `json:"-"`
	XXX_unrecognized     []byte    `json:"-"`
	XXX_sizecache        int32     `json:"-"`
}

func (m *CurrencyValue) Reset()         { *m = CurrencyValue{} }
func (m *CurrencyValue) String() string { return proto.CompactTextString(m) }
func (*CurrencyValue) ProtoMessage()    {}
func (*CurrencyValue) Descriptor() ([]byte, []int) {
	return fileDescriptor_555bd8c177793206, []int{1}
}

func (m *CurrencyValue) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_CurrencyValue.Unmarshal(m, b)
}
func (m *CurrencyValue) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_CurrencyValue.Marshal(b, m, deterministic)
}
func (m *CurrencyValue) XXX_Merge(src proto.Message) {
	xxx_messageInfo_CurrencyValue.Merge(m, src)
}
func (m *CurrencyValue) XXX_Size() int {
	return xxx_messageInfo_CurrencyValue.Size(m)
}
func (m *CurrencyValue) XXX_DiscardUnknown() {
	xxx_messageInfo_CurrencyValue.DiscardUnknown(m)
}

var xxx_messageInfo_CurrencyValue proto.InternalMessageInfo

func (m *CurrencyValue) GetCurrency() *Currency {
	if m != nil {
		return m.Currency
	}
	return nil
}

func (m *CurrencyValue) GetAmount() string {
	if m != nil {
		return m.Amount
	}
	return ""
}

type ID struct {
	PeerID               string      `protobuf:"bytes,1,opt,name=peerID,proto3" json:"peerID,omitempty"`
	Handle               string      `protobuf:"bytes,2,opt,name=handle,proto3" json:"handle,omitempty"`
	Pubkeys              *ID_Pubkeys `protobuf:"bytes,3,opt,name=pubkeys,proto3" json:"pubkeys,omitempty"`
	Sig                  []byte      `protobuf:"bytes,4,opt,name=sig,proto3" json:"sig,omitempty"`
	XXX_NoUnkeyedLiteral struct{}    `json:"-"`
	XXX_unrecognized     []byte      `json:"-"`
	XXX_sizecache        int32       `json:"-"`
}

func (m *ID) Reset()         { *m = ID{} }
func (m *ID) String() string { return proto.CompactTextString(m) }
func (*ID) ProtoMessage()    {}
func (*ID) Descriptor() ([]byte, []int) {
	return fileDescriptor_555bd8c177793206, []int{2}
}

func (m *ID) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_ID.Unmarshal(m, b)
}
func (m *ID) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_ID.Marshal(b, m, deterministic)
}
func (m *ID) XXX_Merge(src proto.Message) {
	xxx_messageInfo_ID.Merge(m, src)
}
func (m *ID) XXX_Size() int {
	return xxx_messageInfo_ID.Size(m)
}
func (m *ID) XXX_DiscardUnknown() {
	xxx_messageInfo_ID.DiscardUnknown(m)
}

var xxx_messageInfo_ID proto.InternalMessageInfo

func (m *ID) GetPeerID() string {
	if m != nil {
		return m.PeerID
	}
	return ""
}

func (m *ID) GetHandle() string {
	if m != nil {
		return m.Handle
	}
	return ""
}

func (m *ID) GetPubkeys() *ID_Pubkeys {
	if m != nil {
		return m.Pubkeys
	}
	return nil
}

func (m *ID) GetSig() []byte {
	if m != nil {
		return m.Sig
	}
	return nil
}

type ID_Pubkeys struct {
	Identity             []byte   `protobuf:"bytes,1,opt,name=identity,proto3" json:"identity,omitempty"`
	Secp256K1            []byte   `protobuf:"bytes,2,opt,name=secp256k1,proto3" json:"secp256k1,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *ID_Pubkeys) Reset()         { *m = ID_Pubkeys{} }
func (m *ID_Pubkeys) String() string { return proto.CompactTextString(m) }
func (*ID_Pubkeys) ProtoMessage()    {}
func (*ID_Pubkeys) Descriptor() ([]byte, []int) {
	return fileDescriptor_555bd8c177793206, []int{2, 0}
}

func (m *ID_Pubkeys) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_ID_Pubkeys.Unmarshal(m, b)
}
func (m *ID_Pubkeys) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_ID_Pubkeys.Marshal(b, m, deterministic)
}
func (m *ID_Pubkeys) XXX_Merge(src proto.Message) {
	xxx_messageInfo_ID_Pubkeys.Merge(m, src)
}
func (m *ID_Pubkeys) XXX_Size() int {
	return xxx_messageInfo_ID_Pubkeys.Size(m)
}
func (m *ID_Pubkeys) XXX_DiscardUnknown() {
	xxx_messageInfo_ID_Pubkeys.DiscardUnknown(m)
}

var xxx_messageInfo_ID_Pubkeys proto.InternalMessageInfo

func (m *ID_Pubkeys) GetIdentity() []byte {
	if m != nil {
		return m.Identity
	}
	return nil
}

func (m *ID_Pubkeys) GetSecp256K1() []byte {
	if m != nil {
		return m.Secp256K1
	}
	return nil
}

func init() {
	proto.RegisterType((*Currency)(nil), "Currency")
	proto.RegisterType((*CurrencyValue)(nil), "CurrencyValue")
	proto.RegisterType((*ID)(nil), "ID")
	proto.RegisterType((*ID_Pubkeys)(nil), "ID.Pubkeys")
}

func init() { proto.RegisterFile("common.proto", fileDescriptor_555bd8c177793206) }

var fileDescriptor_555bd8c177793206 = []byte{
	// 274 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x54, 0x51, 0xcd, 0x6a, 0x83, 0x40,
	0x10, 0x46, 0x23, 0x89, 0x8e, 0x06, 0xca, 0x1e, 0x8a, 0x84, 0x1e, 0x82, 0x10, 0xc8, 0x49, 0xa8,
	0xa5, 0x7d, 0x80, 0xc6, 0x8b, 0x97, 0x52, 0x96, 0xd2, 0x43, 0x6f, 0xfe, 0x0c, 0xed, 0x12, 0xdd,
	0x5d, 0xfc, 0x29, 0xd8, 0xc7, 0xea, 0x13, 0x96, 0x5d, 0x77, 0x53, 0x72, 0x9b, 0xef, 0xcf, 0x99,
	0xcf, 0x85, 0xa8, 0x16, 0x5d, 0x27, 0x78, 0x2a, 0x7b, 0x31, 0x8a, 0xe4, 0x07, 0xfc, 0xd3, 0xd4,
	0xf7, 0xc8, 0xeb, 0x99, 0x10, 0xf0, 0x6a, 0xd1, 0x60, 0xec, 0xec, 0x9d, 0x63, 0x40, 0xf5, 0x4c,
	0x12, 0x88, 0x1a, 0xf6, 0xcd, 0x06, 0x56, 0xb1, 0x96, 0x8d, 0x73, 0xec, 0xee, 0x9d, 0xe3, 0x96,
	0x5e, 0x71, 0x2a, 0xc7, 0xcb, 0x0e, 0xe3, 0xd5, 0x92, 0x53, 0xb3, 0xca, 0xd5, 0xe6, 0xbb, 0x6f,
	0xb3, 0xc4, 0xd8, 0xd3, 0xda, 0x15, 0x97, 0xbc, 0xc0, 0xd6, 0xee, 0x7e, 0x2f, 0xdb, 0x09, 0xc9,
	0x01, 0x7c, 0x6b, 0xd0, 0x47, 0x84, 0x59, 0x90, 0x5a, 0x07, 0xbd, 0x48, 0xe4, 0x16, 0xd6, 0x65,
	0x27, 0x26, 0x3e, 0xea, 0x6b, 0x02, 0x6a, 0x50, 0xf2, 0xeb, 0x80, 0x5b, 0xe4, 0x4a, 0x96, 0x88,
	0x7d, 0x91, 0x9b, 0x22, 0x06, 0x29, 0xfe, 0xab, 0xe4, 0x4d, 0x8b, 0x36, 0xb6, 0x20, 0x72, 0x80,
	0x8d, 0x9c, 0xaa, 0x33, 0xce, 0x83, 0x6e, 0x10, 0x66, 0x61, 0x5a, 0xe4, 0xe9, 0xeb, 0x42, 0x51,
	0xab, 0x91, 0x1b, 0x58, 0x0d, 0xec, 0x53, 0x17, 0x89, 0xa8, 0x1a, 0x77, 0x27, 0xd8, 0x18, 0x17,
	0xd9, 0x81, 0xcf, 0x1a, 0xe4, 0xa3, 0xfa, 0x45, 0x8e, 0x76, 0x5c, 0x30, 0xb9, 0x83, 0x60, 0xc0,
	0x5a, 0x66, 0x8f, 0x4f, 0xe7, 0x7b, 0xbd, 0x3a, 0xa2, 0xff, 0xc4, 0xb3, 0xf7, 0xe1, 0xca, 0xaa,
	0x5a, 0xeb, 0xd7, 0x78, 0xf8, 0x0b, 0x00, 0x00, 0xff, 0xff, 0x3e, 0x83, 0x3f, 0x6e, 0x9d, 0x01,
	0x00, 0x00,
}