// Code generated by protoc-gen-gogo. DO NOT EDIT.
// source: tdrpc/adminrpc.proto

package tdrpc

import (
	context "context"
	fmt "fmt"
	_ "github.com/gogo/protobuf/gogoproto"
	proto "github.com/gogo/protobuf/proto"
	github_com_gogo_protobuf_sortkeys "github.com/gogo/protobuf/sortkeys"
	_ "github.com/grpc-ecosystem/grpc-gateway/protoc-gen-swagger/options"
	_ "google.golang.org/genproto/googleapis/api/annotations"
	grpc "google.golang.org/grpc"
	io "io"
	math "math"
	reflect "reflect"
	strings "strings"
)

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// This is a compile-time assertion to ensure that this generated file
// is compatible with the proto package it is being compiled against.
// A compilation error at this line likely means your copy of the
// proto package needs to be updated.
const _ = proto.GoGoProtoPackageIsVersion2 // please upgrade the proto package

// AdminAccountsRequest is used to request one or more accounts
type AdminAccountsRequest struct {
	// Filter values (id, address)
	Filter map[string]string `protobuf:"bytes,1,rep,name=filter,proto3" json:"filter,omitempty" protobuf_key:"bytes,1,opt,name=key,proto3" protobuf_val:"bytes,2,opt,name=value,proto3"`
	// Offset, Limit for pagination
	Offset int32 `protobuf:"varint,3,opt,name=offset,proto3" json:"offset,omitempty"`
	Limit  int32 `protobuf:"varint,4,opt,name=limit,proto3" json:"limit,omitempty"`
}

func (m *AdminAccountsRequest) Reset()      { *m = AdminAccountsRequest{} }
func (*AdminAccountsRequest) ProtoMessage() {}
func (*AdminAccountsRequest) Descriptor() ([]byte, []int) {
	return fileDescriptor_5f27b1318c7cd0a2, []int{0}
}
func (m *AdminAccountsRequest) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *AdminAccountsRequest) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_AdminAccountsRequest.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalTo(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *AdminAccountsRequest) XXX_Merge(src proto.Message) {
	xxx_messageInfo_AdminAccountsRequest.Merge(m, src)
}
func (m *AdminAccountsRequest) XXX_Size() int {
	return m.Size()
}
func (m *AdminAccountsRequest) XXX_DiscardUnknown() {
	xxx_messageInfo_AdminAccountsRequest.DiscardUnknown(m)
}

var xxx_messageInfo_AdminAccountsRequest proto.InternalMessageInfo

func (m *AdminAccountsRequest) GetFilter() map[string]string {
	if m != nil {
		return m.Filter
	}
	return nil
}

func (m *AdminAccountsRequest) GetOffset() int32 {
	if m != nil {
		return m.Offset
	}
	return 0
}

func (m *AdminAccountsRequest) GetLimit() int32 {
	if m != nil {
		return m.Limit
	}
	return 0
}

type AdminAccountsResponse struct {
	// The list of accounts
	Accounts []*Account `protobuf:"bytes,1,rep,name=accounts,proto3" json:"accounts,omitempty"`
}

func (m *AdminAccountsResponse) Reset()      { *m = AdminAccountsResponse{} }
func (*AdminAccountsResponse) ProtoMessage() {}
func (*AdminAccountsResponse) Descriptor() ([]byte, []int) {
	return fileDescriptor_5f27b1318c7cd0a2, []int{1}
}
func (m *AdminAccountsResponse) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *AdminAccountsResponse) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_AdminAccountsResponse.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalTo(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *AdminAccountsResponse) XXX_Merge(src proto.Message) {
	xxx_messageInfo_AdminAccountsResponse.Merge(m, src)
}
func (m *AdminAccountsResponse) XXX_Size() int {
	return m.Size()
}
func (m *AdminAccountsResponse) XXX_DiscardUnknown() {
	xxx_messageInfo_AdminAccountsResponse.DiscardUnknown(m)
}

var xxx_messageInfo_AdminAccountsResponse proto.InternalMessageInfo

func (m *AdminAccountsResponse) GetAccounts() []*Account {
	if m != nil {
		return m.Accounts
	}
	return nil
}

