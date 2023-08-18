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
// 	protoc-gen-go v1.28.1
// 	protoc        v4.23.3
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
	// Additional info the plugin may want to encapsulate in the Justification.
	// It's not intended for user input.
	Annotation map[string]string `protobuf:"bytes,4,rep,name=annotation,proto3" json:"annotation,omitempty" protobuf_key:"bytes,1,opt,name=key,proto3" protobuf_val:"bytes,2,opt,name=value,proto3"`
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

func (x *ValidateJustificationResponse) GetAnnotation() map[string]string {
	if x != nil {
		return x.Annotation
	}
	return nil
}

// GetUIDataRequest is the request to get the plugin data for display purposes.
type GetUIDataRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields
}

func (x *GetUIDataRequest) Reset() {
	*x = GetUIDataRequest{}
	if protoimpl.UnsafeEnabled {
		mi := &file_jvs_plugin_service_proto_msgTypes[2]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *GetUIDataRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*GetUIDataRequest) ProtoMessage() {}

func (x *GetUIDataRequest) ProtoReflect() protoreflect.Message {
	mi := &file_jvs_plugin_service_proto_msgTypes[2]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use GetUIDataRequest.ProtoReflect.Descriptor instead.
func (*GetUIDataRequest) Descriptor() ([]byte, []int) {
	return file_jvs_plugin_service_proto_rawDescGZIP(), []int{2}
}

// The UIData comprises the data that will be displayed. At present, it exclusively includes the display_name and hint.
type UIData struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// The display name for the plugin, e.g. for the web UI.
	DisplayName string `protobuf:"bytes,1,opt,name=display_name,json=displayName,proto3" json:"display_name,omitempty"`
	// The hint for what value to put as the justification.
	Hint string `protobuf:"bytes,2,opt,name=hint,proto3" json:"hint,omitempty"`
}

func (x *UIData) Reset() {
	*x = UIData{}
	if protoimpl.UnsafeEnabled {
		mi := &file_jvs_plugin_service_proto_msgTypes[3]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *UIData) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*UIData) ProtoMessage() {}

func (x *UIData) ProtoReflect() protoreflect.Message {
	mi := &file_jvs_plugin_service_proto_msgTypes[3]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use UIData.ProtoReflect.Descriptor instead.
func (*UIData) Descriptor() ([]byte, []int) {
	return file_jvs_plugin_service_proto_rawDescGZIP(), []int{3}
}

func (x *UIData) GetDisplayName() string {
	if x != nil {
		return x.DisplayName
	}
	return ""
}

func (x *UIData) GetHint() string {
	if x != nil {
		return x.Hint
	}
	return ""
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
	0x74, 0x69, 0x66, 0x69, 0x63, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x22, 0xff, 0x01, 0x0a, 0x1d, 0x56,
	0x61, 0x6c, 0x69, 0x64, 0x61, 0x74, 0x65, 0x4a, 0x75, 0x73, 0x74, 0x69, 0x66, 0x69, 0x63, 0x61,
	0x74, 0x69, 0x6f, 0x6e, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x12, 0x14, 0x0a, 0x05,
	0x76, 0x61, 0x6c, 0x69, 0x64, 0x18, 0x01, 0x20, 0x01, 0x28, 0x08, 0x52, 0x05, 0x76, 0x61, 0x6c,
	0x69, 0x64, 0x12, 0x18, 0x0a, 0x07, 0x77, 0x61, 0x72, 0x6e, 0x69, 0x6e, 0x67, 0x18, 0x02, 0x20,
	0x03, 0x28, 0x09, 0x52, 0x07, 0x77, 0x61, 0x72, 0x6e, 0x69, 0x6e, 0x67, 0x12, 0x14, 0x0a, 0x05,
	0x65, 0x72, 0x72, 0x6f, 0x72, 0x18, 0x03, 0x20, 0x03, 0x28, 0x09, 0x52, 0x05, 0x65, 0x72, 0x72,
	0x6f, 0x72, 0x12, 0x59, 0x0a, 0x0a, 0x61, 0x6e, 0x6e, 0x6f, 0x74, 0x61, 0x74, 0x69, 0x6f, 0x6e,
	0x18, 0x04, 0x20, 0x03, 0x28, 0x0b, 0x32, 0x39, 0x2e, 0x61, 0x62, 0x63, 0x78, 0x79, 0x7a, 0x2e,
	0x6a, 0x76, 0x73, 0x2e, 0x56, 0x61, 0x6c, 0x69, 0x64, 0x61, 0x74, 0x65, 0x4a, 0x75, 0x73, 0x74,
	0x69, 0x66, 0x69, 0x63, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73,
	0x65, 0x2e, 0x41, 0x6e, 0x6e, 0x6f, 0x74, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x45, 0x6e, 0x74, 0x72,
	0x79, 0x52, 0x0a, 0x61, 0x6e, 0x6e, 0x6f, 0x74, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x1a, 0x3d, 0x0a,
	0x0f, 0x41, 0x6e, 0x6e, 0x6f, 0x74, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x45, 0x6e, 0x74, 0x72, 0x79,
	0x12, 0x10, 0x0a, 0x03, 0x6b, 0x65, 0x79, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x03, 0x6b,
	0x65, 0x79, 0x12, 0x14, 0x0a, 0x05, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28,
	0x09, 0x52, 0x05, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x3a, 0x02, 0x38, 0x01, 0x22, 0x12, 0x0a, 0x10,
	0x47, 0x65, 0x74, 0x55, 0x49, 0x44, 0x61, 0x74, 0x61, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74,
	0x22, 0x3f, 0x0a, 0x06, 0x55, 0x49, 0x44, 0x61, 0x74, 0x61, 0x12, 0x21, 0x0a, 0x0c, 0x64, 0x69,
	0x73, 0x70, 0x6c, 0x61, 0x79, 0x5f, 0x6e, 0x61, 0x6d, 0x65, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09,
	0x52, 0x0b, 0x64, 0x69, 0x73, 0x70, 0x6c, 0x61, 0x79, 0x4e, 0x61, 0x6d, 0x65, 0x12, 0x12, 0x0a,
	0x04, 0x68, 0x69, 0x6e, 0x74, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x04, 0x68, 0x69, 0x6e,
	0x74, 0x32, 0xab, 0x01, 0x0a, 0x09, 0x4a, 0x56, 0x53, 0x50, 0x6c, 0x75, 0x67, 0x69, 0x6e, 0x12,
	0x5f, 0x0a, 0x08, 0x56, 0x61, 0x6c, 0x69, 0x64, 0x61, 0x74, 0x65, 0x12, 0x28, 0x2e, 0x61, 0x62,
	0x63, 0x78, 0x79, 0x7a, 0x2e, 0x6a, 0x76, 0x73, 0x2e, 0x56, 0x61, 0x6c, 0x69, 0x64, 0x61, 0x74,
	0x65, 0x4a, 0x75, 0x73, 0x74, 0x69, 0x66, 0x69, 0x63, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x52, 0x65,
	0x71, 0x75, 0x65, 0x73, 0x74, 0x1a, 0x29, 0x2e, 0x61, 0x62, 0x63, 0x78, 0x79, 0x7a, 0x2e, 0x6a,
	0x76, 0x73, 0x2e, 0x56, 0x61, 0x6c, 0x69, 0x64, 0x61, 0x74, 0x65, 0x4a, 0x75, 0x73, 0x74, 0x69,
	0x66, 0x69, 0x63, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65,
	0x12, 0x3d, 0x0a, 0x09, 0x47, 0x65, 0x74, 0x55, 0x49, 0x44, 0x61, 0x74, 0x61, 0x12, 0x1c, 0x2e,
	0x61, 0x62, 0x63, 0x78, 0x79, 0x7a, 0x2e, 0x6a, 0x76, 0x73, 0x2e, 0x47, 0x65, 0x74, 0x55, 0x49,
	0x44, 0x61, 0x74, 0x61, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x1a, 0x12, 0x2e, 0x61, 0x62,
	0x63, 0x78, 0x79, 0x7a, 0x2e, 0x6a, 0x76, 0x73, 0x2e, 0x55, 0x49, 0x44, 0x61, 0x74, 0x61, 0x42,
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

var file_jvs_plugin_service_proto_msgTypes = make([]protoimpl.MessageInfo, 5)
var file_jvs_plugin_service_proto_goTypes = []interface{}{
	(*ValidateJustificationRequest)(nil),  // 0: abcxyz.jvs.ValidateJustificationRequest
	(*ValidateJustificationResponse)(nil), // 1: abcxyz.jvs.ValidateJustificationResponse
	(*GetUIDataRequest)(nil),              // 2: abcxyz.jvs.GetUIDataRequest
	(*UIData)(nil),                        // 3: abcxyz.jvs.UIData
	nil,                                   // 4: abcxyz.jvs.ValidateJustificationResponse.AnnotationEntry
	(*Justification)(nil),                 // 5: abcxyz.jvs.Justification
}
var file_jvs_plugin_service_proto_depIdxs = []int32{
	5, // 0: abcxyz.jvs.ValidateJustificationRequest.justification:type_name -> abcxyz.jvs.Justification
	4, // 1: abcxyz.jvs.ValidateJustificationResponse.annotation:type_name -> abcxyz.jvs.ValidateJustificationResponse.AnnotationEntry
	0, // 2: abcxyz.jvs.JVSPlugin.Validate:input_type -> abcxyz.jvs.ValidateJustificationRequest
	2, // 3: abcxyz.jvs.JVSPlugin.GetUIData:input_type -> abcxyz.jvs.GetUIDataRequest
	1, // 4: abcxyz.jvs.JVSPlugin.Validate:output_type -> abcxyz.jvs.ValidateJustificationResponse
	3, // 5: abcxyz.jvs.JVSPlugin.GetUIData:output_type -> abcxyz.jvs.UIData
	4, // [4:6] is the sub-list for method output_type
	2, // [2:4] is the sub-list for method input_type
	2, // [2:2] is the sub-list for extension type_name
	2, // [2:2] is the sub-list for extension extendee
	0, // [0:2] is the sub-list for field type_name
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
		file_jvs_plugin_service_proto_msgTypes[2].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*GetUIDataRequest); i {
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
		file_jvs_plugin_service_proto_msgTypes[3].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*UIData); i {
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
			NumMessages:   5,
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
