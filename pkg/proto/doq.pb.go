// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.36.6
// 	protoc        v5.28.1
// source: pkg/proto/doq.proto

package proto

import (
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
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

type CreateQueueRequest struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Name          string                 `protobuf:"bytes,1,opt,name=name,proto3" json:"name,omitempty"`
	Type          string                 `protobuf:"bytes,2,opt,name=type,proto3" json:"type,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *CreateQueueRequest) Reset() {
	*x = CreateQueueRequest{}
	mi := &file_pkg_proto_doq_proto_msgTypes[0]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *CreateQueueRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*CreateQueueRequest) ProtoMessage() {}

func (x *CreateQueueRequest) ProtoReflect() protoreflect.Message {
	mi := &file_pkg_proto_doq_proto_msgTypes[0]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use CreateQueueRequest.ProtoReflect.Descriptor instead.
func (*CreateQueueRequest) Descriptor() ([]byte, []int) {
	return file_pkg_proto_doq_proto_rawDescGZIP(), []int{0}
}

func (x *CreateQueueRequest) GetName() string {
	if x != nil {
		return x.Name
	}
	return ""
}

func (x *CreateQueueRequest) GetType() string {
	if x != nil {
		return x.Type
	}
	return ""
}

type CreateQueueResponse struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Success       bool                   `protobuf:"varint,1,opt,name=success,proto3" json:"success,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *CreateQueueResponse) Reset() {
	*x = CreateQueueResponse{}
	mi := &file_pkg_proto_doq_proto_msgTypes[1]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *CreateQueueResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*CreateQueueResponse) ProtoMessage() {}

func (x *CreateQueueResponse) ProtoReflect() protoreflect.Message {
	mi := &file_pkg_proto_doq_proto_msgTypes[1]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use CreateQueueResponse.ProtoReflect.Descriptor instead.
func (*CreateQueueResponse) Descriptor() ([]byte, []int) {
	return file_pkg_proto_doq_proto_rawDescGZIP(), []int{1}
}

func (x *CreateQueueResponse) GetSuccess() bool {
	if x != nil {
		return x.Success
	}
	return false
}

type DeleteQueueRequest struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Name          string                 `protobuf:"bytes,1,opt,name=name,proto3" json:"name,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *DeleteQueueRequest) Reset() {
	*x = DeleteQueueRequest{}
	mi := &file_pkg_proto_doq_proto_msgTypes[2]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *DeleteQueueRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*DeleteQueueRequest) ProtoMessage() {}

func (x *DeleteQueueRequest) ProtoReflect() protoreflect.Message {
	mi := &file_pkg_proto_doq_proto_msgTypes[2]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use DeleteQueueRequest.ProtoReflect.Descriptor instead.
func (*DeleteQueueRequest) Descriptor() ([]byte, []int) {
	return file_pkg_proto_doq_proto_rawDescGZIP(), []int{2}
}

func (x *DeleteQueueRequest) GetName() string {
	if x != nil {
		return x.Name
	}
	return ""
}

type DeleteQueueResponse struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Success       bool                   `protobuf:"varint,1,opt,name=success,proto3" json:"success,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *DeleteQueueResponse) Reset() {
	*x = DeleteQueueResponse{}
	mi := &file_pkg_proto_doq_proto_msgTypes[3]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *DeleteQueueResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*DeleteQueueResponse) ProtoMessage() {}