func init() {
	proto.RegisterType((*AdminAccountsRequest)(nil), "tdrpc.AdminAccountsRequest")
	proto.RegisterMapType((map[string]string)(nil), "tdrpc.AdminAccountsRequest.FilterEntry")
	proto.RegisterType((*AdminAccountsResponse)(nil), "tdrpc.AdminAccountsResponse")
}

func init() { proto.RegisterFile("tdrpc/adminrpc.proto", fileDescriptor_5f27b1318c7cd0a2) }

var fileDescriptor_5f27b1318c7cd0a2 = []byte{
	// 457 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x7c, 0x92, 0x31, 0x8f, 0xd3, 0x30,
	0x14, 0xc7, 0xe3, 0x96, 0x56, 0xc5, 0x27, 0x38, 0xce, 0x6a, 0x21, 0x2a, 0x27, 0xab, 0xea, 0x42,
	0x05, 0x34, 0x96, 0x8e, 0x05, 0x58, 0xd0, 0x71, 0x82, 0xe9, 0x06, 0x94, 0x91, 0x01, 0xc9, 0x4d,
	0x5e, 0x7d, 0xe6, 0x52, 0x3b, 0x24, 0x0e, 0xe8, 0x36, 0xc4, 0x27, 0x40, 0xe2, 0x4b, 0xf0, 0x11,
	0x18, 0x19, 0x19, 0x2b, 0xb1, 0xdc, 0x48, 0x53, 0x06, 0xc6, 0xfb, 0x08, 0xa8, 0xb6, 0x83, 0xd0,
	0x51, 0xb1, 0x24, 0xfe, 0xff, 0xff, 0xcf, 0x3f, 0xbf, 0x27, 0x3d, 0xdc, 0x37, 0x69, 0x91, 0x27,
	0x8c, 0xa7, 0x0b, 0xa9, 0x8a, 0x3c, 0x89, 0xf2, 0x42, 0x1b, 0x4d, 0x3a, 0xd6, 0x1d, 0xee, 0xb9,
	0xd0, 0x7e, 0x5d, 0x32, 0xdc, 0x17, 0x5a, 0x8b, 0x0c, 0x18, 0xcf, 0x25, 0xe3, 0x4a, 0x69, 0xc3,
	0x8d, 0xd4, 0xaa, 0xf4, 0xe9, 0x54, 0x48, 0x73, 0x52, 0xcd, 0xa2, 0x44, 0x2f, 0x98, 0xd0, 0x42,
	0x33, 0x6b, 0xcf, 0xaa, 0xb9, 0x55, 0x56, 0xd8, 0x93, 0x2f, 0xbf, 0x6f, 0x7f, 0xc9, 0x54, 0x80,
	0x9a, 0x96, 0xef, 0xb8, 0x10, 0x50, 0x30, 0x9d, 0x5b, 0xe0, 0xbf, 0xf0, 0xf1, 0x57, 0x84, 0xfb,
	0x87, 0x9b, 0x3e, 0x0f, 0x93, 0x44, 0x57, 0xca, 0x94, 0x31, 0xbc, 0xa9, 0xa0, 0x34, 0xe4, 0x09,
	0xee, 0xce, 0x65, 0x66, 0xa0, 0x08, 0xd1, 0xa8, 0x3d, 0xd9, 0x39, 0xb8, 0x13, 0xb9, 0x8e, 0xb7,
	0x15, 0x47, 0xcf, 0x6d, 0xe5, 0x33, 0x65, 0x8a, 0xb3, 0xd8, 0x5f, 0x23, 0x37, 0x71, 0x57, 0xcf,
	0xe7, 0x25, 0x98, 0xb0, 0x3d, 0x42, 0x93, 0x4e, 0xec, 0x15, 0xe9, 0xe3, 0x4e, 0x26, 0x17, 0xd2,
	0x84, 0x57, 0xac, 0xed, 0xc4, 0xf0, 0x11, 0xde, 0xf9, 0x0b, 0x42, 0x6e, 0xe0, 0xf6, 0x29, 0x9c,
	0x85, 0x68, 0x84, 0x26, 0x57, 0xe3, 0xcd, 0x71, 0x73, 0xed, 0x2d, 0xcf, 0x2a, 0x08, 0x5b, 0xd6,
	0x73, 0xe2, 0x71, 0xeb, 0x21, 0x1a, 0x1f, 0xe1, 0xc1, 0xa5, 0xa6, 0xca, 0x5c, 0xab, 0x12, 0xc8,
	0x5d, 0xdc, 0xe3, 0xde, 0xf3, 0x43, 0x5c, 0x6f, 0x86, 0x70, 0x76, 0xfc, 0x27, 0x3f, 0xf8, 0x82,
	0x70, 0xcf, 0x52, 0xe2, 0x17, 0x47, 0xe4, 0x15, 0xee, 0x35, 0x30, 0x72, 0xfb, 0x3f, 0x73, 0x0f,
	0xf7, 0xb7, 0x87, 0xee, 0xfd, 0xf1, 0xad, 0x0f, 0xdf, 0x7f, 0x7e, 0x6a, 0xed, 0x91, 0x5d, 0xb7,
	0x09, 0xac, 0x79, 0x8c, 0x1c, 0xe3, 0xee, 0x31, 0xa4, 0x02, 0x0a, 0xd2, 0xf7, 0x00, 0x27, 0x1b,
	0xec, 0xe0, 0x92, 0xeb, 0x79, 0x03, 0xcb, 0xdb, 0x25, 0xd7, 0x3c, 0x2f, 0xb3, 0xf1, 0x53, 0xbe,
	0x5c, 0xd1, 0xe0, 0x7c, 0x45, 0x83, 0x8b, 0x15, 0x45, 0xef, 0x6b, 0x8a, 0x3e, 0xd7, 0x14, 0x7d,
	0xab, 0x29, 0x5a, 0xd6, 0x14, 0xfd, 0xa8, 0x29, 0xfa, 0x55, 0xd3, 0xe0, 0xa2, 0xa6, 0xc1, 0xc7,
	0x35, 0x0d, 0x96, 0x6b, 0x1a, 0x9c, 0xaf, 0x69, 0xf0, 0xf2, 0x9e, 0x90, 0x26, 0x4a, 0xb4, 0x54,
	0x4a, 0xaa, 0xd7, 0x3c, 0x52, 0x60, 0xd8, 0x8c, 0x27, 0xa7, 0xa0, 0x52, 0x66, 0x4e, 0x2a, 0x95,
	0x42, 0x91, 0xea, 0x05, 0xb8, 0x35, 0x9d, 0x75, 0xed, 0xb2, 0x3c, 0xf8, 0x1d, 0x00, 0x00, 0xff,
	0xff, 0x32, 0xdc, 0xb5, 0x27, 0xd9, 0x02, 0x00, 0x00,
}

