// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.36.6
// 	protoc        (unknown)
// source: options/log_options.proto

package options

import (
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	descriptorpb "google.golang.org/protobuf/types/descriptorpb"
	reflect "reflect"
	sync "sync"
	unsafe "unsafe"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

type Sensitive struct {
	state protoimpl.MessageState `protogen:"open.v1"`
	// Types that are valid to be assigned to LogAction:
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
	Pii           bool `protobuf:"varint,5,opt,name=pii,proto3" json:"pii,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
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

func (x *Sensitive) GetLogAction() isSensitive_LogAction {
	if x != nil {
		return x.LogAction
	}
	return nil
}

func (x *Sensitive) GetRedact() bool {
	if x != nil {
		if x, ok := x.LogAction.(*Sensitive_Redact); ok {
			return x.Redact
		}
	}
	return false
}

func (x *Sensitive) GetMask() bool {
	if x != nil {
		if x, ok := x.LogAction.(*Sensitive_Mask); ok {
			return x.Mask
		}
	}
	return false
}

func (x *Sensitive) GetObfuscate() bool {
	if x != nil {
		if x, ok := x.LogAction.(*Sensitive_Obfuscate); ok {
			return x.Obfuscate
		}
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

const file_options_log_options_proto_rawDesc = "" +
	"\n" +
	"\x19options/log_options.proto\x12\aoptions\x1a google/protobuf/descriptor.proto\"\x95\x01\n" +
	"\tSensitive\x12\x18\n" +
	"\x06redact\x18\x01 \x01(\bH\x00R\x06redact\x12\x14\n" +
	"\x04mask\x18\x02 \x01(\bH\x00R\x04mask\x12\x1e\n" +
	"\tobfuscate\x18\x03 \x01(\bH\x00R\tobfuscate\x12\x18\n" +
	"\aencrypt\x18\x04 \x01(\bR\aencrypt\x12\x10\n" +
	"\x03pii\x18\x05 \x01(\bR\x03piiB\f\n" +
	"\n" +
	"log_action:Q\n" +
	"\tsensitive\x12\x1d.google.protobuf.FieldOptions\x18ӆ\x03 \x01(\v2\x12.options.SensitiveR\tsensitiveB\x80\x01\n" +
	"\vcom.optionsB\x0fLogOptionsProtoP\x01Z$github.com/achuala/gosvcextn/options\xa2\x02\x03OXX\xaa\x02\aOptions\xca\x02\aOptions\xe2\x02\x13Options\\GPBMetadata\xea\x02\aOptionsb\x06proto3"

var (
	file_options_log_options_proto_rawDescOnce sync.Once
	file_options_log_options_proto_rawDescData []byte
)

func file_options_log_options_proto_rawDescGZIP() []byte {
	file_options_log_options_proto_rawDescOnce.Do(func() {
		file_options_log_options_proto_rawDescData = protoimpl.X.CompressGZIP(unsafe.Slice(unsafe.StringData(file_options_log_options_proto_rawDesc), len(file_options_log_options_proto_rawDesc)))
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
			RawDescriptor: unsafe.Slice(unsafe.StringData(file_options_log_options_proto_rawDesc), len(file_options_log_options_proto_rawDesc)),
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
	file_options_log_options_proto_goTypes = nil
	file_options_log_options_proto_depIdxs = nil
}
