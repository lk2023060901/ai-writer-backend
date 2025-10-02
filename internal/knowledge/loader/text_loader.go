package loader

import (
	"context"
	"fmt"
	"io"

	kbtypes "github.com/lk2023060901/ai-writer-backend/internal/knowledge/types"
)

// TextLoader 纯文本加载器
type TextLoader struct{}

// NewTextLoader 创建纯文本加载器
func NewTextLoader() *TextLoader {
	return &TextLoader{}
}

// Load 加载纯文本内容
func (l *TextLoader) Load(ctx context.Context, reader io.Reader) (*Document, error) {
	// 读取所有内容
	content, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to read text content: %w", err)
	}

	return &Document{
		Content: string(content),
		Metadata: map[string]interface{}{
			"loader": "text",
		},
	}, nil
}

// SupportedTypes 返回支持的文件类型
func (l *TextLoader) SupportedTypes() []kbtypes.FileType {
	return []kbtypes.FileType{
		kbtypes.FileTypeTxt,
	}
}
