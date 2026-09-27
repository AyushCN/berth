// Package onnx provides ONNX protobuf definitions for building binary ONNX models.
// Generated from ONNX protobuf specification.
package onnx

import (
	"github.com/gogo/protobuf/proto"
)

// ModelProto represents an ONNX model
type ModelProto struct {
	IrVersion      int64                    `protobuf:"varint,1,opt,name=ir_version,json=irVersion,proto3" json:"ir_version,omitempty"`
	OpsetImport    []*OperatorSetIdProto    `protobuf:"bytes,8,rep,name=opset_import,json=opsetImport,proto3" json:"opset_import,omitempty"`
	ProducerName   string                   `protobuf:"bytes,2,opt,name=producer_name,json=producerName,proto3" json:"producer_name,omitempty"`
	ProducerVersion string                  `protobuf:"bytes,3,opt,name=producer_version,json=producerVersion,proto3" json:"producer_version,omitempty"`
	Domain         string                   `protobuf:"bytes,4,opt,name=domain,proto3" json:"domain,omitempty"`
	ModelVersion   int64                    `protobuf:"varint,5,opt,name=model_version,json=modelVersion,proto3" json:"model_version,omitempty"`
	DocString      string                   `protobuf:"bytes,6,opt,name=doc_string,json=docString,proto3" json:"doc_string,omitempty"`
	Graph          *GraphProto              `protobuf:"bytes,7,opt,name=graph,proto3" json:"graph,omitempty"`
	MetadataProps  []*StringStringEntryProto `protobuf:"bytes,14,rep,name=metadata_props,json=metadataProps,proto3" json:"metadata_props,omitempty"`
	XXX_unrecognized []byte                 `json:"-"`
}

func (m *ModelProto) Reset()         { *m = ModelProto{} }
func (m *ModelProto) String() string { return proto.CompactTextString(m) }
func (*ModelProto) ProtoMessage()    {}
func (m *ModelProto) Marshal() ([]byte, error) { return proto.Marshal(m) }
func (m *ModelProto) Unmarshal(data []byte) error { return proto.Unmarshal(data, m) }

// GraphProto represents an ONNX graph
type GraphProto struct {
	Name        string                  `protobuf:"bytes,1,opt,name=name,proto3" json:"name,omitempty"`
	Domain      string                  `protobuf:"bytes,13,opt,name=domain,proto3" json:"domain,omitempty"`
	Node        []*NodeProto            `protobuf:"bytes,2,rep,name=node,proto3" json:"node,omitempty"`
	Input       []*ValueInfoProto       `protobuf:"bytes,3,rep,name=input,proto3" json:"input,omitempty"`
	Output      []*ValueInfoProto       `protobuf:"bytes,4,rep,name=output,proto3" json:"output,omitempty"`
	Initializer []*TensorProto          `protobuf:"bytes,5,rep,name=initializer,proto3" json:"initializer,omitempty"`
	ValueInfo   []*ValueInfoProto       `protobuf:"bytes,6,rep,name=value_info,json=valueInfo,proto3" json:"value_info,omitempty"`
	SparseInitializer []*TensorProto    `protobuf:"bytes,14,rep,name=sparse_initializer,json=sparseInitializer,proto3" json:"sparse_initializer,omitempty"`
	XXX_unrecognized []byte             `json:"-"`
}

func (m *GraphProto) Reset()         { *m = GraphProto{} }
func (m *GraphProto) String() string { return proto.CompactTextString(m) }
func (*GraphProto) ProtoMessage()    {}
func (m *GraphProto) Marshal() ([]byte, error) { return proto.Marshal(m) }
func (m *GraphProto) Unmarshal(data []byte) error { return proto.Unmarshal(data, m) }

// NodeProto represents an ONNX node
type NodeProto struct {
	Name       string   `protobuf:"bytes,1,opt,name=name,proto3" json:"name,omitempty"`
	OpType     string   `protobuf:"bytes,2,opt,name=op_type,json=opType,proto3" json:"op_type,omitempty"`
	Input      []string `protobuf:"bytes,3,rep,name=input,proto3" json:"input,omitempty"`
	Output     []string `protobuf:"bytes,4,rep,name=output,proto3" json:"output,omitempty"`
	Attribute  []*AttributeProto `protobuf:"bytes,5,rep,name=attribute,proto3" json:"attribute,omitempty"`
	Domain     string   `protobuf:"bytes,6,opt,name=domain,proto3" json:"domain,omitempty"`
	XXX_unrecognized []byte `json:"-"`
}

func (m *NodeProto) Reset()         { *m = NodeProto{} }
func (m *NodeProto) String() string { return proto.CompactTextString(m) }
func (*NodeProto) ProtoMessage()    {}
func (m *NodeProto) Marshal() ([]byte, error) { return proto.Marshal(m) }
func (m *NodeProto) Unmarshal(data []byte) error { return proto.Unmarshal(data, m) }

