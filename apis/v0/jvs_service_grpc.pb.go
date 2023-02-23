// Code generated by protoc-gen-go-grpc. DO NOT EDIT.
// versions:
// - protoc-gen-go-grpc v1.2.0
// - protoc             v3.21.12
// source: jvs_service.proto

package v0

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

// JVSServiceClient is the client API for JVSService service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type JVSServiceClient interface {
	CreateJustification(ctx context.Context, in *CreateJustificationRequest, opts ...grpc.CallOption) (*CreateJustificationResponse, error)
}

type jVSServiceClient struct {
	cc grpc.ClientConnInterface
}

func NewJVSServiceClient(cc grpc.ClientConnInterface) JVSServiceClient {
	return &jVSServiceClient{cc}
}

func (c *jVSServiceClient) CreateJustification(ctx context.Context, in *CreateJustificationRequest, opts ...grpc.CallOption) (*CreateJustificationResponse, error) {
	out := new(CreateJustificationResponse)
	err := c.cc.Invoke(ctx, "/abcxyz.jvs.JVSService/CreateJustification", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// JVSServiceServer is the server API for JVSService service.
// All implementations must embed UnimplementedJVSServiceServer
// for forward compatibility
type JVSServiceServer interface {
	CreateJustification(context.Context, *CreateJustificationRequest) (*CreateJustificationResponse, error)
	mustEmbedUnimplementedJVSServiceServer()
}

// UnimplementedJVSServiceServer must be embedded to have forward compatible implementations.
type UnimplementedJVSServiceServer struct {
}

func (UnimplementedJVSServiceServer) CreateJustification(context.Context, *CreateJustificationRequest) (*CreateJustificationResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method CreateJustification not implemented")
}
func (UnimplementedJVSServiceServer) mustEmbedUnimplementedJVSServiceServer() {}

// UnsafeJVSServiceServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to JVSServiceServer will
// result in compilation errors.
type UnsafeJVSServiceServer interface {
	mustEmbedUnimplementedJVSServiceServer()
}

func RegisterJVSServiceServer(s grpc.ServiceRegistrar, srv JVSServiceServer) {
	s.RegisterService(&JVSService_ServiceDesc, srv)
}

func _JVSService_CreateJustification_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(CreateJustificationRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(JVSServiceServer).CreateJustification(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/abcxyz.jvs.JVSService/CreateJustification",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(JVSServiceServer).CreateJustification(ctx, req.(*CreateJustificationRequest))
	}
	return interceptor(ctx, in, info, handler)
}

// JVSService_ServiceDesc is the grpc.ServiceDesc for JVSService service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var JVSService_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "abcxyz.jvs.JVSService",
	HandlerType: (*JVSServiceServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "CreateJustification",
			Handler:    _JVSService_CreateJustification_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "jvs_service.proto",
}
