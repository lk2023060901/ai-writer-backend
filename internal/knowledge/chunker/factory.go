package chunker

import (
	"fmt"

	kbtypes "github.com/lk2023060901/ai-writer-backend/internal/knowledge/types"
)

// Factory Chunker 工厂
type Factory struct{}

// NewFactory 创建 Chunker 工厂
func NewFactory() *Factory {
	return &Factory{}
}

// CreateChunkerConfig 创建 Chunker 配置
type CreateChunkerConfig struct {
	Strategy   kbtypes.ChunkStrategy
	Size       int
	Overlap    int
	Encoding   string
	Separators []string
}

// CreateChunker 创建 Chunker
func (f *Factory) CreateChunker(cfg *CreateChunkerConfig) (Chunker, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config is required")
	}

	switch cfg.Strategy {
	case kbtypes.ChunkStrategyToken:
		return NewTokenChunker(&TokenChunkerConfig{
			Size:     cfg.Size,
			Overlap:  cfg.Overlap,
			Encoding: cfg.Encoding,
		})

	case kbtypes.ChunkStrategyRecursive:
		return NewRecursiveChunker(&RecursiveChunkerConfig{
			Size:       cfg.Size,
			Overlap:    cfg.Overlap,
			Encoding:   cfg.Encoding,
			Separators: cfg.Separators,
		})

	default:
		return nil, fmt.Errorf("unsupported chunk strategy: %s", cfg.Strategy)
	}
}