func (this *AdminAccountsRequest) Equal(that interface{}) bool {
	if that == nil {
		return this == nil
	}

	that1, ok := that.(*AdminAccountsRequest)
	if !ok {
		that2, ok := that.(AdminAccountsRequest)
		if ok {
			that1 = &that2
		} else {
			return false
		}
	}
	if that1 == nil {
		return this == nil
	} else if this == nil {
		return false
	}
	if len(this.Filter) != len(that1.Filter) {
		return false
	}
	for i := range this.Filter {
		if this.Filter[i] != that1.Filter[i] {
			return false
		}
	}
	if this.Offset != that1.Offset {
		return false
	}
	if this.Limit != that1.Limit {
		return false
	}
	return true
}
func (this *AdminAccountsResponse) Equal(that interface{}) bool {
	if that == nil {
		return this == nil
	}

	that1, ok := that.(*AdminAccountsResponse)
	if !ok {
		that2, ok := that.(AdminAccountsResponse)
		if ok {
			that1 = &that2
		} else {
			return false
		}
	}
	if that1 == nil {
		return this == nil
	} else if this == nil {
		return false
	}
	if len(this.Accounts) != len(that1.Accounts) {
		return false
	}
	for i := range this.Accounts {
		if !this.Accounts[i].Equal(that1.Accounts[i]) {
			return false
		}
	}
	return true
}
func (this *AdminAccountsRequest) GoString() string {
	if this == nil {
		return "nil"
	}
	s := make([]string, 0, 7)
	s = append(s, "&tdrpc.AdminAccountsRequest{")
	keysForFilter := make([]string, 0, len(this.Filter))
	for k, _ := range this.Filter {
		keysForFilter = append(keysForFilter, k)
	}
	github_com_gogo_protobuf_sortkeys.Strings(keysForFilter)
	mapStringForFilter := "map[string]string{"
	for _, k := range keysForFilter {
		mapStringForFilter += fmt.Sprintf("%#v: %#v,", k, this.Filter[k])
	}
	mapStringForFilter += "}"
	if this.Filter != nil {
		s = append(s, "Filter: "+mapStringForFilter+",\n")
	}
	s = append(s, "Offset: "+fmt.Sprintf("%#v", this.Offset)+",\n")
	s = append(s, "Limit: "+fmt.Sprintf("%#v", this.Limit)+",\n")
	s = append(s, "}")
	return strings.Join(s, "")
}
func (this *AdminAccountsResponse) GoString() string {
	if this == nil {
		return "nil"
	}
	s := make([]string, 0, 5)
	s = append(s, "&tdrpc.AdminAccountsResponse{")
	if this.Accounts != nil {
		s = append(s, "Accounts: "+fmt.Sprintf("%#v", this.Accounts)+",\n")
	}
	s = append(s, "}")
	return strings.Join(s, "")
}
func valueToGoStringAdminrpc(v interface{}, typ string) string {
	rv := reflect.ValueOf(v)
	if rv.IsNil() {
		return "nil"
	}
	pv := reflect.Indirect(rv).Interface()
	return fmt.Sprintf("func(v %v) *%v { return &v } ( %#v )", typ, typ, pv)
}

