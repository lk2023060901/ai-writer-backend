package biz

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/lk2023060901/ai-writer-backend/internal/assistant/types"
)

// FavoriteRepo defines the repository interface for favorite operations
type FavoriteRepo interface {
	Create(ctx context.Context, favorite *types.AssistantFavorite) error
	Delete(ctx context.Context, userID, assistantID string) error
	ListByUser(ctx context.Context, userID string) ([]*types.AssistantFavoriteWithDetails, error)
	Exists(ctx context.Context, userID, assistantID string) (bool, error)
	GetMaxSortOrder(ctx context.Context, userID string) (int, error)
}

// FavoriteUseCase contains business logic for favorite operations
type FavoriteUseCase struct {
	repo FavoriteRepo
}

// NewFavoriteUseCase creates a new favorite use case
func NewFavoriteUseCase(repo FavoriteRepo) *FavoriteUseCase {
	return &FavoriteUseCase{
		repo: repo,
	}
}

// AddFavorite adds an assistant to user's favorites
func (uc *FavoriteUseCase) AddFavorite(ctx context.Context, userID, assistantID string) (*types.AssistantFavorite, error) {
	if userID == "" {
		return nil, fmt.Errorf("user ID is required")
	}
	if assistantID == "" {
		return nil, fmt.Errorf("assistant ID is required")
	}

	// Check if already exists
	exists, err := uc.repo.Exists(ctx, userID, assistantID)
	if err != nil {
		return nil, fmt.Errorf("failed to check favorite existence: %w", err)
	}
	if exists {
		return nil, fmt.Errorf("assistant already in favorites")
	}

	// Get max sort order and add 1
	maxOrder, err := uc.repo.GetMaxSortOrder(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get max sort order: %w", err)
	}

	favorite := &types.AssistantFavorite{
		ID:          uuid.New().String(),
		UserID:      userID,
		AssistantID: assistantID,
		SortOrder:   maxOrder + 1,
		CreatedAt:   time.Now(),
	}

	if err := uc.repo.Create(ctx, favorite); err != nil {
		return nil, fmt.Errorf("failed to add favorite: %w", err)
	}

	return favorite, nil
}

// RemoveFavorite removes an assistant from user's favorites
func (uc *FavoriteUseCase) RemoveFavorite(ctx context.Context, userID, assistantID string) error {
	if userID == "" {
		return fmt.Errorf("user ID is required")
	}
	if assistantID == "" {
		return fmt.Errorf("assistant ID is required")
	}

	if err := uc.repo.Delete(ctx, userID, assistantID); err != nil {
		return fmt.Errorf("failed to remove favorite: %w", err)
	}

	return nil
}

// ListFavorites lists all favorites for a user
func (uc *FavoriteUseCase) ListFavorites(ctx context.Context, userID string) ([]*types.AssistantFavoriteWithDetails, error) {
	if userID == "" {
		return nil, fmt.Errorf("user ID is required")
	}

	favorites, err := uc.repo.ListByUser(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to list favorites: %w", err)
	}

	return favorites, nil
}
