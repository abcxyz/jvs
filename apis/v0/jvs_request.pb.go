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
// source: protos/v0/jvs_request.proto

package v0

import (
	duration "github.com/golang/protobuf/ptypes/duration"
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

// CreateJustificationRequest provides a justification to the server in order to
// receive a token.
type CreateJustificationRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Justifications []*Justification   `protobuf:"bytes,1,rep,name=justifications,proto3" json:"justifications,omitempty"`
	Ttl            *duration.Duration `protobuf:"bytes,2,opt,name=ttl,proto3" json:"ttl,omitempty"`
}

func (x *CreateJustificationRequest) Reset() {
	*x = CreateJustificationRequest{}
	if protoimpl.UnsafeEnabled {
		mi := &file_protos_v0_jvs_request_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *CreateJustificationRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*CreateJustificationRequest) ProtoMessage() {}

func (x *CreateJustificationRequest) ProtoReflect() protoreflect.Message {
	mi := &file_protos_v0_jvs_request_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use CreateJustificationRequest.ProtoReflect.Descriptor instead.
func (*CreateJustificationRequest) Descriptor() ([]byte, []int) {
	return file_protos_v0_jvs_request_proto_rawDescGZIP(), []int{0}
}

func (x *CreateJustificationRequest) GetJustifications() []*Justification {
	if x != nil {
		return x.Justifications
	}
	return nil
}

func (x *CreateJustificationRequest) GetTtl() *duration.Duration {
	if x != nil {
		return x.Ttl
	}
	return nil
}

// Justification is intended to be used to provide reasons that data access is
// required.
type Justification struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Category string `protobuf:"bytes,1,opt,name=category,proto3" json:"category,omitempty"` // In MVP, the only supported category is "explanation".
	Value    string `protobuf:"bytes,2,opt,name=value,proto3" json:"value,omitempty"`
}

func (x *Justification) Reset() {
	*x = Justification{}
	if protoimpl.UnsafeEnabled {
		mi := &file_protos_v0_jvs_request_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Justification) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Justification) ProtoMessage() {}

func (x *Justification) ProtoReflect() protoreflect.Message {
	mi := &file_protos_v0_jvs_request_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Justification.ProtoReflect.Descriptor instead.
func (*Justification) Descriptor() ([]byte, []int) {
	return file_protos_v0_jvs_request_proto_rawDescGZIP(), []int{1}
}

func (x *Justification) GetCategory() string {
	if x != nil {
		return x.Category
	}
	return ""
}

func (x *Justification) GetValue() string {
	if x != nil {
		return x.Value
	}
	return ""
}

var File_protos_v0_jvs_request_proto protoreflect.FileDescriptor

var file_protos_v0_jvs_request_proto_rawDesc = []byte{
	0x0a, 0x1b, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x73, 0x2f, 0x76, 0x30, 0x2f, 0x6a, 0x76, 0x73, 0x5f,
	0x72, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x0a, 0x61,
	0x62, 0x63, 0x78, 0x79, 0x7a, 0x2e, 0x6a, 0x76, 0x73, 0x1a, 0x1e, 0x67, 0x6f, 0x6f, 0x67, 0x6c,
	0x65, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2f, 0x64, 0x75, 0x72, 0x61, 0x74,
	0x69, 0x6f, 0x6e, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x22, 0x8c, 0x01, 0x0a, 0x1a, 0x43, 0x72,
	0x65, 0x61, 0x74, 0x65, 0x4a, 0x75, 0x73, 0x74, 0x69, 0x66, 0x69, 0x63, 0x61, 0x74, 0x69, 0x6f,
	0x6e, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x12, 0x41, 0x0a, 0x0e, 0x6a, 0x75, 0x73, 0x74,
	0x69, 0x66, 0x69, 0x63, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x73, 0x18, 0x01, 0x20, 0x03, 0x28, 0x0b,
	0x32, 0x19, 0x2e, 0x61, 0x62, 0x63, 0x78, 0x79, 0x7a, 0x2e, 0x6a, 0x76, 0x73, 0x2e, 0x4a, 0x75,
	0x73, 0x74, 0x69, 0x66, 0x69, 0x63, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x52, 0x0e, 0x6a, 0x75, 0x73,
	0x74, 0x69, 0x66, 0x69, 0x63, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x73, 0x12, 0x2b, 0x0a, 0x03, 0x74,
	0x74, 0x6c, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x19, 0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c,
	0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e, 0x44, 0x75, 0x72, 0x61, 0x74,
	0x69, 0x6f, 0x6e, 0x52, 0x03, 0x74, 0x74, 0x6c, 0x22, 0x41, 0x0a, 0x0d, 0x4a, 0x75, 0x73, 0x74,
	0x69, 0x66, 0x69, 0x63, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x12, 0x1a, 0x0a, 0x08, 0x63, 0x61, 0x74,
	0x65, 0x67, 0x6f, 0x72, 0x79, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x08, 0x63, 0x61, 0x74,
	0x65, 0x67, 0x6f, 0x72, 0x79, 0x12, 0x14, 0x0a, 0x05, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x18, 0x02,
	0x20, 0x01, 0x28, 0x09, 0x52, 0x05, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x42, 0x1f, 0x5a, 0x1d, 0x67,
	0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x61, 0x62, 0x63, 0x78, 0x79, 0x7a,
	0x2f, 0x6a, 0x76, 0x73, 0x2f, 0x61, 0x70, 0x69, 0x73, 0x2f, 0x76, 0x30, 0x62, 0x06, 0x70, 0x72,
	0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_protos_v0_jvs_request_proto_rawDescOnce sync.Once
	file_protos_v0_jvs_request_proto_rawDescData = file_protos_v0_jvs_request_proto_rawDesc
)

func file_protos_v0_jvs_request_proto_rawDescGZIP() []byte {
	file_protos_v0_jvs_request_proto_rawDescOnce.Do(func() {
		file_protos_v0_jvs_request_proto_rawDescData = protoimpl.X.CompressGZIP(file_protos_v0_jvs_request_proto_rawDescData)
	})
	return file_protos_v0_jvs_request_proto_rawDescData
}

var file_protos_v0_jvs_request_proto_msgTypes = make([]protoimpl.MessageInfo, 2)
var file_protos_v0_jvs_request_proto_goTypes = []interface{}{
	(*CreateJustificationRequest)(nil), // 0: abcxyz.jvs.CreateJustificationRequest
	(*Justification)(nil),              // 1: abcxyz.jvs.Justification
	(*duration.Duration)(nil),          // 2: google.protobuf.Duration
}
var file_protos_v0_jvs_request_proto_depIdxs = []int32{
	1, // 0: abcxyz.jvs.CreateJustificationRequest.justifications:type_name -> abcxyz.jvs.Justification
	2, // 1: abcxyz.jvs.CreateJustificationRequest.ttl:type_name -> google.protobuf.Duration
	2, // [2:2] is the sub-list for method output_type
	2, // [2:2] is the sub-list for method input_type
	2, // [2:2] is the sub-list for extension type_name
	2, // [2:2] is the sub-list for extension extendee
	0, // [0:2] is the sub-list for field type_name
}

func init() { file_protos_v0_jvs_request_proto_init() }
func file_protos_v0_jvs_request_proto_init() {
	if File_protos_v0_jvs_request_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_protos_v0_jvs_request_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*CreateJustificationRequest); i {
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
		file_protos_v0_jvs_request_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Justification); i {
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
			RawDescriptor: file_protos_v0_jvs_request_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   2,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_protos_v0_jvs_request_proto_goTypes,
		DependencyIndexes: file_protos_v0_jvs_request_proto_depIdxs,
		MessageInfos:      file_protos_v0_jvs_request_proto_msgTypes,
	}.Build()
	File_protos_v0_jvs_request_proto = out.File
	file_protos_v0_jvs_request_proto_rawDesc = nil
	file_protos_v0_jvs_request_proto_goTypes = nil
	file_protos_v0_jvs_request_proto_depIdxs = nil
}
