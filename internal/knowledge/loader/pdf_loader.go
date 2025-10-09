package loader

import (
	"context"
	"fmt"
	"io"
	"strings"

	kbtypes "github.com/lk2023060901/ai-writer-backend/internal/knowledge/types"
	"github.com/gen2brain/go-fitz"
)

// PDFLoader PDF 加载器
type PDFLoader struct{}

// NewPDFLoader 创建 PDF 加载器
func NewPDFLoader() *PDFLoader {
	return &PDFLoader{}
}

// Load 加载 PDF 内容（使用 go-fitz/MuPDF）
func (l *PDFLoader) Load(ctx context.Context, reader io.Reader) (*Document, error) {
	// 将 reader 内容读入内存
	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to read PDF data: %w", err)
	}

	// 从内存打开 PDF 文档
	doc, err := fitz.NewFromMemory(data)
	if err != nil {
		return nil, fmt.Errorf("failed to open PDF: %w", err)
	}
	defer doc.Close()

	// 提取所有页面的文本
	var textBuilder strings.Builder
	numPages := doc.NumPage()

	for i := 0; i < numPages; i++ {
		// 提取页面文本
		text, err := doc.Text(i)
		if err != nil {
			// 跳过无法提取的页面
			continue
		}

		textBuilder.WriteString(text)
		textBuilder.WriteString("\n\n") // 页面之间添加分隔
	}

	return &Document{
		Content: textBuilder.String(),
		Metadata: map[string]interface{}{
			"loader":     "pdf",
			"page_count": numPages,
		},
	}, nil
}

// SupportedTypes 返回支持的文件类型
func (l *PDFLoader) SupportedTypes() []kbtypes.FileType {
	return []kbtypes.FileType{
		kbtypes.FileTypePdf,
	}
}