func (x *DeleteQueueResponse) ProtoReflect() protoreflect.Message {
	mi := &file_pkg_proto_doq_proto_msgTypes[3]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use DeleteQueueResponse.ProtoReflect.Descriptor instead.
func (*DeleteQueueResponse) Descriptor() ([]byte, []int) {
	return file_pkg_proto_doq_proto_rawDescGZIP(), []int{3}
}

func (x *DeleteQueueResponse) GetSuccess() bool {
	if x != nil {
		return x.Success
	}
	return false
}

type EnqueueRequest struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	QueueName     string                 `protobuf:"bytes,1,opt,name=queueName,proto3" json:"queueName,omitempty"`
	Group         string                 `protobuf:"bytes,2,opt,name=group,proto3" json:"group,omitempty"`
	Priority      int64                  `protobuf:"varint,3,opt,name=priority,proto3" json:"priority,omitempty"`
	Content       string                 `protobuf:"bytes,4,opt,name=content,proto3" json:"content,omitempty"`
	Metadata      map[string]string      `protobuf:"bytes,5,rep,name=metadata,proto3" json:"metadata,omitempty" protobuf_key:"bytes,1,opt,name=key" protobuf_val:"bytes,2,opt,name=value"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *EnqueueRequest) Reset() {
	*x = EnqueueRequest{}
	mi := &file_pkg_proto_doq_proto_msgTypes[4]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *EnqueueRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*EnqueueRequest) ProtoMessage() {}

func (x *EnqueueRequest) ProtoReflect() protoreflect.Message {
	mi := &file_pkg_proto_doq_proto_msgTypes[4]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use EnqueueRequest.ProtoReflect.Descriptor instead.
func (*EnqueueRequest) Descriptor() ([]byte, []int) {
	return file_pkg_proto_doq_proto_rawDescGZIP(), []int{4}
}

func (x *EnqueueRequest) GetQueueName() string {
	if x != nil {
		return x.QueueName
	}
	return ""
}

func (x *EnqueueRequest) GetGroup() string {
	if x != nil {
		return x.Group
	}
	return ""
}

func (x *EnqueueRequest) GetPriority() int64 {
	if x != nil {
		return x.Priority
	}
	return 0
}

func (x *EnqueueRequest) GetContent() string {
	if x != nil {
		return x.Content
	}
	return ""
}

func (x *EnqueueRequest) GetMetadata() map[string]string {
	if x != nil {
		return x.Metadata
	}
	return nil
}

type EnqueueResponse struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Success       bool                   `protobuf:"varint,1,opt,name=success,proto3" json:"success,omitempty"`
	QueueName     string                 `protobuf:"bytes,2,opt,name=queueName,proto3" json:"queueName,omitempty"`
	Id            uint64                 `protobuf:"varint,3,opt,name=id,proto3" json:"id,omitempty"`
	Group         string                 `protobuf:"bytes,4,opt,name=group,proto3" json:"group,omitempty"`
	Priority      int64                  `protobuf:"varint,5,opt,name=priority,proto3" json:"priority,omitempty"`
	Content       string                 `protobuf:"bytes,6,opt,name=content,proto3" json:"content,omitempty"`
	Metadata      map[string]string      `protobuf:"bytes,7,rep,name=metadata,proto3" json:"metadata,omitempty" protobuf_key:"bytes,1,opt,name=key" protobuf_val:"bytes,2,opt,name=value"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *EnqueueResponse) Reset() {
	*x = EnqueueResponse{}
	mi := &file_pkg_proto_doq_proto_msgTypes[5]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *EnqueueResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*EnqueueResponse) ProtoMessage() {}

func (x *EnqueueResponse) ProtoReflect() protoreflect.Message {
	mi := &file_pkg_proto_doq_proto_msgTypes[5]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use EnqueueResponse.ProtoReflect.Descriptor instead.
func (*EnqueueResponse) Descriptor() ([]byte, []int) {
	return file_pkg_proto_doq_proto_rawDescGZIP(), []int{5}
}

func (x *EnqueueResponse) GetSuccess() bool {
	if x != nil {
		return x.Success
	}
	return false
}

func (x *EnqueueResponse) GetQueueName() string {
	if x != nil {
		return x.QueueName
	}
	return ""
}

func (x *EnqueueResponse) GetId() uint64 {
	if x != nil {
		return x.Id
	}
	return 0
}

func (x *EnqueueResponse) GetGroup() string {
	if x != nil {
		return x.Group
	}
	return ""
}

func (x *EnqueueResponse) GetPriority() int64 {
	if x != nil {
		return x.Priority
	}
	return 0
}

func (x *EnqueueResponse) GetContent() string {
	if x != nil {
		return x.Content
	}
	return ""
}

func (x *EnqueueResponse) GetMetadata() map[string]string {
	if x != nil {
		return x.Metadata
	}
	return nil
}

type DequeueRequest struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	QueueName     string                 `protobuf:"bytes,1,opt,name=queueName,proto3" json:"queueName,omitempty"`
	Ack           bool                   `protobuf:"varint,2,opt,name=ack,proto3" json:"ack,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *DequeueRequest) Reset() {
	*x = DequeueRequest{}
	mi := &file_pkg_proto_doq_proto_msgTypes[6]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *DequeueRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*DequeueRequest) ProtoMessage() {}

func (x *DequeueRequest) ProtoReflect() protoreflect.Message {
	mi := &file_pkg_proto_doq_proto_msgTypes[6]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use DequeueRequest.ProtoReflect.Descriptor instead.
func (*DequeueRequest) Descriptor() ([]byte, []int) {
	return file_pkg_proto_doq_proto_rawDescGZIP(), []int{6}
}

func (x *DequeueRequest) GetQueueName() string {
	if x != nil {
		return x.QueueName
	}
	return ""
}

func (x *DequeueRequest) GetAck() bool {
	if x != nil {
		return x.Ack
	}
	return false
}

type DequeueResponse struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Success       bool                   `protobuf:"varint,1,opt,name=success,proto3" json:"success,omitempty"`
	QueueName     string                 `protobuf:"bytes,2,opt,name=queueName,proto3" json:"queueName,omitempty"`
	Id            uint64                 `protobuf:"varint,3,opt,name=id,proto3" json:"id,omitempty"`
	Group         string                 `protobuf:"bytes,4,opt,name=group,proto3" json:"group,omitempty"`
	Priority      int64                  `protobuf:"varint,5,opt,name=priority,proto3" json:"priority,omitempty"`
	Content       string                 `protobuf:"bytes,6,opt,name=content,proto3" json:"content,omitempty"`
	Metadata      map[string]string      `protobuf:"bytes,7,rep,name=metadata,proto3" json:"metadata,omitempty" protobuf_key:"bytes,1,opt,name=key" protobuf_val:"bytes,2,opt,name=value"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *DequeueResponse) Reset() {
	*x = DequeueResponse{}
	mi := &file_pkg_proto_doq_proto_msgTypes[7]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *DequeueResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*DequeueResponse) ProtoMessage() {}

