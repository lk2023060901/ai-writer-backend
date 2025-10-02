package service

import "time"

// CreateAgentRequest 创建智能体请求
type CreateAgentRequest struct {
	Name             string   `json:"name" binding:"required,min=1,max=255"`
	Emoji            string   `json:"emoji" binding:"omitempty,max=10"`
	Prompt           string   `json:"prompt" binding:"required,min=10"`
	KnowledgeBaseIDs []string `json:"knowledge_base_ids"`
	Tags             []string `json:"tags" binding:"omitempty,max=10,dive,min=1,max=50"`
}

// UpdateAgentRequest 更新智能体请求
type UpdateAgentRequest struct {
	Name             *string  `json:"name" binding:"omitempty,min=1,max=255"`
	Emoji            *string  `json:"emoji" binding:"omitempty,max=10"`
	Prompt           *string  `json:"prompt" binding:"omitempty,min=10"`
	KnowledgeBaseIDs []string `json:"knowledge_base_ids"`
	Tags             []string `json:"tags" binding:"omitempty,max=10,dive,min=1,max=50"`
}

// ListAgentsRequest 列表查询请求
type ListAgentsRequest struct {
	Page      int      `form:"page" binding:"omitempty,min=1"`
	PageSize  int      `form:"page_size" binding:"omitempty,min=1,max=100"`
	IsEnabled *bool    `form:"is_enabled"`
	Tags      []string `form:"tags"`
	Keyword   string   `form:"keyword"`
}

// AgentResponse 智能体响应
type AgentResponse struct {
	// 公开字段（所有用户可见）
	ID               string   `json:"id"`
	Name             string   `json:"name"`
	Emoji            string   `json:"emoji"`
	Tags             []string `json:"tags"`
	KnowledgeBaseIDs []string `json:"knowledge_base_ids"`
	IsOfficial       bool     `json:"is_official"`
	IsEnabled        bool     `json:"is_enabled"`

	// 私有字段（仅所有者可见，官方智能体返回 nil）
	OwnerID   *string     `json:"owner_id,omitempty"`
	Prompt    *string     `json:"prompt,omitempty"`
	Type      *string     `json:"type,omitempty"`
	CreatedAt *time.Time  `json:"created_at,omitempty"`
	UpdatedAt *time.Time  `json:"updated_at,omitempty"`
}

// ListAgentsResponse 列表响应
type ListAgentsResponse struct {
	Items      []*AgentResponse    `json:"items"`
	Pagination *PaginationResponse `json:"pagination"`
}

// PaginationResponse 分页信息
type PaginationResponse struct {
	Page       int `json:"page"`
	PageSize   int `json:"page_size"`
	Total      int `json:"total"`
	TotalPages int `json:"total_pages"`
}
