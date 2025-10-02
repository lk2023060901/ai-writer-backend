package loader

import (
	"context"
	"fmt"
	"io"
	"strings"

	kbtypes "github.com/lk2023060901/ai-writer-backend/internal/knowledge/types"
	"github.com/ledongthuc/pdf"
)

// PDFLoader PDF 加载器
type PDFLoader struct{}

// NewPDFLoader 创建 PDF 加载器
func NewPDFLoader() *PDFLoader {
	return &PDFLoader{}
}

// Load 加载 PDF 内容
func (l *PDFLoader) Load(ctx context.Context, reader io.Reader) (*Document, error) {
	// 将 reader 内容读入内存
	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to read PDF data: %w", err)
	}

	// 创建 ReaderAt
	readerAt := &bytesReaderAt{data: data}

	// 打开 PDF
	pdfReader, err := pdf.NewReader(readerAt, int64(len(data)))
	if err != nil {
		return nil, fmt.Errorf("failed to open PDF: %w", err)
	}

	// 提取所有页面的文本
	var textBuilder strings.Builder
	numPages := pdfReader.NumPage()

	for i := 1; i <= numPages; i++ {
		page := pdfReader.Page(i)
		if page.V.IsNull() {
			continue
		}

		// 提取页面文本
		text, err := page.GetPlainText(nil)
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

// bytesReaderAt 实现 io.ReaderAt 接口
type bytesReaderAt struct {
	data []byte
}

func (b *bytesReaderAt) ReadAt(p []byte, off int64) (n int, err error) {
	if off < 0 {
		return 0, fmt.Errorf("negative offset")
	}
	if off >= int64(len(b.data)) {
		return 0, io.EOF
	}
	n = copy(p, b.data[off:])
	if n < len(p) {
		err = io.EOF
	}
	return
}