func (x *DequeueResponse) ProtoReflect() protoreflect.Message {
	mi := &file_pkg_proto_doq_proto_msgTypes[7]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use DequeueResponse.ProtoReflect.Descriptor instead.
func (*DequeueResponse) Descriptor() ([]byte, []int) {
	return file_pkg_proto_doq_proto_rawDescGZIP(), []int{7}
}

func (x *DequeueResponse) GetSuccess() bool {
	if x != nil {
		return x.Success
	}
	return false
}

func (x *DequeueResponse) GetQueueName() string {
	if x != nil {
		return x.QueueName
	}
	return ""
}

func (x *DequeueResponse) GetId() uint64 {
	if x != nil {
		return x.Id
	}
	return 0
}

func (x *DequeueResponse) GetGroup() string {
	if x != nil {
		return x.Group
	}
	return ""
}

func (x *DequeueResponse) GetPriority() int64 {
	if x != nil {
		return x.Priority
	}
	return 0
}

func (x *DequeueResponse) GetContent() string {
	if x != nil {
		return x.Content
	}
	return ""
}

func (x *DequeueResponse) GetMetadata() map[string]string {
	if x != nil {
		return x.Metadata
	}
	return nil
}

type GetRequest struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	QueueName     string                 `protobuf:"bytes,1,opt,name=queueName,proto3" json:"queueName,omitempty"`
	Id            uint64                 `protobuf:"varint,2,opt,name=id,proto3" json:"id,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *GetRequest) Reset() {
	*x = GetRequest{}
	mi := &file_pkg_proto_doq_proto_msgTypes[8]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *GetRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*GetRequest) ProtoMessage() {}

func (x *GetRequest) ProtoReflect() protoreflect.Message {
	mi := &file_pkg_proto_doq_proto_msgTypes[8]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use GetRequest.ProtoReflect.Descriptor instead.
func (*GetRequest) Descriptor() ([]byte, []int) {
	return file_pkg_proto_doq_proto_rawDescGZIP(), []int{8}
}

func (x *GetRequest) GetQueueName() string {
	if x != nil {
		return x.QueueName
	}
	return ""
}

func (x *GetRequest) GetId() uint64 {
	if x != nil {
		return x.Id
	}
	return 0
}

type GetResponse struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Success       bool                   `protobuf:"varint,1,opt,name=success,proto3" json:"success,omitempty"`
	QueueName     string                 `protobuf:"bytes,2,opt,name=queueName,proto3" json:"queueName,omitempty"`
	Id            uint64                 `protobuf:"varint,3,opt,name=id,proto3" json:"id,omitempty"`
	Group         string                 `protobuf:"bytes,4,opt,name=group,proto3" json:"group,omitempty"`
	Priority      int64                  `protobuf:"varint,5,opt,name=priority,proto3" json:"priority,omitempty"`
	Content       string                 `protobuf:"bytes,6,opt,name=content,proto3" json:"content,omitempty"`
	Metadata      map[string]string      `protobuf:"bytes,7,rep,name=metadata,proto3" json:"metadata,omitempty" protobuf_key:"bytes,1,opt,name=key" protobuf_val:"bytes,2,opt,name=value"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *GetResponse) Reset() {
	*x = GetResponse{}
	mi := &file_pkg_proto_doq_proto_msgTypes[9]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *GetResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*GetResponse) ProtoMessage() {}

