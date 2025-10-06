package sse

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestEventFormatSSE(t *testing.T) {
	// 测试简单数据
	event := Event{
		Type: "test",
		Data: map[string]interface{}{
			"message": "hello",
			"count":   42,
		},
	}

	result := event.FormatSSE()

	// 验证格式
	lines := strings.Split(result, "\n")
	if len(lines) < 3 {
		t.Fatalf("Expected at least 3 lines, got %d", len(lines))
	}

	// 验证事件类型行
	if lines[0] != "event: test" {
		t.Errorf("Expected 'event: test', got '%s'", lines[0])
	}

	// 验证数据行
	if !strings.HasPrefix(lines[1], "data: ") {
		t.Errorf("Expected data line to start with 'data: ', got '%s'", lines[1])
	}

	// 解析 JSON 数据
	dataJSON := strings.TrimPrefix(lines[1], "data: ")
	var parsedData map[string]interface{}
	err := json.Unmarshal([]byte(dataJSON), &parsedData)
	if err != nil {
		t.Fatalf("Failed to parse JSON data: %v", err)
	}

	// 验证类型字段被添加
	if parsedData["type"] != "test" {
		t.Errorf("Expected type 'test', got '%v'", parsedData["type"])
	}

	// 验证原始数据被保留
	if parsedData["message"] != "hello" {
		t.Errorf("Expected message 'hello', got '%v'", parsedData["message"])
	}

	if parsedData["count"] != float64(42) { // JSON 数字解析为 float64
		t.Errorf("Expected count 42, got '%v'", parsedData["count"])
	}
}

func TestEventFormatSSEWithNonMapData(t *testing.T) {
	// 测试非 map 数据
	event := Event{
		Type: "simple",
		Data: "just a string",
	}

	result := event.FormatSSE()

	// 解析数据
	lines := strings.Split(result, "\n")
	dataJSON := strings.TrimPrefix(lines[1], "data: ")
	var parsedData map[string]interface{}
	err := json.Unmarshal([]byte(dataJSON), &parsedData)
	if err != nil {
		t.Fatalf("Failed to parse JSON data: %v", err)
	}

	// 验证类型字段
	if parsedData["type"] != "simple" {
		t.Errorf("Expected type 'simple', got '%v'", parsedData["type"])
	}

	// 验证数据在 payload 字段中
	if parsedData["payload"] != "just a string" {
		t.Errorf("Expected payload 'just a string', got '%v'", parsedData["payload"])
	}
}
