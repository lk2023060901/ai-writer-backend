package models

import "errors"

// Knowledge Base 错误
var (
	ErrInvalidName                = errors.New("invalid knowledge base name")
	ErrInvalidUserID              = errors.New("invalid user id")
	ErrInvalidEmbeddingProvider   = errors.New("invalid embedding provider")
	ErrInvalidEmbeddingModel      = errors.New("invalid embedding model")
	ErrInvalidEmbeddingDimensions = errors.New("invalid embedding dimensions")
	ErrInvalidChunkStrategy       = errors.New("invalid chunk strategy")
	ErrInvalidChunkSize           = errors.New("invalid chunk size")
	ErrInvalidChunkOverlap        = errors.New("invalid chunk overlap")
)

// Document 错误
var (
	ErrInvalidKnowledgeBaseID = errors.New("invalid knowledge base id")
	ErrInvalidFilename        = errors.New("invalid filename")
	ErrInvalidFileType        = errors.New("invalid file type")
	ErrInvalidFileSize        = errors.New("invalid file size")
	ErrInvalidFileHash        = errors.New("invalid file hash")
	ErrInvalidMinioPath       = errors.New("invalid minio path")
	ErrInvalidDocumentStatus  = errors.New("invalid document status")
)

// Chunk 错误
var (
	ErrInvalidDocumentID  = errors.New("invalid document id")
	ErrEmptyContent       = errors.New("empty content")
	ErrInvalidChunkIndex  = errors.New("invalid chunk index")
	ErrInvalidTokenCount  = errors.New("invalid token count")
	ErrInvalidMilvusID    = errors.New("invalid milvus id")
)
