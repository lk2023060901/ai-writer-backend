package data

import (
	"context"
	"fmt"

	"github.com/lk2023060901/ai-writer-backend/internal/assistant/models"
	"github.com/lk2023060901/ai-writer-backend/internal/assistant/types"

	"gorm.io/gorm"
)

// TopicRepo implements the topic repository using GORM
type TopicRepo struct {
	db *gorm.DB
}

// NewTopicRepo creates a new topic repository
func NewTopicRepo(db *gorm.DB) *TopicRepo {
	return &TopicRepo{db: db}
}

// Create creates a new topic
func (r *TopicRepo) Create(ctx context.Context, topic *types.Topic) error {
	model := r.toModel(topic)
	if err := r.db.WithContext(ctx).Create(model).Error; err != nil {
		return fmt.Errorf("failed to create topic: %w", err)
	}
	return nil
}

// GetByID retrieves a topic by ID
func (r *TopicRepo) GetByID(ctx context.Context, id string) (*types.Topic, error) {
	var model models.Topic
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&model).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("topic not found")
		}
		return nil, fmt.Errorf("failed to get topic: %w", err)
	}

	return r.toDomain(&model), nil
}

// ListByAssistant lists all topics for an assistant
func (r *TopicRepo) ListByAssistant(ctx context.Context, assistantID string) ([]*types.Topic, error) {
	var modelList []models.Topic
	if err := r.db.WithContext(ctx).
		Where("assistant_id = ?", assistantID).
		Order("updated_at DESC").
		Find(&modelList).Error; err != nil {
		return nil, fmt.Errorf("failed to list topics: %w", err)
	}

	topics := make([]*types.Topic, 0, len(modelList))
	for _, model := range modelList {
		topics = append(topics, r.toDomain(&model))
	}

	return topics, nil
}

// Update updates an existing topic
func (r *TopicRepo) Update(ctx context.Context, topic *types.Topic) error {
	model := r.toModel(topic)
	if err := r.db.WithContext(ctx).Save(model).Error; err != nil {
		return fmt.Errorf("failed to update topic: %w", err)
	}
	return nil
}

// Delete deletes a topic by ID
func (r *TopicRepo) Delete(ctx context.Context, id string) error {
	if err := r.db.WithContext(ctx).Where("id = ?", id).Delete(&models.Topic{}).Error; err != nil {
		return fmt.Errorf("failed to delete topic: %w", err)
	}
	return nil
}

// DeleteByAssistant deletes all topics for an assistant
func (r *TopicRepo) DeleteByAssistant(ctx context.Context, assistantID string) error {
	if err := r.db.WithContext(ctx).
		Where("assistant_id = ?", assistantID).
		Delete(&models.Topic{}).Error; err != nil {
		return fmt.Errorf("failed to delete topics: %w", err)
	}
	return nil
}

// toModel converts domain topic to GORM model
func (r *TopicRepo) toModel(topic *types.Topic) *models.Topic {
	return &models.Topic{
		ID:          topic.ID,
		AssistantID: topic.AssistantID,
		Name:        topic.Name,
		CreatedAt:   topic.CreatedAt,
		UpdatedAt:   topic.UpdatedAt,
	}
}

// toDomain converts GORM model to domain topic
func (r *TopicRepo) toDomain(model *models.Topic) *types.Topic {
	return &types.Topic{
		ID:          model.ID,
		AssistantID: model.AssistantID,
		Name:        model.Name,
		CreatedAt:   model.CreatedAt,
		UpdatedAt:   model.UpdatedAt,
	}
}
