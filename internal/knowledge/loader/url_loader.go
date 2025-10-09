package loader

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"golang.org/x/net/html"
)

// URLLoader URL 内容加载器
type URLLoader struct {
	client *http.Client
}

// NewURLLoader 创建 URL 加载器
func NewURLLoader() *URLLoader {
	return &URLLoader{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// LoadContent 加载 URL 内容
func (l *URLLoader) LoadContent(ctx context.Context, url string) (string, error) {
	// 创建请求
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	// 设置 User-Agent
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; AI-Writer/1.0)")

	// 发送请求
	resp, err := l.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to fetch URL: %w", err)
	}
	defer resp.Body.Close()

	// 检查状态码
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP error: %d %s", resp.StatusCode, resp.Status)
	}

	// 读取内容
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	// 检测内容类型
	contentType := resp.Header.Get("Content-Type")
	if strings.Contains(contentType, "text/html") {
		// HTML 内容，提取文本
		return l.extractTextFromHTML(string(body))
	}

	// 纯文本内容
	return string(body), nil
}

// extractTextFromHTML 从 HTML 提取纯文本
func (l *URLLoader) extractTextFromHTML(htmlContent string) (string, error) {
	doc, err := html.Parse(strings.NewReader(htmlContent))
	if err != nil {
		return "", fmt.Errorf("failed to parse HTML: %w", err)
	}

	var text strings.Builder
	var f func(*html.Node)
	f = func(n *html.Node) {
		if n.Type == html.TextNode {
			text.WriteString(n.Data)
			text.WriteString(" ")
		}
		if n.Type == html.ElementNode && (n.Data == "br" || n.Data == "p" || n.Data == "div") {
			text.WriteString("\n")
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			f(c)
		}
	}
	f(doc)

	// 清理多余的空白
	result := strings.TrimSpace(text.String())
	// 移除多余的空行
	lines := strings.Split(result, "\n")
	var cleanedLines []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			cleanedLines = append(cleanedLines, line)
		}
	}

	return strings.Join(cleanedLines, "\n"), nil
}
