package loader

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	kbtypes "github.com/lk2023060901/ai-writer-backend/internal/knowledge/types"
)

// JSONLoader JSON 文件加载器
type JSONLoader struct{}

// NewJSONLoader 创建 JSON 加载器
func NewJSONLoader() *JSONLoader {
	return &JSONLoader{}
}

// Load 加载 JSON 内容，将其格式化为可读文本
func (l *JSONLoader) Load(ctx context.Context, reader io.Reader) (*Document, error) {
	// 读取所有内容
	content, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to read json content: %w", err)
	}

	// 解析 JSON 并格式化
	var data interface{}
	if err := json.Unmarshal(content, &data); err != nil {
		return nil, fmt.Errorf("failed to parse json: %w", err)
	}

	// 将 JSON 转换为格式化的文本
	formattedText := l.formatJSON(data, 0)

	return &Document{
		Content: formattedText,
		Metadata: map[string]interface{}{
			"loader":        "json",
			"original_size": len(content),
		},
	}, nil
}

// formatJSON 递归格式化 JSON 数据为可读文本
func (l *JSONLoader) formatJSON(data interface{}, indent int) string {
	var sb strings.Builder
	indentStr := strings.Repeat("  ", indent)

	switch v := data.(type) {
	case map[string]interface{}:
		for key, value := range v {
			sb.WriteString(fmt.Sprintf("%s%s: ", indentStr, key))

			switch valueType := value.(type) {
			case map[string]interface{}, []interface{}:
				sb.WriteString("\n")
				sb.WriteString(l.formatJSON(value, indent+1))
			default:
				sb.WriteString(fmt.Sprintf("%v\n", valueType))
			}
		}
	case []interface{}:
		for i, item := range v {
			sb.WriteString(fmt.Sprintf("%s[%d]: ", indentStr, i))

			switch itemType := item.(type) {
			case map[string]interface{}, []interface{}:
				sb.WriteString("\n")
				sb.WriteString(l.formatJSON(item, indent+1))
			default:
				sb.WriteString(fmt.Sprintf("%v\n", itemType))
			}
		}
	default:
		sb.WriteString(fmt.Sprintf("%s%v\n", indentStr, v))
	}

	return sb.String()
}

// SupportedTypes 返回支持的文件类型
func (l *JSONLoader) SupportedTypes() []kbtypes.FileType {
	return []kbtypes.FileType{
		kbtypes.FileTypeJson,
	}
}
