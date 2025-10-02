package milvus

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBuildInt64Column(t *testing.T) {
	values := []int64{1, 2, 3, 4, 5}
	col := BuildInt64Column("id", values)

	assert.NotNil(t, col)
	assert.Equal(t, "id", col.Name())
	assert.Equal(t, 5, col.Len())
}

func TestBuildFloatColumn(t *testing.T) {
	values := []float32{1.1, 2.2, 3.3}
	col := BuildFloatColumn("score", values)

	assert.NotNil(t, col)
	assert.Equal(t, "score", col.Name())
	assert.Equal(t, 3, col.Len())
}

func TestBuildVarCharColumn(t *testing.T) {
	values := []string{"hello", "world", "test"}
	col := BuildVarCharColumn("title", values)

	assert.NotNil(t, col)
	assert.Equal(t, "title", col.Name())
	assert.Equal(t, 3, col.Len())
}

func TestBuildBoolColumn(t *testing.T) {
	values := []bool{true, false, true}
	col := BuildBoolColumn("flag", values)

	assert.NotNil(t, col)
	assert.Equal(t, "flag", col.Name())
	assert.Equal(t, 3, col.Len())
}

func TestBuildFloatVectorColumn(t *testing.T) {
	vectors := [][]float32{
		{1.0, 2.0, 3.0},
		{4.0, 5.0, 6.0},
	}
	col := BuildFloatVectorColumn("embedding", 3, vectors)

	assert.NotNil(t, col)
	assert.Equal(t, "embedding", col.Name())
	assert.Equal(t, 2, col.Len())
}

func TestBuildBinaryVectorColumn(t *testing.T) {
	vectors := [][]byte{
		{0xFF, 0x00},
		{0xAB, 0xCD},
	}
	col := BuildBinaryVectorColumn("binary_vec", 16, vectors)

	assert.NotNil(t, col)
	assert.Equal(t, "binary_vec", col.Name())
	assert.Equal(t, 2, col.Len())
}