// Reference imports to suppress errors if they are not otherwise used.
var _ context.Context
var _ grpc.ClientConn

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
const _ = grpc.SupportPackageIsVersion4

// AdminRPCClient is the client API for AdminRPC service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://godoc.org/google.golang.org/grpc#ClientConn.NewStream.
type AdminRPCClient interface {
	// GetUser Accounts
	Accounts(ctx context.Context, in *AdminAccountsRequest, opts ...grpc.CallOption) (*AdminAccountsResponse, error)
	// Decode a payment request
	Ledger(ctx context.Context, in *LedgerRequest, opts ...grpc.CallOption) (*LedgerResponse, error)
}

type adminRPCClient struct {
	cc *grpc.ClientConn
}

func NewAdminRPCClient(cc *grpc.ClientConn) AdminRPCClient {
	return &adminRPCClient{cc}
}

func (c *adminRPCClient) Accounts(ctx context.Context, in *AdminAccountsRequest, opts ...grpc.CallOption) (*AdminAccountsResponse, error) {
	out := new(AdminAccountsResponse)
	err := c.cc.Invoke(ctx, "/tdrpc.AdminRPC/Accounts", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *adminRPCClient) Ledger(ctx context.Context, in *LedgerRequest, opts ...grpc.CallOption) (*LedgerResponse, error) {
	out := new(LedgerResponse)
	err := c.cc.Invoke(ctx, "/tdrpc.AdminRPC/Ledger", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// AdminRPCServer is the server API for AdminRPC service.
type AdminRPCServer interface {
	// GetUser Accounts
	Accounts(context.Context, *AdminAccountsRequest) (*AdminAccountsResponse, error)
	// Decode a payment request
	Ledger(context.Context, *LedgerRequest) (*LedgerResponse, error)
}

func RegisterAdminRPCServer(s *grpc.Server, srv AdminRPCServer) {
	s.RegisterService(&_AdminRPC_serviceDesc, srv)
}

func _AdminRPC_Accounts_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(AdminAccountsRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(AdminRPCServer).Accounts(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/tdrpc.AdminRPC/Accounts",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(AdminRPCServer).Accounts(ctx, req.(*AdminAccountsRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _AdminRPC_Ledger_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(LedgerRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(AdminRPCServer).Ledger(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/tdrpc.AdminRPC/Ledger",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(AdminRPCServer).Ledger(ctx, req.(*LedgerRequest))
	}
	return interceptor(ctx, in, info, handler)
}

var _AdminRPC_serviceDesc = grpc.ServiceDesc{
	ServiceName: "tdrpc.AdminRPC",
	HandlerType: (*AdminRPCServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "Accounts",
			Handler:    _AdminRPC_Accounts_Handler,
		},
		{
			MethodName: "Ledger",
			Handler:    _AdminRPC_Ledger_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "tdrpc/adminrpc.proto",
}

func (m *AdminAccountsRequest) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalTo(dAtA)
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *AdminAccountsRequest) MarshalTo(dAtA []byte) (int, error) {
	var i int
	_ = i
	var l int
	_ = l
	if len(m.Filter) > 0 {
		for k, _ := range m.Filter {
			dAtA[i] = 0xa
			i++
			v := m.Filter[k]
			mapSize := 1 + len(k) + sovAdminrpc(uint64(len(k))) + 1 + len(v) + sovAdminrpc(uint64(len(v)))
			i = encodeVarintAdminrpc(dAtA, i, uint64(mapSize))
			dAtA[i] = 0xa
			i++
			i = encodeVarintAdminrpc(dAtA, i, uint64(len(k)))
			i += copy(dAtA[i:], k)
			dAtA[i] = 0x12
			i++
			i = encodeVarintAdminrpc(dAtA, i, uint64(len(v)))
			i += copy(dAtA[i:], v)
		}
	}
	if m.Offset != 0 {
		dAtA[i] = 0x18
		i++
		i = encodeVarintAdminrpc(dAtA, i, uint64(m.Offset))
	}
	if m.Limit != 0 {
		dAtA[i] = 0x20
		i++
		i = encodeVarintAdminrpc(dAtA, i, uint64(m.Limit))
	}
	return i, nil
}

func (m *AdminAccountsResponse) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalTo(dAtA)
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *AdminAccountsResponse) MarshalTo(dAtA []byte) (int, error) {
	var i int
	_ = i
	var l int
	_ = l
	if len(m.Accounts) > 0 {
		for _, msg := range m.Accounts {
			dAtA[i] = 0xa
			i++
			i = encodeVarintAdminrpc(dAtA, i, uint64(msg.Size()))
			n, err := msg.MarshalTo(dAtA[i:])
			if err != nil {
				return 0, err
			}
			i += n
		}
	}
	return i, nil
}

func encodeVarintAdminrpc(dAtA []byte, offset int, v uint64) int {
	for v >= 1<<7 {
		dAtA[offset] = uint8(v&0x7f | 0x80)
		v >>= 7
		offset++
	}
	dAtA[offset] = uint8(v)
	return offset + 1
}
func (m *AdminAccountsRequest) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	if len(m.Filter) > 0 {
		for k, v := range m.Filter {
			_ = k
			_ = v
			mapEntrySize := 1 + len(k) + sovAdminrpc(uint64(len(k))) + 1 + len(v) + sovAdminrpc(uint64(len(v)))
			n += mapEntrySize + 1 + sovAdminrpc(uint64(mapEntrySize))
		}
	}
	if m.Offset != 0 {
		n += 1 + sovAdminrpc(uint64(m.Offset))
	}
	if m.Limit != 0 {
		n += 1 + sovAdminrpc(uint64(m.Limit))
	}
	return n
}

func (m *AdminAccountsResponse) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	if len(m.Accounts) > 0 {
		for _, e := range m.Accounts {
			l = e.Size()
			n += 1 + l + sovAdminrpc(uint64(l))
		}
	}
	return n
}

