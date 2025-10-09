package service

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/lk2023060901/ai-writer-backend/internal/assistant/biz"
)

// FavoriteService handles HTTP requests for favorite operations
type FavoriteService struct {
	useCase *biz.FavoriteUseCase
}

// NewFavoriteService creates a new favorite service
func NewFavoriteService(useCase *biz.FavoriteUseCase) *FavoriteService {
	return &FavoriteService{
		useCase: useCase,
	}
}

// RegisterRoutes registers favorite routes
func (s *FavoriteService) RegisterRoutes(r *gin.RouterGroup) {
	favorites := r.Group("/favorites")
	{
		favorites.GET("", s.ListFavorites)
		favorites.POST("", s.AddFavorite)
		favorites.DELETE("/:assistant_id", s.RemoveFavorite)
	}
}

// AddFavoriteRequest represents the request to add a favorite
type AddFavoriteRequest struct {
	AssistantID string `json:"assistant_id" binding:"required"`
}

// ListFavorites lists all favorite assistants for the current user
// @Summary List favorite assistants
// @Tags favorites
// @Produce json
// @Success 200 {array} types.AssistantFavoriteWithDetails
// @Router /api/v1/favorites [get]
func (s *FavoriteService) ListFavorites(c *gin.Context) {
	// Get user ID from context (set by JWT middleware)
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not authenticated"})
		return
	}

	favorites, err := s.useCase.ListFavorites(c.Request.Context(), userID.(string))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, favorites)
}

// AddFavorite adds an assistant to favorites
// @Summary Add assistant to favorites
// @Tags favorites
// @Accept json
// @Produce json
// @Param request body AddFavoriteRequest true "Assistant ID"
// @Success 200 {object} types.AssistantFavorite
// @Router /api/v1/favorites [post]
func (s *FavoriteService) AddFavorite(c *gin.Context) {
	// Get user ID from context (set by JWT middleware)
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not authenticated"})
		return
	}

	var req AddFavoriteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	favorite, err := s.useCase.AddFavorite(c.Request.Context(), userID.(string), req.AssistantID)
	if err != nil {
		// Check for specific errors
		if err.Error() == "assistant already in favorites" {
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, favorite)
}

// RemoveFavorite removes an assistant from favorites
// @Summary Remove assistant from favorites
// @Tags favorites
// @Param assistant_id path string true "Assistant ID"
// @Success 200 {object} map[string]string
// @Router /api/v1/favorites/{assistant_id} [delete]
func (s *FavoriteService) RemoveFavorite(c *gin.Context) {
	// Get user ID from context (set by JWT middleware)
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not authenticated"})
		return
	}

	assistantID := c.Param("assistant_id")
	if assistantID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "assistant_id is required"})
		return
	}

	if err := s.useCase.RemoveFavorite(c.Request.Context(), userID.(string), assistantID); err != nil {
		if err.Error() == "favorite not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Favorite removed successfully"})
}
