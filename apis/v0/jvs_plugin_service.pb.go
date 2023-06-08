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

// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.30.0
// 	protoc        v3.20.1
// source: jvs_plugin_service.proto

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

// ValidateJustificationRequest provides a justification for the server to validate.
type ValidateJustificationRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Justification *Justification `protobuf:"bytes,1,opt,name=justification,proto3" json:"justification,omitempty"`
}

func (x *ValidateJustificationRequest) Reset() {
	*x = ValidateJustificationRequest{}
	if protoimpl.UnsafeEnabled {
		mi := &file_jvs_plugin_service_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *ValidateJustificationRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ValidateJustificationRequest) ProtoMessage() {}

func (x *ValidateJustificationRequest) ProtoReflect() protoreflect.Message {
	mi := &file_jvs_plugin_service_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ValidateJustificationRequest.ProtoReflect.Descriptor instead.
func (*ValidateJustificationRequest) Descriptor() ([]byte, []int) {
	return file_jvs_plugin_service_proto_rawDescGZIP(), []int{0}
}

func (x *ValidateJustificationRequest) GetJustification() *Justification {
	if x != nil {
		return x.Justification
	}
	return nil
}

// ValidateJustificationResponse contains the validation result.
type ValidateJustificationResponse struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Valid bool `protobuf:"varint,1,opt,name=valid,proto3" json:"valid,omitempty"`
	// Could be empty if it's valid.
	// Otherwise some warning or error should be provided.
	Warning []string `protobuf:"bytes,2,rep,name=warning,proto3" json:"warning,omitempty"`
	Error   []string `protobuf:"bytes,3,rep,name=error,proto3" json:"error,omitempty"`
}

func (x *ValidateJustificationResponse) Reset() {
	*x = ValidateJustificationResponse{}
	if protoimpl.UnsafeEnabled {
		mi := &file_jvs_plugin_service_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *ValidateJustificationResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ValidateJustificationResponse) ProtoMessage() {}

func (x *ValidateJustificationResponse) ProtoReflect() protoreflect.Message {
	mi := &file_jvs_plugin_service_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ValidateJustificationResponse.ProtoReflect.Descriptor instead.
func (*ValidateJustificationResponse) Descriptor() ([]byte, []int) {
	return file_jvs_plugin_service_proto_rawDescGZIP(), []int{1}
}

func (x *ValidateJustificationResponse) GetValid() bool {
	if x != nil {
		return x.Valid
	}
	return false
}

func (x *ValidateJustificationResponse) GetWarning() []string {
	if x != nil {
		return x.Warning
	}
	return nil
}

func (x *ValidateJustificationResponse) GetError() []string {
	if x != nil {
		return x.Error
	}
	return nil
}

var File_jvs_plugin_service_proto protoreflect.FileDescriptor

var file_jvs_plugin_service_proto_rawDesc = []byte{
	0x0a, 0x18, 0x6a, 0x76, 0x73, 0x5f, 0x70, 0x6c, 0x75, 0x67, 0x69, 0x6e, 0x5f, 0x73, 0x65, 0x72,
	0x76, 0x69, 0x63, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x0a, 0x61, 0x62, 0x63, 0x78,
	0x79, 0x7a, 0x2e, 0x6a, 0x76, 0x73, 0x1a, 0x11, 0x6a, 0x76, 0x73, 0x5f, 0x72, 0x65, 0x71, 0x75,
	0x65, 0x73, 0x74, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x22, 0x5f, 0x0a, 0x1c, 0x56, 0x61, 0x6c,
	0x69, 0x64, 0x61, 0x74, 0x65, 0x4a, 0x75, 0x73, 0x74, 0x69, 0x66, 0x69, 0x63, 0x61, 0x74, 0x69,
	0x6f, 0x6e, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x12, 0x3f, 0x0a, 0x0d, 0x6a, 0x75, 0x73,
	0x74, 0x69, 0x66, 0x69, 0x63, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0b,
	0x32, 0x19, 0x2e, 0x61, 0x62, 0x63, 0x78, 0x79, 0x7a, 0x2e, 0x6a, 0x76, 0x73, 0x2e, 0x4a, 0x75,
	0x73, 0x74, 0x69, 0x66, 0x69, 0x63, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x52, 0x0d, 0x6a, 0x75, 0x73,
	0x74, 0x69, 0x66, 0x69, 0x63, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x22, 0x65, 0x0a, 0x1d, 0x56, 0x61,
	0x6c, 0x69, 0x64, 0x61, 0x74, 0x65, 0x4a, 0x75, 0x73, 0x74, 0x69, 0x66, 0x69, 0x63, 0x61, 0x74,
	0x69, 0x6f, 0x6e, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x12, 0x14, 0x0a, 0x05, 0x76,
	0x61, 0x6c, 0x69, 0x64, 0x18, 0x01, 0x20, 0x01, 0x28, 0x08, 0x52, 0x05, 0x76, 0x61, 0x6c, 0x69,
	0x64, 0x12, 0x18, 0x0a, 0x07, 0x77, 0x61, 0x72, 0x6e, 0x69, 0x6e, 0x67, 0x18, 0x02, 0x20, 0x03,
	0x28, 0x09, 0x52, 0x07, 0x77, 0x61, 0x72, 0x6e, 0x69, 0x6e, 0x67, 0x12, 0x14, 0x0a, 0x05, 0x65,
	0x72, 0x72, 0x6f, 0x72, 0x18, 0x03, 0x20, 0x03, 0x28, 0x09, 0x52, 0x05, 0x65, 0x72, 0x72, 0x6f,
	0x72, 0x32, 0x6c, 0x0a, 0x09, 0x4a, 0x56, 0x53, 0x50, 0x6c, 0x75, 0x67, 0x69, 0x6e, 0x12, 0x5f,
	0x0a, 0x08, 0x56, 0x61, 0x6c, 0x69, 0x64, 0x61, 0x74, 0x65, 0x12, 0x28, 0x2e, 0x61, 0x62, 0x63,
	0x78, 0x79, 0x7a, 0x2e, 0x6a, 0x76, 0x73, 0x2e, 0x56, 0x61, 0x6c, 0x69, 0x64, 0x61, 0x74, 0x65,
	0x4a, 0x75, 0x73, 0x74, 0x69, 0x66, 0x69, 0x63, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x52, 0x65, 0x71,
	0x75, 0x65, 0x73, 0x74, 0x1a, 0x29, 0x2e, 0x61, 0x62, 0x63, 0x78, 0x79, 0x7a, 0x2e, 0x6a, 0x76,
	0x73, 0x2e, 0x56, 0x61, 0x6c, 0x69, 0x64, 0x61, 0x74, 0x65, 0x4a, 0x75, 0x73, 0x74, 0x69, 0x66,
	0x69, 0x63, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x42,
	0x1f, 0x5a, 0x1d, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x61, 0x62,
	0x63, 0x78, 0x79, 0x7a, 0x2f, 0x6a, 0x76, 0x73, 0x2f, 0x61, 0x70, 0x69, 0x73, 0x2f, 0x76, 0x30,
	0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_jvs_plugin_service_proto_rawDescOnce sync.Once
	file_jvs_plugin_service_proto_rawDescData = file_jvs_plugin_service_proto_rawDesc
)

func file_jvs_plugin_service_proto_rawDescGZIP() []byte {
	file_jvs_plugin_service_proto_rawDescOnce.Do(func() {
		file_jvs_plugin_service_proto_rawDescData = protoimpl.X.CompressGZIP(file_jvs_plugin_service_proto_rawDescData)
	})
	return file_jvs_plugin_service_proto_rawDescData
}

var file_jvs_plugin_service_proto_msgTypes = make([]protoimpl.MessageInfo, 2)
var file_jvs_plugin_service_proto_goTypes = []interface{}{
	(*ValidateJustificationRequest)(nil),  // 0: abcxyz.jvs.ValidateJustificationRequest
	(*ValidateJustificationResponse)(nil), // 1: abcxyz.jvs.ValidateJustificationResponse
	(*Justification)(nil),                 // 2: abcxyz.jvs.Justification
}
var file_jvs_plugin_service_proto_depIdxs = []int32{
	2, // 0: abcxyz.jvs.ValidateJustificationRequest.justification:type_name -> abcxyz.jvs.Justification
	0, // 1: abcxyz.jvs.JVSPlugin.Validate:input_type -> abcxyz.jvs.ValidateJustificationRequest
	1, // 2: abcxyz.jvs.JVSPlugin.Validate:output_type -> abcxyz.jvs.ValidateJustificationResponse
	2, // [2:3] is the sub-list for method output_type
	1, // [1:2] is the sub-list for method input_type
	1, // [1:1] is the sub-list for extension type_name
	1, // [1:1] is the sub-list for extension extendee
	0, // [0:1] is the sub-list for field type_name
}

func init() { file_jvs_plugin_service_proto_init() }
func file_jvs_plugin_service_proto_init() {
	if File_jvs_plugin_service_proto != nil {
		return
	}
	file_jvs_request_proto_init()
	if !protoimpl.UnsafeEnabled {
		file_jvs_plugin_service_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*ValidateJustificationRequest); i {
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
		file_jvs_plugin_service_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*ValidateJustificationResponse); i {
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
			RawDescriptor: file_jvs_plugin_service_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   2,
			NumExtensions: 0,
			NumServices:   1,
		},
		GoTypes:           file_jvs_plugin_service_proto_goTypes,
		DependencyIndexes: file_jvs_plugin_service_proto_depIdxs,
		MessageInfos:      file_jvs_plugin_service_proto_msgTypes,
	}.Build()
	File_jvs_plugin_service_proto = out.File
	file_jvs_plugin_service_proto_rawDesc = nil
	file_jvs_plugin_service_proto_goTypes = nil
	file_jvs_plugin_service_proto_depIdxs = nil
}
