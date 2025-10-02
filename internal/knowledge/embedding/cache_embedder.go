package embedding

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	kbtypes "github.com/lk2023060901/ai-writer-backend/internal/knowledge/types"
	"github.com/lk2023060901/ai-writer-backend/internal/pkg/logger"
	"github.com/lk2023060901/ai-writer-backend/internal/pkg/redis"
	"go.uber.org/zap"
)

// CacheEmbedder 带缓存的 Embedder 装饰器
type CacheEmbedder struct {
	embedder Embedder
	cache    *redis.Client
	ttl      time.Duration
	prefix   string
	logger   *logger.Logger
}

// CacheEmbedderConfig 缓存配置
type CacheEmbedderConfig struct {
	TTL    time.Duration // 缓存过期时间
	Prefix string        // 缓存键前缀
}

// NewCacheEmbedder 创建带缓存的 Embedder
func NewCacheEmbedder(embedder Embedder, cache *redis.Client, cfg *CacheEmbedderConfig, lgr *logger.Logger) *CacheEmbedder {
	if cfg == nil {
		cfg = &CacheEmbedderConfig{
			TTL:    24 * time.Hour,
			Prefix: "kb:embedding:",
		}
	}

	if cfg.TTL == 0 {
		cfg.TTL = 24 * time.Hour
	}

	if cfg.Prefix == "" {
		cfg.Prefix = "kb:embedding:"
	}

	var log *logger.Logger
	if lgr == nil {
		log = logger.L()
	} else {
		log = lgr
	}

	return &CacheEmbedder{
		embedder: embedder,
		cache:    cache,
		ttl:      cfg.TTL,
		prefix:   cfg.Prefix,
		logger:   log,
	}
}

// Embed 对单个文本生成向量（带缓存）
func (e *CacheEmbedder) Embed(ctx context.Context, text string) ([]float32, error) {
	// 生成缓存键
	cacheKey := e.cacheKey(text)

	// 尝试从缓存获取
	if e.cache != nil {
		if cached, err := e.getFromCache(ctx, cacheKey); err == nil {
			e.logger.Debug("embedding cache hit",
				zap.String("cache_key", cacheKey))
			return cached, nil
		}
	}

	// 缓存未命中，调用底层 Embedder
	embedding, err := e.embedder.Embed(ctx, text)
	if err != nil {
		return nil, err
	}

	// 写入缓存
	if e.cache != nil {
		if err := e.setToCache(ctx, cacheKey, embedding); err != nil {
			e.logger.Warn("failed to cache embedding",
				zap.String("cache_key", cacheKey),
				zap.Error(err))
		}
	}

	return embedding, nil
}

// BatchEmbed 批量生成向量（带缓存）
func (e *CacheEmbedder) BatchEmbed(ctx context.Context, texts []string) ([][]float32, error) {
	if len(texts) == 0 {
		return [][]float32{}, nil
	}

	results := make([][]float32, len(texts))
	missingIndices := make([]int, 0)
	missingTexts := make([]string, 0)

	// 检查缓存
	if e.cache != nil {
		for i, text := range texts {
			cacheKey := e.cacheKey(text)
			if cached, err := e.getFromCache(ctx, cacheKey); err == nil {
				results[i] = cached
			} else {
				missingIndices = append(missingIndices, i)
				missingTexts = append(missingTexts, text)
			}
		}

		e.logger.Debug("batch embedding cache stats",
			zap.Int("total", len(texts)),
			zap.Int("cache_hits", len(texts)-len(missingTexts)),
			zap.Int("cache_misses", len(missingTexts)))
	} else {
		missingIndices = make([]int, len(texts))
		missingTexts = texts
		for i := range texts {
			missingIndices[i] = i
		}
	}

	// 如果所有都在缓存中，直接返回
	if len(missingTexts) == 0 {
		return results, nil
	}

	// 调用底层 Embedder 处理未命中的文本
	embeddings, err := e.embedder.BatchEmbed(ctx, missingTexts)
	if err != nil {
		return nil, err
	}

	// 填充结果并缓存
	for i, embedding := range embeddings {
		idx := missingIndices[i]
		results[idx] = embedding

		// 写入缓存
		if e.cache != nil {
			cacheKey := e.cacheKey(missingTexts[i])
			if err := e.setToCache(ctx, cacheKey, embedding); err != nil {
				e.logger.Warn("failed to cache embedding",
					zap.String("cache_key", cacheKey),
					zap.Error(err))
			}
		}
	}

	return results, nil
}

// Dimension 返回向量维度
func (e *CacheEmbedder) Dimension() int {
	return e.embedder.Dimension()
}

// Provider 返回 Provider 名称
func (e *CacheEmbedder) Provider() kbtypes.EmbeddingProvider {
	return e.embedder.Provider()
}

// Model 返回模型名称
func (e *CacheEmbedder) Model() string {
	return e.embedder.Model()
}

// cacheKey 生成缓存键
func (e *CacheEmbedder) cacheKey(text string) string {
	// 使用模型名 + 文本 hash 作为缓存键
	hash := sha256.Sum256([]byte(text))
	hashStr := hex.EncodeToString(hash[:])
	return fmt.Sprintf("%s%s:%s", e.prefix, e.Model(), hashStr)
}

// getFromCache 从缓存获取向量
func (e *CacheEmbedder) getFromCache(ctx context.Context, key string) ([]float32, error) {
	data, err := e.cache.Get(ctx, key)
	if err != nil {
		return nil, err
	}

	var embedding []float32
	if err := json.Unmarshal([]byte(data), &embedding); err != nil {
		return nil, fmt.Errorf("failed to unmarshal cached embedding: %w", err)
	}

	return embedding, nil
}

// setToCache 将向量写入缓存
func (e *CacheEmbedder) setToCache(ctx context.Context, key string, embedding []float32) error {
	data, err := json.Marshal(embedding)
	if err != nil {
		return fmt.Errorf("failed to marshal embedding: %w", err)
	}

	if err := e.cache.Set(ctx, key, string(data), e.ttl); err != nil {
		return fmt.Errorf("failed to set cache: %w", err)
	}

	return nil
}
