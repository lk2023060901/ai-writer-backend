package processor

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"strings"

	"github.com/ledongthuc/pdf"
)

// DocumentProcessor 文档处理器实现
type DocumentProcessor struct{}

// NewDocumentProcessor 创建文档处理器
func NewDocumentProcessor() *DocumentProcessor {
	return &DocumentProcessor{}
}

// ExtractText 从文件中提取文本内容
func (p *DocumentProcessor) ExtractText(ctx context.Context, fileData []byte, fileType string) (string, error) {
	switch strings.ToLower(fileType) {
	case "pdf":
		return p.extractPDF(fileData)
	case "docx":
		return p.extractDOCX(fileData)
	case "txt", "md":
		return string(fileData), nil
	default:
		return "", fmt.Errorf("unsupported file type: %s", fileType)
	}
}

// extractPDF 提取 PDF 文本
func (p *DocumentProcessor) extractPDF(fileData []byte) (string, error) {
	reader, err := pdf.NewReader(bytes.NewReader(fileData), int64(len(fileData)))
	if err != nil {
		return "", fmt.Errorf("failed to parse PDF: %w", err)
	}

	var textBuilder strings.Builder
	numPages := reader.NumPage()

	for i := 1; i <= numPages; i++ {
		page := reader.Page(i)
		if page.V.IsNull() {
			continue
		}

		text, err := page.GetPlainText(nil)
		if err != nil {
			continue // 跳过无法提取的页面
		}

		textBuilder.WriteString(text)
		textBuilder.WriteString("\n\n")
	}

	return textBuilder.String(), nil
}

// extractDOCX 提取 DOCX 文本（使用原生 zip + xml 解析，无需第三方库）
func (p *DocumentProcessor) extractDOCX(fileData []byte) (string, error) {
	// DOCX 是一个 ZIP 文件
	zipReader, err := zip.NewReader(bytes.NewReader(fileData), int64(len(fileData)))
	if err != nil {
		return "", fmt.Errorf("failed to open DOCX as ZIP: %w", err)
	}

	// 找到 word/document.xml 文件
	var documentXML *zip.File
	for _, file := range zipReader.File {
		if file.Name == "word/document.xml" {
			documentXML = file
			break
		}
	}

	if documentXML == nil {
		return "", fmt.Errorf("document.xml not found in DOCX")
	}

	// 读取 document.xml 内容
	xmlFile, err := documentXML.Open()
	if err != nil {
		return "", fmt.Errorf("failed to open document.xml: %w", err)
	}
	defer xmlFile.Close()

	xmlData, err := io.ReadAll(xmlFile)
	if err != nil {
		return "", fmt.Errorf("failed to read document.xml: %w", err)
	}

	// 解析 XML 提取文本
	text, err := p.extractTextFromDocumentXML(xmlData)
	if err != nil {
		return "", fmt.Errorf("failed to extract text from XML: %w", err)
	}

	return text, nil
}

// extractTextFromDocumentXML 从 document.xml 中提取纯文本
func (p *DocumentProcessor) extractTextFromDocumentXML(xmlData []byte) (string, error) {
	type Text struct {
		Value string `xml:",chardata"`
	}

	type Run struct {
		Texts []Text `xml:"t"`
	}

	type Paragraph struct {
		Runs []Run `xml:"r"`
	}

	type Body struct {
		Paragraphs []Paragraph `xml:"p"`
	}

	type Document struct {
		Body Body `xml:"body"`
	}

	var doc Document
	if err := xml.Unmarshal(xmlData, &doc); err != nil {
		return "", fmt.Errorf("failed to unmarshal XML: %w", err)
	}

	var textBuilder strings.Builder
	for _, para := range doc.Body.Paragraphs {
		for _, run := range para.Runs {
			for _, text := range run.Texts {
				textBuilder.WriteString(text.Value)
			}
		}
		textBuilder.WriteString("\n")
	}

	return textBuilder.String(), nil
}
