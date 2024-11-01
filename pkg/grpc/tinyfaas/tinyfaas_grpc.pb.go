// Code generated by protoc-gen-go-grpc. DO NOT EDIT.
// versions:
// - protoc-gen-go-grpc v1.2.0
// - protoc             v5.28.3
// source: tinyfaas.proto

package tinyfaas

import (
	context "context"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
)

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
// Requires gRPC-Go v1.32.0 or later.
const _ = grpc.SupportPackageIsVersion7

// TinyFaaSClient is the client API for TinyFaaS service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type TinyFaaSClient interface {
	Request(ctx context.Context, in *Data, opts ...grpc.CallOption) (*Response, error)
}

type tinyFaaSClient struct {
	cc grpc.ClientConnInterface
}

func NewTinyFaaSClient(cc grpc.ClientConnInterface) TinyFaaSClient {
	return &tinyFaaSClient{cc}
}

func (c *tinyFaaSClient) Request(ctx context.Context, in *Data, opts ...grpc.CallOption) (*Response, error) {
	out := new(Response)
	err := c.cc.Invoke(ctx, "/openfogstack.tinyfaas.tinyfaas.TinyFaaS/Request", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// TinyFaaSServer is the server API for TinyFaaS service.
// All implementations should embed UnimplementedTinyFaaSServer
// for forward compatibility
type TinyFaaSServer interface {
	Request(context.Context, *Data) (*Response, error)
}

// UnimplementedTinyFaaSServer should be embedded to have forward compatible implementations.
type UnimplementedTinyFaaSServer struct {
}

func (UnimplementedTinyFaaSServer) Request(context.Context, *Data) (*Response, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Request not implemented")
}

// UnsafeTinyFaaSServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to TinyFaaSServer will
// result in compilation errors.
type UnsafeTinyFaaSServer interface {
	mustEmbedUnimplementedTinyFaaSServer()
}

func RegisterTinyFaaSServer(s grpc.ServiceRegistrar, srv TinyFaaSServer) {
	s.RegisterService(&TinyFaaS_ServiceDesc, srv)
}

func _TinyFaaS_Request_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(Data)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(TinyFaaSServer).Request(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/openfogstack.tinyfaas.tinyfaas.TinyFaaS/Request",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(TinyFaaSServer).Request(ctx, req.(*Data))
	}
	return interceptor(ctx, in, info, handler)
}

// TinyFaaS_ServiceDesc is the grpc.ServiceDesc for TinyFaaS service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var TinyFaaS_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "openfogstack.tinyfaas.tinyfaas.TinyFaaS",
	HandlerType: (*TinyFaaSServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "Request",
			Handler:    _TinyFaaS_Request_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "tinyfaas.proto",
}
