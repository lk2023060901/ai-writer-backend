package factory

import "github.com/lk2023060901/ai-writer-backend/internal/pkg/logger"

// BaseFactory 通用工厂基础结构
type BaseFactory struct {
	logger *logger.Logger
}

// NewBaseFactory 创建基础工厂
func NewBaseFactory(lgr *logger.Logger) *BaseFactory {
	if lgr == nil {
		lgr = logger.L()
	}
	return &BaseFactory{
		logger: lgr,
	}
}

// Logger 获取 logger
func (f *BaseFactory) Logger() *logger.Logger {
	return f.logger
}
