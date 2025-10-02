package milvus

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDataType_IsVector(t *testing.T) {
	tests := []struct {
		name     string
		dataType DataType
		want     bool
	}{
		{"FloatVector is vector", DataTypeFloatVector, true},
		{"BinaryVector is vector", DataTypeBinaryVector, true},
		{"Float16Vector is vector", DataTypeFloat16Vector, true},
		{"BFloat16Vector is vector", DataTypeBFloat16Vector, true},
		{"SparseFloatVector is vector", DataTypeSparseFloatVector, true},
		{"Int64 is not vector", DataTypeInt64, false},
		{"Float is not vector", DataTypeFloat, false},
		{"VarChar is not vector", DataTypeVarChar, false},
		{"JSON is not vector", DataTypeJSON, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.dataType.IsVector()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestDataType_String(t *testing.T) {
	tests := []struct {
		dataType DataType
		want     string
	}{
		{DataTypeInt64, "Int64"},
		{DataTypeFloat, "Float"},
		{DataTypeDouble, "Double"},
		{DataTypeVarChar, "VarChar"},
		{DataTypeBool, "Bool"},
		{DataTypeJSON, "JSON"},
		{DataTypeFloatVector, "FloatVector"},
		{DataTypeBinaryVector, "BinaryVector"},
		{DataTypeFloat16Vector, "Float16Vector"},
		{DataTypeBFloat16Vector, "BFloat16Vector"},
		{DataTypeSparseFloatVector, "SparseFloatVector"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := tt.dataType.String()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestIndexType_String(t *testing.T) {
	tests := []struct {
		indexType IndexType
		want      string
	}{
		{IndexTypeFlat, "FLAT"},
		{IndexTypeIVFFlat, "IVF_FLAT"},
		{IndexTypeIVFSQ8, "IVF_SQ8"},
		{IndexTypeHNSW, "HNSW"},
		{IndexTypeDiskANN, "DISKANN"},
		{IndexTypeAutoIndex, "AUTOINDEX"},
		{IndexTypeGPUIVFFlat, "GPU_IVF_FLAT"},
		{IndexTypeGPUIVFPQ, "GPU_IVF_PQ"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := tt.indexType.String()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestMetricType_String(t *testing.T) {
	tests := []struct {
		metricType MetricType
		want       string
	}{
		{MetricTypeL2, "L2"},
		{MetricTypeIP, "IP"},
		{MetricTypeCosine, "COSINE"},
		{MetricTypeJaccard, "JACCARD"},
		{MetricTypeHamming, "HAMMING"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := tt.metricType.String()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestFloatVector(t *testing.T) {
	data := []float32{1.0, 2.0, 3.0, 4.0}
	vec := NewFloatVector(data)

	assert.Equal(t, 4, vec.Dim())
	assert.Equal(t, data, vec.Data())
	assert.Equal(t, DataTypeFloatVector, vec.Type())
}

func TestBinaryVector(t *testing.T) {
	data := []byte{0xFF, 0x00, 0xAB, 0xCD}
	vec := NewBinaryVector(data, 32) // 4 bytes * 8 bits

	assert.Equal(t, 32, vec.Dim())
	assert.Equal(t, data, vec.Data())
	assert.Equal(t, DataTypeBinaryVector, vec.Type())
}

func TestFloat16Vector(t *testing.T) {
	data := []byte{0x01, 0x02, 0x03, 0x04}
	vec := NewFloat16Vector(data)

	assert.Equal(t, 2, vec.Dim()) // 4 bytes / 2
	assert.Equal(t, data, vec.Data())
	assert.Equal(t, DataTypeFloat16Vector, vec.Type())
}

func TestBFloat16Vector(t *testing.T) {
	data := []byte{0x01, 0x02, 0x03, 0x04}
	vec := NewBFloat16Vector(data)

	assert.Equal(t, 2, vec.Dim()) // 4 bytes / 2
	assert.Equal(t, data, vec.Data())
	assert.Equal(t, DataTypeBFloat16Vector, vec.Type())
}

func TestSparseFloatVector(t *testing.T) {
	indices := []uint32{0, 10, 20, 30}
	values := []float32{1.0, 2.0, 3.0, 4.0}
	vec := NewSparseFloatVector(indices, values)

	assert.Equal(t, 4, vec.Dim())
	data := vec.Data().(map[uint32]float32)
	assert.Equal(t, float32(1.0), data[0])
	assert.Equal(t, float32(2.0), data[10])
	assert.Equal(t, float32(3.0), data[20])
	assert.Equal(t, float32(4.0), data[30])
	assert.Equal(t, DataTypeSparseFloatVector, vec.Type())
}

func TestVector_EdgeCases(t *testing.T) {
	t.Run("empty float vector", func(t *testing.T) {
		vec := NewFloatVector([]float32{})
		assert.Equal(t, 0, vec.Dim())
		assert.NotNil(t, vec.Data())
	})

	t.Run("empty binary vector", func(t *testing.T) {
		vec := NewBinaryVector([]byte{}, 0)
		assert.Equal(t, 0, vec.Dim())
		assert.NotNil(t, vec.Data())
	})

	t.Run("sparse vector with mismatched lengths", func(t *testing.T) {
		indices := []uint32{0, 1}
		values := []float32{1.0, 2.0, 3.0}
		vec := NewSparseFloatVector(indices, values)
		// 应该取最小长度
		assert.Equal(t, 2, vec.Dim())
	})

	t.Run("large dimension vector", func(t *testing.T) {
		data := make([]float32, 2048)
		for i := range data {
			data[i] = float32(i)
		}
		vec := NewFloatVector(data)
		assert.Equal(t, 2048, vec.Dim())
	})
}

func TestDataType_Validation(t *testing.T) {
	tests := []struct {
		name     string
		dataType DataType
		isValid  bool
	}{
		{"valid Int64", DataTypeInt64, true},
		{"valid Float", DataTypeFloat, true},
		{"valid FloatVector", DataTypeFloatVector, true},
		{"invalid type", DataType(999), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 基本验证：所有定义的类型都应该有字符串表示
			str := tt.dataType.String()
			if tt.isValid {
				assert.NotEmpty(t, str)
			}
		})
	}
}

func TestIndexType_Validation(t *testing.T) {
	validIndexTypes := []IndexType{
		IndexTypeFlat,
		IndexTypeIVFFlat,
		IndexTypeIVFSQ8,
		IndexTypeHNSW,
		IndexTypeDiskANN,
		IndexTypeAutoIndex,
	}

	for _, indexType := range validIndexTypes {
		t.Run(indexType.String(), func(t *testing.T) {
			str := indexType.String()
			assert.NotEmpty(t, str)
		})
	}
}

func TestMetricType_Validation(t *testing.T) {
	validMetricTypes := []MetricType{
		MetricTypeL2,
		MetricTypeIP,
		MetricTypeCosine,
		MetricTypeJaccard,
		MetricTypeHamming,
	}

	for _, metricType := range validMetricTypes {
		t.Run(metricType.String(), func(t *testing.T) {
			str := metricType.String()
			assert.NotEmpty(t, str)
		})
	}
}
