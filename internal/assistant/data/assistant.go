package data

import (
	"context"
	"fmt"

	"github.com/lk2023060901/ai-writer-backend/internal/assistant/models"
	"github.com/lk2023060901/ai-writer-backend/internal/assistant/types"

	"gorm.io/gorm"
)

// AssistantRepo implements the assistant repository using GORM
type AssistantRepo struct {
	db *gorm.DB
}

// NewAssistantRepo creates a new assistant repository
func NewAssistantRepo(db *gorm.DB) *AssistantRepo {
	return &AssistantRepo{db: db}
}

// Create creates a new assistant
func (r *AssistantRepo) Create(ctx context.Context, assistant *types.Assistant) error {
	model := r.toModel(assistant)
	if err := r.db.WithContext(ctx).Create(model).Error; err != nil {
		return fmt.Errorf("failed to create assistant: %w", err)
	}
	return nil
}

// GetByID retrieves an assistant by ID and user ID
func (r *AssistantRepo) GetByID(ctx context.Context, id, userID string) (*types.Assistant, error) {
	var model models.Assistant
	if err := r.db.WithContext(ctx).
		Where("id = ? AND user_id = ?", id, userID).
		First(&model).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("assistant not found")
		}
		return nil, fmt.Errorf("failed to get assistant: %w", err)
	}

	return r.toDomain(&model)
}

// List lists assistants for a user with optional filtering
func (r *AssistantRepo) List(ctx context.Context, userID string, filter *types.AssistantFilter) ([]*types.Assistant, error) {
	query := r.db.WithContext(ctx).Model(&models.Assistant{}).Where("user_id = ?", userID)

	// Apply filters
	if filter != nil {
		if len(filter.Tags) > 0 {
			// Query JSON array for tag membership
			for _, tag := range filter.Tags {
				query = query.Where("JSON_CONTAINS(tags, ?)", fmt.Sprintf(`"%s"`, tag))
			}
		}
		if filter.Keyword != "" {
			keyword := "%" + filter.Keyword + "%"
			query = query.Where("name LIKE ?", keyword)
		}
	}

	var modelList []models.Assistant
	if err := query.Order("updated_at DESC").Find(&modelList).Error; err != nil {
		return nil, fmt.Errorf("failed to list assistants: %w", err)
	}

	assistants := make([]*types.Assistant, 0, len(modelList))
	for _, model := range modelList {
		assistant, err := r.toDomain(&model)
		if err != nil {
			return nil, err
		}
		assistants = append(assistants, assistant)
	}

	return assistants, nil
}

// Update updates an existing assistant
func (r *AssistantRepo) Update(ctx context.Context, assistant *types.Assistant) error {
	model := r.toModel(assistant)
	if err := r.db.WithContext(ctx).Save(model).Error; err != nil {
		return fmt.Errorf("failed to update assistant: %w", err)
	}
	return nil
}

// Delete deletes an assistant by ID and user ID
func (r *AssistantRepo) Delete(ctx context.Context, id, userID string) error {
	if err := r.db.WithContext(ctx).
		Where("id = ? AND user_id = ?", id, userID).
		Delete(&models.Assistant{}).Error; err != nil {
		return fmt.Errorf("failed to delete assistant: %w", err)
	}
	return nil
}

// toModel converts domain assistant to GORM model
func (r *AssistantRepo) toModel(assistant *types.Assistant) *models.Assistant {
	return &models.Assistant{
		ID:               assistant.ID,
		UserID:           assistant.UserID,
		Name:             assistant.Name,
		Emoji:            assistant.Emoji,
		Prompt:           assistant.Prompt,
		Type:             assistant.Type,
		Tags:             models.StringArray(assistant.Tags),
		KnowledgeBaseIDs: models.StringArray(assistant.KnowledgeBaseIDs),
		CreatedAt:        assistant.CreatedAt,
		UpdatedAt:        assistant.UpdatedAt,
	}
}

// toDomain converts GORM model to domain assistant
func (r *AssistantRepo) toDomain(model *models.Assistant) (*types.Assistant, error) {
	return &types.Assistant{
		ID:               model.ID,
		UserID:           model.UserID,
		Name:             model.Name,
		Emoji:            model.Emoji,
		Prompt:           model.Prompt,
		Type:             model.Type,
		Tags:             []string(model.Tags),
		KnowledgeBaseIDs: []string(model.KnowledgeBaseIDs),
		CreatedAt:        model.CreatedAt,
		UpdatedAt:        model.UpdatedAt,
	}, nil
}
