package biz

import (
	"context"
	"fmt"
	"time"

	"github.com/lk2023060901/ai-writer-backend/internal/assistant/types"

	"github.com/google/uuid"
)

// TopicUseCase contains business logic for topic operations
type TopicUseCase struct {
	repo TopicRepo
}

// NewTopicUseCase creates a new topic use case
func NewTopicUseCase(repo TopicRepo) *TopicUseCase {
	return &TopicUseCase{
		repo: repo,
	}
}

// CreateTopic creates a new topic
func (uc *TopicUseCase) CreateTopic(ctx context.Context, userID, assistantID, name string) (*types.Topic, error) {
	if assistantID == "" {
		return nil, fmt.Errorf("assistant ID is required")
	}

	if userID == "" {
		return nil, fmt.Errorf("user ID is required")
	}

	if name == "" {
		name = "New Topic"
	}

	topic := &types.Topic{
		ID:          uuid.New().String(),
		AssistantID: assistantID,
		UserID:      userID,
		Name:        name,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if err := uc.repo.Create(ctx, topic); err != nil {
		return nil, fmt.Errorf("failed to create topic: %w", err)
	}

	return topic, nil
}

// GetTopic retrieves a topic by ID
func (uc *TopicUseCase) GetTopic(ctx context.Context, id string) (*types.Topic, error) {
	topic, err := uc.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get topic: %w", err)
	}

	return topic, nil
}

// ListTopics lists topics for an assistant
func (uc *TopicUseCase) ListTopics(ctx context.Context, assistantID string) ([]*types.Topic, error) {
	topics, err := uc.repo.ListByAssistant(ctx, assistantID)
	if err != nil {
		return nil, fmt.Errorf("failed to list topics: %w", err)
	}

	return topics, nil
}

// ListTopicsByUser lists all topics for a user (across all their assistants)
func (uc *TopicUseCase) ListTopicsByUser(ctx context.Context, userID string) ([]*types.Topic, error) {
	topics, err := uc.repo.ListByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to list user topics: %w", err)
	}

	return topics, nil
}

// UpdateTopic updates a topic
func (uc *TopicUseCase) UpdateTopic(ctx context.Context, id, name string) (*types.Topic, error) {
	topic, err := uc.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get topic: %w", err)
	}

	topic.Name = name
	topic.UpdatedAt = time.Now()

	if err := uc.repo.Update(ctx, topic); err != nil {
		return nil, fmt.Errorf("failed to update topic: %w", err)
	}

	return topic, nil
}

// DeleteTopic deletes a topic
func (uc *TopicUseCase) DeleteTopic(ctx context.Context, id string) error {
	if err := uc.repo.Delete(ctx, id); err != nil {
		return fmt.Errorf("failed to delete topic: %w", err)
	}

	return nil
}

// DeleteAllTopics deletes all topics for an assistant
func (uc *TopicUseCase) DeleteAllTopics(ctx context.Context, userID, assistantID string) error {
	if err := uc.repo.DeleteByAssistant(ctx, assistantID); err != nil {
		return fmt.Errorf("failed to delete all topics: %w", err)
	}

	// Create a new default topic
	topic := &types.Topic{
		ID:          uuid.New().String(),
		AssistantID: assistantID,
		UserID:      userID,
		Name:        "Default Topic",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if err := uc.repo.Create(ctx, topic); err != nil {
		return fmt.Errorf("failed to create default topic: %w", err)
	}

	return nil
}

// UpdateTopicTime updates the last activity time of a topic
func (uc *TopicUseCase) UpdateTopicTime(ctx context.Context, id string) error {
	topic, err := uc.repo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to get topic: %w", err)
	}

	topic.UpdatedAt = time.Now()

	if err := uc.repo.Update(ctx, topic); err != nil {
		return fmt.Errorf("failed to update topic time: %w", err)
	}

	return nil
}
