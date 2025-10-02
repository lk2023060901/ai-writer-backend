package biz

import (
	"context"
	"fmt"
	"time"

	"github.com/lk2023060901/ai-writer-backend/internal/agent/types"

	"github.com/google/uuid"
)

// AgentRepo defines the repository interface for agent data operations
type AgentRepo interface {
	Create(ctx context.Context, agent *types.Agent) error
	GetByID(ctx context.Context, id string) (*types.Agent, error)
	List(ctx context.Context, filter *types.AgentFilter) ([]*types.Agent, error)
	Update(ctx context.Context, agent *types.Agent) error
	Delete(ctx context.Context, id string) error
	ListGroups(ctx context.Context) ([]*types.AgentGroup, error)
}

// AgentUseCase contains business logic for agent operations
type AgentUseCase struct {
	repo AgentRepo
}

// NewAgentUseCase creates a new agent use case
func NewAgentUseCase(repo AgentRepo) *AgentUseCase {
	return &AgentUseCase{
		repo: repo,
	}
}

// CreateAgent creates a new agent
func (uc *AgentUseCase) CreateAgent(ctx context.Context, req *CreateAgentRequest) (*types.Agent, error) {
	// Validate request
	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("invalid request: %w", err)
	}

	agent := &types.Agent{
		ID:          uuid.New().String(),
		Name:        req.Name,
		Description: req.Description,
		Emoji:       req.Emoji,
		Prompt:      req.Prompt,
		Group:       req.Group,
		Settings:    req.Settings,
		IsBuiltin:   false, // User-created agents are not builtin
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if err := uc.repo.Create(ctx, agent); err != nil {
		return nil, fmt.Errorf("failed to create agent: %w", err)
	}

	return agent, nil
}

// GetAgent retrieves an agent by ID
func (uc *AgentUseCase) GetAgent(ctx context.Context, id string) (*types.Agent, error) {
	if id == "" {
		return nil, fmt.Errorf("agent ID is required")
	}

	agent, err := uc.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get agent: %w", err)
	}

	return agent, nil
}

// ListAgents lists agents with optional filtering
func (uc *AgentUseCase) ListAgents(ctx context.Context, filter *types.AgentFilter) ([]*types.Agent, error) {
	agents, err := uc.repo.List(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to list agents: %w", err)
	}

	return agents, nil
}

// UpdateAgent updates an existing agent
func (uc *AgentUseCase) UpdateAgent(ctx context.Context, id string, req *UpdateAgentRequest) (*types.Agent, error) {
	// Get existing agent
	agent, err := uc.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get agent: %w", err)
	}

	// Don't allow updating builtin agents
	if agent.IsBuiltin {
		return nil, fmt.Errorf("cannot update builtin agent")
	}

	// Update fields
	if req.Name != "" {
		agent.Name = req.Name
	}
	if req.Description != "" {
		agent.Description = req.Description
	}
	if req.Emoji != "" {
		agent.Emoji = req.Emoji
	}
	if req.Prompt != "" {
		agent.Prompt = req.Prompt
	}
	if len(req.Group) > 0 {
		agent.Group = req.Group
	}
	if req.Settings != nil {
		agent.Settings = req.Settings
	}

	agent.UpdatedAt = time.Now()

	if err := uc.repo.Update(ctx, agent); err != nil {
		return nil, fmt.Errorf("failed to update agent: %w", err)
	}

	return agent, nil
}

// DeleteAgent deletes an agent by ID
func (uc *AgentUseCase) DeleteAgent(ctx context.Context, id string) error {
	// Get agent to check if it's builtin
	agent, err := uc.repo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to get agent: %w", err)
	}

	// Don't allow deleting builtin agents
	if agent.IsBuiltin {
		return fmt.Errorf("cannot delete builtin agent")
	}

	if err := uc.repo.Delete(ctx, id); err != nil {
		return fmt.Errorf("failed to delete agent: %w", err)
	}

	return nil
}

// ListGroups returns all agent groups with counts
func (uc *AgentUseCase) ListGroups(ctx context.Context) ([]*types.AgentGroup, error) {
	groups, err := uc.repo.ListGroups(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list groups: %w", err)
	}

	return groups, nil
}

// ListByGroup lists agents in a specific group
func (uc *AgentUseCase) ListByGroup(ctx context.Context, groupName string) ([]*types.Agent, error) {
	filter := &types.AgentFilter{
		Group: groupName,
	}

	agents, err := uc.repo.List(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to list agents by group: %w", err)
	}

	return agents, nil
}

// CreateAgentRequest represents a request to create an agent
type CreateAgentRequest struct {
	Name        string               `json:"name" binding:"required"`
	Description string               `json:"description"`
	Emoji       string               `json:"emoji"`
	Prompt      string               `json:"prompt" binding:"required"`
	Group       []string             `json:"group"`
	Settings    *types.AgentSettings `json:"settings"`
}

// Validate validates the create agent request
func (r *CreateAgentRequest) Validate() error {
	if r.Name == "" {
		return fmt.Errorf("name is required")
	}
	if r.Prompt == "" {
		return fmt.Errorf("prompt is required")
	}
	return nil
}

// UpdateAgentRequest represents a request to update an agent
type UpdateAgentRequest struct {
	Name        string               `json:"name"`
	Description string               `json:"description"`
	Emoji       string               `json:"emoji"`
	Prompt      string               `json:"prompt"`
	Group       []string             `json:"group"`
	Settings    *types.AgentSettings `json:"settings"`
}
