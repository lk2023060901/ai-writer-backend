package biz

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// SystemOwnerID 系统所有者 ID，用于标识官方智能体
const SystemOwnerID = "00000000-0000-0000-0000-000000000000"

// Agent 智能体业务模型（用户智能体和官方智能体共用）
type Agent struct {
	ID               string    // UUID v7
	OwnerID          string    // 所有者 UUID（官方智能体为 SystemOwnerID）
	Name             string    // 智能体名称
	Emoji            string    // emoji 图标
	Prompt           string    // 提示词
	KnowledgeBaseIDs []string  // 关联的知识库 UUID 数组
	Tags             []string  // 标签数组
	Type             string    // 类型（固定为 'agent'）
	IsEnabled        bool      // 是否启用
	CreatedAt        time.Time // 创建时间
	UpdatedAt        time.Time // 更新时间
}

// IsOfficial 判断是否为官方智能体
func (a *Agent) IsOfficial() bool {
	return a.OwnerID == SystemOwnerID
}

// ListAgentsRequest 列表查询请求
type ListAgentsRequest struct {
	UserID    string   // 当前用户 ID
	Page      int      // 页码
	PageSize  int      // 每页数量
	IsEnabled *bool    // 启用状态过滤（可选）
	Tags      []string // 标签过滤（可选）
	Keyword   string   // 关键词搜索（可选）
}

// AgentRepo 用户智能体仓储接口
type AgentRepo interface {
	Create(ctx context.Context, agent *Agent) error
	GetByID(ctx context.Context, id string, ownerID string) (*Agent, error)
	List(ctx context.Context, req *ListAgentsRequest) ([]*Agent, int64, error)
	Update(ctx context.Context, agent *Agent) error
	Delete(ctx context.Context, id string, ownerID string) error
	UpdateEnabled(ctx context.Context, id string, ownerID string, enabled bool) error
}

// OfficialAgentRepo 官方智能体仓储接口
type OfficialAgentRepo interface {
	GetByID(ctx context.Context, id string) (*Agent, error)
	List(ctx context.Context, req *ListAgentsRequest) ([]*Agent, int64, error)
}

// AgentUseCase 智能体业务逻辑
type AgentUseCase struct {
	agentRepo         AgentRepo
	officialAgentRepo OfficialAgentRepo
}

// NewAgentUseCase 创建智能体用例
func NewAgentUseCase(agentRepo AgentRepo, officialAgentRepo OfficialAgentRepo) *AgentUseCase {
	return &AgentUseCase{
		agentRepo:         agentRepo,
		officialAgentRepo: officialAgentRepo,
	}
}

// CreateAgent 创建智能体
func (uc *AgentUseCase) CreateAgent(ctx context.Context, userID string, name string, emoji string, prompt string, knowledgeBaseIDs []string, tags []string) (*Agent, error) {
	// 验证
	if name == "" {
		return nil, ErrAgentNameRequired
	}
	if prompt == "" {
		return nil, ErrAgentPromptRequired
	}
	if len(prompt) < 10 {
		return nil, ErrAgentPromptTooShort
	}

	agent := &Agent{
		ID:               uuid.Must(uuid.NewV7()).String(),
		OwnerID:          userID,
		Name:             name,
		Emoji:            emoji,
		Prompt:           prompt,
		KnowledgeBaseIDs: knowledgeBaseIDs,
		Tags:             tags,
		Type:             "agent",
		IsEnabled:        true,
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}

	if err := uc.agentRepo.Create(ctx, agent); err != nil {
		return nil, err
	}

	return agent, nil
}

// GetAgent 获取智能体详情（支持用户智能体和官方智能体）
func (uc *AgentUseCase) GetAgent(ctx context.Context, id string, userID string) (*Agent, error) {
	// 先尝试从用户智能体查询
	agent, err := uc.agentRepo.GetByID(ctx, id, userID)
	if err == nil {
		return agent, nil
	}

	// 如果不是 NotFound 错误，直接返回
	if err != ErrAgentNotFound {
		return nil, err
	}

	// 从官方智能体查询
	return uc.officialAgentRepo.GetByID(ctx, id)
}

