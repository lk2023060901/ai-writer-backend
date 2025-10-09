package processor

import (
	"context"
	"fmt"
	"strings"

	"github.com/gen2brain/go-fitz"
	"github.com/tidwall/gjson"
	"github.com/yuin/goldmark"
	goldmarktext "github.com/yuin/goldmark/text"
)

// func init() {
// 	// 设置 UniOffice 许可证密钥（已禁用）
// 	err := license.SetMeteredKey("c1609bf36881094add1da9ca73148904a289319d80e190b55c99687c84143e1c")
// 	if err != nil {
// 		panic(fmt.Sprintf("failed to set unioffice license: %v", err))
// 	}
// }

// DocumentProcessor 文档处理器实现
type DocumentProcessor struct{}

// NewDocumentProcessor 创建文档处理器
func NewDocumentProcessor() *DocumentProcessor {
	return &DocumentProcessor{}
}

// ExtractText 从文件中提取文本内容
func (p *DocumentProcessor) ExtractText(ctx context.Context, fileData []byte, fileType string) (string, error) {
	switch strings.ToLower(fileType) {
	case "pdf":
		return p.extractPDF(fileData)
	// case "docx":
	// 	return p.extractDOCX(fileData)
	case "txt":
		return string(fileData), nil
	case "md":
		return p.extractMarkdown(fileData)
	case "json":
		return p.extractJSON(fileData)
	default:
		return "", fmt.Errorf("unsupported file type: %s", fileType)
	}
}

// extractPDF 提取 PDF 文本（使用 go-fitz/MuPDF）
func (p *DocumentProcessor) extractPDF(fileData []byte) (string, error) {
	// 从内存打开 PDF 文档
	doc, err := fitz.NewFromMemory(fileData)
	if err != nil {
		return "", fmt.Errorf("failed to open PDF: %w", err)
	}
	defer doc.Close()

	var textBuilder strings.Builder
	numPages := doc.NumPage()

	// 提取每一页的文本
	for i := 0; i < numPages; i++ {
		text, err := doc.Text(i)
		if err != nil {
			// 跳过无法提取的页面
			continue
		}

		textBuilder.WriteString(text)
		textBuilder.WriteString("\n\n")
	}

	return textBuilder.String(), nil
}

// extractDOCX 提取 DOCX 文本（已禁用 UniOffice，仅支持 MinerU）
// func (p *DocumentProcessor) extractDOCX(fileData []byte) (string, error) {
// 	// 打开 DOCX 文档
// 	doc, err := document.Read(bytes.NewReader(fileData), int64(len(fileData)))
// 	if err != nil {
// 		return "", fmt.Errorf("failed to open DOCX document: %w", err)
// 	}
// 	defer doc.Close()
//
// 	// 提取所有段落的文本
// 	var textBuilder strings.Builder
// 	for _, para := range doc.Paragraphs() {
// 		for _, run := range para.Runs() {
// 			textBuilder.WriteString(run.Text())
// 		}
// 		textBuilder.WriteString("\n")
// 	}
//
// 	// 提取表格内容
// 	for _, table := range doc.Tables() {
// 		for _, row := range table.Rows() {
// 			for _, cell := range row.Cells() {
// 				for _, para := range cell.Paragraphs() {
// 					for _, run := range para.Runs() {
// 						textBuilder.WriteString(run.Text())
// 					}
// 					textBuilder.WriteString("\t")
// 				}
// 			}
// 			textBuilder.WriteString("\n")
// 		}
// 		textBuilder.WriteString("\n")
// 	}
//
// 	return textBuilder.String(), nil
// }

// extractMarkdown 提取 Markdown 文本（使用 goldmark 解析）
func (p *DocumentProcessor) extractMarkdown(fileData []byte) (string, error) {
	// Goldmark 用于解析和验证 Markdown
	md := goldmark.New()
	reader := goldmarktext.NewReader(fileData)
	_ = md.Parser().Parse(reader) // 验证格式

	// 直接返回原始 Markdown 内容（保留格式更有助于 RAG 理解上下文）
	return string(fileData), nil
}

// extractJSON 提取 JSON 内容（使用 gjson 美化）
func (p *DocumentProcessor) extractJSON(fileData []byte) (string, error) {
	// 验证 JSON 格式
	if !gjson.ValidBytes(fileData) {
		return "", fmt.Errorf("invalid JSON format")
	}

	// 解析 JSON 并格式化
	result := gjson.ParseBytes(fileData)

	// 将 JSON 转换为可读文本
	var textBuilder strings.Builder
	var extractValues func(key string, value gjson.Result, depth int)
	extractValues = func(key string, value gjson.Result, depth int) {
		indent := strings.Repeat("  ", depth)

		switch value.Type {
		case gjson.String:
			textBuilder.WriteString(fmt.Sprintf("%s%s: %s\n", indent, key, value.String()))
		case gjson.Number:
			textBuilder.WriteString(fmt.Sprintf("%s%s: %v\n", indent, key, value.Num))
		case gjson.True, gjson.False:
			textBuilder.WriteString(fmt.Sprintf("%s%s: %v\n", indent, key, value.Bool()))
		case gjson.JSON:
			if value.IsArray() {
				textBuilder.WriteString(fmt.Sprintf("%s%s: [\n", indent, key))
				for i, item := range value.Array() {
					extractValues(fmt.Sprintf("[%d]", i), item, depth+1)
				}
				textBuilder.WriteString(fmt.Sprintf("%s]\n", indent))
			} else if value.IsObject() {
				textBuilder.WriteString(fmt.Sprintf("%s%s: {\n", indent, key))
				value.ForEach(func(k, v gjson.Result) bool {
					extractValues(k.String(), v, depth+1)
					return true
				})
				textBuilder.WriteString(fmt.Sprintf("%s}\n", indent))
			}
		}
	}

	// 处理顶层
	if result.IsArray() {
		for i, item := range result.Array() {
			extractValues(fmt.Sprintf("Item %d", i), item, 0)
		}
	} else if result.IsObject() {
		result.ForEach(func(key, value gjson.Result) bool {
			extractValues(key.String(), value, 0)
			return true
		})
	} else {
		// 简单值，直接返回
		return result.String(), nil
	}

	return textBuilder.String(), nil
}
