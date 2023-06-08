// Copyright 2023 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Code generated by protoc-gen-go-grpc. DO NOT EDIT.
// versions:
// - protoc-gen-go-grpc v1.3.0
// - protoc             v3.20.1
// source: jvs_plugin_service.proto

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

const (
	JVSPlugin_Validate_FullMethodName = "/abcxyz.jvs.JVSPlugin/Validate"
)

// JVSPluginClient is the client API for JVSPlugin service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type JVSPluginClient interface {
	Validate(ctx context.Context, in *ValidateJustificationRequest, opts ...grpc.CallOption) (*ValidateJustificationResponse, error)
}

type jVSPluginClient struct {
	cc grpc.ClientConnInterface
}

func NewJVSPluginClient(cc grpc.ClientConnInterface) JVSPluginClient {
	return &jVSPluginClient{cc}
}

func (c *jVSPluginClient) Validate(ctx context.Context, in *ValidateJustificationRequest, opts ...grpc.CallOption) (*ValidateJustificationResponse, error) {
	out := new(ValidateJustificationResponse)
	err := c.cc.Invoke(ctx, JVSPlugin_Validate_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// JVSPluginServer is the server API for JVSPlugin service.
// All implementations must embed UnimplementedJVSPluginServer
// for forward compatibility
type JVSPluginServer interface {
	Validate(context.Context, *ValidateJustificationRequest) (*ValidateJustificationResponse, error)
	mustEmbedUnimplementedJVSPluginServer()
}

// UnimplementedJVSPluginServer must be embedded to have forward compatible implementations.
type UnimplementedJVSPluginServer struct {
}

func (UnimplementedJVSPluginServer) Validate(context.Context, *ValidateJustificationRequest) (*ValidateJustificationResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Validate not implemented")
}
func (UnimplementedJVSPluginServer) mustEmbedUnimplementedJVSPluginServer() {}

// UnsafeJVSPluginServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to JVSPluginServer will
// result in compilation errors.
type UnsafeJVSPluginServer interface {
	mustEmbedUnimplementedJVSPluginServer()
}

func RegisterJVSPluginServer(s grpc.ServiceRegistrar, srv JVSPluginServer) {
	s.RegisterService(&JVSPlugin_ServiceDesc, srv)
}

func _JVSPlugin_Validate_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ValidateJustificationRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(JVSPluginServer).Validate(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: JVSPlugin_Validate_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(JVSPluginServer).Validate(ctx, req.(*ValidateJustificationRequest))
	}
	return interceptor(ctx, in, info, handler)
}

// JVSPlugin_ServiceDesc is the grpc.ServiceDesc for JVSPlugin service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var JVSPlugin_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "abcxyz.jvs.JVSPlugin",
	HandlerType: (*JVSPluginServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "Validate",
			Handler:    _JVSPlugin_Validate_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "jvs_plugin_service.proto",
}
