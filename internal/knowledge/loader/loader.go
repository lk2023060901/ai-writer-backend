package loader

import (
	"context"
	"io"

	kbtypes "github.com/lk2023060901/ai-writer-backend/internal/knowledge/types"
)

// Loader 文档加载器接口
type Loader interface {
	// Load 加载文档内容
	Load(ctx context.Context, reader io.Reader) (*Document, error)

	// SupportedTypes 返回支持的文件类型
	SupportedTypes() []kbtypes.FileType
}

// Document 加载后的文档
type Document struct {
	Content  string                 // 文档文本内容
	Metadata map[string]interface{} // 文档元数据
}

// LoaderFactory Loader 工厂接口
type LoaderFactory interface {
	// CreateLoader 根据文件类型创建 Loader
	CreateLoader(fileType kbtypes.FileType) (Loader, error)

	// SupportedTypes 返回所有支持的文件类型
	SupportedTypes() []kbtypes.FileType
}
