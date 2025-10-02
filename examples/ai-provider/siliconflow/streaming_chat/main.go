package main

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/lk2023060901/ai-writer-backend/internal/ai/provider/factory"
	"github.com/lk2023060901/ai-writer-backend/internal/ai/provider/openai"
	"github.com/lk2023060901/ai-writer-backend/internal/ai/provider/types"
	"github.com/lk2023060901/ai-writer-backend/internal/pkg/logger"
	"go.uber.org/zap"
)

func main() {
	// 初始化日志
	cfg := logger.DefaultConfig()
	cfg.Output = "both"
	cfg.File.Filename = "logs/siliconflow_streaming_chat.log"
	if err := logger.InitGlobal(cfg); err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}
	defer logger.Sync()

	logger.Info("SiliconFlow 流式聊天示例启动")

	// 创建 Provider（使用 Factory）
	config := factory.SiliconFlow(
		"sk-gkqnwrnkmxqdeuqcpnntuzjtsfmbloyemaolyaxpuicfczxo",
		factory.WithModel("Qwen/Qwen2.5-7B-Instruct"),
	)
	provider, err := openai.New(config)
	if err != nil {
		log.Fatalf("Failed to create provider: %v", err)
	}
	defer provider.Close()

	fmt.Println("=== SiliconFlow 流式聊天测试 ===")
	fmt.Println("问题: 请讲一个有趣的故事，大约200字。")
	fmt.Println("\n回答:")
	fmt.Println("---")

	userMessage := "请讲一个有趣的故事，大约200字。"

	logger.Info("发送流式请求",
		zap.String("provider", "siliconflow"),
		zap.String("message", userMessage),
	)

	// 构建请求
	req := types.ChatCompletionRequest{
		Messages: []types.Message{
			{Role: "user", Content: userMessage},
		},
		MaxTokens:   1024,
		Temperature: 0.7,
	}

	// 发送流式请求
	stream, err := provider.CreateChatCompletionStream(context.Background(), req)
	if err != nil {
		logger.Error("流式请求失败", zap.Error(err))
		log.Fatalf("Error: %v", err)
	}

	// 接收流式响应
	var responseBuilder strings.Builder
	var messageID string
	var model string
	var usage *types.Usage

	for chunk := range stream {
		if chunk.Error != nil {
			logger.Error("流式响应错误", zap.Error(chunk.Error))
			log.Fatalf("Stream error: %v", chunk.Error)
		}

		if chunk.Done {
			break
		}

		// 记录基本信息
		if messageID == "" && chunk.ID != "" {
			messageID = chunk.ID
			model = chunk.Model
		}

		// 处理内容增量
		if len(chunk.Choices) > 0 && chunk.Choices[0].Delta.Content != "" {
			content := chunk.Choices[0].Delta.Content
			fmt.Print(content)
			responseBuilder.WriteString(content)
		}

		// 记录使用统计
		if chunk.Usage != nil {
			usage = chunk.Usage
		}
	}

	fmt.Println("\n---")

	logger.Info("收到完整流式响应",
		zap.String("provider", "siliconflow"),
		zap.String("id", messageID),
		zap.String("model", model),
		zap.String("response", responseBuilder.String()),
		zap.Any("usage", usage),
	)

	// 打印最终统计
	if usage != nil {
		fmt.Printf("\n使用统计:\n")
		fmt.Printf("  输入 Tokens: %d\n", usage.PromptTokens)
		fmt.Printf("  输出 Tokens: %d\n", usage.CompletionTokens)
		fmt.Printf("  总计 Tokens: %d\n", usage.TotalTokens)
	}
}
