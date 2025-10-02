package service

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/lk2023060901/ai-writer-backend/internal/user/biz"
	"go.uber.org/zap"
)

type UserService struct {
	uc     *biz.UserUseCase
	logger *zap.Logger
}

func NewUserService(uc *biz.UserUseCase, logger *zap.Logger) *UserService {
	return &UserService{
		uc:     uc,
		logger: logger,
	}
}

type CreateUserRequest struct {
	Name  string `json:"name" binding:"required"`
	Email string `json:"email" binding:"required,email"`
}

type UserResponse struct {
	ID        string  `json:"id"`
	Name      string `json:"name"`
	Email     string `json:"email"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

func (s *UserService) CreateUser(c *gin.Context) {
	var req CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user, err := s.uc.CreateUser(c.Request.Context(), req.Name, req.Email)
	if err != nil {
		s.logger.Error("failed to create user", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create user"})
		return
	}

	c.JSON(http.StatusCreated, s.toResponse(user))
}

func (s *UserService) GetUser(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user id"})
		return
	}

	user, err := s.uc.GetUser(c.Request.Context(), id)
	if err != nil {
		s.logger.Error("failed to get user", zap.Error(err))
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}

	c.JSON(http.StatusOK, s.toResponse(user))
}

func (s *UserService) ListUsers(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))

	users, err := s.uc.ListUsers(c.Request.Context(), page, pageSize)
	if err != nil {
		s.logger.Error("failed to list users", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list users"})
		return
	}

	responses := make([]*UserResponse, len(users))
	for i, user := range users {
		responses[i] = s.toResponse(user)
	}

	c.JSON(http.StatusOK, gin.H{"users": responses})
}

func (s *UserService) UpdateUser(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user id"})
		return
	}

	var req CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user := &biz.User{
		ID:    id,
		Name:  req.Name,
		Email: req.Email,
	}

	if err := s.uc.UpdateUser(c.Request.Context(), user); err != nil {
		s.logger.Error("failed to update user", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update user"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "user updated successfully"})
}

func (s *UserService) DeleteUser(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user id"})
		return
	}

	if err := s.uc.DeleteUser(c.Request.Context(), id); err != nil {
		s.logger.Error("failed to delete user", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete user"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "user deleted successfully"})
}

func (s *UserService) toResponse(user *biz.User) *UserResponse {
	return &UserResponse{
		ID:        user.ID,
		Name:      user.Name,
		Email:     user.Email,
		CreatedAt: user.CreatedAt.Format("2006-01-02 15:04:05"),
		UpdatedAt: user.UpdatedAt.Format("2006-01-02 15:04:05"),
	}
}

func (s *UserService) RegisterRoutes(r *gin.RouterGroup) {
	users := r.Group("/users")
	{
		users.POST("", s.CreateUser)
		users.GET("/:id", s.GetUser)
		users.GET("", s.ListUsers)
		users.PUT("/:id", s.UpdateUser)
		users.DELETE("/:id", s.DeleteUser)
	}
}
