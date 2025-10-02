package milvus

import "time"

// DataType represents the data type of a field
type DataType int32

const (
	DataTypeNone           DataType = 0
	DataTypeBool           DataType = 1
	DataTypeInt8           DataType = 2
	DataTypeInt16          DataType = 3
	DataTypeInt32          DataType = 4
	DataTypeInt64          DataType = 5
	DataTypeFloat          DataType = 6
	DataTypeDouble         DataType = 7
	DataTypeString         DataType = 20
	DataTypeVarChar        DataType = 21
	DataTypeArray          DataType = 22
	DataTypeJSON           DataType = 23
	DataTypeFloatVector       DataType = 100
	DataTypeBinaryVector      DataType = 101
	DataTypeFloat16Vector     DataType = 102
	DataTypeBFloat16Vector    DataType = 103
	DataTypeSparseFloatVector DataType = 104
)

// IsVector returns true if this is a vector type
func (dt DataType) IsVector() bool {
	return dt == DataTypeFloatVector ||
		dt == DataTypeBinaryVector ||
		dt == DataTypeFloat16Vector ||
		dt == DataTypeBFloat16Vector ||
		dt == DataTypeSparseFloatVector
}

// String returns the string representation of DataType
func (dt DataType) String() string {
	switch dt {
	case DataTypeBool:
		return "Bool"
	case DataTypeInt8:
		return "Int8"
	case DataTypeInt16:
		return "Int16"
	case DataTypeInt32:
		return "Int32"
	case DataTypeInt64:
		return "Int64"
	case DataTypeFloat:
		return "Float"
	case DataTypeDouble:
		return "Double"
	case DataTypeString:
		return "String"
	case DataTypeVarChar:
		return "VarChar"
	case DataTypeArray:
		return "Array"
	case DataTypeJSON:
		return "JSON"
	case DataTypeFloatVector:
		return "FloatVector"
	case DataTypeBinaryVector:
		return "BinaryVector"
	case DataTypeFloat16Vector:
		return "Float16Vector"
	case DataTypeBFloat16Vector:
		return "BFloat16Vector"
	case DataTypeSparseFloatVector:
		return "SparseFloatVector"
	default:
		return "Unknown"
	}
}

// IndexType represents the type of index
type IndexType string

const (
	IndexTypeFlat        IndexType = "FLAT"
	IndexTypeIVFFlat     IndexType = "IVF_FLAT"
	IndexTypeIVFSQ8      IndexType = "IVF_SQ8"
	IndexTypeIVFPQ       IndexType = "IVF_PQ"
	IndexTypeHNSW        IndexType = "HNSW"
	IndexTypeDiskANN     IndexType = "DISKANN"
	IndexTypeAUTOINDEX   IndexType = "AUTOINDEX"
	IndexTypeAutoIndex   IndexType = "AUTOINDEX" // Alias for backward compatibility
	IndexTypeSCANN       IndexType = "SCANN"
	IndexTypeBinFlat     IndexType = "BIN_FLAT"
	IndexTypeBinIVFFlat  IndexType = "BIN_IVF_FLAT"
	IndexTypeGPUIVFFlat  IndexType = "GPU_IVF_FLAT"
	IndexTypeGPUIVFPQ    IndexType = "GPU_IVF_PQ"
)

// String returns the string representation of IndexType
func (it IndexType) String() string {
	return string(it)
}

// MetricType represents the distance metric type
type MetricType string

const (
	MetricTypeL2       MetricType = "L2"
	MetricTypeIP       MetricType = "IP"
	MetricTypeCosine   MetricType = "COSINE"
	MetricTypeJaccard  MetricType = "JACCARD"
	MetricTypeHamming  MetricType = "HAMMING"
	MetricTypeTANIMOTO MetricType = "TANIMOTO"
	MetricTypeSUPERSTRUCTURE MetricType = "SUPERSTRUCTURE"
	MetricTypeSUBSTRUCTURE   MetricType = "SUBSTRUCTURE"
)

// String returns the string representation of MetricType
func (mt MetricType) String() string {
	return string(mt)
}

// ConsistencyLevel represents the consistency level
type ConsistencyLevel string

const (
	ConsistencyLevelStrong     ConsistencyLevel = "Strong"
	ConsistencyLevelBounded    ConsistencyLevel = "Bounded"
	ConsistencyLevelSession    ConsistencyLevel = "Session"
	ConsistencyLevelEventually ConsistencyLevel = "Eventually"
)

// IndexState represents the state of index building
type IndexState string

