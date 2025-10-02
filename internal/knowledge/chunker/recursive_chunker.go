package chunker

import (
	"context"
	"fmt"
	"strings"

	"github.com/pkoukk/tiktoken-go"
)

// RecursiveChunker 递归分块器（按分隔符递归分割）
type RecursiveChunker struct {
	encoding   *tiktoken.Tiktoken
	size       int
	overlap    int
	separators []string
}

// RecursiveChunkerConfig 递归分块器配置
type RecursiveChunkerConfig struct {
	Size       int      // 每块的 token 数量
	Overlap    int      // 重叠的 token 数量
	Encoding   string   // 编码方式
	Separators []string // 分隔符列表（按优先级）
}

// NewRecursiveChunker 创建递归分块器
func NewRecursiveChunker(cfg *RecursiveChunkerConfig) (*RecursiveChunker, error) {
	if cfg == nil {
		cfg = &RecursiveChunkerConfig{
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

	// 默认分隔符（按优先级从高到低）
	if len(cfg.Separators) == 0 {
		cfg.Separators = []string{
			"\n\n",   // 段落
			"\n",     // 换行
			". ",     // 句子
			"! ",     // 感叹句
			"? ",     // 疑问句
			"; ",     // 分号
			", ",     // 逗号
			" ",      // 空格
			"",       // 字符
		}
	}

	// 创建 tiktoken 编码器
	encoding, err := tiktoken.GetEncoding(cfg.Encoding)
	if err != nil {
		return nil, fmt.Errorf("failed to get encoding: %w", err)
	}

	return &RecursiveChunker{
		encoding:   encoding,
		size:       cfg.Size,
		overlap:    cfg.Overlap,
		separators: cfg.Separators,
	}, nil
}

// Chunk 将文本分块
func (c *RecursiveChunker) Chunk(ctx context.Context, text string) ([]*TextChunk, error) {
	if text == "" {
		return []*TextChunk{}, nil
	}

	// 递归分割文本
	splits := c.splitText(text, c.separators)

	// 合并分割结果为块
	chunks := c.mergeChunks(splits)

	return chunks, nil
}

// splitText 递归分割文本
func (c *RecursiveChunker) splitText(text string, separators []string) []string {
	if len(separators) == 0 {
		return []string{text}
	}

	separator := separators[0]
	remainingSeparators := separators[1:]

	var splits []string

	if separator == "" {
		// 按字符分割
		for _, char := range text {
			splits = append(splits, string(char))
		}
	} else {
		// 按分隔符分割
		parts := strings.Split(text, separator)
		for i, part := range parts {
			if part != "" {
				splits = append(splits, part)
			}
			// 保留分隔符（除了最后一个）
			if i < len(parts)-1 && separator != "" {
				splits = append(splits, separator)
			}
		}
	}

	// 检查每个分割是否需要继续递归分割
	var finalSplits []string
	for _, split := range splits {
		tokens := c.encoding.Encode(split, nil, nil)
		if len(tokens) > c.size && len(remainingSeparators) > 0 {
			// 需要继续分割
			subSplits := c.splitText(split, remainingSeparators)
			finalSplits = append(finalSplits, subSplits...)
		} else {
			finalSplits = append(finalSplits, split)
		}
	}

	return finalSplits
}

// mergeChunks 合并分割结果为块
func (c *RecursiveChunker) mergeChunks(splits []string) []*TextChunk {
	chunks := make([]*TextChunk, 0)
	currentChunk := ""
	currentTokens := 0
	chunkIndex := 0
	textPosition := 0

	for _, split := range splits {
		splitTokens := len(c.encoding.Encode(split, nil, nil))

		// 如果当前分割本身就超过块大小，单独成块
		if splitTokens > c.size {
			// 先保存当前块（如果有内容）
			if currentChunk != "" {
				chunks = append(chunks, &TextChunk{
					Index:      chunkIndex,
					Content:    currentChunk,
					TokenCount: currentTokens,
					Start:      textPosition - len(currentChunk),
					End:        textPosition,
				})
				chunkIndex++
			}

			// 大分割单独成块
			chunks = append(chunks, &TextChunk{
				Index:      chunkIndex,
				Content:    split,
				TokenCount: splitTokens,
				Start:      textPosition,
				End:        textPosition + len(split),
			})
			chunkIndex++
			textPosition += len(split)
			currentChunk = ""
			currentTokens = 0
			continue
		}

		// 检查是否会超过块大小
		if currentTokens+splitTokens > c.size && currentChunk != "" {
			// 保存当前块
			chunks = append(chunks, &TextChunk{
				Index:      chunkIndex,
				Content:    currentChunk,
				TokenCount: currentTokens,
				Start:      textPosition - len(currentChunk),
				End:        textPosition,
			})
			chunkIndex++

			// 处理重叠
			if c.overlap > 0 {
				overlapText := c.getOverlapText(currentChunk, c.overlap)
				currentChunk = overlapText + split
				currentTokens = len(c.encoding.Encode(currentChunk, nil, nil))
			} else {
				currentChunk = split
				currentTokens = splitTokens
			}
		} else {
			// 添加到当前块
			currentChunk += split
			currentTokens += splitTokens
		}

		textPosition += len(split)
	}

	// 保存最后一个块
	if currentChunk != "" {
		chunks = append(chunks, &TextChunk{
			Index:      chunkIndex,
			Content:    currentChunk,
			TokenCount: currentTokens,
			Start:      textPosition - len(currentChunk),
			End:        textPosition,
		})
	}

	return chunks
}

// getOverlapText 获取重叠文本
func (c *RecursiveChunker) getOverlapText(text string, overlapTokens int) string {
	tokens := c.encoding.Encode(text, nil, nil)
	if len(tokens) <= overlapTokens {
		return text
	}

	// 取最后 overlapTokens 个 tokens
	overlapTokenSlice := tokens[len(tokens)-overlapTokens:]
	return c.encoding.Decode(overlapTokenSlice)
}

// ChunkSize 返回分块大小
func (c *RecursiveChunker) ChunkSize() int {
	return c.size
}

// ChunkOverlap 返回分块重叠大小
func (c *RecursiveChunker) ChunkOverlap() int {
	return c.overlap
}

// Close 关闭分块器
func (c *RecursiveChunker) Close() error {
	// tiktoken-go 不需要手动释放
	return nil
}
