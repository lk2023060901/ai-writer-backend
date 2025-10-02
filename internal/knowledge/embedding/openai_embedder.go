package embedding

import (
	"context"
	"fmt"

	kbtypes "github.com/lk2023060901/ai-writer-backend/internal/knowledge/types"
	"github.com/lk2023060901/ai-writer-backend/internal/pkg/logger"
	"github.com/sashabaranov/go-openai"
	"go.uber.org/zap"
)

// OpenAIEmbedder OpenAI Embedder 实现
type OpenAIEmbedder struct {
	client    *openai.Client
	model     string
	dimension int
	logger    *logger.Logger
}

// OpenAIEmbedderConfig OpenAI Embedder 配置
type OpenAIEmbedderConfig struct {
	APIKey    string
	BaseURL   string
	Model     string
	Dimension int
}

// NewOpenAIEmbedder 创建 OpenAI Embedder
func NewOpenAIEmbedder(cfg *OpenAIEmbedderConfig, lgr *logger.Logger) (*OpenAIEmbedder, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config is required")
	}

	if cfg.APIKey == "" {
		return nil, fmt.Errorf("api key is required")
	}

	if cfg.Model == "" {
		cfg.Model = string(openai.SmallEmbedding3) // text-embedding-3-small
	}

	if cfg.Dimension == 0 {
		cfg.Dimension = 1536 // 默认维度
	}

	var log *logger.Logger
	if lgr == nil {
		log = logger.L()
	} else {
		log = lgr
	}

	// 创建 OpenAI 客户端配置
	clientCfg := openai.DefaultConfig(cfg.APIKey)
	if cfg.BaseURL != "" {
		clientCfg.BaseURL = cfg.BaseURL
	}

	client := openai.NewClientWithConfig(clientCfg)

	log.Info("openai embedder created",
		zap.String("model", cfg.Model),
		zap.Int("dimension", cfg.Dimension))

	return &OpenAIEmbedder{
		client:    client,
		model:     cfg.Model,
		dimension: cfg.Dimension,
		logger:    log,
	}, nil
}

// Embed 对单个文本生成向量
func (e *OpenAIEmbedder) Embed(ctx context.Context, text string) ([]float32, error) {
	embeddings, err := e.BatchEmbed(ctx, []string{text})
	if err != nil {
		return nil, err
	}

	if len(embeddings) == 0 {
		return nil, fmt.Errorf("no embedding generated")
	}

	return embeddings[0], nil
}

// BatchEmbed 批量生成向量
func (e *OpenAIEmbedder) BatchEmbed(ctx context.Context, texts []string) ([][]float32, error) {
	if len(texts) == 0 {
		return [][]float32{}, nil
	}

	// 创建请求
	req := openai.EmbeddingRequestStrings{
		Input:      texts,
		Model:      openai.EmbeddingModel(e.model),
		Dimensions: e.dimension,
	}

	// 调用 API
	resp, err := e.client.CreateEmbeddings(ctx, req)
	if err != nil {
		e.logger.Error("failed to create embeddings",
			zap.Error(err),
			zap.Int("text_count", len(texts)))
		return nil, fmt.Errorf("failed to create embeddings: %w", err)
	}

	// 提取向量
	embeddings := make([][]float32, len(resp.Data))
	for i, data := range resp.Data {
		embeddings[i] = data.Embedding
	}

	e.logger.Info("embeddings created successfully",
		zap.Int("count", len(embeddings)),
		zap.Int("prompt_tokens", resp.Usage.PromptTokens),
		zap.Int("total_tokens", resp.Usage.TotalTokens))

	return embeddings, nil
}

// Dimension 返回向量维度
func (e *OpenAIEmbedder) Dimension() int {
	return e.dimension
}

// Provider 返回 Provider 名称
func (e *OpenAIEmbedder) Provider() kbtypes.EmbeddingProvider {
	return kbtypes.EmbeddingProviderOpenAI
}

// Model 返回模型名称
func (e *OpenAIEmbedder) Model() string {
	return e.model
}
