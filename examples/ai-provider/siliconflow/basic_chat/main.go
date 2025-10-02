package main

import (
	"context"
	"fmt"
	"log"
	"time"

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
	cfg.File.Filename = "logs/siliconflow_basic_chat.log"
	if err := logger.InitGlobal(cfg); err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}
	defer logger.Sync()

	logger.Info("SiliconFlow 基本聊天示例启动")

	// 创建 SiliconFlow Provider（使用 Factory 快速创建）
	config := factory.SiliconFlow(
		"sk-gkqnwrnkmxqdeuqcpnntuzjtsfmbloyemaolyaxpuicfczxo",
		factory.WithModel("Qwen/Qwen2.5-7B-Instruct"),
		factory.WithTimeout(30*time.Second),
	)
	provider, err := openai.New(config)
	if err != nil {
		log.Fatalf("Failed to create provider: %v", err)
	}
	defer provider.Close()

	fmt.Println("=== SiliconFlow 基本聊天测试 ===\n")

	userMessage := "你好！请用中文介绍一下你自己。"

	logger.Info("发送请求",
		zap.String("provider", "siliconflow"),
		zap.String("message", userMessage),
	)

	// 构建请求
	req := types.ChatCompletionRequest{
		Messages: []types.Message{
			{Role: "user", Content: userMessage},
		},
		MaxTokens:   100,
		Temperature: 0.7,
	}

	// 发送请求
	resp, err := provider.CreateChatCompletion(context.Background(), req)
	if err != nil {
		logger.Error("请求失败", zap.Error(err))
		log.Fatalf("Error: %v", err)
	}

	// 提取响应内容
	responseText := resp.Choices[0].Message.GetTextContent()

	logger.Info("收到响应",
		zap.String("provider", "siliconflow"),
		zap.String("id", resp.ID),
		zap.String("model", resp.Model),
		zap.String("response", responseText),
		zap.Int("prompt_tokens", resp.Usage.PromptTokens),
		zap.Int("completion_tokens", resp.Usage.CompletionTokens),
		zap.Int("total_tokens", resp.Usage.TotalTokens),
	)

	// 打印响应
	fmt.Printf("Model: %s\n", resp.Model)
	fmt.Printf("ID: %s\n", resp.ID)
	fmt.Println("\n内容:")
	fmt.Println(responseText)

	// 打印使用统计
	fmt.Printf("\n使用统计:\n")
	fmt.Printf("  输入 Tokens: %d\n", resp.Usage.PromptTokens)
	fmt.Printf("  输出 Tokens: %d\n", resp.Usage.CompletionTokens)
	fmt.Printf("  总计 Tokens: %d\n", resp.Usage.TotalTokens)
}
