package embedding

import (
	"context"
	"fmt"

	"github.com/lk2023060901/ai-writer-backend/internal/knowledge/biz"
	"github.com/sashabaranov/go-openai"
)

// EmbeddingService Embedding 生成服务
type EmbeddingService struct{}

// NewEmbeddingService 创建 Embedding 服务
func NewEmbeddingService() *EmbeddingService {
	return &EmbeddingService{}
}

// GenerateEmbeddings 批量生成 Embeddings
func (s *EmbeddingService) GenerateEmbeddings(ctx context.Context, texts []string, config *biz.AIProviderConfig) ([][]float32, error) {
	if len(texts) == 0 {
		return nil, fmt.Errorf("no texts to embed")
	}

	// 创建 OpenAI 客户端
	clientConfig := openai.DefaultConfig(config.APIKey)
	if config.APIBaseURL != "" {
		clientConfig.BaseURL = config.APIBaseURL
	}
	client := openai.NewClientWithConfig(clientConfig)

	// 批量处理（OpenAI API 限制单次请求数量）
	batchSize := 100
	var allEmbeddings [][]float32

	for i := 0; i < len(texts); i += batchSize {
		end := i + batchSize
		if end > len(texts) {
			end = len(texts)
		}

		batch := texts[i:end]
		req := openai.EmbeddingRequest{
			Input: batch,
			Model: openai.EmbeddingModel(config.EmbeddingModel),
		}

		resp, err := client.CreateEmbeddings(ctx, req)
		if err != nil {
			return nil, fmt.Errorf("failed to create embeddings: %w", err)
		}

		// 提取 embedding 向量
		for _, data := range resp.Data {
			embedding := make([]float32, len(data.Embedding))
			for j, val := range data.Embedding {
				embedding[j] = float32(val)
			}
			allEmbeddings = append(allEmbeddings, embedding)
		}
	}

	return allEmbeddings, nil
}
