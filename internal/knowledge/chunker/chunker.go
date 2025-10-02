package chunker

import (
	"context"
)

// Chunker 文本分块接口
type Chunker interface {
	// Chunk 将文本分块
	Chunk(ctx context.Context, text string) ([]*TextChunk, error)

	// ChunkSize 返回分块大小
	ChunkSize() int

	// ChunkOverlap 返回分块重叠大小
	ChunkOverlap() int
}

// TextChunk 文本分块
type TextChunk struct {
	Index      int    // 块序号（从 0 开始）
	Content    string // 块内容
	TokenCount int    // Token 数量
	Start      int    // 在原文中的起始位置
	End        int    // 在原文中的结束位置
}

// ChunkConfig 分块配置
type ChunkConfig struct {
	Size     int // 分块大小
	Overlap  int // 重叠大小
}