func (x *GetResponse) ProtoReflect() protoreflect.Message {
	mi := &file_pkg_proto_doq_proto_msgTypes[9]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use GetResponse.ProtoReflect.Descriptor instead.
func (*GetResponse) Descriptor() ([]byte, []int) {
	return file_pkg_proto_doq_proto_rawDescGZIP(), []int{9}
}

func (x *GetResponse) GetSuccess() bool {
	if x != nil {
		return x.Success
	}
	return false
}

func (x *GetResponse) GetQueueName() string {
	if x != nil {
		return x.QueueName
	}
	return ""
}

func (x *GetResponse) GetId() uint64 {
	if x != nil {
		return x.Id
	}
	return 0
}

func (x *GetResponse) GetGroup() string {
	if x != nil {
		return x.Group
	}
	return ""
}

func (x *GetResponse) GetPriority() int64 {
	if x != nil {
		return x.Priority
	}
	return 0
}

func (x *GetResponse) GetContent() string {
	if x != nil {
		return x.Content
	}
	return ""
}

func (x *GetResponse) GetMetadata() map[string]string {
	if x != nil {
		return x.Metadata
	}
	return nil
}

type DeleteRequest struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	QueueName     string                 `protobuf:"bytes,1,opt,name=queueName,proto3" json:"queueName,omitempty"`
	Id            uint64                 `protobuf:"varint,2,opt,name=id,proto3" json:"id,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *DeleteRequest) Reset() {
	*x = DeleteRequest{}
	mi := &file_pkg_proto_doq_proto_msgTypes[10]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *DeleteRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*DeleteRequest) ProtoMessage() {}

func (x *DeleteRequest) ProtoReflect() protoreflect.Message {
	mi := &file_pkg_proto_doq_proto_msgTypes[10]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use DeleteRequest.ProtoReflect.Descriptor instead.
func (*DeleteRequest) Descriptor() ([]byte, []int) {
	return file_pkg_proto_doq_proto_rawDescGZIP(), []int{10}
}

func (x *DeleteRequest) GetQueueName() string {
	if x != nil {
		return x.QueueName
	}
	return ""
}

func (x *DeleteRequest) GetId() uint64 {
	if x != nil {
		return x.Id
	}
	return 0
}

type DeleteResponse struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Success       bool                   `protobuf:"varint,1,opt,name=success,proto3" json:"success,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *DeleteResponse) Reset() {
	*x = DeleteResponse{}
	mi := &file_pkg_proto_doq_proto_msgTypes[11]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *DeleteResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*DeleteResponse) ProtoMessage() {}

func (x *DeleteResponse) ProtoReflect() protoreflect.Message {
	mi := &file_pkg_proto_doq_proto_msgTypes[11]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use DeleteResponse.ProtoReflect.Descriptor instead.
func (*DeleteResponse) Descriptor() ([]byte, []int) {
	return file_pkg_proto_doq_proto_rawDescGZIP(), []int{11}
}

func (x *DeleteResponse) GetSuccess() bool {
	if x != nil {
		return x.Success
	}
	return false
}

type AckRequest struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	QueueName     string                 `protobuf:"bytes,1,opt,name=queueName,proto3" json:"queueName,omitempty"`
	Id            uint64                 `protobuf:"varint,2,opt,name=id,proto3" json:"id,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *AckRequest) Reset() {
	*x = AckRequest{}
	mi := &file_pkg_proto_doq_proto_msgTypes[12]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *AckRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*AckRequest) ProtoMessage() {}

func (x *AckRequest) ProtoReflect() protoreflect.Message {
	mi := &file_pkg_proto_doq_proto_msgTypes[12]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use AckRequest.ProtoReflect.Descriptor instead.
func (*AckRequest) Descriptor() ([]byte, []int) {
	return file_pkg_proto_doq_proto_rawDescGZIP(), []int{12}
}

func (x *AckRequest) GetQueueName() string {
	if x != nil {
		return x.QueueName
	}
	return ""
}

func (x *AckRequest) GetId() uint64 {
	if x != nil {
		return x.Id
	}
	return 0
}

type AckResponse struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Success       bool                   `protobuf:"varint,1,opt,name=success,proto3" json:"success,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *AckResponse) Reset() {
	*x = AckResponse{}
	mi := &file_pkg_proto_doq_proto_msgTypes[13]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *AckResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*AckResponse) ProtoMessage() {}

func (x *AckResponse) ProtoReflect() protoreflect.Message {
	mi := &file_pkg_proto_doq_proto_msgTypes[13]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use AckResponse.ProtoReflect.Descriptor instead.
func (*AckResponse) Descriptor() ([]byte, []int) {
	return file_pkg_proto_doq_proto_rawDescGZIP(), []int{13}
}

func (x *AckResponse) GetSuccess() bool {
	if x != nil {
		return x.Success
	}
	return false
}

type NackRequest struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	QueueName     string                 `protobuf:"bytes,1,opt,name=queueName,proto3" json:"queueName,omitempty"`
	Id            uint64                 `protobuf:"varint,2,opt,name=id,proto3" json:"id,omitempty"`
	Priority      int64                  `protobuf:"varint,3,opt,name=priority,proto3" json:"priority,omitempty"`
	Metadata      map[string]string      `protobuf:"bytes,4,rep,name=metadata,proto3" json:"metadata,omitempty" protobuf_key:"bytes,1,opt,name=key" protobuf_val:"bytes,2,opt,name=value"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *NackRequest) Reset() {
	*x = NackRequest{}
	mi := &file_pkg_proto_doq_proto_msgTypes[14]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *NackRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*NackRequest) ProtoMessage() {}

func (x *NackRequest) ProtoReflect() protoreflect.Message {
	mi := &file_pkg_proto_doq_proto_msgTypes[14]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use NackRequest.ProtoReflect.Descriptor instead.
func (*NackRequest) Descriptor() ([]byte, []int) {
	return file_pkg_proto_doq_proto_rawDescGZIP(), []int{14}
}

func (x *NackRequest) GetQueueName() string {
	if x != nil {
		return x.QueueName
	}
	return ""
}

func (x *NackRequest) GetId() uint64 {
	if x != nil {
		return x.Id
	}
	return 0
}

func (x *NackRequest) GetPriority() int64 {
	if x != nil {
		return x.Priority
	}
	return 0
}

func (x *NackRequest) GetMetadata() map[string]string {
	if x != nil {
		return x.Metadata
	}
	return nil
}

type NackResponse struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Success       bool                   `protobuf:"varint,1,opt,name=success,proto3" json:"success,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *NackResponse) Reset() {
	*x = NackResponse{}
	mi := &file_pkg_proto_doq_proto_msgTypes[15]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *NackResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*NackResponse) ProtoMessage() {}