const (
	IndexStateNone       IndexState = "IndexStateNone"
	IndexStateUnissued   IndexState = "Unissued"
	IndexStateInProgress IndexState = "InProgress"
	IndexStateFinished   IndexState = "Finished"
	IndexStateFailed     IndexState = "Failed"
)

// LoadState represents the load state of a collection or partition
type LoadState string

const (
	LoadStateNotExist  LoadState = "LoadStateNotExist"
	LoadStateNotLoad   LoadState = "LoadStateNotLoad"
	LoadStateLoading   LoadState = "LoadStateLoading"
	LoadStateLoaded    LoadState = "LoadStateLoaded"
)

// Constants for default values
const (
	DefaultShardsNum         int32         = 2
	DefaultTimeout           time.Duration = 30 * time.Second
	DefaultConsistencyLevel                = ConsistencyLevelBounded
	DefaultRetries                         = 3
	DefaultRetryDelay                      = time.Second
	DefaultMaxVectorDim                    = 32768
	DefaultMinVectorDim                    = 1
	MaxCollectionNameLength                = 255
	MaxFieldNameLength                     = 255
)

// Vector represents a vector interface
type Vector interface {
	Dim() int
	Data() interface{}
	Type() DataType
}

// FloatVector represents a float32 vector
type FloatVector struct {
	dim  int
	data []float32
}

// NewFloatVector creates a new FloatVector
func NewFloatVector(data []float32) *FloatVector {
	return &FloatVector{
		dim:  len(data),
		data: data,
	}
}

// Dim returns the dimension of the vector
func (v *FloatVector) Dim() int {
	return v.dim
}

// Data returns the underlying data
func (v *FloatVector) Data() interface{} {
	return v.data
}

// Type returns the data type
func (v *FloatVector) Type() DataType {
	return DataTypeFloatVector
}

// BinaryVector represents a binary vector
type BinaryVector struct {
	dim  int
	data []byte
}

// NewBinaryVector creates a new BinaryVector
func NewBinaryVector(data []byte, dim int) *BinaryVector {
	return &BinaryVector{
		dim:  dim,
		data: data,
	}
}

// Dim returns the dimension of the vector
func (v *BinaryVector) Dim() int {
	return v.dim
}

// Data returns the underlying data
func (v *BinaryVector) Data() interface{} {
	return v.data
}

// Type returns the data type
func (v *BinaryVector) Type() DataType {
	return DataTypeBinaryVector
}

// Float16Vector represents a float16 vector
type Float16Vector struct {
	dim  int
	data []byte
}

// NewFloat16Vector creates a new Float16Vector
func NewFloat16Vector(data []byte) *Float16Vector {
	return &Float16Vector{
		dim:  len(data) / 2, // 2 bytes per float16
		data: data,
	}
}

// Dim returns the dimension of the vector
func (v *Float16Vector) Dim() int {
	return v.dim
}

// Data returns the underlying data
func (v *Float16Vector) Data() interface{} {
	return v.data
}

// Type returns the data type
func (v *Float16Vector) Type() DataType {
	return DataTypeFloat16Vector
}

// BFloat16Vector represents a bfloat16 vector
type BFloat16Vector struct {
	dim  int
	data []byte
}

// NewBFloat16Vector creates a new BFloat16Vector
func NewBFloat16Vector(data []byte) *BFloat16Vector {
	return &BFloat16Vector{
		dim:  len(data) / 2, // 2 bytes per bfloat16
		data: data,
	}
}

// Dim returns the dimension of the vector
func (v *BFloat16Vector) Dim() int {
	return v.dim
}

// Data returns the underlying data
func (v *BFloat16Vector) Data() interface{} {
	return v.data
}

// Type returns the data type
func (v *BFloat16Vector) Type() DataType {
	return DataTypeBFloat16Vector
}

// SparseFloatVector represents a sparse float vector
type SparseFloatVector struct {
	dim     int
	indices []uint32
	values  []float32
}

// NewSparseFloatVector creates a new SparseFloatVector
func NewSparseFloatVector(indices []uint32, values []float32) *SparseFloatVector {
	dim := len(indices)
	if len(values) < dim {
		dim = len(values)
	}
	return &SparseFloatVector{
		dim:     dim,
		indices: indices,
		values:  values,
	}
}

// Dim returns the dimension of the vector
func (v *SparseFloatVector) Dim() int {
	return v.dim
}

// Data returns the underlying data as a map
func (v *SparseFloatVector) Data() interface{} {
	data := make(map[uint32]float32)
	for i := 0; i < v.dim; i++ {
		data[v.indices[i]] = v.values[i]
	}
	return data
}

// Type returns the data type
func (v *SparseFloatVector) Type() DataType {
	return DataTypeSparseFloatVector
}
