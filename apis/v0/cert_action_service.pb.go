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
// 	protoc-gen-go v1.26.0
// 	protoc        v3.19.4
// source: protos/v0/cert_action_service.proto

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

// CertificateActionResponse is a blank response.
type CertificateActionResponse struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields
}

func (x *CertificateActionResponse) Reset() {
	*x = CertificateActionResponse{}
	if protoimpl.UnsafeEnabled {
		mi := &file_protos_v0_cert_action_service_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *CertificateActionResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*CertificateActionResponse) ProtoMessage() {}

func (x *CertificateActionResponse) ProtoReflect() protoreflect.Message {
	mi := &file_protos_v0_cert_action_service_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use CertificateActionResponse.ProtoReflect.Descriptor instead.
func (*CertificateActionResponse) Descriptor() ([]byte, []int) {
	return file_protos_v0_cert_action_service_proto_rawDescGZIP(), []int{0}
}

var File_protos_v0_cert_action_service_proto protoreflect.FileDescriptor

var file_protos_v0_cert_action_service_proto_rawDesc = []byte{
	0x0a, 0x23, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x73, 0x2f, 0x76, 0x30, 0x2f, 0x63, 0x65, 0x72, 0x74,
	0x5f, 0x61, 0x63, 0x74, 0x69, 0x6f, 0x6e, 0x5f, 0x73, 0x65, 0x72, 0x76, 0x69, 0x63, 0x65, 0x2e,
	0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x0a, 0x61, 0x62, 0x63, 0x78, 0x79, 0x7a, 0x2e, 0x6a, 0x76,
	0x73, 0x1a, 0x23, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x73, 0x2f, 0x76, 0x30, 0x2f, 0x63, 0x65, 0x72,
	0x74, 0x5f, 0x61, 0x63, 0x74, 0x69, 0x6f, 0x6e, 0x5f, 0x72, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74,
	0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x22, 0x1b, 0x0a, 0x19, 0x43, 0x65, 0x72, 0x74, 0x69, 0x66,
	0x69, 0x63, 0x61, 0x74, 0x65, 0x41, 0x63, 0x74, 0x69, 0x6f, 0x6e, 0x52, 0x65, 0x73, 0x70, 0x6f,
	0x6e, 0x73, 0x65, 0x32, 0x7c, 0x0a, 0x18, 0x43, 0x65, 0x72, 0x74, 0x69, 0x66, 0x69, 0x63, 0x61,
	0x74, 0x65, 0x41, 0x63, 0x74, 0x69, 0x6f, 0x6e, 0x53, 0x65, 0x72, 0x76, 0x69, 0x63, 0x65, 0x12,
	0x60, 0x0a, 0x11, 0x43, 0x65, 0x72, 0x74, 0x69, 0x66, 0x69, 0x63, 0x61, 0x74, 0x65, 0x41, 0x63,
	0x74, 0x69, 0x6f, 0x6e, 0x12, 0x24, 0x2e, 0x61, 0x62, 0x63, 0x78, 0x79, 0x7a, 0x2e, 0x6a, 0x76,
	0x73, 0x2e, 0x43, 0x65, 0x72, 0x74, 0x69, 0x66, 0x69, 0x63, 0x61, 0x74, 0x65, 0x41, 0x63, 0x74,
	0x69, 0x6f, 0x6e, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x1a, 0x25, 0x2e, 0x61, 0x62, 0x63,
	0x78, 0x79, 0x7a, 0x2e, 0x6a, 0x76, 0x73, 0x2e, 0x43, 0x65, 0x72, 0x74, 0x69, 0x66, 0x69, 0x63,
	0x61, 0x74, 0x65, 0x41, 0x63, 0x74, 0x69, 0x6f, 0x6e, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73,
	0x65, 0x42, 0x1f, 0x5a, 0x1d, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f,
	0x61, 0x62, 0x63, 0x78, 0x79, 0x7a, 0x2f, 0x6a, 0x76, 0x73, 0x2f, 0x61, 0x70, 0x69, 0x73, 0x2f,
	0x76, 0x30, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_protos_v0_cert_action_service_proto_rawDescOnce sync.Once
	file_protos_v0_cert_action_service_proto_rawDescData = file_protos_v0_cert_action_service_proto_rawDesc
)

func file_protos_v0_cert_action_service_proto_rawDescGZIP() []byte {
	file_protos_v0_cert_action_service_proto_rawDescOnce.Do(func() {
		file_protos_v0_cert_action_service_proto_rawDescData = protoimpl.X.CompressGZIP(file_protos_v0_cert_action_service_proto_rawDescData)
	})
	return file_protos_v0_cert_action_service_proto_rawDescData
}

var file_protos_v0_cert_action_service_proto_msgTypes = make([]protoimpl.MessageInfo, 1)
var file_protos_v0_cert_action_service_proto_goTypes = []interface{}{
	(*CertificateActionResponse)(nil), // 0: abcxyz.jvs.CertificateActionResponse
	(*CertificateActionRequest)(nil),  // 1: abcxyz.jvs.CertificateActionRequest
}
var file_protos_v0_cert_action_service_proto_depIdxs = []int32{
	1, // 0: abcxyz.jvs.CertificateActionService.CertificateAction:input_type -> abcxyz.jvs.CertificateActionRequest
	0, // 1: abcxyz.jvs.CertificateActionService.CertificateAction:output_type -> abcxyz.jvs.CertificateActionResponse
	1, // [1:2] is the sub-list for method output_type
	0, // [0:1] is the sub-list for method input_type
	0, // [0:0] is the sub-list for extension type_name
	0, // [0:0] is the sub-list for extension extendee
	0, // [0:0] is the sub-list for field type_name
}

func init() { file_protos_v0_cert_action_service_proto_init() }
func file_protos_v0_cert_action_service_proto_init() {
	if File_protos_v0_cert_action_service_proto != nil {
		return
	}
	file_protos_v0_cert_action_request_proto_init()
	if !protoimpl.UnsafeEnabled {
		file_protos_v0_cert_action_service_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*CertificateActionResponse); i {
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
			RawDescriptor: file_protos_v0_cert_action_service_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   1,
			NumExtensions: 0,
			NumServices:   1,
		},
		GoTypes:           file_protos_v0_cert_action_service_proto_goTypes,
		DependencyIndexes: file_protos_v0_cert_action_service_proto_depIdxs,
		MessageInfos:      file_protos_v0_cert_action_service_proto_msgTypes,
	}.Build()
	File_protos_v0_cert_action_service_proto = out.File
	file_protos_v0_cert_action_service_proto_rawDesc = nil
	file_protos_v0_cert_action_service_proto_goTypes = nil
	file_protos_v0_cert_action_service_proto_depIdxs = nil
}