// ValueInfoProto represents tensor value info
type ValueInfoProto struct {
	Name       string         `protobuf:"bytes,1,opt,name=name,proto3" json:"name,omitempty"`
	Type       *TypeProto     `protobuf:"bytes,2,opt,name=type,proto3" json:"type,omitempty"`
	DocString  string         `protobuf:"bytes,3,opt,name=doc_string,json=docString,proto3" json:"doc_string,omitempty"`
	XXX_unrecognized []byte   `json:"-"`
}

func (m *ValueInfoProto) Reset()         { *m = ValueInfoProto{} }
func (m *ValueInfoProto) String() string { return proto.CompactTextString(m) }
func (*ValueInfoProto) ProtoMessage()    {}
func (m *ValueInfoProto) Marshal() ([]byte, error) { return proto.Marshal(m) }
func (m *ValueInfoProto) Unmarshal(data []byte) error { return proto.Unmarshal(data, m) }

// TypeProto represents a type proto
type TypeProto struct {
	Value *TypeProto_TensorType `protobuf:"bytes,1,opt,name=tensor_type,json=tensorType,proto3" json:"tensor_type,omitempty"`
	XXX_unrecognized []byte     `json:"-"`
}

func (m *TypeProto) Reset()         { *m = TypeProto{} }
func (m *TypeProto) String() string { return proto.CompactTextString(m) }
func (*TypeProto) ProtoMessage()    {}
func (m *TypeProto) Marshal() ([]byte, error) { return proto.Marshal(m) }
func (m *TypeProto) Unmarshal(data []byte) error { return proto.Unmarshal(data, m) }

// TypeProto_TensorType represents tensor type
type TypeProto_TensorType struct {
	ElemType int32            `protobuf:"varint,1,opt,name=elem_type,json=elemType,proto3" json:"elem_type,omitempty"`
	Shape    *TensorShapeProto `protobuf:"bytes,2,opt,name=shape,proto3" json:"shape,omitempty"`
	XXX_unrecognized []byte   `json:"-"`
}

func (m *TypeProto_TensorType) Reset()         { *m = TypeProto_TensorType{} }
func (m *TypeProto_TensorType) String() string { return proto.CompactTextString(m) }
func (*TypeProto_TensorType) ProtoMessage()    {}
func (m *TypeProto_TensorType) Marshal() ([]byte, error) { return proto.Marshal(m) }
func (m *TypeProto_TensorType) Unmarshal(data []byte) error { return proto.Unmarshal(data, m) }

// TensorShapeProto represents tensor shape
type TensorShapeProto struct {
	Dim []*TensorShapeProto_Dimension `protobuf:"bytes,1,rep,name=dim,proto3" json:"dim,omitempty"`
	XXX_unrecognized []byte `json:"-"`
}

func (m *TensorShapeProto) Reset()         { *m = TensorShapeProto{} }
func (m *TensorShapeProto) String() string { return proto.CompactTextString(m) }
func (*TensorShapeProto) ProtoMessage()    {}
func (m *TensorShapeProto) Marshal() ([]byte, error) { return proto.Marshal(m) }
func (m *TensorShapeProto) Unmarshal(data []byte) error { return proto.Unmarshal(data, m) }

// TensorShapeProto_Dimension represents tensor shape dimension
type TensorShapeProto_Dimension struct {
	DimValue int64 `protobuf:"varint,1,opt,name=dim_value,json=dimValue,proto3" json:"dim_value,omitempty"`
	DimParam string `protobuf:"bytes,2,opt,name=dim_param,json=dimParam,proto3" json:"dim_param,omitempty"`
	XXX_unrecognized []byte `json:"-"`
}

func (m *TensorShapeProto_Dimension) Reset()         { *m = TensorShapeProto_Dimension{} }
func (m *TensorShapeProto_Dimension) String() string { return proto.CompactTextString(m) }
func (*TensorShapeProto_Dimension) ProtoMessage()    {}
func (m *TensorShapeProto_Dimension) Marshal() ([]byte, error) { return proto.Marshal(m) }
func (m *TensorShapeProto_Dimension) Unmarshal(data []byte) error { return proto.Unmarshal(data, m) }

// TensorProto represents a tensor
type TensorProto struct {
	DataType   int32    `protobuf:"varint,1,opt,name=data_type,json=dataType,proto3" json:"data_type,omitempty"`
	Dims       []int64  `protobuf:"varint,2,rep,name=dims,proto3" json:"dims,omitempty"`
	FloatData  []float32 `protobuf:"fixed32,10,rep,name=float_data,json=floatData,proto3" json:"float_data,omitempty"`
	Int64Data  []int64  `protobuf:"varint,11,rep,name=int64_data,json=int64Data,proto3" json:"int64_data,omitempty"`
	Name       string   `protobuf:"bytes,8,opt,name=name,proto3" json:"name,omitempty"`
	XXX_unrecognized []byte `json:"-"`
}

