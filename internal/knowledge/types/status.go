package types

// DocumentStatus 文档处理状态
type DocumentStatus string

const (
	// DocumentStatusPending 待处理
	DocumentStatusPending DocumentStatus = "pending"
	// DocumentStatusProcessing 处理中
	DocumentStatusProcessing DocumentStatus = "processing"
	// DocumentStatusCompleted 处理完成
	DocumentStatusCompleted DocumentStatus = "completed"
	// DocumentStatusFailed 处理失败
	DocumentStatusFailed DocumentStatus = "failed"
)

// Valid 检查状态是否有效
func (s DocumentStatus) Valid() bool {
	switch s {
	case DocumentStatusPending, DocumentStatusProcessing, DocumentStatusCompleted, DocumentStatusFailed:
		return true
	}
	return false
}

// String 返回字符串表示
func (s DocumentStatus) String() string {
	return string(s)
}

// FileType 文件类型
type FileType string

const (
	FileTypeTxt  FileType = "txt"
	FileTypePdf  FileType = "pdf"
	FileTypeDocx FileType = "docx"
	FileTypeMd   FileType = "md"
	FileTypeHtml FileType = "html"
	FileTypeJson FileType = "json"
)

// Valid 检查文件类型是否有效
func (ft FileType) Valid() bool {
	switch ft {
	case FileTypeTxt, FileTypePdf, FileTypeDocx, FileTypeMd, FileTypeHtml, FileTypeJson:
		return true
	}
	return false
}

// String 返回字符串表示
func (ft FileType) String() string {
	return string(ft)
}

// ChunkStrategy 分块策略
type ChunkStrategy string

const (
	// ChunkStrategyToken 基于 Token 分块
	ChunkStrategyToken ChunkStrategy = "token"
	// ChunkStrategyRecursive 递归分块
	ChunkStrategyRecursive ChunkStrategy = "recursive"
)

// Valid 检查分块策略是否有效
func (cs ChunkStrategy) Valid() bool {
	switch cs {
	case ChunkStrategyToken, ChunkStrategyRecursive:
		return true
	}
	return false
}

// String 返回字符串表示
func (cs ChunkStrategy) String() string {
	return string(cs)
}

// EmbeddingProvider Embedding 提供商（仅支持 API 调用）
type EmbeddingProvider string

const (
	// EmbeddingProviderOpenAI OpenAI Embedding API
	EmbeddingProviderOpenAI EmbeddingProvider = "openai"
	// EmbeddingProviderAnthropic Anthropic Embedding API（如果支持）
	EmbeddingProviderAnthropic EmbeddingProvider = "anthropic"
)

// Valid 检查 Embedding 提供商是否有效
func (ep EmbeddingProvider) Valid() bool {
	switch ep {
	case EmbeddingProviderOpenAI, EmbeddingProviderAnthropic:
		return true
	}
	return false
}

// String 返回字符串表示
func (ep EmbeddingProvider) String() string {
	return string(ep)
}
