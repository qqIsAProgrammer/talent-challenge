// Code generated by protoc-gen-go. DO NOT EDIT.
// source: messagepb.proto

package raftstorepb

import proto "github.com/golang/protobuf/proto"
import fmt "fmt"
import math "math"
import raftpb "go.etcd.io/etcd/raft/v3/raftpb"

import (
	context "golang.org/x/net/context"
	grpc "google.golang.org/grpc"
)

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// This is a compile-time assertion to ensure that this generated file
// is compatible with the proto package it is being compiled against.
// A compilation error at this line likely means your copy of the
// proto package needs to be updated.
const _ = proto.ProtoPackageIsVersion2 // please upgrade the proto package

type RaftMsgReq struct {
	Message              *raftpb.Message `protobuf:"bytes,1,opt,name=message" json:"message,omitempty"`
	FromPeer             uint64          `protobuf:"varint,2,opt,name=from_peer,json=fromPeer" json:"from_peer,omitempty"`
	ToPeer               uint64          `protobuf:"varint,3,opt,name=to_peer,json=toPeer" json:"to_peer,omitempty"`
	XXX_NoUnkeyedLiteral struct{}        `json:"-"`
	XXX_unrecognized     []byte          `json:"-"`
	XXX_sizecache        int32           `json:"-"`
}

func (m *RaftMsgReq) Reset()         { *m = RaftMsgReq{} }
func (m *RaftMsgReq) String() string { return proto.CompactTextString(m) }
func (*RaftMsgReq) ProtoMessage()    {}
func (*RaftMsgReq) Descriptor() ([]byte, []int) {
	return fileDescriptor_messagepb_5a79634d65d7d3a1, []int{0}
}
func (m *RaftMsgReq) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_RaftMsgReq.Unmarshal(m, b)
}
func (m *RaftMsgReq) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_RaftMsgReq.Marshal(b, m, deterministic)
}
func (dst *RaftMsgReq) XXX_Merge(src proto.Message) {
	xxx_messageInfo_RaftMsgReq.Merge(dst, src)
}
func (m *RaftMsgReq) XXX_Size() int {
	return xxx_messageInfo_RaftMsgReq.Size(m)
}
func (m *RaftMsgReq) XXX_DiscardUnknown() {
	xxx_messageInfo_RaftMsgReq.DiscardUnknown(m)
}

var xxx_messageInfo_RaftMsgReq proto.InternalMessageInfo

func (m *RaftMsgReq) GetMessage() *raftpb.Message {
	if m != nil {
		return m.Message
	}
	return nil
}

func (m *RaftMsgReq) GetFromPeer() uint64 {
	if m != nil {
		return m.FromPeer
	}
	return 0
}

func (m *RaftMsgReq) GetToPeer() uint64 {
	if m != nil {
		return m.ToPeer
	}
	return 0
}

type Empty struct {
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *Empty) Reset()         { *m = Empty{} }
func (m *Empty) String() string { return proto.CompactTextString(m) }
func (*Empty) ProtoMessage()    {}
func (*Empty) Descriptor() ([]byte, []int) {
	return fileDescriptor_messagepb_5a79634d65d7d3a1, []int{1}
}
func (m *Empty) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_Empty.Unmarshal(m, b)
}
func (m *Empty) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_Empty.Marshal(b, m, deterministic)
}
func (dst *Empty) XXX_Merge(src proto.Message) {
	xxx_messageInfo_Empty.Merge(dst, src)
}
func (m *Empty) XXX_Size() int {
	return xxx_messageInfo_Empty.Size(m)
}
func (m *Empty) XXX_DiscardUnknown() {
	xxx_messageInfo_Empty.DiscardUnknown(m)
}

var xxx_messageInfo_Empty proto.InternalMessageInfo

func init() {
	proto.RegisterType((*RaftMsgReq)(nil), "raftstorepb.RaftMsgReq")
	proto.RegisterType((*Empty)(nil), "raftstorepb.Empty")
}

// Reference imports to suppress errors if they are not otherwise used.
var _ context.Context
var _ grpc.ClientConn

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
const _ = grpc.SupportPackageIsVersion4

// Client API for Message service

type MessageClient interface {
	// peer to peer
	RaftMessage(ctx context.Context, opts ...grpc.CallOption) (Message_RaftMessageClient, error)
}

type messageClient struct {
	cc *grpc.ClientConn
}

func NewMessageClient(cc *grpc.ClientConn) MessageClient {
	return &messageClient{cc}
}

func (c *messageClient) RaftMessage(ctx context.Context, opts ...grpc.CallOption) (Message_RaftMessageClient, error) {
	stream, err := grpc.NewClientStream(ctx, &_Message_serviceDesc.Streams[0], c.cc, "/raftstorepb.Message/RaftMessage", opts...)
	if err != nil {
		return nil, err
	}
	x := &messageRaftMessageClient{stream}
	return x, nil
}