// ListAgents 获取智能体列表（包含官方智能体 + 用户自己创建的）
func (uc *AgentUseCase) ListAgents(ctx context.Context, req *ListAgentsRequest) ([]*Agent, int64, error) {
	// 查询用户智能体
	userAgents, userTotal, err := uc.agentRepo.List(ctx, req)
	if err != nil {
		return nil, 0, err
	}

	// 查询官方智能体
	officialAgents, officialTotal, err := uc.officialAgentRepo.List(ctx, req)
	if err != nil {
		return nil, 0, err
	}

	// 合并结果（官方智能体在前）
	allAgents := append(officialAgents, userAgents...)
	totalCount := officialTotal + userTotal

	return allAgents, totalCount, nil
}

// UpdateAgent 更新智能体（仅用户自己创建的）
func (uc *AgentUseCase) UpdateAgent(ctx context.Context, id string, userID string, name *string, emoji *string, prompt *string, knowledgeBaseIDs []string, tags []string) (*Agent, error) {
	// 获取现有智能体
	agent, err := uc.agentRepo.GetByID(ctx, id, userID)
	if err != nil {
		return nil, err
	}

	// 更新字段
	if name != nil && *name != "" {
		agent.Name = *name
	}
	if emoji != nil {
		agent.Emoji = *emoji
	}
	if prompt != nil {
		if *prompt == "" {
			return nil, ErrAgentPromptRequired
		}
		if len(*prompt) < 10 {
			return nil, ErrAgentPromptTooShort
		}
		agent.Prompt = *prompt
	}
	if knowledgeBaseIDs != nil {
		agent.KnowledgeBaseIDs = knowledgeBaseIDs
	}
	if tags != nil {
		agent.Tags = tags
	}

	agent.UpdatedAt = time.Now()

	if err := uc.agentRepo.Update(ctx, agent); err != nil {
		return nil, err
	}

	return agent, nil
}

// DeleteAgent 删除智能体（仅用户自己创建的）
func (uc *AgentUseCase) DeleteAgent(ctx context.Context, id string, userID string) error {
	return uc.agentRepo.Delete(ctx, id, userID)
}

// EnableAgent 启用智能体
func (uc *AgentUseCase) EnableAgent(ctx context.Context, id string, userID string) error {
	return uc.agentRepo.UpdateEnabled(ctx, id, userID, true)
}

// DisableAgent 禁用智能体
func (uc *AgentUseCase) DisableAgent(ctx context.Context, id string, userID string) error {
	return uc.agentRepo.UpdateEnabled(ctx, id, userID, false)
}

// BatchCreateAgents 批量创建智能体
func (uc *AgentUseCase) BatchCreateAgents(ctx context.Context, userID string, items []struct {
	Name   string
	Emoji  string
	Prompt string
	Tags   []string
}) ([]*Agent, []error) {
	agents := make([]*Agent, 0, len(items))
	errors := make([]error, 0)

	for i, item := range items {
		// 验证
		if item.Name == "" {
			errors = append(errors, ErrAgentNameRequired)
			continue
		}
		if item.Prompt == "" {
			errors = append(errors, ErrAgentPromptRequired)
			continue
		}
		if len(item.Prompt) < 10 {
			errors = append(errors, ErrAgentPromptTooShort)
			continue
		}

		agent := &Agent{
			ID:               uuid.Must(uuid.NewV7()).String(),
			OwnerID:          userID,
			Name:             item.Name,
			Emoji:            item.Emoji,
			Prompt:           item.Prompt,
			KnowledgeBaseIDs: []string{},
			Tags:             item.Tags,
			Type:             "agent",
			IsEnabled:        true,
			CreatedAt:        time.Now(),
			UpdatedAt:        time.Now(),
		}

		if err := uc.agentRepo.Create(ctx, agent); err != nil {
			errors = append(errors, err)
			continue
		}

		agents = append(agents, agent)

		// 确保每个智能体的创建时间略有不同（UUID v7 基于时间戳）
		if i < len(items)-1 {
			time.Sleep(time.Millisecond)
		}
	}

	return agents, errors
}
