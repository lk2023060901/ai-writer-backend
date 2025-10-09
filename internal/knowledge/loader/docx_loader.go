package loader

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"

	kbtypes "github.com/lk2023060901/ai-writer-backend/internal/knowledge/types"
	"github.com/unidoc/unioffice/common/license"
	"github.com/unidoc/unioffice/document"
)

func init() {
	// 设置 UniOffice 许可证密钥
	err := license.SetMeteredKey("c1609bf36881094add1da9ca73148904a289319d80e190b55c99687c84143e1c")
	if err != nil {
		panic(fmt.Sprintf("failed to set unioffice license: %v", err))
	}
}

// DOCXLoader Word 文档加载器
type DOCXLoader struct{}

// NewDOCXLoader 创建 Word 文档加载器
func NewDOCXLoader() *DOCXLoader {
	return &DOCXLoader{}
}

// Load 加载 Word 文档内容
func (l *DOCXLoader) Load(ctx context.Context, reader io.Reader) (*Document, error) {
	// 读取所有数据
	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to read DOCX data: %w", err)
	}

	// 打开 DOCX 文档
	doc, err := document.Read(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, fmt.Errorf("failed to open DOCX document: %w", err)
	}
	defer doc.Close()

	// 提取所有段落的文本
	var textBuilder strings.Builder
	for _, para := range doc.Paragraphs() {
		for _, run := range para.Runs() {
			textBuilder.WriteString(run.Text())
		}
		textBuilder.WriteString("\n")
	}

	return &Document{
		Content: textBuilder.String(),
		Metadata: map[string]interface{}{
			"loader": "docx",
		},
	}, nil
}

// SupportedTypes 返回支持的文件类型
func (l *DOCXLoader) SupportedTypes() []kbtypes.FileType {
	return []kbtypes.FileType{
		kbtypes.FileTypeDocx,
	}
}
