package milvus

import (
	"fmt"
	"math"

	"github.com/milvus-io/milvus/client/v2/column"
)

// BuildInt64Column 构建 Int64 列
func BuildInt64Column(name string, values []int64) column.Column {
	return column.NewColumnInt64(name, values)
}

// BuildFloatColumn 构建 Float 列
func BuildFloatColumn(name string, values []float32) column.Column {
	return column.NewColumnFloat(name, values)
}

// BuildDoubleColumn 构建 Double 列
func BuildDoubleColumn(name string, values []float64) column.Column {
	return column.NewColumnDouble(name, values)
}

// BuildVarCharColumn 构建 VarChar 列
func BuildVarCharColumn(name string, values []string) column.Column {
	return column.NewColumnVarChar(name, values)
}

// BuildBoolColumn 构建 Bool 列
func BuildBoolColumn(name string, values []bool) column.Column {
	return column.NewColumnBool(name, values)
}

// BuildFloatVectorColumn 构建 FloatVector 列
func BuildFloatVectorColumn(name string, dim int, vectors [][]float32) column.Column {
	return column.NewColumnFloatVector(name, dim, vectors)
}

// BuildBinaryVectorColumn 构建 BinaryVector 列
func BuildBinaryVectorColumn(name string, dim int, vectors [][]byte) column.Column {
	return column.NewColumnBinaryVector(name, dim, vectors)
}

// BuildJSONColumn 构建 JSON 列
func BuildJSONColumn(name string, values [][]byte) column.Column {
	return column.NewColumnJSONBytes(name, values)
}

// ValidateVectorDimension 验证向量维度
func ValidateVectorDimension(vectors [][]float32, expectedDim int) error {
	for i, vec := range vectors {
		if len(vec) != expectedDim {
			return fmt.Errorf("vector at index %d has dimension %d, expected %d", i, len(vec), expectedDim)
		}
	}
	return nil
}

// NormalizeVector L2 归一化向量
func NormalizeVector(vector []float32) []float32 {
	var norm float32
	for _, v := range vector {
		norm += v * v
	}
	if norm == 0 {
		return vector
	}

	// L2 范数是平方和的平方根
	norm = float32(math.Sqrt(float64(norm)))
	normalized := make([]float32, len(vector))
	for i, v := range vector {
		normalized[i] = v / norm
	}
	return normalized
}

// NormalizeVectors L2 归一化多个向量
func NormalizeVectors(vectors [][]float32) [][]float32 {
	normalized := make([][]float32, len(vectors))
	for i, vec := range vectors {
		normalized[i] = NormalizeVector(vec)
	}
	return normalized
}

// BuildExprIn 构建 IN 表达式
func BuildExprIn(field string, values []interface{}) string {
	if len(values) == 0 {
		return ""
	}

	expr := fmt.Sprintf("%s in [", field)
	for i, v := range values {
		if i > 0 {
			expr += ", "
		}
		switch val := v.(type) {
		case string:
			expr += fmt.Sprintf("\"%s\"", val)
		default:
			expr += fmt.Sprintf("%v", val)
		}
	}
	expr += "]"
	return expr
}

// BuildExprRange 构建范围表达式
func BuildExprRange(field string, min, max interface{}) string {
	return fmt.Sprintf("%s >= %v && %s <= %v", field, min, field, max)
}

// BuildExprAnd 构建 AND 表达式
func BuildExprAnd(exprs ...string) string {
	if len(exprs) == 0 {
		return ""
	}
	if len(exprs) == 1 {
		return exprs[0]
	}

	result := "(" + exprs[0] + ")"
	for i := 1; i < len(exprs); i++ {
		result += " && (" + exprs[i] + ")"
	}
	return result
}

// BuildExprOr 构建 OR 表达式
func BuildExprOr(exprs ...string) string {
	if len(exprs) == 0 {
		return ""
	}
	if len(exprs) == 1 {
		return exprs[0]
	}

	result := "(" + exprs[0] + ")"
	for i := 1; i < len(exprs); i++ {
		result += " || (" + exprs[i] + ")"
	}
	return result
}

// ChunkSlice 将切片分块
func ChunkSlice[T any](slice []T, chunkSize int) [][]T {
	if chunkSize <= 0 {
		return [][]T{slice}
	}

	var chunks [][]T
	for i := 0; i < len(slice); i += chunkSize {
		end := i + chunkSize
		if end > len(slice) {
			end = len(slice)
		}
		chunks = append(chunks, slice[i:end])
	}
	return chunks
}
