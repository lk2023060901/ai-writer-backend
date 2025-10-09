package embedding

import (
	"context"
	"fmt"

	"github.com/lk2023060901/ai-writer-backend/internal/knowledge/biz"
	"github.com/lk2023060901/ai-writer-backend/internal/pkg/logger"
	"github.com/sashabaranov/go-openai"
	"go.uber.org/zap"
)

// EmbeddingService Embedding 生成服务
type EmbeddingService struct{}

// NewEmbeddingService 创建 Embedding 服务
func NewEmbeddingService() *EmbeddingService {
	return &EmbeddingService{}
}

// GenerateEmbeddings 批量生成 Embeddings
func (s *EmbeddingService) GenerateEmbeddings(ctx context.Context, texts []string, provider *biz.AIProvider, model *biz.AIModel) ([][]float32, error) {
	// 记录向量化请求
	logger.Info("生成向量嵌入请求",
		zap.String("provider", provider.ProviderType),
		zap.String("model", model.ModelName),
		zap.Int("text_count", len(texts)))

	if len(texts) == 0 {
		return nil, fmt.Errorf("no texts to embed")
	}

	// 从 provider 获取 API Key
	apiKey := provider.APIKey
	if apiKey == "" {
		return nil, fmt.Errorf("API key is empty for provider %s", provider.ProviderType)
	}

	apiBaseURL := ""

	// 根据 provider type 设置不同的 base URL
	switch provider.ProviderType {
	case "siliconflow":
		apiBaseURL = "https://api.siliconflow.cn/v1"
	case "openai":
		apiBaseURL = "https://api.openai.com/v1"
	case "anthropic":
		// Anthropic 不支持 Embedding，应该在验证阶段就拦截
		return nil, fmt.Errorf("anthropic does not support embeddings")
	}

	// 创建 OpenAI 客户端
	clientConfig := openai.DefaultConfig(apiKey)
	if apiBaseURL != "" {
		clientConfig.BaseURL = apiBaseURL
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
			Model: openai.EmbeddingModel(model.ModelName),
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

	// 记录向量化完成
	dimension := 0
	if len(allEmbeddings) > 0 {
		dimension = len(allEmbeddings[0])
	}
	logger.Info("向量嵌入生成完成",
		zap.String("provider", provider.ProviderType),
		zap.String("model", model.ModelName),
		zap.Int("embedding_count", len(allEmbeddings)),
		zap.Int("dimension", dimension))

	return allEmbeddings, nil
}
