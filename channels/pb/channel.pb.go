// Code generated by protoc-gen-go. DO NOT EDIT.
// source: channel.proto

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

type ChannelMessage struct {
	Message              string               `protobuf:"bytes,1,opt,name=message,proto3" json:"message,omitempty"`
	Topic                string               `protobuf:"bytes,2,opt,name=topic,proto3" json:"topic,omitempty"`
	PeerID               string               `protobuf:"bytes,3,opt,name=peerID,proto3" json:"peerID,omitempty"`
	Timestamp            *timestamp.Timestamp `protobuf:"bytes,4,opt,name=timestamp,proto3" json:"timestamp,omitempty"`
	Signature            []byte               `protobuf:"bytes,5,opt,name=signature,proto3" json:"signature,omitempty"`
	XXX_NoUnkeyedLiteral struct{}             `json:"-"`
	XXX_unrecognized     []byte               `json:"-"`
	XXX_sizecache        int32                `json:"-"`
}

func (m *ChannelMessage) Reset()         { *m = ChannelMessage{} }
func (m *ChannelMessage) String() string { return proto.CompactTextString(m) }
func (*ChannelMessage) ProtoMessage()    {}
func (*ChannelMessage) Descriptor() ([]byte, []int) {
	return fileDescriptor_c8f385724121f37b, []int{0}
}

func (m *ChannelMessage) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_ChannelMessage.Unmarshal(m, b)
}
func (m *ChannelMessage) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_ChannelMessage.Marshal(b, m, deterministic)
}
func (m *ChannelMessage) XXX_Merge(src proto.Message) {
	xxx_messageInfo_ChannelMessage.Merge(m, src)
}
func (m *ChannelMessage) XXX_Size() int {
	return xxx_messageInfo_ChannelMessage.Size(m)
}
func (m *ChannelMessage) XXX_DiscardUnknown() {
	xxx_messageInfo_ChannelMessage.DiscardUnknown(m)
}

var xxx_messageInfo_ChannelMessage proto.InternalMessageInfo

func (m *ChannelMessage) GetMessage() string {
	if m != nil {
		return m.Message
	}
	return ""
}

func (m *ChannelMessage) GetTopic() string {
	if m != nil {
		return m.Topic
	}
	return ""
}

func (m *ChannelMessage) GetPeerID() string {
	if m != nil {
		return m.PeerID
	}
	return ""
}

func (m *ChannelMessage) GetTimestamp() *timestamp.Timestamp {
	if m != nil {
		return m.Timestamp
	}
	return nil
}

func (m *ChannelMessage) GetSignature() []byte {
	if m != nil {
		return m.Signature
	}
	return nil
}

func init() {
	proto.RegisterType((*ChannelMessage)(nil), "ChannelMessage")
}

func init() { proto.RegisterFile("channel.proto", fileDescriptor_c8f385724121f37b) }

var fileDescriptor_c8f385724121f37b = []byte{
	// 186 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0xe2, 0xe2, 0x4d, 0xce, 0x48, 0xcc,
	0xcb, 0x4b, 0xcd, 0xd1, 0x2b, 0x28, 0xca, 0x2f, 0xc9, 0x97, 0x92, 0x4f, 0xcf, 0xcf, 0x4f, 0xcf,
	0x49, 0xd5, 0x07, 0xf3, 0x92, 0x4a, 0xd3, 0xf4, 0x4b, 0x32, 0x73, 0x53, 0x8b, 0x4b, 0x12, 0x73,
	0x0b, 0x20, 0x0a, 0x94, 0x36, 0x30, 0x72, 0xf1, 0x39, 0x43, 0xb4, 0xf8, 0xa6, 0x16, 0x17, 0x27,
	0xa6, 0xa7, 0x0a, 0x49, 0x70, 0xb1, 0xe7, 0x42, 0x98, 0x12, 0x8c, 0x0a, 0x8c, 0x1a, 0x9c, 0x41,
	0x30, 0xae, 0x90, 0x08, 0x17, 0x6b, 0x49, 0x7e, 0x41, 0x66, 0xb2, 0x04, 0x13, 0x58, 0x1c, 0xc2,
	0x11, 0x12, 0xe3, 0x62, 0x2b, 0x48, 0x4d, 0x2d, 0xf2, 0x74, 0x91, 0x60, 0x06, 0x0b, 0x43, 0x79,
	0x42, 0x16, 0x5c, 0x9c, 0x70, 0xdb, 0x24, 0x58, 0x14, 0x18, 0x35, 0xb8, 0x8d, 0xa4, 0xf4, 0x20,
	0xee, 0xd1, 0x83, 0xb9, 0x47, 0x2f, 0x04, 0xa6, 0x22, 0x08, 0xa1, 0x58, 0x48, 0x86, 0x8b, 0xb3,
	0x38, 0x33, 0x3d, 0x2f, 0xb1, 0xa4, 0xb4, 0x28, 0x55, 0x82, 0x55, 0x81, 0x51, 0x83, 0x27, 0x08,
	0x21, 0xe0, 0xc4, 0x12, 0xc5, 0x54, 0x90, 0x94, 0xc4, 0x06, 0x36, 0xc2, 0x18, 0x10, 0x00, 0x00,
	0xff, 0xff, 0x92, 0xe6, 0x08, 0x5a, 0xf1, 0x00, 0x00, 0x00,
}
