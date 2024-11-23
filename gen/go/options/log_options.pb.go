// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.35.2
// 	protoc        (unknown)
// source: options/log_options.proto

package options

import (
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	descriptorpb "google.golang.org/protobuf/types/descriptorpb"
	reflect "reflect"
	sync "sync"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

type Sensitive struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// Types that are assignable to LogAction:
	//
	//	*Sensitive_Redact
	//	*Sensitive_Mask
	//	*Sensitive_Obfuscate
	LogAction isSensitive_LogAction `protobuf_oneof:"log_action"`
	// Indicates to encrypt the data while storing in permanent storage
	// Note, this will also apply to the logging of the element
	Encrypt bool `protobuf:"varint,4,opt,name=encrypt,proto3" json:"encrypt,omitempty"`
	// Indicates the field is a PII, field with this option will
	// expect the data to be encrypted and not logged in plain text
	Pii bool `protobuf:"varint,5,opt,name=pii,proto3" json:"pii,omitempty"`
}

func (x *Sensitive) Reset() {
	*x = Sensitive{}
	mi := &file_options_log_options_proto_msgTypes[0]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Sensitive) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Sensitive) ProtoMessage() {}

func (x *Sensitive) ProtoReflect() protoreflect.Message {
	mi := &file_options_log_options_proto_msgTypes[0]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Sensitive.ProtoReflect.Descriptor instead.
func (*Sensitive) Descriptor() ([]byte, []int) {
	return file_options_log_options_proto_rawDescGZIP(), []int{0}
}

func (m *Sensitive) GetLogAction() isSensitive_LogAction {
	if m != nil {
		return m.LogAction
	}
	return nil
}

func (x *Sensitive) GetRedact() bool {
	if x, ok := x.GetLogAction().(*Sensitive_Redact); ok {
		return x.Redact
	}
	return false
}

func (x *Sensitive) GetMask() bool {
	if x, ok := x.GetLogAction().(*Sensitive_Mask); ok {
		return x.Mask
	}
	return false
}

func (x *Sensitive) GetObfuscate() bool {
	if x, ok := x.GetLogAction().(*Sensitive_Obfuscate); ok {
		return x.Obfuscate
	}
	return false
}

func (x *Sensitive) GetEncrypt() bool {
	if x != nil {
		return x.Encrypt
	}
	return false
}

func (x *Sensitive) GetPii() bool {
	if x != nil {
		return x.Pii
	}
	return false
}

type isSensitive_LogAction interface {
	isSensitive_LogAction()
}

type Sensitive_Redact struct {
	// Indicates to clear the data while logging
	Redact bool `protobuf:"varint,1,opt,name=redact,proto3,oneof"`
}

type Sensitive_Mask struct {
	// Indicates to mask the data while logging
	Mask bool `protobuf:"varint,2,opt,name=mask,proto3,oneof"`
}

type Sensitive_Obfuscate struct {
	// Indicates to obfuscate the data while logging
	Obfuscate bool `protobuf:"varint,3,opt,name=obfuscate,proto3,oneof"` // Indicates whether data is PII or not
}

func (*Sensitive_Redact) isSensitive_LogAction() {}

func (*Sensitive_Mask) isSensitive_LogAction() {}

func (*Sensitive_Obfuscate) isSensitive_LogAction() {}

var file_options_log_options_proto_extTypes = []protoimpl.ExtensionInfo{
	{
		ExtendedType:  (*descriptorpb.FieldOptions)(nil),
		ExtensionType: (*Sensitive)(nil),
		Field:         50003,
		Name:          "options.sensitive",
		Tag:           "bytes,50003,opt,name=sensitive",
		Filename:      "options/log_options.proto",
	},
}

// Extension fields to descriptorpb.FieldOptions.
var (
	// When set to true, `sensitive` indicates that this field contains sensitive data, such as
	// personally identifiable information, passwords, or private keys, and should be redacted for
	// display by tools aware of this annotation. Note that that this has no effect on standard
	// Protobuf functions such as `TextFormat::PrintToString`.
	//
	// # For example this to be used as below
	//
	//	message SensitiveTestData {
	//	   string name = 1 [(options.sensitive).mask = true];
	//	   string secret = 2 [(options.sensitive).encrypt = true];
	//	 }
	//
	// optional options.Sensitive sensitive = 50003;
	E_Sensitive = &file_options_log_options_proto_extTypes[0]
)

var File_options_log_options_proto protoreflect.FileDescriptor

var file_options_log_options_proto_rawDesc = []byte{
	0x0a, 0x19, 0x6f, 0x70, 0x74, 0x69, 0x6f, 0x6e, 0x73, 0x2f, 0x6c, 0x6f, 0x67, 0x5f, 0x6f, 0x70,
	0x74, 0x69, 0x6f, 0x6e, 0x73, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x07, 0x6f, 0x70, 0x74,
	0x69, 0x6f, 0x6e, 0x73, 0x1a, 0x20, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2f, 0x70, 0x72, 0x6f,
	0x74, 0x6f, 0x62, 0x75, 0x66, 0x2f, 0x64, 0x65, 0x73, 0x63, 0x72, 0x69, 0x70, 0x74, 0x6f, 0x72,
	0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x22, 0x95, 0x01, 0x0a, 0x09, 0x53, 0x65, 0x6e, 0x73, 0x69,
	0x74, 0x69, 0x76, 0x65, 0x12, 0x18, 0x0a, 0x06, 0x72, 0x65, 0x64, 0x61, 0x63, 0x74, 0x18, 0x01,
	0x20, 0x01, 0x28, 0x08, 0x48, 0x00, 0x52, 0x06, 0x72, 0x65, 0x64, 0x61, 0x63, 0x74, 0x12, 0x14,
	0x0a, 0x04, 0x6d, 0x61, 0x73, 0x6b, 0x18, 0x02, 0x20, 0x01, 0x28, 0x08, 0x48, 0x00, 0x52, 0x04,
	0x6d, 0x61, 0x73, 0x6b, 0x12, 0x1e, 0x0a, 0x09, 0x6f, 0x62, 0x66, 0x75, 0x73, 0x63, 0x61, 0x74,
	0x65, 0x18, 0x03, 0x20, 0x01, 0x28, 0x08, 0x48, 0x00, 0x52, 0x09, 0x6f, 0x62, 0x66, 0x75, 0x73,
	0x63, 0x61, 0x74, 0x65, 0x12, 0x18, 0x0a, 0x07, 0x65, 0x6e, 0x63, 0x72, 0x79, 0x70, 0x74, 0x18,
	0x04, 0x20, 0x01, 0x28, 0x08, 0x52, 0x07, 0x65, 0x6e, 0x63, 0x72, 0x79, 0x70, 0x74, 0x12, 0x10,
	0x0a, 0x03, 0x70, 0x69, 0x69, 0x18, 0x05, 0x20, 0x01, 0x28, 0x08, 0x52, 0x03, 0x70, 0x69, 0x69,
	0x42, 0x0c, 0x0a, 0x0a, 0x6c, 0x6f, 0x67, 0x5f, 0x61, 0x63, 0x74, 0x69, 0x6f, 0x6e, 0x3a, 0x51,
	0x0a, 0x09, 0x73, 0x65, 0x6e, 0x73, 0x69, 0x74, 0x69, 0x76, 0x65, 0x12, 0x1d, 0x2e, 0x67, 0x6f,
	0x6f, 0x67, 0x6c, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e, 0x46, 0x69,
	0x65, 0x6c, 0x64, 0x4f, 0x70, 0x74, 0x69, 0x6f, 0x6e, 0x73, 0x18, 0xd3, 0x86, 0x03, 0x20, 0x01,
	0x28, 0x0b, 0x32, 0x12, 0x2e, 0x6f, 0x70, 0x74, 0x69, 0x6f, 0x6e, 0x73, 0x2e, 0x53, 0x65, 0x6e,
	0x73, 0x69, 0x74, 0x69, 0x76, 0x65, 0x52, 0x09, 0x73, 0x65, 0x6e, 0x73, 0x69, 0x74, 0x69, 0x76,
	0x65, 0x42, 0x80, 0x01, 0x0a, 0x0b, 0x63, 0x6f, 0x6d, 0x2e, 0x6f, 0x70, 0x74, 0x69, 0x6f, 0x6e,
	0x73, 0x42, 0x0f, 0x4c, 0x6f, 0x67, 0x4f, 0x70, 0x74, 0x69, 0x6f, 0x6e, 0x73, 0x50, 0x72, 0x6f,
	0x74, 0x6f, 0x50, 0x01, 0x5a, 0x24, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d,
	0x2f, 0x61, 0x63, 0x68, 0x75, 0x61, 0x6c, 0x61, 0x2f, 0x67, 0x6f, 0x73, 0x76, 0x63, 0x65, 0x78,
	0x74, 0x6e, 0x2f, 0x6f, 0x70, 0x74, 0x69, 0x6f, 0x6e, 0x73, 0xa2, 0x02, 0x03, 0x4f, 0x58, 0x58,
	0xaa, 0x02, 0x07, 0x4f, 0x70, 0x74, 0x69, 0x6f, 0x6e, 0x73, 0xca, 0x02, 0x07, 0x4f, 0x70, 0x74,
	0x69, 0x6f, 0x6e, 0x73, 0xe2, 0x02, 0x13, 0x4f, 0x70, 0x74, 0x69, 0x6f, 0x6e, 0x73, 0x5c, 0x47,
	0x50, 0x42, 0x4d, 0x65, 0x74, 0x61, 0x64, 0x61, 0x74, 0x61, 0xea, 0x02, 0x07, 0x4f, 0x70, 0x74,
	0x69, 0x6f, 0x6e, 0x73, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_options_log_options_proto_rawDescOnce sync.Once
	file_options_log_options_proto_rawDescData = file_options_log_options_proto_rawDesc
)

func file_options_log_options_proto_rawDescGZIP() []byte {
	file_options_log_options_proto_rawDescOnce.Do(func() {
		file_options_log_options_proto_rawDescData = protoimpl.X.CompressGZIP(file_options_log_options_proto_rawDescData)
	})
	return file_options_log_options_proto_rawDescData
}

var file_options_log_options_proto_msgTypes = make([]protoimpl.MessageInfo, 1)
var file_options_log_options_proto_goTypes = []any{
	(*Sensitive)(nil),                 // 0: options.Sensitive
	(*descriptorpb.FieldOptions)(nil), // 1: google.protobuf.FieldOptions
}
var file_options_log_options_proto_depIdxs = []int32{
	1, // 0: options.sensitive:extendee -> google.protobuf.FieldOptions
	0, // 1: options.sensitive:type_name -> options.Sensitive
	2, // [2:2] is the sub-list for method output_type
	2, // [2:2] is the sub-list for method input_type
	1, // [1:2] is the sub-list for extension type_name
	0, // [0:1] is the sub-list for extension extendee
	0, // [0:0] is the sub-list for field type_name
}

func init() { file_options_log_options_proto_init() }
func file_options_log_options_proto_init() {
	if File_options_log_options_proto != nil {
		return
	}
	file_options_log_options_proto_msgTypes[0].OneofWrappers = []any{
		(*Sensitive_Redact)(nil),
		(*Sensitive_Mask)(nil),
		(*Sensitive_Obfuscate)(nil),
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: file_options_log_options_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   1,
			NumExtensions: 1,
			NumServices:   0,
		},
		GoTypes:           file_options_log_options_proto_goTypes,
		DependencyIndexes: file_options_log_options_proto_depIdxs,
		MessageInfos:      file_options_log_options_proto_msgTypes,
		ExtensionInfos:    file_options_log_options_proto_extTypes,
	}.Build()
	File_options_log_options_proto = out.File
	file_options_log_options_proto_rawDesc = nil
	file_options_log_options_proto_goTypes = nil
	file_options_log_options_proto_depIdxs = nil
}
