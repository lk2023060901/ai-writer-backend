package loader

import (
	"fmt"

	kbtypes "github.com/lk2023060901/ai-writer-backend/internal/knowledge/types"
)

// Factory Loader 工厂
type Factory struct {
	loaders map[kbtypes.FileType]Loader
}

// NewFactory 创建 Loader 工厂
func NewFactory() *Factory {
	factory := &Factory{
		loaders: make(map[kbtypes.FileType]Loader),
	}

	// 注册所有 Loaders
	factory.registerLoader(NewTextLoader())
	factory.registerLoader(NewMarkdownLoader())
	factory.registerLoader(NewPDFLoader())
	factory.registerLoader(NewDOCXLoader())
	factory.registerLoader(NewJSONLoader())

	return factory
}

// registerLoader 注册 Loader
func (f *Factory) registerLoader(loader Loader) {
	for _, fileType := range loader.SupportedTypes() {
		f.loaders[fileType] = loader
	}
}

// CreateLoader 根据文件类型创建 Loader
func (f *Factory) CreateLoader(fileType kbtypes.FileType) (Loader, error) {
	loader, ok := f.loaders[fileType]
	if !ok {
		return nil, fmt.Errorf("unsupported file type: %s", fileType)
	}
	return loader, nil
}

// SupportedTypes 返回所有支持的文件类型
func (f *Factory) SupportedTypes() []kbtypes.FileType {
	types := make([]kbtypes.FileType, 0, len(f.loaders))
	for fileType := range f.loaders {
		types = append(types, fileType)
	}
	return types
}
