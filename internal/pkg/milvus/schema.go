package milvus

import (
	"fmt"

	"github.com/milvus-io/milvus/client/v2/entity"
)

// FieldSchema 字段 Schema 定义
type FieldSchema struct {
	Name         string
	DataType     DataType
	IsPrimaryKey bool
	IsAutoID     bool
	Description  string
	Dimension    int                    // 向量维度
	MaxLength    int                    // 字符串最大长度
	TypeParams   map[string]interface{} // 类型参数
}

// CollectionSchema Collection Schema 定义
type CollectionSchema struct {
	Name              string
	Description       string
	Fields            []*FieldSchema
	EnableDynamicField bool
	AutoID            bool
}

// NewFieldSchema 创建字段 Schema
func NewFieldSchema(name string, dataType DataType) *FieldSchema {
	return &FieldSchema{
		Name:       name,
		DataType:   dataType,
		TypeParams: make(map[string]interface{}),
	}
}

// WithPrimaryKey 设置为主键
func (f *FieldSchema) WithPrimaryKey(isPrimary bool) *FieldSchema {
	f.IsPrimaryKey = isPrimary
	return f
}

// WithAutoID 设置自动 ID
func (f *FieldSchema) WithAutoID(autoID bool) *FieldSchema {
	f.IsAutoID = autoID
	return f
}

// WithDescription 设置描述
func (f *FieldSchema) WithDescription(desc string) *FieldSchema {
	f.Description = desc
	return f
}

// WithDimension 设置向量维度
func (f *FieldSchema) WithDimension(dim int) *FieldSchema {
	if !f.DataType.IsVector() {
		return f
	}
	f.Dimension = dim
	return f
}

// WithMaxLength 设置字符串最大长度
func (f *FieldSchema) WithMaxLength(maxLen int) *FieldSchema {
	if f.DataType != DataTypeVarChar {
		return f
	}
	f.MaxLength = maxLen
	return f
}

// Validate 验证字段 Schema
func (f *FieldSchema) Validate() error {
	if f.Name == "" {
		return fmt.Errorf("field name cannot be empty")
	}

	// 验证主键字段
	if f.IsPrimaryKey {
		if f.DataType != DataTypeInt64 && f.DataType != DataTypeVarChar {
			return fmt.Errorf("primary key must be Int64 or VarChar type")
		}
	}

	// 验证向量字段
	if f.DataType.IsVector() {
		if f.Dimension <= 0 {
			return fmt.Errorf("vector field %s must have positive dimension", f.Name)
		}
	}

	// 验证字符串字段
	if f.DataType == DataTypeVarChar {
		if f.MaxLength <= 0 {
			return fmt.Errorf("varchar field %s must have positive max_length", f.Name)
		}
	}

	return nil
}

// ToEntity 转换为 entity.Schema
func (f *FieldSchema) ToEntity() *entity.Field {
	field := entity.NewField()
	field.WithName(f.Name)
	field.WithDataType(entity.FieldType(f.DataType))

	if f.IsPrimaryKey {
		field.WithIsPrimaryKey(true)
	}

	if f.IsAutoID {
		field.WithIsAutoID(true)
	}

	if f.Description != "" {
		field.WithDescription(f.Description)
	}

	// 设置向量维度
	if f.DataType.IsVector() && f.Dimension > 0 {
		field.WithDim(int64(f.Dimension))
	}

	// 设置字符串最大长度
	if f.DataType == DataTypeVarChar && f.MaxLength > 0 {
		field.WithMaxLength(int64(f.MaxLength))
	}

	return field
}

// NewCollectionSchema 创建 Collection Schema
func NewCollectionSchema(name, description string) *CollectionSchema {
	return &CollectionSchema{
		Name:        name,
		Description: description,
		Fields:      make([]*FieldSchema, 0),
	}
}

// AddField 添加字段
func (s *CollectionSchema) AddField(field *FieldSchema) *CollectionSchema {
	s.Fields = append(s.Fields, field)
	return s
}

// WithEnableDynamicField 启用动态字段
func (s *CollectionSchema) WithEnableDynamicField(enable bool) *CollectionSchema {
	s.EnableDynamicField = enable
	return s
}

// WithAutoID 设置自动 ID
func (s *CollectionSchema) WithAutoID(autoID bool) *CollectionSchema {
	s.AutoID = autoID
	return s
}

// Validate 验证 Collection Schema
func (s *CollectionSchema) Validate() error {
	if s.Name == "" {
		return fmt.Errorf("collection name cannot be empty")
	}

	if len(s.Fields) == 0 {
		return fmt.Errorf("collection must have at least one field")
	}

	// 检查主键
	hasPrimaryKey := false
	hasVector := false

	for _, field := range s.Fields {
		if err := field.Validate(); err != nil {
			return err
		}

		if field.IsPrimaryKey {
			if hasPrimaryKey {
				return fmt.Errorf("collection can only have one primary key")
			}
			hasPrimaryKey = true
		}

		if field.DataType.IsVector() {
			hasVector = true
		}
	}

	if !hasPrimaryKey {
		return fmt.Errorf("collection must have a primary key")
	}

	if !hasVector {
		return fmt.Errorf("collection must have at least one vector field")
	}

	return nil
}

