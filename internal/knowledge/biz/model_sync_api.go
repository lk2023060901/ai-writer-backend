package biz

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/google/uuid"
)

// fetchLatestModels 从 AI 服务商 API 获取最新模型列表
func (uc *ModelSyncUseCase) fetchLatestModels(ctx context.Context, provider *AIProvider) ([]*AIModel, error) {
	if provider.APIKey == "" {
		return nil, fmt.Errorf("provider %s has no API key configured", provider.ProviderName)
	}

	switch provider.ProviderType {
	case "siliconflow":
		return uc.fetchSiliconFlowModels(ctx, provider)
	case "anthropic":
		return uc.fetchAnthropicModels(ctx, provider)
	case "zhipu":
		return uc.fetchZhipuModels(ctx, provider)
	case "openai":
		return nil, fmt.Errorf("OpenAI API key not provided, skipping")
	default:
		return nil, fmt.Errorf("unsupported provider type: %s", provider.ProviderType)
	}
}

// SiliconFlowModelData 硅基流动模型原始数据
type SiliconFlowModelData struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	OwnedBy string `json:"owned_by"`
}

// SiliconFlowModelsResponse 硅基流动模型列表响应
type SiliconFlowModelsResponse struct {
	Object string                 `json:"object"`
	Data   []SiliconFlowModelData `json:"data"`
}

// fetchSiliconFlowModels 从硅基流动获取模型列表（按 sub_type 分批获取并聚合）
func (uc *ModelSyncUseCase) fetchSiliconFlowModels(ctx context.Context, provider *AIProvider) ([]*AIModel, error) {
	// 定义要获取的 sub_type 列表
	subTypes := []struct {
		name           string
		capabilityType string
	}{
		{"embedding", CapabilityTypeEmbedding},
		{"reranker", CapabilityTypeRerank},
		{"chat", CapabilityTypeChat},
	}

	// 用于聚合模型的 map（key: model_name）
	modelMap := make(map[string]*AIModel)

	// 分批次获取每个 sub_type 的模型
	for _, st := range subTypes {
		apiModels, err := uc.fetchSiliconFlowModelsBySubType(ctx, provider, st.name)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch %s models: %w", st.name, err)
		}

		// 聚合模型能力
		for _, m := range apiModels {
			// 如果模型已存在，添加能力类型
			if model, exists := modelMap[m.ID]; exists {
				// 检查能力类型是否已存在
				capExists := false
				for _, cap := range model.Capabilities {
					if cap == st.capabilityType {
						capExists = true
						break
					}
				}
				if !capExists {
					model.Capabilities = append(model.Capabilities, st.capabilityType)
				}
			} else {
				// 创建新模型
				now := time.Now()
				model := &AIModel{
					ID:                      uuid.New().String(),
					ProviderID:              provider.ID,
					ModelName:               m.ID,
					DisplayName:             m.ID,
					IsEnabled:               true,
					VerificationStatus:      "available",
					Capabilities:            []string{st.capabilityType},
					SupportsStream:          false,
					SupportsVision:          false,
					SupportsFunctionCalling: false,
					SupportsReasoning:       false,
					SupportsWebSearch:       false,
					CreatedAt:               now,
					UpdatedAt:               now,
				}

				// 根据能力类型设置特定字段
				if st.capabilityType == CapabilityTypeEmbedding {
					// 获取 embedding 维度
					if dim, err := uc.getEmbeddingDimensions(ctx, provider, m.ID); err == nil {
						model.EmbeddingDimensions = &dim
					}
				} else if st.capabilityType == CapabilityTypeChat {
					// Chat 模型默认支持流式
					model.SupportsStream = true
					// 推断其他能力
					model.SupportsVision = uc.inferVisionSupport(m.ID)
					model.SupportsFunctionCalling = uc.inferFunctionCallingSupport(m.ID)
					model.SupportsReasoning = uc.inferReasoningSupport(m.ID)
				}

				modelMap[m.ID] = model
			}
		}
	}

	// 转换为数组
	models := make([]*AIModel, 0, len(modelMap))
	for _, model := range modelMap {
		models = append(models, model)
	}

	return models, nil
}

// fetchSiliconFlowModelsBySubType 获取指定 sub_type 的模型列表
func (uc *ModelSyncUseCase) fetchSiliconFlowModelsBySubType(ctx context.Context, provider *AIProvider, subType string) ([]SiliconFlowModelData, error) {
	url := fmt.Sprintf("%s/models?sub_type=%s", provider.APIBaseURL, subType)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+provider.APIKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to call API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	var result SiliconFlowModelsResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return result.Data, nil
}

