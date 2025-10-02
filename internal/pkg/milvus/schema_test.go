package milvus

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewFieldSchema(t *testing.T) {
	field := NewFieldSchema("test_field", DataTypeInt64)

	assert.NotNil(t, field)
	assert.Equal(t, "test_field", field.Name)
	assert.Equal(t, DataTypeInt64, field.DataType)
	assert.False(t, field.IsPrimaryKey)
	assert.False(t, field.IsAutoID)
	assert.NotNil(t, field.TypeParams)
}

func TestFieldSchema_WithPrimaryKey(t *testing.T) {
	field := NewFieldSchema("id", DataTypeInt64).
		WithPrimaryKey(true)

	assert.True(t, field.IsPrimaryKey)
}

func TestFieldSchema_WithAutoID(t *testing.T) {
	field := NewFieldSchema("id", DataTypeInt64).
		WithAutoID(true)

	assert.True(t, field.IsAutoID)
}

func TestFieldSchema_WithDescription(t *testing.T) {
	desc := "This is a test field"
	field := NewFieldSchema("test", DataTypeFloat).
		WithDescription(desc)

	assert.Equal(t, desc, field.Description)
}

func TestFieldSchema_WithDimension(t *testing.T) {
	field := NewFieldSchema("embedding", DataTypeFloatVector).
		WithDimension(768)

	assert.Equal(t, 768, field.Dimension)

	// 非向量类型不应设置维度
	field2 := NewFieldSchema("id", DataTypeInt64).
		WithDimension(100)
	assert.Equal(t, 0, field2.Dimension)
}

func TestFieldSchema_WithMaxLength(t *testing.T) {
	field := NewFieldSchema("title", DataTypeVarChar).
		WithMaxLength(256)

	assert.Equal(t, 256, field.MaxLength)

	// 非 VarChar 类型不应设置最大长度
	field2 := NewFieldSchema("id", DataTypeInt64).
		WithMaxLength(100)
	assert.Equal(t, 0, field2.MaxLength)
}