func sovAdminrpc(x uint64) (n int) {
	for {
		n++
		x >>= 7
		if x == 0 {
			break
		}
	}
	return n
}
func sozAdminrpc(x uint64) (n int) {
	return sovAdminrpc(uint64((x << 1) ^ uint64((int64(x) >> 63))))
}
func (this *AdminAccountsRequest) String() string {
	if this == nil {
		return "nil"
	}
	keysForFilter := make([]string, 0, len(this.Filter))
	for k, _ := range this.Filter {
		keysForFilter = append(keysForFilter, k)
	}
	github_com_gogo_protobuf_sortkeys.Strings(keysForFilter)
	mapStringForFilter := "map[string]string{"
	for _, k := range keysForFilter {
		mapStringForFilter += fmt.Sprintf("%v: %v,", k, this.Filter[k])
	}
	mapStringForFilter += "}"
	s := strings.Join([]string{`&AdminAccountsRequest{`,
		`Filter:` + mapStringForFilter + `,`,
		`Offset:` + fmt.Sprintf("%v", this.Offset) + `,`,
		`Limit:` + fmt.Sprintf("%v", this.Limit) + `,`,
		`}`,
	}, "")
	return s
}
func (this *AdminAccountsResponse) String() string {
	if this == nil {
		return "nil"
	}
	s := strings.Join([]string{`&AdminAccountsResponse{`,
		`Accounts:` + strings.Replace(fmt.Sprintf("%v", this.Accounts), "Account", "Account", 1) + `,`,
		`}`,
	}, "")
	return s
}
func valueToStringAdminrpc(v interface{}) string {
	rv := reflect.ValueOf(v)
	if rv.IsNil() {
		return "nil"
	}
	pv := reflect.Indirect(rv).Interface()
	return fmt.Sprintf("*%v", pv)
}
func (m *AdminAccountsRequest) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowAdminrpc
			}
			if iNdEx >= l {
				return io.ErrUnexpectedEOF
			}
			b := dAtA[iNdEx]
			iNdEx++
			wire |= uint64(b&0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		fieldNum := int32(wire >> 3)
		wireType := int(wire & 0x7)
		if wireType == 4 {
			return fmt.Errorf("proto: AdminAccountsRequest: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: AdminAccountsRequest: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Filter", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowAdminrpc
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				msglen |= int(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if msglen < 0 {
				return ErrInvalidLengthAdminrpc
			}
			postIndex := iNdEx + msglen
			if postIndex < 0 {
				return ErrInvalidLengthAdminrpc
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			if m.Filter == nil {
				m.Filter = make(map[string]string)
			}
			var mapkey string
			var mapvalue string
			for iNdEx < postIndex {
				entryPreIndex := iNdEx
				var wire uint64
				for shift := uint(0); ; shift += 7 {
					if shift >= 64 {
						return ErrIntOverflowAdminrpc
					}
					if iNdEx >= l {
						return io.ErrUnexpectedEOF
					}
					b := dAtA[iNdEx]
					iNdEx++
					wire |= uint64(b&0x7F) << shift
					if b < 0x80 {
						break
					}
				}
				fieldNum := int32(wire >> 3)
				if fieldNum == 1 {
					var stringLenmapkey uint64
					for shift := uint(0); ; shift += 7 {
						if shift >= 64 {
							return ErrIntOverflowAdminrpc
						}
						if iNdEx >= l {
							return io.ErrUnexpectedEOF
						}
						b := dAtA[iNdEx]
						iNdEx++
						stringLenmapkey |= uint64(b&0x7F) << shift
						if b < 0x80 {
							break
						}
					}
					intStringLenmapkey := int(stringLenmapkey)
					if intStringLenmapkey < 0 {
						return ErrInvalidLengthAdminrpc
					}
					postStringIndexmapkey := iNdEx + intStringLenmapkey
					if postStringIndexmapkey < 0 {
						return ErrInvalidLengthAdminrpc
					}
					if postStringIndexmapkey > l {
						return io.ErrUnexpectedEOF
					}
					mapkey = string(dAtA[iNdEx:postStringIndexmapkey])
					iNdEx = postStringIndexmapkey
				} else if fieldNum == 2 {
					var stringLenmapvalue uint64
					for shift := uint(0); ; shift += 7 {
						if shift >= 64 {
							return ErrIntOverflowAdminrpc
						}
						if iNdEx >= l {
							return io.ErrUnexpectedEOF
						}
						b := dAtA[iNdEx]
						iNdEx++
						stringLenmapvalue |= uint64(b&0x7F) << shift
						if b < 0x80 {
							break
						}
					}
					intStringLenmapvalue := int(stringLenmapvalue)
					if intStringLenmapvalue < 0 {
						return ErrInvalidLengthAdminrpc
					}
					postStringIndexmapvalue := iNdEx + intStringLenmapvalue
					if postStringIndexmapvalue < 0 {
						return ErrInvalidLengthAdminrpc
					}
					if postStringIndexmapvalue > l {
						return io.ErrUnexpectedEOF
					}
					mapvalue = string(dAtA[iNdEx:postStringIndexmapvalue])
					iNdEx = postStringIndexmapvalue
				} else {
					iNdEx = entryPreIndex
					skippy, err := skipAdminrpc(dAtA[iNdEx:])
					if err != nil {
						return err
					}
					if skippy < 0 {
						return ErrInvalidLengthAdminrpc
					}
					if (iNdEx + skippy) > postIndex {
						return io.ErrUnexpectedEOF
					}
					iNdEx += skippy
				}
			}
			m.Filter[mapkey] = mapvalue
			iNdEx = postIndex
		case 3:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field Offset", wireType)
			}
			m.Offset = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowAdminrpc
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.Offset |= int32(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		case 4:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field Limit", wireType)
			}
			m.Limit = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowAdminrpc
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.Limit |= int32(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		default:
			iNdEx = preIndex
			skippy, err := skipAdminrpc(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if skippy < 0 {
				return ErrInvalidLengthAdminrpc
			}
			if (iNdEx + skippy) < 0 {
				return ErrInvalidLengthAdminrpc
			}
			if (iNdEx + skippy) > l {
				return io.ErrUnexpectedEOF
			}
			iNdEx += skippy
		}
	}

	if iNdEx > l {
		return io.ErrUnexpectedEOF
	}
	return nil
}
func (m *AdminAccountsResponse) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowAdminrpc
			}
			if iNdEx >= l {
				return io.ErrUnexpectedEOF
			}
			b := dAtA[iNdEx]
			iNdEx++
			wire |= uint64(b&0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		fieldNum := int32(wire >> 3)
		wireType := int(wire & 0x7)
		if wireType == 4 {
			return fmt.Errorf("proto: AdminAccountsResponse: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: AdminAccountsResponse: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Accounts", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowAdminrpc
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				msglen |= int(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if msglen < 0 {
				return ErrInvalidLengthAdminrpc
			}
			postIndex := iNdEx + msglen
			if postIndex < 0 {
				return ErrInvalidLengthAdminrpc
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Accounts = append(m.Accounts, &Account{})
			if err := m.Accounts[len(m.Accounts)-1].Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skipAdminrpc(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if skippy < 0 {
				return ErrInvalidLengthAdminrpc
			}
			if (iNdEx + skippy) < 0 {
				return ErrInvalidLengthAdminrpc
			}
			if (iNdEx + skippy) > l {
				return io.ErrUnexpectedEOF
			}
			iNdEx += skippy
		}
	}

	if iNdEx > l {
		return io.ErrUnexpectedEOF
	}
	return nil
}
func skipAdminrpc(dAtA []byte) (n int, err error) {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return 0, ErrIntOverflowAdminrpc
			}
			if iNdEx >= l {
				return 0, io.ErrUnexpectedEOF
			}
			b := dAtA[iNdEx]
			iNdEx++
			wire |= (uint64(b) & 0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		wireType := int(wire & 0x7)
		switch wireType {
		case 0:
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return 0, ErrIntOverflowAdminrpc
				}
				if iNdEx >= l {
					return 0, io.ErrUnexpectedEOF
				}
				iNdEx++
				if dAtA[iNdEx-1] < 0x80 {
					break
				}
			}
			return iNdEx, nil
		case 1:
			iNdEx += 8
			return iNdEx, nil
		case 2:
			var length int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return 0, ErrIntOverflowAdminrpc
				}
				if iNdEx >= l {
					return 0, io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				length |= (int(b) & 0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if length < 0 {
				return 0, ErrInvalidLengthAdminrpc
			}
			iNdEx += length
			if iNdEx < 0 {
				return 0, ErrInvalidLengthAdminrpc
			}
			return iNdEx, nil
		case 3:
			for {
				var innerWire uint64
				var start int = iNdEx
				for shift := uint(0); ; shift += 7 {
					if shift >= 64 {
						return 0, ErrIntOverflowAdminrpc
					}
					if iNdEx >= l {
						return 0, io.ErrUnexpectedEOF
					}
					b := dAtA[iNdEx]
					iNdEx++
					innerWire |= (uint64(b) & 0x7F) << shift
					if b < 0x80 {
						break
					}
				}
				innerWireType := int(innerWire & 0x7)
				if innerWireType == 4 {
					break
				}
				next, err := skipAdminrpc(dAtA[start:])
				if err != nil {
					return 0, err
				}
				iNdEx = start + next
				if iNdEx < 0 {
					return 0, ErrInvalidLengthAdminrpc
				}
			}
			return iNdEx, nil
		case 4:
			return iNdEx, nil
		case 5:
			iNdEx += 4
			return iNdEx, nil
		default:
			return 0, fmt.Errorf("proto: illegal wireType %d", wireType)
		}
	}
	panic("unreachable")
}

var (
	ErrInvalidLengthAdminrpc = fmt.Errorf("proto: negative length found during unmarshaling")
	ErrIntOverflowAdminrpc   = fmt.Errorf("proto: integer overflow")
)