func (x *NackResponse) ProtoReflect() protoreflect.Message {
	mi := &file_pkg_proto_doq_proto_msgTypes[15]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use NackResponse.ProtoReflect.Descriptor instead.
func (*NackResponse) Descriptor() ([]byte, []int) {
	return file_pkg_proto_doq_proto_rawDescGZIP(), []int{15}
}

func (x *NackResponse) GetSuccess() bool {
	if x != nil {
		return x.Success
	}
	return false
}

type UpdatePriorityRequest struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	QueueName     string                 `protobuf:"bytes,1,opt,name=queueName,proto3" json:"queueName,omitempty"`
	Id            uint64                 `protobuf:"varint,2,opt,name=id,proto3" json:"id,omitempty"`
	Priority      int64                  `protobuf:"varint,3,opt,name=priority,proto3" json:"priority,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *UpdatePriorityRequest) Reset() {
	*x = UpdatePriorityRequest{}
	mi := &file_pkg_proto_doq_proto_msgTypes[16]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *UpdatePriorityRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*UpdatePriorityRequest) ProtoMessage() {}

func (x *UpdatePriorityRequest) ProtoReflect() protoreflect.Message {
	mi := &file_pkg_proto_doq_proto_msgTypes[16]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use UpdatePriorityRequest.ProtoReflect.Descriptor instead.
func (*UpdatePriorityRequest) Descriptor() ([]byte, []int) {
	return file_pkg_proto_doq_proto_rawDescGZIP(), []int{16}
}

func (x *UpdatePriorityRequest) GetQueueName() string {
	if x != nil {
		return x.QueueName
	}
	return ""
}

func (x *UpdatePriorityRequest) GetId() uint64 {
	if x != nil {
		return x.Id
	}
	return 0
}

func (x *UpdatePriorityRequest) GetPriority() int64 {
	if x != nil {
		return x.Priority
	}
	return 0
}

type UpdatePriorityResponse struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Success       bool                   `protobuf:"varint,1,opt,name=success,proto3" json:"success,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *UpdatePriorityResponse) Reset() {
	*x = UpdatePriorityResponse{}
	mi := &file_pkg_proto_doq_proto_msgTypes[17]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *UpdatePriorityResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*UpdatePriorityResponse) ProtoMessage() {}

