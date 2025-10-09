package data

import (
	"context"
	"fmt"

	"github.com/lk2023060901/ai-writer-backend/internal/knowledge/biz"
	"github.com/lk2023060901/ai-writer-backend/internal/pkg/milvus"
	"github.com/milvus-io/milvus/client/v2/column"
	"github.com/milvus-io/milvus/client/v2/entity"
	"github.com/milvus-io/milvus/client/v2/index"
	"github.com/milvus-io/milvus/client/v2/milvusclient"
)

// MilvusVectorDBService 实现 biz.VectorDBService 接口
type MilvusVectorDBService struct {
	client *milvus.Client
}

// NewMilvusVectorDBService 创建 Milvus 向量数据库服务
func NewMilvusVectorDBService(client *milvus.Client) *MilvusVectorDBService {
	return &MilvusVectorDBService{
		client: client,
	}
}

// CreateCollection 创建向量 collection
func (s *MilvusVectorDBService) CreateCollection(ctx context.Context, collectionName string, dimension int) error {
	cli := s.client.GetClient()
	if cli == nil {
		return fmt.Errorf("milvus client is not available")
	}

	// 检查 collection 是否已存在
	has, err := cli.HasCollection(ctx, milvusclient.NewHasCollectionOption(collectionName))
	if err != nil {
		return fmt.Errorf("failed to check collection: %w", err)
	}

	if has {
		return nil // 已存在，直接返回
	}

	// 创建 schema
	schema := entity.NewSchema().
		WithName(collectionName).
		WithField(entity.NewField().WithName("id").WithDataType(entity.FieldTypeVarChar).WithMaxLength(64).WithIsPrimaryKey(true)).
		WithField(entity.NewField().WithName("document_id").WithDataType(entity.FieldTypeVarChar).WithMaxLength(64)).
		WithField(entity.NewField().WithName("chunk_id").WithDataType(entity.FieldTypeVarChar).WithMaxLength(64)).
		WithField(entity.NewField().WithName("content").WithDataType(entity.FieldTypeVarChar).WithMaxLength(65535)).
		WithField(entity.NewField().WithName("embedding").WithDataType(entity.FieldTypeFloatVector).WithDim(int64(dimension)))

	// 创建 collection
	err = cli.CreateCollection(ctx, milvusclient.NewCreateCollectionOption(collectionName, schema))
	if err != nil {
		return fmt.Errorf("failed to create collection: %w", err)
	}

	// 创建向量索引
	idx := index.NewAutoIndex(entity.COSINE)
	_, err = cli.CreateIndex(ctx, milvusclient.NewCreateIndexOption(collectionName, "embedding", idx))
	if err != nil {
		return fmt.Errorf("failed to create index: %w", err)
	}

	// 加载 collection
	loadTask, err := cli.LoadCollection(ctx, milvusclient.NewLoadCollectionOption(collectionName))
	if err != nil {
		return fmt.Errorf("failed to load collection: %w", err)
	}

	// 等待加载完成
	err = loadTask.Await(ctx)
	if err != nil {
		return fmt.Errorf("failed to wait for collection load: %w", err)
	}

	return nil
}

// InsertVectors 批量插入向量
func (s *MilvusVectorDBService) InsertVectors(ctx context.Context, collectionName string, chunks []*biz.Chunk) error {
	if len(chunks) == 0 {
		return nil
	}

	cli := s.client.GetClient()
	if cli == nil {
		return fmt.Errorf("milvus client is not available")
	}

	// 准备数据列
	ids := make([]string, len(chunks))
	documentIDs := make([]string, len(chunks))
	chunkIDs := make([]string, len(chunks))
	contents := make([]string, len(chunks))
	embeddings := make([][]float32, len(chunks))

	for i, chunk := range chunks {
		ids[i] = chunk.ID
		documentIDs[i] = chunk.DocumentID
		chunkIDs[i] = chunk.ID
		contents[i] = chunk.Content
		embeddings[i] = chunk.Embedding
	}

	// 创建 column 数据
	idColumn := column.NewColumnVarChar("id", ids)
	documentIDColumn := column.NewColumnVarChar("document_id", documentIDs)
	chunkIDColumn := column.NewColumnVarChar("chunk_id", chunkIDs)
	contentColumn := column.NewColumnVarChar("content", contents)
	embeddingColumn := column.NewColumnFloatVector("embedding", len(embeddings[0]), embeddings)

	// 插入数据
	_, err := cli.Insert(ctx, milvusclient.NewColumnBasedInsertOption(collectionName).
		WithColumns(idColumn, documentIDColumn, chunkIDColumn, contentColumn, embeddingColumn))

	if err != nil {
		return fmt.Errorf("failed to insert vectors: %w", err)
	}

	// 刷新 collection 以确保数据持久化
	flushTask, err := cli.Flush(ctx, milvusclient.NewFlushOption(collectionName))
	if err != nil {
		return fmt.Errorf("failed to flush: %w", err)
	}

	err = flushTask.Await(ctx)
	if err != nil {
		return fmt.Errorf("failed to wait for flush: %w", err)
	}

	return nil
}

