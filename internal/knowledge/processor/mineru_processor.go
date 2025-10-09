package processor

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/lk2023060901/ai-writer-backend/internal/pkg/logger"
	"github.com/lk2023060901/ai-writer-backend/internal/pkg/mineru"
	"go.uber.org/zap"
	"archive/zip"
)

// MinerUProcessor MinerU 文档处理器
type MinerUProcessor struct {
	client *mineru.Client
	logger *logger.Logger
	// 嵌入基础 processor 用于 fallback
	baseProcessor *DocumentProcessor
}

// NewMinerUProcessor 创建 MinerU 文档处理器
func NewMinerUProcessor(client *mineru.Client, log *logger.Logger) *MinerUProcessor {
	return &MinerUProcessor{
		client:        client,
		logger:        log,
		baseProcessor: NewDocumentProcessor(),
	}
}

// ExtractText 使用 MinerU 提取文本内容
func (p *MinerUProcessor) ExtractText(ctx context.Context, fileData []byte, fileType string) (string, error) {
	fileType = strings.ToLower(fileType)

	// 对于简单文本文件，直接使用本地处理（无需调用 MinerU API）
	if fileType == "txt" || fileType == "md" || fileType == "json" {
		p.logger.Info("using local processor for simple text file", zap.String("type", fileType))
		return p.baseProcessor.ExtractText(ctx, fileData, fileType)
	}

	// 对于 PDF/DOCX 等复杂文档，使用 MinerU
	if fileType == "pdf" || fileType == "docx" || fileType == "doc" || fileType == "ppt" || fileType == "pptx" {
		p.logger.Info("using MinerU for document processing", zap.String("type", fileType))
		return p.extractWithMinerU(ctx, fileData, fileType)
	}

	// 不支持的文件类型
	return "", fmt.Errorf("unsupported file type: %s", fileType)
}

// extractWithMinerU 使用 MinerU 提取文本
func (p *MinerUProcessor) extractWithMinerU(ctx context.Context, fileData []byte, fileType string) (string, error) {
	// 1. 创建临时文件
	tmpFile, err := p.createTempFile(fileData, fileType)
	if err != nil {
		return "", fmt.Errorf("failed to create temp file: %w", err)
	}
	defer os.Remove(tmpFile) // 清理临时文件

	p.logger.Info("created temp file for MinerU processing", zap.String("path", tmpFile))

	// 2. 创建 MinerU 任务（上传文件）
	// 注意：OCR 仅对扫描的 PDF 有用，对于 DOCX/PPTX 等原生文档应该关闭
	// 免费账户可能有 OCR 配额限制
	isOCR := fileType == "pdf" // 仅对 PDF 启用 OCR
	req := &mineru.BatchUploadRequest{
		Language: "ch",
		Files: []mineru.BatchFileInfo{
			{
				Name:   filepath.Base(tmpFile),
				IsOCR:  isOCR,
				DataID: "doc-" + fileType,
			},
		},
	}

	p.logger.Info("starting MinerU batch upload task")

	// 3. 创建批量任务并上传文件，等待完成
	results, err := p.client.CreateBatchWithFilesAndWait(ctx, req, []string{tmpFile}, nil)
	if err != nil {
		p.logger.Error("MinerU task failed", zap.Error(err))
		// DOCX fallback 已禁用，因为 UniOffice 许可证已过期
		// p.logger.Warn("falling back to local processor")
		// return p.baseProcessor.ExtractText(ctx, fileData, fileType)
		return "", fmt.Errorf("MinerU processing failed: %w", err)
	}

	// 4. 检查结果
	if len(results.Data.ExtractResult) == 0 {
		return "", fmt.Errorf("no extraction results returned from MinerU")
	}

	result := results.Data.ExtractResult[0]

	// 5. 检查任务状态
	if result.State != mineru.TaskStateDone {
		p.logger.Error("MinerU task not completed",
			zap.String("state", string(result.State)),
			zap.String("error", result.ErrMsg))
		// DOCX fallback 已禁用，因为 UniOffice 许可证已过期
		// p.logger.Warn("falling back to local processor")
		// return p.baseProcessor.ExtractText(ctx, fileData, fileType)
		return "", fmt.Errorf("MinerU task failed with state %s: %s", result.State, result.ErrMsg)
	}

	// 6. 下载并解析结果
	text, err := p.downloadAndExtractText(ctx, result.FullZipURL)
	if err != nil {
		p.logger.Error("failed to extract text from MinerU result", zap.Error(err))
		// DOCX fallback 已禁用，因为 UniOffice 许可证已过期
		// p.logger.Warn("falling back to local processor")
		// return p.baseProcessor.ExtractText(ctx, fileData, fileType)
		return "", fmt.Errorf("failed to extract text from MinerU result: %w", err)
	}

	p.logger.Info("successfully extracted text via MinerU",
		zap.Int("text_length", len(text)),
		zap.String("state", string(result.State)))

	return text, nil
}