func TestValidateVectorDimension(t *testing.T) {
	tests := []struct {
		name        string
		vectors     [][]float32
		expectedDim int
		wantErr     bool
	}{
		{
			name: "valid vectors",
			vectors: [][]float32{
				{1.0, 2.0, 3.0},
				{4.0, 5.0, 6.0},
			},
			expectedDim: 3,
			wantErr:     false,
		},
		{
			name: "dimension mismatch",
			vectors: [][]float32{
				{1.0, 2.0, 3.0},
				{4.0, 5.0},
			},
			expectedDim: 3,
			wantErr:     true,
		},
		{
			name:        "empty vectors",
			vectors:     [][]float32{},
			expectedDim: 3,
			wantErr:     false,
		},
		{
			name: "zero dimension",
			vectors: [][]float32{
				{},
				{},
			},
			expectedDim: 0,
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateVectorDimension(tt.vectors, tt.expectedDim)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestNormalizeVector(t *testing.T) {
	tests := []struct {
		name     string
		vector   []float32
		checkNorm bool
	}{
		{
			name:      "normal vector",
			vector:    []float32{3.0, 4.0},
			checkNorm: true,
		},
		{
			name:      "already normalized",
			vector:    []float32{1.0, 0.0},
			checkNorm: true,
		},
		{
			name:      "zero vector",
			vector:    []float32{0.0, 0.0},
			checkNorm: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			normalized := NormalizeVector(tt.vector)
			assert.NotNil(t, normalized)
			assert.Equal(t, len(tt.vector), len(normalized))

			if tt.checkNorm {
				// 计算 L2 范数
				var norm float32
				for _, v := range normalized {
					norm += v * v
				}
				// 归一化后的向量范数应该接近 1
				assert.InDelta(t, 1.0, norm, 0.0001)
			}
		})
	}
}

func TestNormalizeVectors(t *testing.T) {
	vectors := [][]float32{
		{3.0, 4.0},
		{1.0, 0.0},
		{0.0, 1.0},
	}

	normalized := NormalizeVectors(vectors)
	assert.Len(t, normalized, 3)

	for i, vec := range normalized {
		assert.Len(t, vec, len(vectors[i]))
	}
}

func TestBuildExprIn(t *testing.T) {
	tests := []struct {
		name   string
		field  string
		values []interface{}
		want   string
	}{
		{
			name:   "int values",
			field:  "id",
			values: []interface{}{1, 2, 3},
			want:   "id in [1, 2, 3]",
		},
		{
			name:   "string values",
			field:  "name",
			values: []interface{}{"Alice", "Bob"},
			want:   `name in ["Alice", "Bob"]`,
		},
		{
			name:   "single value",
			field:  "id",
			values: []interface{}{1},
			want:   "id in [1]",
		},
		{
			name:   "empty values",
			field:  "id",
			values: []interface{}{},
			want:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := BuildExprIn(tt.field, tt.values)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestBuildExprRange(t *testing.T) {
	tests := []struct {
		name  string
		field string
		min   interface{}
		max   interface{}
		want  string
	}{
		{
			name:  "int range",
			field: "age",
			min:   18,
			max:   65,
			want:  "age >= 18 && age <= 65",
		},
		{
			name:  "float range",
			field: "score",
			min:   0.5,
			max:   1.0,
			want:  "score >= 0.5 && score <= 1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := BuildExprRange(tt.field, tt.min, tt.max)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestBuildExprAnd(t *testing.T) {
	tests := []struct {
		name  string
		exprs []string
		want  string
	}{
		{
			name:  "two expressions",
			exprs: []string{"id > 0", "age < 100"},
			want:  "(id > 0) && (age < 100)",
		},
		{
			name:  "three expressions",
			exprs: []string{"a == 1", "b == 2", "c == 3"},
			want:  "(a == 1) && (b == 2) && (c == 3)",
		},
		{
			name:  "single expression",
			exprs: []string{"id > 0"},
			want:  "id > 0",
		},
		{
			name:  "empty expressions",
			exprs: []string{},
			want:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := BuildExprAnd(tt.exprs...)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestBuildExprOr(t *testing.T) {
	tests := []struct {
		name  string
		exprs []string
		want  string
	}{
		{
			name:  "two expressions",
			exprs: []string{"name == 'Alice'", "name == 'Bob'"},
			want:  "(name == 'Alice') || (name == 'Bob')",
		},
		{
			name:  "three expressions",
			exprs: []string{"a == 1", "b == 2", "c == 3"},
			want:  "(a == 1) || (b == 2) || (c == 3)",
		},
		{
			name:  "single expression",
			exprs: []string{"id > 0"},
			want:  "id > 0",
		},
		{
			name:  "empty expressions",
			exprs: []string{},
			want:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := BuildExprOr(tt.exprs...)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestChunkSlice(t *testing.T) {
	tests := []struct {
		name      string
		slice     []int
		chunkSize int
		wantLen   int
	}{
		{
			name:      "exact chunks",
			slice:     []int{1, 2, 3, 4, 5, 6},
			chunkSize: 2,
			wantLen:   3,
		},
		{
			name:      "uneven chunks",
			slice:     []int{1, 2, 3, 4, 5},
			chunkSize: 2,
			wantLen:   3,
		},
		{
			name:      "chunk larger than slice",
			slice:     []int{1, 2, 3},
			chunkSize: 10,
			wantLen:   1,
		},
		{
			name:      "chunk size 1",
			slice:     []int{1, 2, 3},
			chunkSize: 1,
			wantLen:   3,
		},
		{
			name:      "negative chunk size",
			slice:     []int{1, 2, 3},
			chunkSize: -1,
			wantLen:   1,
		},
		{
			name:      "zero chunk size",
			slice:     []int{1, 2, 3},
			chunkSize: 0,
			wantLen:   1,
		},
		{
			name:      "empty slice",
			slice:     []int{},
			chunkSize: 2,
			wantLen:   0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			chunks := ChunkSlice(tt.slice, tt.chunkSize)
			assert.Len(t, chunks, tt.wantLen)

			// 验证所有元素都被包含
			total := 0
			for _, chunk := range chunks {
				total += len(chunk)
			}
			assert.Equal(t, len(tt.slice), total)
		})
	}
}

func TestChunkSlice_Verification(t *testing.T) {
	slice := []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
	chunks := ChunkSlice(slice, 3)

	assert.Len(t, chunks, 4)
	assert.Equal(t, []int{1, 2, 3}, chunks[0])
	assert.Equal(t, []int{4, 5, 6}, chunks[1])
	assert.Equal(t, []int{7, 8, 9}, chunks[2])
	assert.Equal(t, []int{10}, chunks[3])
}

func TestNormalizeVector_EdgeCases(t *testing.T) {
	t.Run("single element", func(t *testing.T) {
		vec := []float32{5.0}
		normalized := NormalizeVector(vec)
		assert.Len(t, normalized, 1)
		// 单元素向量归一化后应该是 1.0
		var norm float32
		for _, v := range normalized {
			norm += v * v
		}
		assert.InDelta(t, 1.0, norm, 0.0001)
	})

	t.Run("very small values", func(t *testing.T) {
		vec := []float32{0.0001, 0.0002, 0.0003}
		normalized := NormalizeVector(vec)
		assert.Len(t, normalized, 3)
	})

	t.Run("very large values", func(t *testing.T) {
		vec := []float32{1000000.0, 2000000.0}
		normalized := NormalizeVector(vec)
		assert.Len(t, normalized, 2)

		var norm float32
		for _, v := range normalized {
			norm += v * v
		}
		assert.InDelta(t, 1.0, norm, 0.0001)
	})

	t.Run("negative values", func(t *testing.T) {
		vec := []float32{-3.0, -4.0}
		normalized := NormalizeVector(vec)
		assert.Len(t, normalized, 2)

		var norm float32
		for _, v := range normalized {
			norm += v * v
		}
		assert.InDelta(t, 1.0, norm, 0.0001)
	})

	t.Run("mixed positive and negative", func(t *testing.T) {
		vec := []float32{3.0, -4.0}
		normalized := NormalizeVector(vec)
		assert.Len(t, normalized, 2)

		var norm float32
		for _, v := range normalized {
			norm += v * v
		}
		assert.InDelta(t, 1.0, norm, 0.0001)
	})
}

func TestBuildExpr_Complex(t *testing.T) {
	// 构建复杂表达式
	expr1 := BuildExprIn("category", []interface{}{"tech", "science"})
	expr2 := BuildExprRange("score", 0.5, 1.0)
	expr3 := "status == 'active'"

	combined := BuildExprAnd(expr1, expr2, expr3)

	assert.Contains(t, combined, "category in")
	assert.Contains(t, combined, "score >=")
	assert.Contains(t, combined, "status == 'active'")
	assert.Contains(t, combined, "&&")
}

func TestValidateVectorDimension_LargeVectors(t *testing.T) {
	// 测试大维度向量
	dim := 768
	vectors := make([][]float32, 100)
	for i := range vectors {
		vectors[i] = make([]float32, dim)
		for j := range vectors[i] {
			vectors[i][j] = float32(j)
		}
	}

	err := ValidateVectorDimension(vectors, dim)
	assert.NoError(t, err)

	// 添加一个错误维度的向量
	vectors[50] = make([]float32, dim+1)
	err = ValidateVectorDimension(vectors, dim)
	assert.Error(t, err)
}

func TestNormalizeVector_Precision(t *testing.T) {
	vec := []float32{1.0, 1.0, 1.0, 1.0}
	normalized := NormalizeVector(vec)

	// 计算精确的 L2 范数
	var norm float64
	for _, v := range normalized {
		norm += float64(v) * float64(v)
	}
	norm = math.Sqrt(norm)

	// 归一化后范数应该非常接近 1.0
	assert.InDelta(t, 1.0, norm, 0.000001)
}