// ToEntity 转换为 entity.Schema
func (s *CollectionSchema) ToEntity() *entity.Schema {
	schema := &entity.Schema{
		CollectionName: s.Name,
		Description:    s.Description,
		AutoID:         s.AutoID,
		Fields:         make([]*entity.Field, 0, len(s.Fields)),
	}

	for _, field := range s.Fields {
		schema.Fields = append(schema.Fields, field.ToEntity())
	}

	if s.EnableDynamicField {
		schema.EnableDynamicField = true
	}

	return schema
}

// GetPrimaryKey 获取主键字段
func (s *CollectionSchema) GetPrimaryKey() *FieldSchema {
	for _, field := range s.Fields {
		if field.IsPrimaryKey {
			return field
		}
	}
	return nil
}

// GetVectorFields 获取所有向量字段
func (s *CollectionSchema) GetVectorFields() []*FieldSchema {
	vectors := make([]*FieldSchema, 0)
	for _, field := range s.Fields {
		if field.DataType.IsVector() {
			vectors = append(vectors, field)
		}
	}
	return vectors
}

// GetField 根据名称获取字段
func (s *CollectionSchema) GetField(name string) *FieldSchema {
	for _, field := range s.Fields {
		if field.Name == name {
			return field
		}
	}
	return nil
}

// SchemaBuilder Schema 构建器
type SchemaBuilder struct {
	schema *CollectionSchema
}

// NewSchemaBuilder 创建 Schema 构建器
func NewSchemaBuilder(name, description string) *SchemaBuilder {
	return &SchemaBuilder{
		schema: NewCollectionSchema(name, description),
	}
}

// AddInt64Field 添加 Int64 字段
func (b *SchemaBuilder) AddInt64Field(name string, isPrimaryKey, isAutoID bool) *SchemaBuilder {
	field := NewFieldSchema(name, DataTypeInt64).
		WithPrimaryKey(isPrimaryKey).
		WithAutoID(isAutoID)
	b.schema.AddField(field)
	return b
}

// AddVarCharField 添加 VarChar 字段
func (b *SchemaBuilder) AddVarCharField(name string, maxLength int, isPrimaryKey bool) *SchemaBuilder {
	field := NewFieldSchema(name, DataTypeVarChar).
		WithMaxLength(maxLength).
		WithPrimaryKey(isPrimaryKey)
	b.schema.AddField(field)
	return b
}

// AddFloatVectorField 添加 FloatVector 字段
func (b *SchemaBuilder) AddFloatVectorField(name string, dimension int) *SchemaBuilder {
	field := NewFieldSchema(name, DataTypeFloatVector).
		WithDimension(dimension)
	b.schema.AddField(field)
	return b
}

// AddBinaryVectorField 添加 BinaryVector 字段
func (b *SchemaBuilder) AddBinaryVectorField(name string, dimension int) *SchemaBuilder {
	field := NewFieldSchema(name, DataTypeBinaryVector).
		WithDimension(dimension)
	b.schema.AddField(field)
	return b
}

// AddFloat16VectorField 添加 Float16Vector 字段
func (b *SchemaBuilder) AddFloat16VectorField(name string, dimension int) *SchemaBuilder {
	field := NewFieldSchema(name, DataTypeFloat16Vector).
		WithDimension(dimension)
	b.schema.AddField(field)
	return b
}

// AddBFloat16VectorField 添加 BFloat16Vector 字段
func (b *SchemaBuilder) AddBFloat16VectorField(name string, dimension int) *SchemaBuilder {
	field := NewFieldSchema(name, DataTypeBFloat16Vector).
		WithDimension(dimension)
	b.schema.AddField(field)
	return b
}

// AddSparseFloatVectorField 添加 SparseFloatVector 字段
func (b *SchemaBuilder) AddSparseFloatVectorField(name string) *SchemaBuilder {
	field := NewFieldSchema(name, DataTypeSparseFloatVector)
	b.schema.AddField(field)
	return b
}

// AddJSONField 添加 JSON 字段
func (b *SchemaBuilder) AddJSONField(name string) *SchemaBuilder {
	field := NewFieldSchema(name, DataTypeJSON)
	b.schema.AddField(field)
	return b
}

// EnableDynamicField 启用动态字段
func (b *SchemaBuilder) EnableDynamicField() *SchemaBuilder {
	b.schema.WithEnableDynamicField(true)
	return b
}

// Build 构建 Schema
func (b *SchemaBuilder) Build() (*CollectionSchema, error) {
	if err := b.schema.Validate(); err != nil {
		return nil, err
	}
	return b.schema, nil
}
