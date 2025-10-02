package data

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/lk2023060901/ai-writer-backend/internal/agent/models"
	"github.com/lk2023060901/ai-writer-backend/internal/agent/types"

	"gorm.io/gorm"
)

// AgentRepo implements the agent repository using GORM
type AgentRepo struct {
	db *gorm.DB
}

// NewAgentRepo creates a new agent repository
func NewAgentRepo(db *gorm.DB) *AgentRepo {
	return &AgentRepo{db: db}
}

// Create creates a new agent
func (r *AgentRepo) Create(ctx context.Context, agent *types.Agent) error {
	model := r.toModel(agent)
	if err := r.db.WithContext(ctx).Create(model).Error; err != nil {
		return fmt.Errorf("failed to create agent: %w", err)
	}
	return nil
}

// GetByID retrieves an agent by ID
func (r *AgentRepo) GetByID(ctx context.Context, id string) (*types.Agent, error) {
	var model models.Agent
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&model).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("agent not found")
		}
		return nil, fmt.Errorf("failed to get agent: %w", err)
	}

	return r.toDomain(&model)
}

// List lists agents with optional filtering
func (r *AgentRepo) List(ctx context.Context, filter *types.AgentFilter) ([]*types.Agent, error) {
	query := r.db.WithContext(ctx).Model(&models.Agent{})

	// Apply filters
	if filter != nil {
		if filter.Group != "" {
			// Query JSON array for group membership
			query = query.Where("JSON_CONTAINS(groups, ?)", fmt.Sprintf(`"%s"`, filter.Group))
		}
		if filter.IsBuiltin != nil {
			query = query.Where("is_builtin = ?", *filter.IsBuiltin)
		}
		if filter.Keyword != "" {
			keyword := "%" + filter.Keyword + "%"
			query = query.Where("name LIKE ? OR description LIKE ?", keyword, keyword)
		}
	}

	var modelList []models.Agent
	if err := query.Order("created_at DESC").Find(&modelList).Error; err != nil {
		return nil, fmt.Errorf("failed to list agents: %w", err)
	}

	agents := make([]*types.Agent, 0, len(modelList))
	for _, model := range modelList {
		agent, err := r.toDomain(&model)
		if err != nil {
			return nil, err
		}
		agents = append(agents, agent)
	}

	return agents, nil
}

// Update updates an existing agent
func (r *AgentRepo) Update(ctx context.Context, agent *types.Agent) error {
	model := r.toModel(agent)
	if err := r.db.WithContext(ctx).Save(model).Error; err != nil {
		return fmt.Errorf("failed to update agent: %w", err)
	}
	return nil
}

// Delete deletes an agent by ID
func (r *AgentRepo) Delete(ctx context.Context, id string) error {
	if err := r.db.WithContext(ctx).Where("id = ?", id).Delete(&models.Agent{}).Error; err != nil {
		return fmt.Errorf("failed to delete agent: %w", err)
	}
	return nil
}

// ListGroups returns all agent groups with counts
func (r *AgentRepo) ListGroups(ctx context.Context) ([]*types.AgentGroup, error) {
	// Fetch all agents and process in-memory
	var agents []models.Agent
	if err := r.db.WithContext(ctx).Find(&agents).Error; err != nil {
		return nil, fmt.Errorf("failed to get agents: %w", err)
	}

	groupCounts := make(map[string]int)
	for _, agent := range agents {
		for _, group := range agent.Groups {
			groupCounts[group]++
		}
	}

	groups := make([]*types.AgentGroup, 0, len(groupCounts))
	for name, count := range groupCounts {
		groups = append(groups, &types.AgentGroup{
			Name:  name,
			Count: count,
		})
	}

	return groups, nil
}

// toModel converts domain agent to GORM model
func (r *AgentRepo) toModel(agent *types.Agent) *models.Agent {
	settingsJSON := make(models.JSON)
	if agent.Settings != nil {
		settingsJSON["temperature"] = agent.Settings.Temperature
		settingsJSON["max_tokens"] = agent.Settings.MaxTokens
		settingsJSON["context_count"] = agent.Settings.ContextCount
		settingsJSON["enable_web_search"] = agent.Settings.EnableWebSearch
		settingsJSON["tool_use_mode"] = agent.Settings.ToolUseMode
	}

	return &models.Agent{
		ID:          agent.ID,
		Name:        agent.Name,
		Description: agent.Description,
		Emoji:       agent.Emoji,
		Prompt:      agent.Prompt,
		Groups:      models.StringArray(agent.Group),
		Settings:    settingsJSON,
		IsBuiltin:   agent.IsBuiltin,
		CreatedAt:   agent.CreatedAt,
		UpdatedAt:   agent.UpdatedAt,
	}
}

// toDomain converts GORM model to domain agent
func (r *AgentRepo) toDomain(model *models.Agent) (*types.Agent, error) {
	var settings *types.AgentSettings
	if model.Settings != nil {
		settingsBytes, _ := json.Marshal(model.Settings)
		settings = &types.AgentSettings{}
		if err := json.Unmarshal(settingsBytes, settings); err != nil {
			return nil, fmt.Errorf("failed to unmarshal settings: %w", err)
		}
	}

	return &types.Agent{
		ID:          model.ID,
		Name:        model.Name,
		Description: model.Description,
		Emoji:       model.Emoji,
		Prompt:      model.Prompt,
		Group:       []string(model.Groups),
		Settings:    settings,
		IsBuiltin:   model.IsBuiltin,
		CreatedAt:   model.CreatedAt,
		UpdatedAt:   model.UpdatedAt,
	}, nil
}