func TestFieldSchema_Validate(t *testing.T) {
	tests := []struct {
		name    string
		field   *FieldSchema
		wantErr bool
	}{
		{
			name: "valid int64 field",
			field: NewFieldSchema("id", DataTypeInt64).
				WithPrimaryKey(true).
				WithAutoID(true),
			wantErr: false,
		},
		{
			name: "valid varchar field",
			field: NewFieldSchema("title", DataTypeVarChar).
				WithMaxLength(256),
			wantErr: false,
		},
		{
			name: "valid vector field",
			field: NewFieldSchema("embedding", DataTypeFloatVector).
				WithDimension(768),
			wantErr: false,
		},
		{
			name:    "empty field name",
			field:   NewFieldSchema("", DataTypeInt64),
			wantErr: true,
		},
		{
			name: "invalid primary key type",
			field: NewFieldSchema("id", DataTypeFloat).
				WithPrimaryKey(true),
			wantErr: true,
		},
		{
			name: "vector without dimension",
			field: NewFieldSchema("embedding", DataTypeFloatVector),
			wantErr: true,
		},
		{
			name:    "varchar without max length",
			field:   NewFieldSchema("title", DataTypeVarChar),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.field.Validate()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestNewCollectionSchema(t *testing.T) {
	schema := NewCollectionSchema("test_collection", "Test description")

	assert.NotNil(t, schema)
	assert.Equal(t, "test_collection", schema.Name)
	assert.Equal(t, "Test description", schema.Description)
	assert.NotNil(t, schema.Fields)
	assert.Len(t, schema.Fields, 0)
}

func TestCollectionSchema_AddField(t *testing.T) {
	schema := NewCollectionSchema("test", "")

	field1 := NewFieldSchema("id", DataTypeInt64)
	field2 := NewFieldSchema("vector", DataTypeFloatVector).WithDimension(128)

	schema.AddField(field1).AddField(field2)

	assert.Len(t, schema.Fields, 2)
	assert.Equal(t, "id", schema.Fields[0].Name)
	assert.Equal(t, "vector", schema.Fields[1].Name)
}

func TestCollectionSchema_WithEnableDynamicField(t *testing.T) {
	schema := NewCollectionSchema("test", "").
		WithEnableDynamicField(true)

	assert.True(t, schema.EnableDynamicField)
}

func TestCollectionSchema_WithAutoID(t *testing.T) {
	schema := NewCollectionSchema("test", "").
		WithAutoID(true)

	assert.True(t, schema.AutoID)
}

func TestCollectionSchema_Validate(t *testing.T) {
	tests := []struct {
		name    string
		schema  *CollectionSchema
		wantErr bool
	}{
		{
			name: "valid schema",
			schema: NewCollectionSchema("test", "").
				AddField(NewFieldSchema("id", DataTypeInt64).WithPrimaryKey(true)).
				AddField(NewFieldSchema("vector", DataTypeFloatVector).WithDimension(128)),
			wantErr: false,
		},
		{
			name:    "empty collection name",
			schema:  NewCollectionSchema("", ""),
			wantErr: true,
		},
		{
			name:    "no fields",
			schema:  NewCollectionSchema("test", ""),
			wantErr: true,
		},
		{
			name: "no primary key",
			schema: NewCollectionSchema("test", "").
				AddField(NewFieldSchema("vector", DataTypeFloatVector).WithDimension(128)),
			wantErr: true,
		},
		{
			name: "no vector field",
			schema: NewCollectionSchema("test", "").
				AddField(NewFieldSchema("id", DataTypeInt64).WithPrimaryKey(true)),
			wantErr: true,
		},
		{
			name: "multiple primary keys",
			schema: NewCollectionSchema("test", "").
				AddField(NewFieldSchema("id1", DataTypeInt64).WithPrimaryKey(true)).
				AddField(NewFieldSchema("id2", DataTypeInt64).WithPrimaryKey(true)).
				AddField(NewFieldSchema("vector", DataTypeFloatVector).WithDimension(128)),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.schema.Validate()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestCollectionSchema_GetPrimaryKey(t *testing.T) {
	schema := NewCollectionSchema("test", "").
		AddField(NewFieldSchema("id", DataTypeInt64).WithPrimaryKey(true)).
		AddField(NewFieldSchema("vector", DataTypeFloatVector).WithDimension(128))

	pk := schema.GetPrimaryKey()
	assert.NotNil(t, pk)
	assert.Equal(t, "id", pk.Name)
	assert.True(t, pk.IsPrimaryKey)
}

func TestCollectionSchema_GetVectorFields(t *testing.T) {
	schema := NewCollectionSchema("test", "").
		AddField(NewFieldSchema("id", DataTypeInt64).WithPrimaryKey(true)).
		AddField(NewFieldSchema("vector1", DataTypeFloatVector).WithDimension(128)).
		AddField(NewFieldSchema("vector2", DataTypeBinaryVector).WithDimension(256))

	vectors := schema.GetVectorFields()
	assert.Len(t, vectors, 2)
	assert.Equal(t, "vector1", vectors[0].Name)
	assert.Equal(t, "vector2", vectors[1].Name)
}

func TestCollectionSchema_GetField(t *testing.T) {
	schema := NewCollectionSchema("test", "").
		AddField(NewFieldSchema("id", DataTypeInt64)).
		AddField(NewFieldSchema("title", DataTypeVarChar).WithMaxLength(256))

	field := schema.GetField("title")
	assert.NotNil(t, field)
	assert.Equal(t, "title", field.Name)
	assert.Equal(t, DataTypeVarChar, field.DataType)

	notFound := schema.GetField("nonexistent")
	assert.Nil(t, notFound)
}

func TestSchemaBuilder(t *testing.T) {
	builder := NewSchemaBuilder("test_collection", "Test description")

	assert.NotNil(t, builder)
	assert.NotNil(t, builder.schema)
	assert.Equal(t, "test_collection", builder.schema.Name)
}

func TestSchemaBuilder_AddInt64Field(t *testing.T) {
	builder := NewSchemaBuilder("test", "")
	builder.AddInt64Field("id", true, true)

	assert.Len(t, builder.schema.Fields, 1)
	field := builder.schema.Fields[0]
	assert.Equal(t, "id", field.Name)
	assert.Equal(t, DataTypeInt64, field.DataType)
	assert.True(t, field.IsPrimaryKey)
	assert.True(t, field.IsAutoID)
}

func TestSchemaBuilder_AddVarCharField(t *testing.T) {
	builder := NewSchemaBuilder("test", "")
	builder.AddVarCharField("title", 256, false)

	assert.Len(t, builder.schema.Fields, 1)
	field := builder.schema.Fields[0]
	assert.Equal(t, "title", field.Name)
	assert.Equal(t, DataTypeVarChar, field.DataType)
	assert.Equal(t, 256, field.MaxLength)
	assert.False(t, field.IsPrimaryKey)
}

func TestSchemaBuilder_AddFloatVectorField(t *testing.T) {
	builder := NewSchemaBuilder("test", "")
	builder.AddFloatVectorField("embedding", 768)

	assert.Len(t, builder.schema.Fields, 1)
	field := builder.schema.Fields[0]
	assert.Equal(t, "embedding", field.Name)
	assert.Equal(t, DataTypeFloatVector, field.DataType)
	assert.Equal(t, 768, field.Dimension)
}

func TestSchemaBuilder_AddBinaryVectorField(t *testing.T) {
	builder := NewSchemaBuilder("test", "")
	builder.AddBinaryVectorField("binary_vec", 256)

	assert.Len(t, builder.schema.Fields, 1)
	field := builder.schema.Fields[0]
	assert.Equal(t, "binary_vec", field.Name)
	assert.Equal(t, DataTypeBinaryVector, field.DataType)
	assert.Equal(t, 256, field.Dimension)
}

func TestSchemaBuilder_AddJSONField(t *testing.T) {
	builder := NewSchemaBuilder("test", "")
	builder.AddJSONField("metadata")

	assert.Len(t, builder.schema.Fields, 1)
	field := builder.schema.Fields[0]
	assert.Equal(t, "metadata", field.Name)
	assert.Equal(t, DataTypeJSON, field.DataType)
}

func TestSchemaBuilder_EnableDynamicField(t *testing.T) {
	builder := NewSchemaBuilder("test", "")
	builder.EnableDynamicField()

	assert.True(t, builder.schema.EnableDynamicField)
}

func TestSchemaBuilder_Build(t *testing.T) {
	tests := []struct {
		name    string
		setup   func() *SchemaBuilder
		wantErr bool
	}{
		{
			name: "valid schema",
			setup: func() *SchemaBuilder {
				return NewSchemaBuilder("test", "").
					AddInt64Field("id", true, true).
					AddFloatVectorField("embedding", 768)
			},
			wantErr: false,
		},
		{
			name: "invalid schema - no primary key",
			setup: func() *SchemaBuilder {
				return NewSchemaBuilder("test", "").
					AddFloatVectorField("embedding", 768)
			},
			wantErr: true,
		},
		{
			name: "invalid schema - no vector field",
			setup: func() *SchemaBuilder {
				return NewSchemaBuilder("test", "").
					AddInt64Field("id", true, true)
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			builder := tt.setup()
			schema, err := builder.Build()

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, schema)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, schema)
			}
		})
	}
}

func TestSchemaBuilder_ComplexSchema(t *testing.T) {
	schema, err := NewSchemaBuilder("complex_collection", "Complex schema example").
		AddInt64Field("id", true, true).
		AddVarCharField("title", 512, false).
		AddVarCharField("content", 2048, false).
		AddFloatVectorField("title_embedding", 384).
		AddFloatVectorField("content_embedding", 768).
		AddJSONField("metadata").
		EnableDynamicField().
		Build()

	assert.NoError(t, err)
	assert.NotNil(t, schema)
	assert.Len(t, schema.Fields, 6)
	assert.True(t, schema.EnableDynamicField)

	// 验证主键
	pk := schema.GetPrimaryKey()
	assert.NotNil(t, pk)
	assert.Equal(t, "id", pk.Name)

	// 验证向量字段
	vectors := schema.GetVectorFields()
	assert.Len(t, vectors, 2)
	assert.Equal(t, 384, vectors[0].Dimension)
	assert.Equal(t, 768, vectors[1].Dimension)
}

func TestSchemaBuilder_ChainedCalls(t *testing.T) {
	builder := NewSchemaBuilder("test", "Test collection").
		AddInt64Field("id", true, true).
		AddVarCharField("name", 100, false).
		AddFloatVectorField("vec", 128).
		EnableDynamicField()

	schema, err := builder.Build()

	assert.NoError(t, err)
	assert.NotNil(t, schema)
	assert.Len(t, schema.Fields, 3)
	assert.True(t, schema.EnableDynamicField)
}