func (m *TensorProto) Reset()         { *m = TensorProto{} }
func (m *TensorProto) String() string { return proto.CompactTextString(m) }
func (*TensorProto) ProtoMessage()    {}
func (m *TensorProto) Marshal() ([]byte, error) { return proto.Marshal(m) }
func (m *TensorProto) Unmarshal(data []byte) error { return proto.Unmarshal(data, m) }

// OperatorSetIdProto represents an operator set import
type OperatorSetIdProto struct {
	Domain  string `protobuf:"bytes,1,opt,name=domain,proto3" json:"domain,omitempty"`
	Version int64  `protobuf:"varint,2,opt,name=version,proto3" json:"version,omitempty"`
	XXX_unrecognized []byte `json:"-"`
}

func (m *OperatorSetIdProto) Reset()         { *m = OperatorSetIdProto{} }
func (m *OperatorSetIdProto) String() string { return proto.CompactTextString(m) }
func (*OperatorSetIdProto) ProtoMessage()    {}
func (m *OperatorSetIdProto) Marshal() ([]byte, error) { return proto.Marshal(m) }
func (m *OperatorSetIdProto) Unmarshal(data []byte) error { return proto.Unmarshal(data, m) }

// AttributeProto represents an attribute
type AttributeProto struct {
	Name      string  `protobuf:"bytes,1,opt,name=name,proto3" json:"name,omitempty"`
	RefAttrName string `protobuf:"bytes,2,opt,name=ref_attr_name,json=refAttrName,proto3" json:"ref_attr_name,omitempty"`
	Type      int32   `protobuf:"varint,3,opt,name=type,proto3" json:"type,omitempty"`
	F         float32 `protobuf:"fixed32,4,opt,name=f,proto3" json:"f,omitempty"`
	I         int64   `protobuf:"varint,5,opt,name=i,proto3" json:"i,omitempty"`
	S         []byte  `protobuf:"bytes,6,opt,name=s,proto3" json:"s,omitempty"`
	T         *TensorProto `protobuf:"bytes,7,opt,name=t,proto3" json:"t,omitempty"`
	Floats    []float32 `protobuf:"fixed32,8,rep,name=floats,proto3" json:"floats,omitempty"`
	Ints      []int64  `protobuf:"varint,9,rep,name=ints,proto3" json:"ints,omitempty"`
	Strings   []string `protobuf:"bytes,10,rep,name=strings,proto3" json:"strings,omitempty"`
	Tensors   []*TensorProto `protobuf:"bytes,11,rep,name=tensors,proto3" json:"tensors,omitempty"`
	Graphs    []*GraphProto `protobuf:"bytes,12,rep,name=graphs,proto3" json:"graphs,omitempty"`
	XXX_unrecognized []byte `json:"-"`
}

func (m *AttributeProto) Reset()         { *m = AttributeProto{} }
func (m *AttributeProto) String() string { return proto.CompactTextString(m) }
func (*AttributeProto) ProtoMessage()    {}
func (m *AttributeProto) Marshal() ([]byte, error) { return proto.Marshal(m) }
func (m *AttributeProto) Unmarshal(data []byte) error { return proto.Unmarshal(data, m) }

// StringStringEntryProto for metadata
type StringStringEntryProto struct {
	Key   string `protobuf:"bytes,1,opt,name=key,proto3" json:"key,omitempty"`
	Value string `protobuf:"bytes,2,opt,name=value,proto3" json:"value,omitempty"`
	XXX_unrecognized []byte `json:"-"`
}

func (m *StringStringEntryProto) Reset()         { *m = StringStringEntryProto{} }
func (m *StringStringEntryProto) String() string { return proto.CompactTextString(m) }
func (*StringStringEntryProto) ProtoMessage()    {}
func (m *StringStringEntryProto) Marshal() ([]byte, error) { return proto.Marshal(m) }
func (m *StringStringEntryProto) Unmarshal(data []byte) error { return proto.Unmarshal(data, m) }

// TensorProto data types
const (
	TensorProto_UNDEFINED = 0
	TensorProto_FLOAT     = 1
	TensorProto_UINT8     = 2
	TensorProto_INT8      = 3
	TensorProto_UINT16    = 4
	TensorProto_INT16     = 5
	TensorProto_INT32     = 6
	TensorProto_INT64     = 7
	TensorProto_STRING    = 7
	TensorProto_BOOL      = 9
	TensorProto_FLOAT16   = 10
	TensorProto_DOUBLE    = 11
	TensorProto_UINT32    = 12
	TensorProto_UINT64    = 13
	TensorProto_COMPLEX64 = 14
	TensorProto_COMPLEX128 = 15
)

