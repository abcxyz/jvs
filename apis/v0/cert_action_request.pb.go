// Copyright 2022 Google LLC
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

// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.27.1
// 	protoc        v3.19.1
// source: protos/v0/cert_action_request.proto

package v0

import (
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	reflect "reflect"
	sync "sync"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

type Action_ACTION int32

const (
	// This rotates the specified key gracefully. it will make a new primary and promote that new key to primary,
	// but will not immediately disable the version specified (so it can still be used for JWT validation).
	Action_ROTATE Action_ACTION = 0
	// This will immediately disable the version specified. If the version is primary, it will make a new primary and
	// promote that new key to primary. This is intended to make it invalid for use in JWT validation as soon as possible.
	// However, until client caches are removed, JWTs could still be validated using the version.
	Action_FORCE_DISABLE Action_ACTION = 1
	// This will immediately destroy the version specified. If the version is primary, it will make a new primary and
	// promote that new key to primary. This is intended to make it invalid for use in JWT validation as soon as possible.
	// However, until client caches are removed, JWTs could still be validated using the version.
	Action_FORCE_DESTROY Action_ACTION = 2
)

// Enum value maps for Action_ACTION.
var (
	Action_ACTION_name = map[int32]string{
		0: "ROTATE",
		1: "FORCE_DISABLE",
		2: "FORCE_DESTROY",
	}
	Action_ACTION_value = map[string]int32{
		"ROTATE":        0,
		"FORCE_DISABLE": 1,
		"FORCE_DESTROY": 2,
	}
)

func (x Action_ACTION) Enum() *Action_ACTION {
	p := new(Action_ACTION)
	*p = x
	return p
}

func (x Action_ACTION) String() string {
	return protoimpl.X.EnumStringOf(x.Descriptor(), protoreflect.EnumNumber(x))
}

func (Action_ACTION) Descriptor() protoreflect.EnumDescriptor {
	return file_protos_v0_cert_action_request_proto_enumTypes[0].Descriptor()
}

func (Action_ACTION) Type() protoreflect.EnumType {
	return &file_protos_v0_cert_action_request_proto_enumTypes[0]
}

func (x Action_ACTION) Number() protoreflect.EnumNumber {
	return protoreflect.EnumNumber(x)
}

// Deprecated: Use Action_ACTION.Descriptor instead.
func (Action_ACTION) EnumDescriptor() ([]byte, []int) {
	return file_protos_v0_cert_action_request_proto_rawDescGZIP(), []int{1, 0}
}

// CertificateActionRequest is a request to do a manual action on a certificate.
type CertificateActionRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Actions []*Action `protobuf:"bytes,1,rep,name=actions,proto3" json:"actions,omitempty"`
}

