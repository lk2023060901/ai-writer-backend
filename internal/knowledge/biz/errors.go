package biz

import "errors"

// SystemOwnerID 官方资源的所有者 ID（与智能体模块一致）
const SystemOwnerID = "00000000-0000-0000-0000-000000000000"

// AI Provider Config 相关错误
var (
	ErrAIProviderConfigNotFound        = errors.New("ai provider config not found")
	ErrAIProviderConfigNameRequired    = errors.New("ai provider config name is required")
	ErrAIProviderConfigAPIKeyRequired  = errors.New("api key is required")
	ErrAIProviderConfigInvalidProvider = errors.New("invalid provider type")
	ErrAIProviderConfigInUse           = errors.New("ai provider config is in use by knowledge bases")
	ErrNoDefaultAIConfig               = errors.New("no default ai provider config found")
)

// Knowledge Base 相关错误
var (
	ErrKnowledgeBaseNotFound         = errors.New("knowledge base not found")
	ErrKnowledgeBaseNameRequired     = errors.New("knowledge base name is required")
	ErrKnowledgeBaseInvalidChunkSize = errors.New("invalid chunk size")
	ErrKnowledgeBaseInvalidOverlap   = errors.New("invalid chunk overlap")
)

// Document 相关错误
var (
	ErrDocumentNotFound       = errors.New("document not found")
	ErrDocumentInvalidType    = errors.New("invalid document type")
	ErrDocumentTooLarge       = errors.New("document too large")
	ErrDocumentHashExists     = errors.New("document with same hash already exists")
	ErrDocumentProcessing     = errors.New("document is being processed")
	ErrDocumentAlreadyFailed  = errors.New("document processing already failed")
)

// 权限相关错误
var (
	ErrUnauthorized                 = errors.New("unauthorized")
	ErrCannotEditOfficialResource   = errors.New("cannot edit official resource")
	ErrCannotDeleteOfficialResource = errors.New("cannot delete official resource")
)

// Milvus 相关错误
var (
	ErrMilvusCollectionExists    = errors.New("milvus collection already exists")
	ErrMilvusCollectionNotFound  = errors.New("milvus collection not found")
	ErrMilvusCreateFailed        = errors.New("failed to create milvus collection")
	ErrMilvusInsertFailed        = errors.New("failed to insert vectors to milvus")
	ErrMilvusSearchFailed        = errors.New("failed to search vectors in milvus")
)
