package processor

import (
	"fmt"
	"strings"
	"unicode"

	"github.com/pkoukk/tiktoken-go"
)

// ChunkText 将文本分块
func (p *DocumentProcessor) ChunkText(text string, chunkSize, chunkOverlap int, strategy string) ([]string, error) {
	switch strategy {
	case "recursive":
		return p.recursiveChunk(text, chunkSize, chunkOverlap)
	case "fixed":
		return p.fixedChunk(text, chunkSize, chunkOverlap)
	default:
		return p.recursiveChunk(text, chunkSize, chunkOverlap)
	}
}

// recursiveChunk 递归分块（按段落、句子）
func (p *DocumentProcessor) recursiveChunk(text string, chunkSize, chunkOverlap int) ([]string, error) {
	// 初始化 tiktoken 编码器
	encoding, err := tiktoken.GetEncoding("cl100k_base")
	if err != nil {
		return nil, fmt.Errorf("failed to get tiktoken encoding: %w", err)
	}

	// 按段落分割
	paragraphs := strings.Split(text, "\n\n")
	var chunks []string
	var currentChunk strings.Builder
	var currentTokens int

	for _, para := range paragraphs {
		para = strings.TrimSpace(para)
		if para == "" {
			continue
		}

		paraTokens := len(encoding.Encode(para, nil, nil))

		// 如果当前段落加上现有内容超过了 chunk 大小
		if currentTokens+paraTokens > chunkSize && currentChunk.Len() > 0 {
			// 保存当前 chunk
			chunks = append(chunks, currentChunk.String())

			// 处理 overlap
			if chunkOverlap > 0 {
				overlapText := p.getOverlapText(currentChunk.String(), chunkOverlap, encoding)
				currentChunk.Reset()
				currentChunk.WriteString(overlapText)
				currentTokens = len(encoding.Encode(overlapText, nil, nil))
			} else {
				currentChunk.Reset()
				currentTokens = 0
			}
		}

		// 如果单个段落就超过了 chunk 大小，需要按句子分割
		if paraTokens > chunkSize {
			sentences := p.splitSentences(para)
			for _, sentence := range sentences {
				sentTokens := len(encoding.Encode(sentence, nil, nil))
				if currentTokens+sentTokens > chunkSize && currentChunk.Len() > 0 {
					chunks = append(chunks, currentChunk.String())
					if chunkOverlap > 0 {
						overlapText := p.getOverlapText(currentChunk.String(), chunkOverlap, encoding)
						currentChunk.Reset()
						currentChunk.WriteString(overlapText)
						currentTokens = len(encoding.Encode(overlapText, nil, nil))
					} else {
						currentChunk.Reset()
						currentTokens = 0
					}
				}
				if currentChunk.Len() > 0 {
					currentChunk.WriteString(" ")
				}
				currentChunk.WriteString(sentence)
				currentTokens += sentTokens
			}
		} else {
			// 添加段落
			if currentChunk.Len() > 0 {
				currentChunk.WriteString("\n\n")
			}
			currentChunk.WriteString(para)
			currentTokens += paraTokens
		}
	}

	// 添加最后一个 chunk
	if currentChunk.Len() > 0 {
		chunks = append(chunks, currentChunk.String())
	}

	return chunks, nil
}

// fixedChunk 固定大小分块
func (p *DocumentProcessor) fixedChunk(text string, chunkSize, chunkOverlap int) ([]string, error) {
	encoding, err := tiktoken.GetEncoding("cl100k_base")
	if err != nil {
		return nil, fmt.Errorf("failed to get tiktoken encoding: %w", err)
	}

	tokens := encoding.Encode(text, nil, nil)
	var chunks []string

	for i := 0; i < len(tokens); i += chunkSize - chunkOverlap {
		end := i + chunkSize
		if end > len(tokens) {
			end = len(tokens)
		}

		chunkTokens := tokens[i:end]
		chunkText := encoding.Decode(chunkTokens)
		chunks = append(chunks, chunkText)

		if end >= len(tokens) {
			break
		}
	}

	return chunks, nil
}

// splitSentences 分割句子
func (p *DocumentProcessor) splitSentences(text string) []string {
	var sentences []string
	var currentSentence strings.Builder

	runes := []rune(text)
	for i, r := range runes {
		currentSentence.WriteRune(r)

		// 句子结束符
		if r == '。' || r == '！' || r == '？' || r == '.' || r == '!' || r == '?' {
			// 检查下一个字符是否是空格或结束
			if i+1 >= len(runes) || unicode.IsSpace(runes[i+1]) {
				sentences = append(sentences, strings.TrimSpace(currentSentence.String()))
				currentSentence.Reset()
			}
		}
	}

	// 添加剩余内容
	if currentSentence.Len() > 0 {
		sentences = append(sentences, strings.TrimSpace(currentSentence.String()))
	}

	return sentences
}

// getOverlapText 获取重叠文本
func (p *DocumentProcessor) getOverlapText(text string, overlapTokens int, encoding *tiktoken.Tiktoken) string {
	tokens := encoding.Encode(text, nil, nil)
	if len(tokens) <= overlapTokens {
		return text
	}

	overlapTokenSlice := tokens[len(tokens)-overlapTokens:]
	return encoding.Decode(overlapTokenSlice)
}
