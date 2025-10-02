package chunker

import (
	"context"
	"fmt"

	"github.com/pkoukk/tiktoken-go"
)

// TokenChunker 基于 Token 的分块器
type TokenChunker struct {
	encoding *tiktoken.Tiktoken
	size     int
	overlap  int
}

// TokenChunkerConfig Token 分块器配置
type TokenChunkerConfig struct {
	Size     int    // 每块的 token 数量
	Overlap  int    // 重叠的 token 数量
	Encoding string // 编码方式（默认 cl100k_base，用于 OpenAI）
}

// NewTokenChunker 创建 Token 分块器
func NewTokenChunker(cfg *TokenChunkerConfig) (*TokenChunker, error) {
	if cfg == nil {
		cfg = &TokenChunkerConfig{
			Size:     512,
			Overlap:  50,
			Encoding: "cl100k_base",
		}
	}

	if cfg.Size <= 0 {
		return nil, fmt.Errorf("chunk size must be positive")
	}

	if cfg.Overlap < 0 {
		return nil, fmt.Errorf("chunk overlap cannot be negative")
	}

	if cfg.Overlap >= cfg.Size {
		return nil, fmt.Errorf("chunk overlap must be less than chunk size")
	}

	if cfg.Encoding == "" {
		cfg.Encoding = "cl100k_base"
	}

	// 创建 tiktoken 编码器
	encoding, err := tiktoken.GetEncoding(cfg.Encoding)
	if err != nil {
		return nil, fmt.Errorf("failed to get encoding: %w", err)
	}

	return &TokenChunker{
		encoding: encoding,
		size:     cfg.Size,
		overlap:  cfg.Overlap,
	}, nil
}

// Chunk 将文本分块
func (c *TokenChunker) Chunk(ctx context.Context, text string) ([]*TextChunk, error) {
	if text == "" {
		return []*TextChunk{}, nil
	}

	// 对整个文本进行编码
	tokens := c.encoding.Encode(text, nil, nil)
	totalTokens := len(tokens)

	if totalTokens == 0 {
		return []*TextChunk{}, nil
	}

	chunks := make([]*TextChunk, 0)
	chunkIndex := 0
	start := 0

	for start < totalTokens {
		// 计算当前块的结束位置
		end := start + c.size
		if end > totalTokens {
			end = totalTokens
		}

		// 提取当前块的 tokens
		chunkTokens := tokens[start:end]

		// 解码为文本
		chunkText := c.encoding.Decode(chunkTokens)

		// 计算在原文中的位置（近似）
		textStart := 0
		textEnd := len(text)

		if start > 0 {
			// 解码之前的所有 tokens 来找到起始位置
			beforeText := c.encoding.Decode(tokens[:start])
			textStart = len(beforeText)
		}

		if end < totalTokens {
			// 解码到当前位置的所有 tokens 来找到结束位置
			beforeAndCurrentText := c.encoding.Decode(tokens[:end])
			textEnd = len(beforeAndCurrentText)
		}

		chunks = append(chunks, &TextChunk{
			Index:      chunkIndex,
			Content:    chunkText,
			TokenCount: len(chunkTokens),
			Start:      textStart,
			End:        textEnd,
		})

		chunkIndex++

		// 移动到下一个块的起始位置（考虑重叠）
		start += c.size - c.overlap

		// 避免无限循环
		if c.size-c.overlap <= 0 {
			break
		}
	}

	return chunks, nil
}

// ChunkSize 返回分块大小
func (c *TokenChunker) ChunkSize() int {
	return c.size
}

// ChunkOverlap 返回分块重叠大小
func (c *TokenChunker) ChunkOverlap() int {
	return c.overlap
}

// Close 关闭分块器
func (c *TokenChunker) Close() error {
	// tiktoken-go 不需要手动释放
	return nil
}
