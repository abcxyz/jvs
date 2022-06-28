// Code generated by protoc-gen-go-grpc. DO NOT EDIT.

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

// CertificateActionServiceClient is the client API for CertificateActionService service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type CertificateActionServiceClient interface {
	CertificateAction(ctx context.Context, in *CertificateActionRequest, opts ...grpc.CallOption) (*CertificateActionResponse, error)
}

type certificateActionServiceClient struct {
	cc grpc.ClientConnInterface
}

func NewCertificateActionServiceClient(cc grpc.ClientConnInterface) CertificateActionServiceClient {
	return &certificateActionServiceClient{cc}
}

func (c *certificateActionServiceClient) CertificateAction(ctx context.Context, in *CertificateActionRequest, opts ...grpc.CallOption) (*CertificateActionResponse, error) {
	out := new(CertificateActionResponse)
	err := c.cc.Invoke(ctx, "/jvs.CertificateActionService/CertificateAction", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// CertificateActionServiceServer is the server API for CertificateActionService service.
// All implementations must embed UnimplementedCertificateActionServiceServer
// for forward compatibility
type CertificateActionServiceServer interface {
	CertificateAction(context.Context, *CertificateActionRequest) (*CertificateActionResponse, error)
	mustEmbedUnimplementedCertificateActionServiceServer()
}

// UnimplementedCertificateActionServiceServer must be embedded to have forward compatible implementations.
type UnimplementedCertificateActionServiceServer struct {
}

func (UnimplementedCertificateActionServiceServer) CertificateAction(context.Context, *CertificateActionRequest) (*CertificateActionResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method CertificateAction not implemented")
}
func (UnimplementedCertificateActionServiceServer) mustEmbedUnimplementedCertificateActionServiceServer() {
}

// UnsafeCertificateActionServiceServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to CertificateActionServiceServer will
// result in compilation errors.
type UnsafeCertificateActionServiceServer interface {
	mustEmbedUnimplementedCertificateActionServiceServer()
}

func RegisterCertificateActionServiceServer(s grpc.ServiceRegistrar, srv CertificateActionServiceServer) {
	s.RegisterService(&CertificateActionService_ServiceDesc, srv)
}

func _CertificateActionService_CertificateAction_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(CertificateActionRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(CertificateActionServiceServer).CertificateAction(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/jvs.CertificateActionService/CertificateAction",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(CertificateActionServiceServer).CertificateAction(ctx, req.(*CertificateActionRequest))
	}
	return interceptor(ctx, in, info, handler)
}

// CertificateActionService_ServiceDesc is the grpc.ServiceDesc for CertificateActionService service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var CertificateActionService_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "jvs.CertificateActionService",
	HandlerType: (*CertificateActionServiceServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "CertificateAction",
			Handler:    _CertificateActionService_CertificateAction_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "protos/v0/cert_action_service.proto",
}