package loader

import (
	"context"
	"fmt"
	"io"
	"regexp"
	"strings"

	kbtypes "github.com/lk2023060901/ai-writer-backend/internal/knowledge/types"
	"github.com/russross/blackfriday/v2"
)

// MarkdownLoader Markdown 加载器
type MarkdownLoader struct{}

// NewMarkdownLoader 创建 Markdown 加载器
func NewMarkdownLoader() *MarkdownLoader {
	return &MarkdownLoader{}
}

// Load 加载 Markdown 内容
func (l *MarkdownLoader) Load(ctx context.Context, reader io.Reader) (*Document, error) {
	// 读取所有内容
	content, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to read markdown content: %w", err)
	}

	// 将 Markdown 转换为 HTML
	html := blackfriday.Run(content)

	// 将 HTML 转换为纯文本
	plainText := l.htmlToPlainText(string(html))

	return &Document{
		Content: plainText,
		Metadata: map[string]interface{}{
			"loader":         "markdown",
			"original_format": "markdown",
		},
	}, nil
}

// htmlToPlainText 将 HTML 转换为纯文本
func (l *MarkdownLoader) htmlToPlainText(html string) string {
	// 移除 script 和 style 标签及其内容
	reScript := regexp.MustCompile(`(?i)<script[^>]*>.*?</script>`)
	html = reScript.ReplaceAllString(html, "")
	reStyle := regexp.MustCompile(`(?i)<style[^>]*>.*?</style>`)
	html = reStyle.ReplaceAllString(html, "")

	// 将 <br> 和 </p> 转换为换行
	html = regexp.MustCompile(`(?i)<br\s*/?>|</p>`).ReplaceAllString(html, "\n")

	// 将 </h1> - </h6> 转换为双换行
	html = regexp.MustCompile(`(?i)</h[1-6]>`).ReplaceAllString(html, "\n\n")

	// 将 </li> 转换为换行
	html = regexp.MustCompile(`(?i)</li>`).ReplaceAllString(html, "\n")

	// 移除所有 HTML 标签
	reTag := regexp.MustCompile(`<[^>]+>`)
	text := reTag.ReplaceAllString(html, "")

	// 解码 HTML 实体
	text = l.decodeHTMLEntities(text)

	// 清理多余的空白
	text = l.cleanWhitespace(text)

	return text
}

// decodeHTMLEntities 解码常见的 HTML 实体
func (l *MarkdownLoader) decodeHTMLEntities(text string) string {
	entities := map[string]string{
		"&nbsp;":  " ",
		"&lt;":    "<",
		"&gt;":    ">",
		"&amp;":   "&",
		"&quot;":  "\"",
		"&apos;":  "'",
		"&ndash;": "–",
		"&mdash;": "—",
	}

	for entity, replacement := range entities {
		text = strings.ReplaceAll(text, entity, replacement)
	}

	return text
}

// cleanWhitespace 清理多余的空白字符
func (l *MarkdownLoader) cleanWhitespace(text string) string {
	// 移除行首行尾空白
	lines := strings.Split(text, "\n")
	var cleanedLines []string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" {
			cleanedLines = append(cleanedLines, trimmed)
		}
	}

	// 重新组合，段落之间保留单个换行
	text = strings.Join(cleanedLines, "\n")

	// 将多个连续换行替换为双换行（段落分隔）
	reMultiNewline := regexp.MustCompile(`\n{3,}`)
	text = reMultiNewline.ReplaceAllString(text, "\n\n")

	return strings.TrimSpace(text)
}

// SupportedTypes 返回支持的文件类型
func (l *MarkdownLoader) SupportedTypes() []kbtypes.FileType {
	return []kbtypes.FileType{
		kbtypes.FileTypeMd,
	}
}