// Search 向量搜索
func (s *MilvusVectorDBService) Search(ctx context.Context, collectionName string, vector []float32, topK int) ([]*biz.SearchResult, error) {
	return s.SearchWithThreshold(ctx, collectionName, vector, topK, 0.0)
}

// SearchWithThreshold 向量搜索（带阈值过滤）
func (s *MilvusVectorDBService) SearchWithThreshold(ctx context.Context, collectionName string, vector []float32, topK int, minScore float32) ([]*biz.SearchResult, error) {
	cli := s.client.GetClient()
	if cli == nil {
		return nil, fmt.Errorf("milvus client is not available")
	}

	// 执行搜索
	searchResult, err := cli.Search(ctx, milvusclient.NewSearchOption(
		collectionName,
		topK,
		[]entity.Vector{entity.FloatVector(vector)},
	).WithOutputFields("document_id", "chunk_id", "content"))

	if err != nil {
		return nil, fmt.Errorf("failed to search: %w", err)
	}

	// 解析结果并应用阈值过滤
	var results []*biz.SearchResult
	for _, resultSet := range searchResult {
		docIDs := resultSet.GetColumn("document_id")
		chunkIDs := resultSet.GetColumn("chunk_id")
		contents := resultSet.GetColumn("content")

		for i := 0; i < resultSet.ResultCount; i++ {
			score := resultSet.Scores[i]

			// 应用最小分数过滤（COSINE 相似度：0-1，越高越相似）
			if minScore > 0 && score < minScore {
				continue
			}

			documentID, _ := docIDs.GetAsString(i)
			chunkID, _ := chunkIDs.GetAsString(i)
			content, _ := contents.GetAsString(i)

			results = append(results, &biz.SearchResult{
				ChunkID:    chunkID,
				DocumentID: documentID,
				Content:    content,
				Score:      score,
			})
		}
	}

	return results, nil
}

// DeleteByDocumentID 根据文档 ID 删除向量（完整实现）
func (s *MilvusVectorDBService) DeleteByDocumentID(ctx context.Context, collectionName, documentID string) error {
	cli := s.client.GetClient()
	if cli == nil {
		return fmt.Errorf("milvus client is not available")
	}

	// 使用表达式删除所有匹配的向量
	expr := fmt.Sprintf("document_id == '%s'", documentID)
	deleteOpt := milvusclient.NewDeleteOption(collectionName)
	deleteOpt.WithExpr(expr)
	
	_, err := cli.Delete(ctx, deleteOpt)
	if err != nil {
		return fmt.Errorf("failed to delete by document_id: %w", err)
	}

	// 刷新以确保删除立即生效
	flushTask, err := cli.Flush(ctx, milvusclient.NewFlushOption(collectionName))
	if err != nil {
		return fmt.Errorf("failed to flush after delete: %w", err)
	}

	err = flushTask.Await(ctx)
	if err != nil {
		return fmt.Errorf("failed to wait for flush after delete: %w", err)
	}

	return nil
}

// DropCollection 删除 collection
func (s *MilvusVectorDBService) DropCollection(ctx context.Context, collectionName string) error {
	cli := s.client.GetClient()
	if cli == nil {
		return fmt.Errorf("milvus client is not available")
	}

	err := cli.DropCollection(ctx, milvusclient.NewDropCollectionOption(collectionName))
	if err != nil {
		return fmt.Errorf("failed to drop collection: %w", err)
	}

	return nil
}