// createTempFile 创建临时文件
func (p *MinerUProcessor) createTempFile(data []byte, fileType string) (string, error) {
	tmpDir := os.TempDir()

	f, err := os.CreateTemp(tmpDir, fmt.Sprintf("mineru-*.%s", fileType))
	if err != nil {
		return "", fmt.Errorf("failed to create temp file: %w", err)
	}
	defer f.Close()

	if _, err := f.Write(data); err != nil {
		os.Remove(f.Name())
		return "", fmt.Errorf("failed to write temp file: %w", err)
	}

	return f.Name(), nil
}

// downloadAndExtractText 下载 MinerU 结果并提取文本
func (p *MinerUProcessor) downloadAndExtractText(ctx context.Context, zipURL string) (string, error) {
	// 1. 下载结果 ZIP
	tmpZip := filepath.Join(os.TempDir(), "mineru-result-*.zip")
	f, err := os.CreateTemp(os.TempDir(), "mineru-result-*.zip")
	if err != nil {
		return "", fmt.Errorf("failed to create temp zip file: %w", err)
	}
	tmpZip = f.Name()
	f.Close()
	defer os.Remove(tmpZip)

	p.logger.Info("downloading MinerU result", zap.String("url", zipURL))

	err = p.client.DownloadResult(ctx, zipURL, tmpZip)
	if err != nil {
		return "", fmt.Errorf("failed to download result: %w", err)
	}

	// 2. 解压并读取 Markdown 文件
	text, err := p.extractTextFromZip(tmpZip)
	if err != nil {
		return "", fmt.Errorf("failed to extract text from zip: %w", err)
	}

	return text, nil
}

// extractTextFromZip 从 ZIP 中提取文本
func (p *MinerUProcessor) extractTextFromZip(zipPath string) (string, error) {
	// MinerU 返回的 ZIP 中通常包含：
	// - auto/ (目录)
	//   - xxx.md (Markdown 格式的提取结果)
	//   - images/ (图片)
	//   - content_list.json (元数据)

	// 打开 ZIP 文件
	zipReader, err := zip.OpenReader(zipPath)
	if err != nil {
		return "", fmt.Errorf("failed to open zip reader: %w", err)
	}
	defer zipReader.Close()

	// 查找并读取所有 .md 文件
	var mdContent strings.Builder
	foundMarkdown := false

	for _, file := range zipReader.File {
		// 只处理 .md 文件
		if !strings.HasSuffix(strings.ToLower(file.Name), ".md") {
			continue
		}

		p.logger.Info("found markdown file in MinerU result",
			zap.String("filename", file.Name),
			zap.Int64("size", int64(file.UncompressedSize64)))

		// 打开文件
		rc, err := file.Open()
		if err != nil {
			p.logger.Warn("failed to open markdown file",
				zap.String("filename", file.Name),
				zap.Error(err))
			continue
		}

		// 读取内容
		content, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			p.logger.Warn("failed to read markdown file",
				zap.String("filename", file.Name),
				zap.Error(err))
			continue
		}

		// 追加内容
		if mdContent.Len() > 0 {
			mdContent.WriteString("\n\n")
		}
		mdContent.Write(content)
		foundMarkdown = true
	}

	if !foundMarkdown {
		return "", fmt.Errorf("no markdown files found in MinerU result ZIP")
	}

	p.logger.Info("successfully extracted markdown from ZIP",
		zap.Int("total_length", mdContent.Len()))

	return mdContent.String(), nil
}

// ChunkText 文本分块（复用基础实现）
func (p *MinerUProcessor) ChunkText(text string, chunkSize, chunkOverlap int, strategy string) ([]string, error) {
	return p.baseProcessor.ChunkText(text, chunkSize, chunkOverlap, strategy)
}
