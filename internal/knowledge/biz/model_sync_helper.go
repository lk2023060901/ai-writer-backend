package biz

import (
	"context"
)

// compareModels 对比当前模型和最新模型列表
func (uc *ModelSyncUseCase) compareModels(current, latest []*AIModel) *ModelSyncResult {
	result := &ModelSyncResult{
		NewModels:        []*AIModel{},
		DeprecatedModels: []*AIModel{},
		UpdatedModels:    []*AIModel{},
		Errors:           []error{},
	}

	// 创建当前模型的 map（以 model_name 为 key）
	currentMap := make(map[string]*AIModel)
	for _, m := range current {
		currentMap[m.ModelName] = m
	}

	// 创建最新模型的 map
	latestMap := make(map[string]*AIModel)
	for _, m := range latest {
		latestMap[m.ModelName] = m
	}

	// 找出新增的模型
	for _, m := range latest {
		if _, exists := currentMap[m.ModelName]; !exists {
			result.NewModels = append(result.NewModels, m)
		}
	}

	// 找出弃用的模型
	for _, m := range current {
		if _, exists := latestMap[m.ModelName]; !exists {
			result.DeprecatedModels = append(result.DeprecatedModels, m)
		}
	}

	// 找出需要更新的模型（比较 max_tokens 等字段）
	for _, latest := range latestMap {
		if current, exists := currentMap[latest.ModelName]; exists {
			needUpdate := false

			// 比较 max_tokens
			if latest.MaxTokens != nil && current.MaxTokens != nil {
				if *latest.MaxTokens != *current.MaxTokens {
					needUpdate = true
				}
			} else if (latest.MaxTokens == nil) != (current.MaxTokens == nil) {
				needUpdate = true
			}

			// 比较 display_name
			if latest.DisplayName != current.DisplayName {
				needUpdate = true
			}

			// 比较 capabilities
			if !capabilitiesEqual(current.Capabilities, latest.Capabilities) {
				needUpdate = true
			}

			// 比较能力标志
			if current.SupportsStream != latest.SupportsStream ||
				current.SupportsVision != latest.SupportsVision ||
				current.SupportsFunctionCalling != latest.SupportsFunctionCalling ||
				current.SupportsReasoning != latest.SupportsReasoning ||
				current.SupportsWebSearch != latest.SupportsWebSearch {
				needUpdate = true
			}

			// 比较 embedding_dimensions
			if !embeddingDimensionsEqual(current.EmbeddingDimensions, latest.EmbeddingDimensions) {
				needUpdate = true
			}

			if needUpdate {
				// 保留原 ID，更新字段
				latest.ID = current.ID
				result.UpdatedModels = append(result.UpdatedModels, latest)
			}
		}
	}

	return result
}

// applyChanges 应用变更到数据库
func (uc *ModelSyncUseCase) applyChanges(ctx context.Context, result *ModelSyncResult) error {
	// 插入新模型（能力已包含在模型中，不需要单独插入）
	for _, model := range result.NewModels {
		if err := uc.aiModelRepo.Create(ctx, model); err != nil {
			result.Errors = append(result.Errors, err)
		}
	}

	// 标记弃用的模型为 disabled
	for _, model := range result.DeprecatedModels {
		model.IsEnabled = false
		model.VerificationStatus = "deprecated"
		if err := uc.aiModelRepo.Update(ctx, model); err != nil {
			result.Errors = append(result.Errors, err)
		}
	}

	// 更新模型信息
	for _, model := range result.UpdatedModels {
		if err := uc.aiModelRepo.Update(ctx, model); err != nil {
			result.Errors = append(result.Errors, err)
		}
	}

	return nil
}

// intPtr 返回 int 指针
func intPtr(i int) *int {
	return &i
}

// capabilitiesEqual 比较两个 capabilities 数组是否相等
func capabilitiesEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}

	// 创建 map 用于比较（顺序无关）
	aMap := make(map[string]bool)
	for _, v := range a {
		aMap[v] = true
	}

	for _, v := range b {
		if !aMap[v] {
			return false
		}
	}

	return true
}

// embeddingDimensionsEqual 比较两个 embedding dimensions 是否相等
func embeddingDimensionsEqual(a, b *int) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return *a == *b
}