func (x *UpdatePriorityResponse) ProtoReflect() protoreflect.Message {
	mi := &file_pkg_proto_doq_proto_msgTypes[17]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use UpdatePriorityResponse.ProtoReflect.Descriptor instead.
func (*UpdatePriorityResponse) Descriptor() ([]byte, []int) {
	return file_pkg_proto_doq_proto_rawDescGZIP(), []int{17}
}

func (x *UpdatePriorityResponse) GetSuccess() bool {
	if x != nil {
		return x.Success
	}
	return false
}

var File_pkg_proto_doq_proto protoreflect.FileDescriptor

const file_pkg_proto_doq_proto_rawDesc = "" +
	"\n" +
	"\x13pkg/proto/doq.proto\x12\x05queue\"<\n" +
	"\x12CreateQueueRequest\x12\x12\n" +
	"\x04name\x18\x01 \x01(\tR\x04name\x12\x12\n" +
	"\x04type\x18\x02 \x01(\tR\x04type\"/\n" +
	"\x13CreateQueueResponse\x12\x18\n" +
	"\asuccess\x18\x01 \x01(\bR\asuccess\"(\n" +
	"\x12DeleteQueueRequest\x12\x12\n" +
	"\x04name\x18\x01 \x01(\tR\x04name\"/\n" +
	"\x13DeleteQueueResponse\x12\x18\n" +
	"\asuccess\x18\x01 \x01(\bR\asuccess\"\xf8\x01\n" +
	"\x0eEnqueueRequest\x12\x1c\n" +
	"\tqueueName\x18\x01 \x01(\tR\tqueueName\x12\x14\n" +
	"\x05group\x18\x02 \x01(\tR\x05group\x12\x1a\n" +
	"\bpriority\x18\x03 \x01(\x03R\bpriority\x12\x18\n" +
	"\acontent\x18\x04 \x01(\tR\acontent\x12?\n" +
	"\bmetadata\x18\x05 \x03(\v2#.queue.EnqueueRequest.MetadataEntryR\bmetadata\x1a;\n" +
	"\rMetadataEntry\x12\x10\n" +
	"\x03key\x18\x01 \x01(\tR\x03key\x12\x14\n" +
	"\x05value\x18\x02 \x01(\tR\x05value:\x028\x01\"\xa4\x02\n" +
	"\x0fEnqueueResponse\x12\x18\n" +
	"\asuccess\x18\x01 \x01(\bR\asuccess\x12\x1c\n" +
	"\tqueueName\x18\x02 \x01(\tR\tqueueName\x12\x0e\n" +
	"\x02id\x18\x03 \x01(\x04R\x02id\x12\x14\n" +
	"\x05group\x18\x04 \x01(\tR\x05group\x12\x1a\n" +
	"\bpriority\x18\x05 \x01(\x03R\bpriority\x12\x18\n" +
	"\acontent\x18\x06 \x01(\tR\acontent\x12@\n" +
	"\bmetadata\x18\a \x03(\v2$.queue.EnqueueResponse.MetadataEntryR\bmetadata\x1a;\n" +
	"\rMetadataEntry\x12\x10\n" +
	"\x03key\x18\x01 \x01(\tR\x03key\x12\x14\n" +
	"\x05value\x18\x02 \x01(\tR\x05value:\x028\x01\"@\n" +
	"\x0eDequeueRequest\x12\x1c\n" +
	"\tqueueName\x18\x01 \x01(\tR\tqueueName\x12\x10\n" +
	"\x03ack\x18\x02 \x01(\bR\x03ack\"\xa4\x02\n" +
	"\x0fDequeueResponse\x12\x18\n" +
	"\asuccess\x18\x01 \x01(\bR\asuccess\x12\x1c\n" +
	"\tqueueName\x18\x02 \x01(\tR\tqueueName\x12\x0e\n" +
	"\x02id\x18\x03 \x01(\x04R\x02id\x12\x14\n" +
	"\x05group\x18\x04 \x01(\tR\x05group\x12\x1a\n" +
	"\bpriority\x18\x05 \x01(\x03R\bpriority\x12\x18\n" +
	"\acontent\x18\x06 \x01(\tR\acontent\x12@\n" +
	"\bmetadata\x18\a \x03(\v2$.queue.DequeueResponse.MetadataEntryR\bmetadata\x1a;\n" +
	"\rMetadataEntry\x12\x10\n" +
	"\x03key\x18\x01 \x01(\tR\x03key\x12\x14\n" +
	"\x05value\x18\x02 \x01(\tR\x05value:\x028\x01\":\n" +
	"\n" +
	"GetRequest\x12\x1c\n" +
	"\tqueueName\x18\x01 \x01(\tR\tqueueName\x12\x0e\n" +
	"\x02id\x18\x02 \x01(\x04R\x02id\"\x9c\x02\n" +
	"\vGetResponse\x12\x18\n" +
	"\asuccess\x18\x01 \x01(\bR\asuccess\x12\x1c\n" +
	"\tqueueName\x18\x02 \x01(\tR\tqueueName\x12\x0e\n" +
	"\x02id\x18\x03 \x01(\x04R\x02id\x12\x14\n" +
	"\x05group\x18\x04 \x01(\tR\x05group\x12\x1a\n" +
	"\bpriority\x18\x05 \x01(\x03R\bpriority\x12\x18\n" +
	"\acontent\x18\x06 \x01(\tR\acontent\x12<\n" +
	"\bmetadata\x18\a \x03(\v2 .queue.GetResponse.MetadataEntryR\bmetadata\x1a;\n" +
	"\rMetadataEntry\x12\x10\n" +
	"\x03key\x18\x01 \x01(\tR\x03key\x12\x14\n" +
	"\x05value\x18\x02 \x01(\tR\x05value:\x028\x01\"=\n" +
	"\rDeleteRequest\x12\x1c\n" +
	"\tqueueName\x18\x01 \x01(\tR\tqueueName\x12\x0e\n" +
	"\x02id\x18\x02 \x01(\x04R\x02id\"*\n" +
	"\x0eDeleteResponse\x12\x18\n" +
	"\asuccess\x18\x01 \x01(\bR\asuccess\":\n" +
	"\n" +
	"AckRequest\x12\x1c\n" +
	"\tqueueName\x18\x01 \x01(\tR\tqueueName\x12\x0e\n" +
	"\x02id\x18\x02 \x01(\x04R\x02id\"'\n" +
	"\vAckResponse\x12\x18\n" +
	"\asuccess\x18\x01 \x01(\bR\asuccess\"\xd2\x01\n" +
	"\vNackRequest\x12\x1c\n" +
	"\tqueueName\x18\x01 \x01(\tR\tqueueName\x12\x0e\n" +
	"\x02id\x18\x02 \x01(\x04R\x02id\x12\x1a\n" +
	"\bpriority\x18\x03 \x01(\x03R\bpriority\x12<\n" +
	"\bmetadata\x18\x04 \x03(\v2 .queue.NackRequest.MetadataEntryR\bmetadata\x1a;\n" +
	"\rMetadataEntry\x12\x10\n" +
	"\x03key\x18\x01 \x01(\tR\x03key\x12\x14\n" +
	"\x05value\x18\x02 \x01(\tR\x05value:\x028\x01\"(\n" +
	"\fNackResponse\x12\x18\n" +
	"\asuccess\x18\x01 \x01(\bR\asuccess\"a\n" +
	"\x15UpdatePriorityRequest\x12\x1c\n" +
	"\tqueueName\x18\x01 \x01(\tR\tqueueName\x12\x0e\n" +
	"\x02id\x18\x02 \x01(\x04R\x02id\x12\x1a\n" +
	"\bpriority\x18\x03 \x01(\x03R\bpriority\"2\n" +
	"\x16UpdatePriorityResponse\x12\x18\n" +
	"\asuccess\x18\x01 \x01(\bR\asuccess2\xac\x05\n" +
	"\x03DOQ\x12D\n" +
	"\vCreateQueue\x12\x19.queue.CreateQueueRequest\x1a\x1a.queue.CreateQueueResponse\x12D\n" +
	"\vDeleteQueue\x12\x19.queue.DeleteQueueRequest\x1a\x1a.queue.DeleteQueueResponse\x12:\n" +
	"\aEnqueue\x12\x15.queue.EnqueueRequest\x1a\x16.queue.EnqueueResponse\"\x00\x12D\n" +
	"\rEnqueueStream\x12\x15.queue.EnqueueRequest\x1a\x16.queue.EnqueueResponse\"\x00(\x010\x01\x12:\n" +
	"\aDequeue\x12\x15.queue.DequeueRequest\x1a\x16.queue.DequeueResponse\"\x00\x12D\n" +
	"\rDequeueStream\x12\x15.queue.DequeueRequest\x1a\x16.queue.DequeueResponse\"\x00(\x010\x01\x12.\n" +
	"\x03Get\x12\x11.queue.GetRequest\x1a\x12.queue.GetResponse\"\x00\x127\n" +
	"\x06Delete\x12\x14.queue.DeleteRequest\x1a\x15.queue.DeleteResponse\"\x00\x12,\n" +
	"\x03Ack\x12\x11.queue.AckRequest\x1a\x12.queue.AckResponse\x12/\n" +
	"\x04Nack\x12\x12.queue.NackRequest\x1a\x13.queue.NackResponse\x12M\n" +
	"\x0eUpdatePriority\x12\x1c.queue.UpdatePriorityRequest\x1a\x1d.queue.UpdatePriorityResponseB)Z'github.com/kgantsov/doq/pkg/proto;protob\x06proto3"

var (
	file_pkg_proto_doq_proto_rawDescOnce sync.Once
	file_pkg_proto_doq_proto_rawDescData []byte
)

func file_pkg_proto_doq_proto_rawDescGZIP() []byte {
	file_pkg_proto_doq_proto_rawDescOnce.Do(func() {
		file_pkg_proto_doq_proto_rawDescData = protoimpl.X.CompressGZIP(unsafe.Slice(unsafe.StringData(file_pkg_proto_doq_proto_rawDesc), len(file_pkg_proto_doq_proto_rawDesc)))
	})
	return file_pkg_proto_doq_proto_rawDescData
}

var file_pkg_proto_doq_proto_msgTypes = make([]protoimpl.MessageInfo, 23)
var file_pkg_proto_doq_proto_goTypes = []any{
	(*CreateQueueRequest)(nil),     // 0: queue.CreateQueueRequest
	(*CreateQueueResponse)(nil),    // 1: queue.CreateQueueResponse
	(*DeleteQueueRequest)(nil),     // 2: queue.DeleteQueueRequest
	(*DeleteQueueResponse)(nil),    // 3: queue.DeleteQueueResponse
	(*EnqueueRequest)(nil),         // 4: queue.EnqueueRequest
	(*EnqueueResponse)(nil),        // 5: queue.EnqueueResponse
	(*DequeueRequest)(nil),         // 6: queue.DequeueRequest
	(*DequeueResponse)(nil),        // 7: queue.DequeueResponse
	(*GetRequest)(nil),             // 8: queue.GetRequest
	(*GetResponse)(nil),            // 9: queue.GetResponse
	(*DeleteRequest)(nil),          // 10: queue.DeleteRequest
	(*DeleteResponse)(nil),         // 11: queue.DeleteResponse
	(*AckRequest)(nil),             // 12: queue.AckRequest
	(*AckResponse)(nil),            // 13: queue.AckResponse
	(*NackRequest)(nil),            // 14: queue.NackRequest
	(*NackResponse)(nil),           // 15: queue.NackResponse
	(*UpdatePriorityRequest)(nil),  // 16: queue.UpdatePriorityRequest
	(*UpdatePriorityResponse)(nil), // 17: queue.UpdatePriorityResponse
	nil,                            // 18: queue.EnqueueRequest.MetadataEntry
	nil,                            // 19: queue.EnqueueResponse.MetadataEntry
	nil,                            // 20: queue.DequeueResponse.MetadataEntry
	nil,                            // 21: queue.GetResponse.MetadataEntry
	nil,                            // 22: queue.NackRequest.MetadataEntry
}
var file_pkg_proto_doq_proto_depIdxs = []int32{
	18, // 0: queue.EnqueueRequest.metadata:type_name -> queue.EnqueueRequest.MetadataEntry
	19, // 1: queue.EnqueueResponse.metadata:type_name -> queue.EnqueueResponse.MetadataEntry
	20, // 2: queue.DequeueResponse.metadata:type_name -> queue.DequeueResponse.MetadataEntry
	21, // 3: queue.GetResponse.metadata:type_name -> queue.GetResponse.MetadataEntry
	22, // 4: queue.NackRequest.metadata:type_name -> queue.NackRequest.MetadataEntry
	0,  // 5: queue.DOQ.CreateQueue:input_type -> queue.CreateQueueRequest
	2,  // 6: queue.DOQ.DeleteQueue:input_type -> queue.DeleteQueueRequest
	4,  // 7: queue.DOQ.Enqueue:input_type -> queue.EnqueueRequest
	4,  // 8: queue.DOQ.EnqueueStream:input_type -> queue.EnqueueRequest
	6,  // 9: queue.DOQ.Dequeue:input_type -> queue.DequeueRequest
	6,  // 10: queue.DOQ.DequeueStream:input_type -> queue.DequeueRequest
	8,  // 11: queue.DOQ.Get:input_type -> queue.GetRequest
	10, // 12: queue.DOQ.Delete:input_type -> queue.DeleteRequest
	12, // 13: queue.DOQ.Ack:input_type -> queue.AckRequest
	14, // 14: queue.DOQ.Nack:input_type -> queue.NackRequest
	16, // 15: queue.DOQ.UpdatePriority:input_type -> queue.UpdatePriorityRequest
	1,  // 16: queue.DOQ.CreateQueue:output_type -> queue.CreateQueueResponse
	3,  // 17: queue.DOQ.DeleteQueue:output_type -> queue.DeleteQueueResponse
	5,  // 18: queue.DOQ.Enqueue:output_type -> queue.EnqueueResponse
	5,  // 19: queue.DOQ.EnqueueStream:output_type -> queue.EnqueueResponse
	7,  // 20: queue.DOQ.Dequeue:output_type -> queue.DequeueResponse
	7,  // 21: queue.DOQ.DequeueStream:output_type -> queue.DequeueResponse
	9,  // 22: queue.DOQ.Get:output_type -> queue.GetResponse
	11, // 23: queue.DOQ.Delete:output_type -> queue.DeleteResponse
	13, // 24: queue.DOQ.Ack:output_type -> queue.AckResponse
	15, // 25: queue.DOQ.Nack:output_type -> queue.NackResponse
	17, // 26: queue.DOQ.UpdatePriority:output_type -> queue.UpdatePriorityResponse
	16, // [16:27] is the sub-list for method output_type
	5,  // [5:16] is the sub-list for method input_type
	5,  // [5:5] is the sub-list for extension type_name
	5,  // [5:5] is the sub-list for extension extendee
	0,  // [0:5] is the sub-list for field type_name
}

func init() { file_pkg_proto_doq_proto_init() }
func file_pkg_proto_doq_proto_init() {
	if File_pkg_proto_doq_proto != nil {
		return
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: unsafe.Slice(unsafe.StringData(file_pkg_proto_doq_proto_rawDesc), len(file_pkg_proto_doq_proto_rawDesc)),
			NumEnums:      0,
			NumMessages:   23,
			NumExtensions: 0,
			NumServices:   1,
		},
		GoTypes:           file_pkg_proto_doq_proto_goTypes,
		DependencyIndexes: file_pkg_proto_doq_proto_depIdxs,
		MessageInfos:      file_pkg_proto_doq_proto_msgTypes,
	}.Build()
	File_pkg_proto_doq_proto = out.File
	file_pkg_proto_doq_proto_goTypes = nil
	file_pkg_proto_doq_proto_depIdxs = nil
}
