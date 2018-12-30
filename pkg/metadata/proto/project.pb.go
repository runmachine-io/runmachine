// Code generated by protoc-gen-go. DO NOT EDIT.
// source: project.proto

package runm

import proto "github.com/golang/protobuf/proto"
import fmt "fmt"
import math "math"

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// This is a compile-time assertion to ensure that this generated file
// is compatible with the proto package it is being compiled against.
// A compilation error at this line likely means your copy of the
// proto package needs to be updated.
const _ = proto.ProtoPackageIsVersion2 // please upgrade the proto package

// A grouping of users of the system. A user may have permissions to read or
// take action within one or more Projects. Projects may be parents of other
// Projects, creating a tree structure.
type Project struct {
	Uuid                 string   `protobuf:"bytes,1,opt,name=uuid" json:"uuid,omitempty"`
	DisplayName          string   `protobuf:"bytes,2,opt,name=display_name,json=displayName" json:"display_name,omitempty"`
	Slug                 string   `protobuf:"bytes,3,opt,name=slug" json:"slug,omitempty"`
	Parent               *Project `protobuf:"bytes,4,opt,name=parent" json:"parent,omitempty"`
	Generation           uint32   `protobuf:"varint,100,opt,name=generation" json:"generation,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *Project) Reset()         { *m = Project{} }
func (m *Project) String() string { return proto.CompactTextString(m) }
func (*Project) ProtoMessage()    {}
func (*Project) Descriptor() ([]byte, []int) {
	return fileDescriptor_project_7a6f13a82e0f2c26, []int{0}
}
func (m *Project) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_Project.Unmarshal(m, b)
}
func (m *Project) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_Project.Marshal(b, m, deterministic)
}
func (dst *Project) XXX_Merge(src proto.Message) {
	xxx_messageInfo_Project.Merge(dst, src)
}
func (m *Project) XXX_Size() int {
	return xxx_messageInfo_Project.Size(m)
}
func (m *Project) XXX_DiscardUnknown() {
	xxx_messageInfo_Project.DiscardUnknown(m)
}

var xxx_messageInfo_Project proto.InternalMessageInfo

func (m *Project) GetUuid() string {
	if m != nil {
		return m.Uuid
	}
	return ""
}

func (m *Project) GetDisplayName() string {
	if m != nil {
		return m.DisplayName
	}
	return ""
}

func (m *Project) GetSlug() string {
	if m != nil {
		return m.Slug
	}
	return ""
}

func (m *Project) GetParent() *Project {
	if m != nil {
		return m.Parent
	}
	return nil
}

func (m *Project) GetGeneration() uint32 {
	if m != nil {
		return m.Generation
	}
	return 0
}

func init() {
	proto.RegisterType((*Project)(nil), "runm.Project")
}

func init() { proto.RegisterFile("project.proto", fileDescriptor_project_7a6f13a82e0f2c26) }

var fileDescriptor_project_7a6f13a82e0f2c26 = []byte{
	// 164 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0xe2, 0xe2, 0x2d, 0x28, 0xca, 0xcf,
	0x4a, 0x4d, 0x2e, 0xd1, 0x2b, 0x28, 0xca, 0x2f, 0xc9, 0x17, 0x62, 0x29, 0x2a, 0xcd, 0xcb, 0x55,
	0x9a, 0xcd, 0xc8, 0xc5, 0x1e, 0x00, 0x11, 0x17, 0x12, 0xe2, 0x62, 0x29, 0x2d, 0xcd, 0x4c, 0x91,
	0x60, 0x54, 0x60, 0xd4, 0xe0, 0x0c, 0x02, 0xb3, 0x85, 0x14, 0xb9, 0x78, 0x52, 0x32, 0x8b, 0x0b,
	0x72, 0x12, 0x2b, 0xe3, 0xf3, 0x12, 0x73, 0x53, 0x25, 0x98, 0xc0, 0x72, 0xdc, 0x50, 0x31, 0xbf,
	0xc4, 0xdc, 0x54, 0x90, 0xb6, 0xe2, 0x9c, 0xd2, 0x74, 0x09, 0x66, 0x88, 0x36, 0x10, 0x5b, 0x48,
	0x95, 0x8b, 0xad, 0x20, 0xb1, 0x28, 0x35, 0xaf, 0x44, 0x82, 0x45, 0x81, 0x51, 0x83, 0xdb, 0x88,
	0x57, 0x0f, 0x64, 0x9b, 0x1e, 0xd4, 0xa6, 0x20, 0xa8, 0xa4, 0x90, 0x1c, 0x17, 0x57, 0x7a, 0x6a,
	0x5e, 0x6a, 0x51, 0x62, 0x49, 0x66, 0x7e, 0x9e, 0x44, 0x8a, 0x02, 0xa3, 0x06, 0x6f, 0x10, 0x92,
	0x48, 0x12, 0x1b, 0xd8, 0xa9, 0xc6, 0x80, 0x00, 0x00, 0x00, 0xff, 0xff, 0x66, 0x62, 0x6f, 0x9e,
	0xbb, 0x00, 0x00, 0x00,
}