type Message_RaftMessageClient interface {
	Send(*RaftMsgReq) error
	CloseAndRecv() (*Empty, error)
	grpc.ClientStream
}

type messageRaftMessageClient struct {
	grpc.ClientStream
}

func (x *messageRaftMessageClient) Send(m *RaftMsgReq) error {
	return x.ClientStream.SendMsg(m)
}

func (x *messageRaftMessageClient) CloseAndRecv() (*Empty, error) {
	if err := x.ClientStream.CloseSend(); err != nil {
		return nil, err
	}
	m := new(Empty)
	if err := x.ClientStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

// Server API for Message service

type MessageServer interface {
	// peer to peer
	RaftMessage(Message_RaftMessageServer) error
}

func RegisterMessageServer(s *grpc.Server, srv MessageServer) {
	s.RegisterService(&_Message_serviceDesc, srv)
}

func _Message_RaftMessage_Handler(srv interface{}, stream grpc.ServerStream) error {
	return srv.(MessageServer).RaftMessage(&messageRaftMessageServer{stream})
}

type Message_RaftMessageServer interface {
	SendAndClose(*Empty) error
	Recv() (*RaftMsgReq, error)
	grpc.ServerStream
}

type messageRaftMessageServer struct {
	grpc.ServerStream
}

func (x *messageRaftMessageServer) SendAndClose(m *Empty) error {
	return x.ServerStream.SendMsg(m)
}

func (x *messageRaftMessageServer) Recv() (*RaftMsgReq, error) {
	m := new(RaftMsgReq)
	if err := x.ServerStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

var _Message_serviceDesc = grpc.ServiceDesc{
	ServiceName: "raftstorepb.Message",
	HandlerType: (*MessageServer)(nil),
	Methods:     []grpc.MethodDesc{},
	Streams: []grpc.StreamDesc{
		{
			StreamName:    "RaftMessage",
			Handler:       _Message_RaftMessage_Handler,
			ClientStreams: true,
		},
	},
	Metadata: "messagepb.proto",
}

func init() { proto.RegisterFile("messagepb.proto", fileDescriptor_messagepb_5a79634d65d7d3a1) }

var fileDescriptor_messagepb_5a79634d65d7d3a1 = []byte{
	// 206 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x54, 0x8f, 0x3f, 0x4b, 0xc5, 0x30,
	0x14, 0xc5, 0x8d, 0x7f, 0x5e, 0xf4, 0x76, 0x78, 0x90, 0xe5, 0x3d, 0xea, 0x52, 0x3a, 0xb5, 0x4b,
	0x0a, 0xed, 0xee, 0xe6, 0xe0, 0x50, 0x90, 0x7c, 0x01, 0x69, 0xf4, 0xb6, 0x38, 0x84, 0x1b, 0x93,
	0x8b, 0xe0, 0xb7, 0x97, 0x24, 0x15, 0x75, 0x3a, 0xdc, 0xfb, 0xe3, 0x1c, 0xce, 0x81, 0xa3, 0xc3,
	0x18, 0x97, 0x0d, 0xbd, 0xd5, 0x3e, 0x10, 0x93, 0xaa, 0xc2, 0xb2, 0x72, 0x64, 0x0a, 0xe8, 0x6d,
	0xdd, 0x6f, 0xa4, 0x91, 0x5f, 0xdf, 0xf4, 0x3b, 0x0d, 0x49, 0x87, 0x04, 0x87, 0xcf, 0x29, 0xab,
	0xb7, 0x59, 0x8a, 0xaf, 0x75, 0x00, 0x66, 0x59, 0x79, 0x8e, 0x9b, 0xc1, 0x0f, 0xd5, 0x83, 0xdc,
	0x83, 0xcf, 0xa2, 0x11, 0x5d, 0x35, 0x1e, 0x75, 0xb1, 0xe8, 0xb9, 0xbc, 0xcd, 0x0f, 0x57, 0xf7,
	0x70, 0xb7, 0x06, 0x72, 0x2f, 0x1e, 0x31, 0x9c, 0x2f, 0x1b, 0xd1, 0x5d, 0x9b, 0xdb, 0xf4, 0x78,
	0x46, 0x0c, 0xea, 0x04, 0x92, 0xa9, 0xa0, 0xab, 0x8c, 0x0e, 0x4c, 0x09, 0xb4, 0x12, 0x6e, 0x1e,
	0x9d, 0xe7, 0xaf, 0xf1, 0x09, 0xe4, 0x1e, 0xa9, 0x1e, 0xa0, 0xca, 0x15, 0xf6, 0xf3, 0xa4, 0xff,
	0x4c, 0xd1, 0xbf, 0xe5, 0x6a, 0xf5, 0x0f, 0xe4, 0x98, 0xf6, 0xa2, 0x13, 0xf6, 0x90, 0x97, 0x4c,
	0xdf, 0x01, 0x00, 0x00, 0xff, 0xff, 0x01, 0x1d, 0x9f, 0x18, 0x14, 0x01, 0x00, 0x00,
}
