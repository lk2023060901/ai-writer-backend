package biz

import (
	"context"
	"fmt"
	"time"

	"github.com/lk2023060901/ai-writer-backend/internal/assistant/types"

	"github.com/google/uuid"
)

// AssistantRepo defines the repository interface for assistant data operations
type AssistantRepo interface {
	Create(ctx context.Context, assistant *types.Assistant) error
	GetByID(ctx context.Context, id, userID string) (*types.Assistant, error)
	List(ctx context.Context, userID string, filter *types.AssistantFilter) ([]*types.Assistant, error)
	Update(ctx context.Context, assistant *types.Assistant) error
	Delete(ctx context.Context, id, userID string) error
}

// TopicRepo defines the interface for topic repository
type TopicRepo interface {
	Create(ctx context.Context, topic *types.Topic) error
	GetByID(ctx context.Context, id string) (*types.Topic, error)
	ListByAssistant(ctx context.Context, assistantID string) ([]*types.Topic, error)
	ListByUserID(ctx context.Context, userID string) ([]*types.Topic, error)
	Update(ctx context.Context, topic *types.Topic) error
	Delete(ctx context.Context, id string) error
	DeleteByAssistant(ctx context.Context, assistantID string) error
}

// AssistantUseCase contains business logic for assistant operations
type AssistantUseCase struct {
	repo      AssistantRepo
	topicRepo TopicRepo
}

// NewAssistantUseCase creates a new assistant use case
func NewAssistantUseCase(
	repo AssistantRepo,
	topicRepo TopicRepo,
) *AssistantUseCase {
	return &AssistantUseCase{
		repo:      repo,
		topicRepo: topicRepo,
	}
}

// CreateAssistant creates a new assistant
func (uc *AssistantUseCase) CreateAssistant(ctx context.Context, userID string, req *CreateAssistantRequest) (*types.Assistant, error) {
	// Validate request
	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("invalid request: %w", err)
	}

	assistantID := uuid.New().String()

	assistant := &types.Assistant{
		ID:               assistantID,
		UserID:           userID,
		Name:             req.Name,
		Emoji:            req.Emoji,
		Prompt:           req.Prompt,
		Type:             "assistant",
		Tags:             req.Tags,
		KnowledgeBaseIDs: req.KnowledgeBaseIDs,
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}

	if err := uc.repo.Create(ctx, assistant); err != nil {
		return nil, fmt.Errorf("failed to create assistant: %w", err)
	}

	// Create default topic
	if err := uc.createDefaultTopic(ctx, assistantID); err != nil {
		return nil, fmt.Errorf("failed to create default topic: %w", err)
	}

	return assistant, nil
}

// GetAssistant retrieves an assistant by ID
func (uc *AssistantUseCase) GetAssistant(ctx context.Context, id, userID string) (*types.Assistant, error) {
	assistant, err := uc.repo.GetByID(ctx, id, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get assistant: %w", err)
	}

	return assistant, nil
}

// ListAssistants lists user's assistants with optional filtering
func (uc *AssistantUseCase) ListAssistants(ctx context.Context, userID string, filter *types.AssistantFilter) ([]*types.Assistant, error) {
	assistants, err := uc.repo.List(ctx, userID, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to list assistants: %w", err)
	}

	return assistants, nil
}

// UpdateAssistant updates an existing assistant
func (uc *AssistantUseCase) UpdateAssistant(ctx context.Context, id, userID string, req *UpdateAssistantRequest) (*types.Assistant, error) {
	// Get existing assistant
	assistant, err := uc.repo.GetByID(ctx, id, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get assistant: %w", err)
	}

	// Update fields
	if req.Name != "" {
		assistant.Name = req.Name
	}
	if req.Emoji != "" {
		assistant.Emoji = req.Emoji
	}
	if req.Prompt != "" {
		assistant.Prompt = req.Prompt
	}
	if len(req.Tags) > 0 {
		assistant.Tags = req.Tags
	}

	assistant.UpdatedAt = time.Now()

	if err := uc.repo.Update(ctx, assistant); err != nil {
		return nil, fmt.Errorf("failed to update assistant: %w", err)
	}

	return assistant, nil
}


// DeleteAssistant deletes an assistant
func (uc *AssistantUseCase) DeleteAssistant(ctx context.Context, id, userID string) error {
	// Delete all topics first
	if err := uc.topicRepo.DeleteByAssistant(ctx, id); err != nil {
		return fmt.Errorf("failed to delete topics: %w", err)
	}

	if err := uc.repo.Delete(ctx, id, userID); err != nil {
		return fmt.Errorf("failed to delete assistant: %w", err)
	}

	return nil
}

// createDefaultTopic creates a default topic for an assistant
func (uc *AssistantUseCase) createDefaultTopic(ctx context.Context, assistantID string) error {
	topic := &types.Topic{
		ID:          uuid.New().String(),
		AssistantID: assistantID,
		Name:        "Default Topic",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	return uc.topicRepo.Create(ctx, topic)
}

// CreateAssistantRequest represents a request to create an assistant
type CreateAssistantRequest struct {
	Name             string   `json:"name" binding:"required"`
	Emoji            string   `json:"emoji"`
	Prompt           string   `json:"prompt"`
	Tags             []string `json:"tags"`
	KnowledgeBaseIDs []string `json:"knowledge_base_ids"`
}

// Validate validates the create assistant request
func (r *CreateAssistantRequest) Validate() error {
	if r.Name == "" {
		return fmt.Errorf("name is required")
	}
	return nil
}

// UpdateAssistantRequest represents a request to update an assistant
type UpdateAssistantRequest struct {
	Name   string   `json:"name"`
	Emoji  string   `json:"emoji"`
	Prompt string   `json:"prompt"`
	Tags   []string `json:"tags"`
}