func (x *CertificateActionRequest) Reset() {
	*x = CertificateActionRequest{}
	if protoimpl.UnsafeEnabled {
		mi := &file_protos_v0_cert_action_request_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *CertificateActionRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*CertificateActionRequest) ProtoMessage() {}

func (x *CertificateActionRequest) ProtoReflect() protoreflect.Message {
	mi := &file_protos_v0_cert_action_request_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use CertificateActionRequest.ProtoReflect.Descriptor instead.
func (*CertificateActionRequest) Descriptor() ([]byte, []int) {
	return file_protos_v0_cert_action_request_proto_rawDescGZIP(), []int{0}
}

func (x *CertificateActionRequest) GetActions() []*Action {
	if x != nil {
		return x.Actions
	}
	return nil
}

// Justification is intended to be used to provide reasons that data access is required.
type Action struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Version uint32        `protobuf:"varint,1,opt,name=version,proto3" json:"version,omitempty"`
	Action  Action_ACTION `protobuf:"varint,2,opt,name=action,proto3,enum=jvs.Action_ACTION" json:"action,omitempty"`
}

func (x *Action) Reset() {
	*x = Action{}
	if protoimpl.UnsafeEnabled {
		mi := &file_protos_v0_cert_action_request_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Action) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Action) ProtoMessage() {}

func (x *Action) ProtoReflect() protoreflect.Message {
	mi := &file_protos_v0_cert_action_request_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Action.ProtoReflect.Descriptor instead.
func (*Action) Descriptor() ([]byte, []int) {
	return file_protos_v0_cert_action_request_proto_rawDescGZIP(), []int{1}
}

func (x *Action) GetVersion() uint32 {
	if x != nil {
		return x.Version
	}
	return 0
}

func (x *Action) GetAction() Action_ACTION {
	if x != nil {
		return x.Action
	}
	return Action_ROTATE
}

var File_protos_v0_cert_action_request_proto protoreflect.FileDescriptor

var file_protos_v0_cert_action_request_proto_rawDesc = []byte{
	0x0a, 0x23, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x73, 0x2f, 0x76, 0x30, 0x2f, 0x63, 0x65, 0x72, 0x74,
	0x5f, 0x61, 0x63, 0x74, 0x69, 0x6f, 0x6e, 0x5f, 0x72, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x2e,
	0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x03, 0x6a, 0x76, 0x73, 0x22, 0x41, 0x0a, 0x18, 0x43, 0x65,
	0x72, 0x74, 0x69, 0x66, 0x69, 0x63, 0x61, 0x74, 0x65, 0x41, 0x63, 0x74, 0x69, 0x6f, 0x6e, 0x52,
	0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x12, 0x25, 0x0a, 0x07, 0x61, 0x63, 0x74, 0x69, 0x6f, 0x6e,
	0x73, 0x18, 0x01, 0x20, 0x03, 0x28, 0x0b, 0x32, 0x0b, 0x2e, 0x6a, 0x76, 0x73, 0x2e, 0x41, 0x63,
	0x74, 0x69, 0x6f, 0x6e, 0x52, 0x07, 0x61, 0x63, 0x74, 0x69, 0x6f, 0x6e, 0x73, 0x22, 0x8a, 0x01,
	0x0a, 0x06, 0x41, 0x63, 0x74, 0x69, 0x6f, 0x6e, 0x12, 0x18, 0x0a, 0x07, 0x76, 0x65, 0x72, 0x73,
	0x69, 0x6f, 0x6e, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0d, 0x52, 0x07, 0x76, 0x65, 0x72, 0x73, 0x69,
	0x6f, 0x6e, 0x12, 0x2a, 0x0a, 0x06, 0x61, 0x63, 0x74, 0x69, 0x6f, 0x6e, 0x18, 0x02, 0x20, 0x01,
	0x28, 0x0e, 0x32, 0x12, 0x2e, 0x6a, 0x76, 0x73, 0x2e, 0x41, 0x63, 0x74, 0x69, 0x6f, 0x6e, 0x2e,
	0x41, 0x43, 0x54, 0x49, 0x4f, 0x4e, 0x52, 0x06, 0x61, 0x63, 0x74, 0x69, 0x6f, 0x6e, 0x22, 0x3a,
	0x0a, 0x06, 0x41, 0x43, 0x54, 0x49, 0x4f, 0x4e, 0x12, 0x0a, 0x0a, 0x06, 0x52, 0x4f, 0x54, 0x41,
	0x54, 0x45, 0x10, 0x00, 0x12, 0x11, 0x0a, 0x0d, 0x46, 0x4f, 0x52, 0x43, 0x45, 0x5f, 0x44, 0x49,
	0x53, 0x41, 0x42, 0x4c, 0x45, 0x10, 0x01, 0x12, 0x11, 0x0a, 0x0d, 0x46, 0x4f, 0x52, 0x43, 0x45,
	0x5f, 0x44, 0x45, 0x53, 0x54, 0x52, 0x4f, 0x59, 0x10, 0x02, 0x42, 0x1f, 0x5a, 0x1d, 0x67, 0x69,
	0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x61, 0x62, 0x63, 0x78, 0x79, 0x7a, 0x2f,
	0x6a, 0x76, 0x73, 0x2f, 0x61, 0x70, 0x69, 0x73, 0x2f, 0x76, 0x30, 0x62, 0x06, 0x70, 0x72, 0x6f,
	0x74, 0x6f, 0x33,
}

var (
	file_protos_v0_cert_action_request_proto_rawDescOnce sync.Once
	file_protos_v0_cert_action_request_proto_rawDescData = file_protos_v0_cert_action_request_proto_rawDesc
)

func file_protos_v0_cert_action_request_proto_rawDescGZIP() []byte {
	file_protos_v0_cert_action_request_proto_rawDescOnce.Do(func() {
		file_protos_v0_cert_action_request_proto_rawDescData = protoimpl.X.CompressGZIP(file_protos_v0_cert_action_request_proto_rawDescData)
	})
	return file_protos_v0_cert_action_request_proto_rawDescData
}

var file_protos_v0_cert_action_request_proto_enumTypes = make([]protoimpl.EnumInfo, 1)
var file_protos_v0_cert_action_request_proto_msgTypes = make([]protoimpl.MessageInfo, 2)
var file_protos_v0_cert_action_request_proto_goTypes = []interface{}{
	(Action_ACTION)(0),               // 0: jvs.Action.ACTION
	(*CertificateActionRequest)(nil), // 1: jvs.CertificateActionRequest
	(*Action)(nil),                   // 2: jvs.Action
}
var file_protos_v0_cert_action_request_proto_depIdxs = []int32{
	2, // 0: jvs.CertificateActionRequest.actions:type_name -> jvs.Action
	0, // 1: jvs.Action.action:type_name -> jvs.Action.ACTION
	2, // [2:2] is the sub-list for method output_type
	2, // [2:2] is the sub-list for method input_type
	2, // [2:2] is the sub-list for extension type_name
	2, // [2:2] is the sub-list for extension extendee
	0, // [0:2] is the sub-list for field type_name
}

func init() { file_protos_v0_cert_action_request_proto_init() }
func file_protos_v0_cert_action_request_proto_init() {
	if File_protos_v0_cert_action_request_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_protos_v0_cert_action_request_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*CertificateActionRequest); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_protos_v0_cert_action_request_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Action); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: file_protos_v0_cert_action_request_proto_rawDesc,
			NumEnums:      1,
			NumMessages:   2,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_protos_v0_cert_action_request_proto_goTypes,
		DependencyIndexes: file_protos_v0_cert_action_request_proto_depIdxs,
		EnumInfos:         file_protos_v0_cert_action_request_proto_enumTypes,
		MessageInfos:      file_protos_v0_cert_action_request_proto_msgTypes,
	}.Build()
	File_protos_v0_cert_action_request_proto = out.File
	file_protos_v0_cert_action_request_proto_rawDesc = nil
	file_protos_v0_cert_action_request_proto_goTypes = nil
	file_protos_v0_cert_action_request_proto_depIdxs = nil
}
