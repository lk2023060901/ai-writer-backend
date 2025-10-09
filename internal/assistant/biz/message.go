package biz

import (
	"context"
	"fmt"
	"time"

	"github.com/lk2023060901/ai-writer-backend/internal/assistant/types"

	"github.com/google/uuid"
)

// MessageRepo defines the repository interface for message operations
type MessageRepo interface {
	Create(ctx context.Context, message *types.Message) error
	GetByID(ctx context.Context, id string) (*types.Message, error)
	ListByTopic(ctx context.Context, topicID string, limit, offset int) ([]*types.Message, error)
	CountByTopic(ctx context.Context, topicID string) (int64, error)
	DeleteByTopic(ctx context.Context, topicID string) error
}

// MessageUseCase contains business logic for message operations
type MessageUseCase struct {
	repo      MessageRepo
	topicRepo TopicRepo
}

// NewMessageUseCase creates a new message use case
func NewMessageUseCase(repo MessageRepo, topicRepo TopicRepo) *MessageUseCase {
	return &MessageUseCase{
		repo:      repo,
		topicRepo: topicRepo,
	}
}

// CreateMessage creates a new message
func (uc *MessageUseCase) CreateMessage(ctx context.Context, topicID, role string, contentBlocks []types.ContentBlock, tokenCount *int) (*types.Message, error) {
	return uc.CreateMessageWithModel(ctx, topicID, role, contentBlocks, tokenCount, "", "")
}

// CreateMessageWithModel creates a new message with provider and model information
func (uc *MessageUseCase) CreateMessageWithModel(ctx context.Context, topicID, role string, contentBlocks []types.ContentBlock, tokenCount *int, provider, model string) (*types.Message, error) {
	// Validate topic exists
	if _, err := uc.topicRepo.GetByID(ctx, topicID); err != nil {
		return nil, fmt.Errorf("topic not found: %w", err)
	}

	// Validate role
	if role != "user" && role != "assistant" {
		return nil, fmt.Errorf("invalid role: must be 'user' or 'assistant'")
	}

	// Validate content blocks
	if len(contentBlocks) == 0 {
		return nil, fmt.Errorf("content blocks cannot be empty")
	}

	message := &types.Message{
		ID:            uuid.New().String(),
		TopicID:       topicID,
		Role:          role,
		ContentBlocks: contentBlocks,
		TokenCount:    tokenCount,
		Provider:      provider,
		Model:         model,
		CreatedAt:     time.Now(),
	}

	if err := uc.repo.Create(ctx, message); err != nil {
		return nil, fmt.Errorf("failed to create message: %w", err)
	}

	return message, nil
}

// GetMessage retrieves a message by ID
func (uc *MessageUseCase) GetMessage(ctx context.Context, id string) (*types.Message, error) {
	message, err := uc.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get message: %w", err)
	}

	return message, nil
}

// ListMessages lists messages for a topic with pagination
func (uc *MessageUseCase) ListMessages(ctx context.Context, topicID string, limit, offset int) ([]*types.Message, int64, error) {
	// Validate topic exists
	if _, err := uc.topicRepo.GetByID(ctx, topicID); err != nil {
		return nil, 0, fmt.Errorf("topic not found: %w", err)
	}

	// Get messages
	messages, err := uc.repo.ListByTopic(ctx, topicID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list messages: %w", err)
	}

	// Get total count
	total, err := uc.repo.CountByTopic(ctx, topicID)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count messages: %w", err)
	}

	return messages, total, nil
}

// DeleteMessagesInTopic deletes all messages in a topic
func (uc *MessageUseCase) DeleteMessagesInTopic(ctx context.Context, topicID string) error {
	// Validate topic exists
	if _, err := uc.topicRepo.GetByID(ctx, topicID); err != nil {
		return fmt.Errorf("topic not found: %w", err)
	}

	if err := uc.repo.DeleteByTopic(ctx, topicID); err != nil {
		return fmt.Errorf("failed to delete messages: %w", err)
	}

	return nil
}
