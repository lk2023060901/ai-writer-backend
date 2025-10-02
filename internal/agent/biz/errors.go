package biz

import "errors"

var (
	// ErrAgentNameRequired 智能体名称必填
	ErrAgentNameRequired = errors.New("agent name is required")

	// ErrAgentPromptRequired 提示词必填
	ErrAgentPromptRequired = errors.New("agent prompt is required")

	// ErrAgentPromptTooShort 提示词过短
	ErrAgentPromptTooShort = errors.New("agent prompt must be at least 10 characters")

	// ErrAgentNotFound 智能体不存在
	ErrAgentNotFound = errors.New("agent not found")

	// ErrAgentUnauthorized 无权访问该智能体
	ErrAgentUnauthorized = errors.New("unauthorized to access this agent")

	// ErrAgentTagsInvalid 标签格式错误
	ErrAgentTagsInvalid = errors.New("agent tags invalid")

	// ErrAgentKnowledgeBaseInvalid 知识库ID无效
	ErrAgentKnowledgeBaseInvalid = errors.New("knowledge base id invalid")
)