// fetchAnthropicModels 从 Anthropic 获取模型列表（尝试调用 /models API，失败则使用预定义列表）
func (uc *ModelSyncUseCase) fetchAnthropicModels(ctx context.Context, provider *AIProvider) ([]*AIModel, error) {
	// 尝试调用 OpenAI 兼容的 /models API（用于中转服务）
	url := provider.APIBaseURL + "/v1/models"

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err == nil {
		req.Header.Set("Authorization", "Bearer "+provider.APIKey)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("anthropic-version", "2023-06-01")

		client := &http.Client{Timeout: 30 * time.Second}
		resp, err := client.Do(req)
		if err == nil && resp.StatusCode == http.StatusOK {
			defer resp.Body.Close()

			var result struct {
				Object string `json:"object"`
				Data   []struct {
					ID         string `json:"id"`
					Object     string `json:"object"`
					Created    int64  `json:"created"`
					MaxTokens  int    `json:"max_tokens,omitempty"`
					OwnedBy    string `json:"owned_by"`
				} `json:"data"`
			}

			if err := json.NewDecoder(resp.Body).Decode(&result); err == nil && len(result.Data) > 0 {
				// 成功获取模型列表
				models := []*AIModel{}
				now := time.Now()

				for _, m := range result.Data {
					model := &AIModel{
						ID:                 uuid.New().String(),
						ProviderID:         provider.ID,
						ModelName:          m.ID,
						DisplayName:        m.ID,
						IsEnabled:          true,
						VerificationStatus: "available",
						CreatedAt:          now,
						UpdatedAt:          now,
					}

					if m.MaxTokens > 0 {
						model.MaxTokens = &m.MaxTokens
					}

					// 推断模型能力
					model.Capabilities = []string{CapabilityTypeChat}
					model.SupportsStream = true
					model.SupportsVision = uc.inferVisionSupport(m.ID)
					model.SupportsFunctionCalling = uc.inferFunctionCallingSupport(m.ID)
					model.SupportsReasoning = uc.inferReasoningSupport(m.ID)
					model.SupportsWebSearch = false

					models = append(models, model)
				}

				return models, nil
			}
			if resp != nil && resp.Body != nil {
				resp.Body.Close()
			}
		}
	}

	// API 调用失败，使用官方预定义列表
	// https://docs.anthropic.com/en/docs/about-claude/models
	now := time.Now()

	models := []*AIModel{
		{
			ID:                      uuid.New().String(),
			ProviderID:              provider.ID,
			ModelName:               "claude-3-5-sonnet-20241022",
			DisplayName:             "Claude 3.5 Sonnet (Oct 2024)",
			MaxTokens:               intPtr(200000),
			IsEnabled:               true,
			VerificationStatus:      "available",
			Capabilities:            []string{CapabilityTypeChat},
			SupportsStream:          true,
			SupportsVision:          true,
			SupportsFunctionCalling: true,
			SupportsReasoning:       false,
			SupportsWebSearch:       false,
			CreatedAt:               now,
			UpdatedAt:               now,
		},
		{
			ID:                      uuid.New().String(),
			ProviderID:              provider.ID,
			ModelName:               "claude-3-5-haiku-20241022",
			DisplayName:             "Claude 3.5 Haiku (Oct 2024)",
			MaxTokens:               intPtr(200000),
			IsEnabled:               true,
			VerificationStatus:      "available",
			Capabilities:            []string{CapabilityTypeChat},
			SupportsStream:          true,
			SupportsVision:          false,
			SupportsFunctionCalling: true,
			SupportsReasoning:       false,
			SupportsWebSearch:       false,
			CreatedAt:               now,
			UpdatedAt:               now,
		},
		{
			ID:                      uuid.New().String(),
			ProviderID:              provider.ID,
			ModelName:               "claude-3-opus-20240229",
			DisplayName:             "Claude 3 Opus",
			MaxTokens:               intPtr(200000),
			IsEnabled:               true,
			VerificationStatus:      "available",
			Capabilities:            []string{CapabilityTypeChat},
			SupportsStream:          true,
			SupportsVision:          true,
			SupportsFunctionCalling: false,
			SupportsReasoning:       false,
			SupportsWebSearch:       false,
			CreatedAt:               now,
			UpdatedAt:               now,
		},
	}

	return models, nil
}

// ZhipuModelsResponse 智谱 AI 模型列表响应
type ZhipuModelsResponse struct {
	Object string `json:"object"`
	Data   []struct {
		ID         string `json:"id"`
		Object     string `json:"object"`
		Created    int64  `json:"created"`
		MaxTokens  int    `json:"max_tokens,omitempty"`
		OwnedBy    string `json:"owned_by"`
	} `json:"data"`
}

// fetchZhipuModels 从智谱 AI 获取模型列表
func (uc *ModelSyncUseCase) fetchZhipuModels(ctx context.Context, provider *AIProvider) ([]*AIModel, error) {
	url := provider.APIBaseURL + "/models"

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+provider.APIKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to call API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	var result ZhipuModelsResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	models := []*AIModel{}
	now := time.Now()

	for _, m := range result.Data {
		model := &AIModel{
			ID:                 uuid.New().String(),
			ProviderID:         provider.ID,
			ModelName:          m.ID,
			DisplayName:        m.ID,
			IsEnabled:          true,
			VerificationStatus: "available",
			CreatedAt:          now,
			UpdatedAt:          now,
		}

		if m.MaxTokens > 0 {
			model.MaxTokens = &m.MaxTokens
		}

		// 智谱 API 不返回模型类型，默认为 chat 模型
		model.Capabilities = []string{CapabilityTypeChat}
		model.SupportsStream = true
		model.SupportsVision = uc.inferVisionSupport(m.ID)
		model.SupportsFunctionCalling = uc.inferFunctionCallingSupport(m.ID)
		model.SupportsReasoning = uc.inferReasoningSupport(m.ID)
		model.SupportsWebSearch = false

		models = append(models, model)
	}

	return models, nil
}

// getEmbeddingDimensions 通过测试调用获取 embedding 维度
func (uc *ModelSyncUseCase) getEmbeddingDimensions(ctx context.Context, provider *AIProvider, modelName string) (int, error) {
	url := provider.APIBaseURL + "/embeddings"

	requestBody := map[string]interface{}{
		"model": modelName,
		"input": "hi",
	}

	bodyBytes, err := json.Marshal(requestBody)
	if err != nil {
		return 0, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(bodyBytes))
	if err != nil {
		return 0, err
	}

	req.Header.Set("Authorization", "Bearer "+provider.APIKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	var result struct {
		Data []struct {
			Embedding []float64 `json:"embedding"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return 0, err
	}

	if len(result.Data) == 0 || len(result.Data[0].Embedding) == 0 {
		return 0, fmt.Errorf("no embedding data returned")
	}

	return len(result.Data[0].Embedding), nil
}

// contains 字符串包含（不区分大小写）
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr ||
		(len(s) > 0 && len(substr) > 0 &&
			bytes.Contains(bytes.ToLower([]byte(s)), bytes.ToLower([]byte(substr)))))
}

// inferVisionSupport 推断是否支持视觉理解
func (uc *ModelSyncUseCase) inferVisionSupport(modelName string) bool {
	return contains(modelName, "vision") ||
		contains(modelName, "vl") ||
		contains(modelName, "-VL-") ||
		contains(modelName, "4o") || // GPT-4o
		contains(modelName, "QVQ") // Qwen Visual Question
}

// inferFunctionCallingSupport 推断是否支持函数调用
func (uc *ModelSyncUseCase) inferFunctionCallingSupport(modelName string) bool {
	return contains(modelName, "turbo") ||
		contains(modelName, "plus") ||
		contains(modelName, "sonnet") || // Claude Sonnet 系列
		contains(modelName, "gpt-4") || // GPT-4 系列
		contains(modelName, "gpt-3.5") // GPT-3.5 系列
}

// inferReasoningSupport 推断是否支持推理能力
func (uc *ModelSyncUseCase) inferReasoningSupport(modelName string) bool {
	return contains(modelName, "-R1") || // DeepSeek-R1 系列
		contains(modelName, "R1-") ||
		contains(modelName, "Thinking") || // Qwen Thinking / GLM Thinking
		contains(modelName, "QwQ") || // Qwen with Question
		contains(modelName, "Rumination") || // GLM Rumination (反思)
		contains(modelName, "-o1") || // GPT-o1
		contains(modelName, "-o3") // GPT-o3
}
