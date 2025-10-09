package llm

import (
	"context"
	"io"

	"github.com/lk2023060901/ai-writer-backend/internal/assistant/types"
)

// Provider 服务商适配器接口
type Provider interface {
	// Name 返回服务商名称
	Name() string

	// ChatStream 流式聊天（返回 channel）
	ChatStream(ctx context.Context, req *ChatRequest) (<-chan StreamEvent, error)

	// ValidateConfig 验证配置是否有效
	ValidateConfig() error

	// SupportedModels 返回支持的模型列表
	SupportedModels() []string

	// SupportsMultimodal 是否支持多模态
	SupportsMultimodal() bool
}

// ChatRequest 统一的聊天请求格式
type ChatRequest struct {
	// 消息内容
	Messages []Message `json:"messages"`

	// 模型配置
	Model       string   `json:"model"`
	Temperature *float64 `json:"temperature,omitempty"`
	MaxTokens   *int     `json:"max_tokens,omitempty"`
	TopP        *float64 `json:"top_p,omitempty"`
	Stream      bool     `json:"stream"`

	// 系统提示
	SystemPrompt string `json:"system,omitempty"`

	// 工具调用（可选）
	Tools []Tool `json:"tools,omitempty"`

	// 服务商特定选项
	ProviderOptions map[string]interface{} `json:"provider_options,omitempty"`
}

// Message 消息结构
type Message struct {
	Role    string         `json:"role"` // system | user | assistant
	Content []ContentBlock `json:"content"`
}

// ContentBlock 内容块
type ContentBlock struct {
	Type string `json:"type"` // text | image_url | file

	// 文本
	Text string `json:"text,omitempty"`

	// 图片（OpenAI/Anthropic 格式）
	ImageURL *ImageURL `json:"image_url,omitempty"`

	// 文件（Gemini/自定义格式）
	FileURL      string `json:"file_url,omitempty"`
	FileMimeType string `json:"mime_type,omitempty"`
}

// ImageURL 图片 URL 结构
type ImageURL struct {
	URL    string `json:"url"`
	Detail string `json:"detail,omitempty"` // auto | low | high
}

// Tool 工具定义
type Tool struct {
	Type     string   `json:"type"` // function
	Function Function `json:"function"`
}

// Function 函数定义
type Function struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters"`
}

// StreamEvent 流式事件
type StreamEvent struct {
	Type    EventType `json:"type"`
	Content string    `json:"content,omitempty"`
	Index   int       `json:"index,omitempty"`

	// 完成信息
	FinishReason string `json:"finish_reason,omitempty"`
	TokenCount   *int   `json:"token_count,omitempty"`

	// 错误信息
	Error error `json:"error,omitempty"`

	// 原始响应（用于调试）
	Raw interface{} `json:"raw,omitempty"`
}

// EventType 事件类型
type EventType string

const (
	EventStart  EventType = "start"
	EventToken  EventType = "token"
	EventDone   EventType = "done"
	EventError  EventType = "error"
	EventThink  EventType = "think" // 思考过程（Claude）
)

// ProviderConfig 服务商配置
type ProviderConfig struct {
	Provider string `json:"provider"`
	APIKey   string `json:"api_key"`
	BaseURL  string `json:"base_url,omitempty"`
	OrgID    string `json:"org_id,omitempty"` // OpenAI 组织 ID
}

// ProviderFactory 服务商工厂
type ProviderFactory interface {
	CreateProvider(config ProviderConfig) (Provider, error)
}

// StreamWriter SSE 流写入器
type StreamWriter interface {
	WriteEvent(eventType string, data interface{}) error
	WriteError(err error) error
	Flush() error
	Close() error
}

// MultiProviderOrchestrator 多服务商编排器
type MultiProviderOrchestrator interface {
	// ChatStreamMulti 并发调用多个服务商
	ChatStreamMulti(ctx context.Context, req *types.ChatRequest) (<-chan *types.ChatResponse, error)

	// RegisterProvider 注册服务商
	RegisterProvider(provider Provider) error

	// GetProvider 获取服务商实例
	GetProvider(name string) (Provider, error)
}

// ContextManager 上下文管理器
type ContextManager interface {
	// GetHistory 获取历史消息
	GetHistory(topicID string, limit int) ([]Message, error)

	// SaveMessage 保存消息
	SaveMessage(topicID string, role string, content []ContentBlock) error

	// BuildContext 构建完整上下文
	BuildContext(topicID string, newMessage string, contentBlocks []types.MessageContentBlock) ([]Message, error)
}

// WebSearchProvider 联网搜索提供者
type WebSearchProvider interface {
	// Search 执行搜索
	Search(ctx context.Context, query string, depth string) ([]types.WebSearchResult, error)

	// IsAvailable 检查服务是否可用
	IsAvailable() bool
}

// FileProcessor 文件处理器
type FileProcessor interface {
	// ProcessFile 处理上传的文件
	ProcessFile(ctx context.Context, fileURL string) (*ProcessedFile, error)

	// SupportedMimeTypes 支持的文件类型
	SupportedMimeTypes() []string
}

// ProcessedFile 处理后的文件
type ProcessedFile struct {
	URL          string
	MimeType     string
	Size         int64
	Content      string // 提取的文本内容
	Base64Data   string // Base64 编码数据（用于 API 调用）
	ThumbnailURL string // 缩略图（图片/视频）
}

// RateLimiter 速率限制器
type RateLimiter interface {
	// Allow 检查是否允许请求
	Allow(provider string, userID string) bool

	// Wait 等待直到允许请求
	Wait(ctx context.Context, provider string, userID string) error
}

// TokenCounter Token 计数器
type TokenCounter interface {
	// Count 计算消息的 token 数量
	Count(messages []Message, model string) (int, error)

	// EstimateCost 估算成本
	EstimateCost(inputTokens, outputTokens int, model string) (float64, error)
}

// ResponseCache 响应缓存
type ResponseCache interface {
	// Get 获取缓存的响应
	Get(key string) (*CachedResponse, bool)

	// Set 设置缓存
	Set(key string, response *CachedResponse) error

	// GenerateKey 生成缓存键
	GenerateKey(req *ChatRequest) string
}

// CachedResponse 缓存的响应
type CachedResponse struct {
	Content      string
	Provider     string
	Model        string
	TokenCount   int
	CachedAt     int64
	ExpiresAt    int64
}

// ErrorHandler 错误处理器
type ErrorHandler interface {
	// HandleError 处理错误
	HandleError(err error, provider string) *types.ChatResponse

	// IsRetryable 判断错误是否可重试
	IsRetryable(err error) bool

	// ShouldFallback 是否应该降级到其他服务商
	ShouldFallback(err error) bool
}

// MetricsCollector 指标收集器
type MetricsCollector interface {
	// RecordRequest 记录请求
	RecordRequest(provider, model string)

	// RecordLatency 记录延迟
	RecordLatency(provider, model string, duration float64)

	// RecordTokens 记录 token 使用
	RecordTokens(provider, model string, inputTokens, outputTokens int)

	// RecordError 记录错误
	RecordError(provider, model string, errType string)
}

// StreamMerger 流合并器（将多个服务商的流合并为一个）
type StreamMerger interface {
	// Merge 合并多个流
	Merge(streams []<-chan StreamEvent) <-chan *types.ChatResponse

	// Close 关闭合并器
	Close() error
}

// ProviderHealthChecker 服务商健康检查
type ProviderHealthChecker interface {
	// Check 检查服务商是否健康
	Check(provider string) error

	// GetStatus 获取所有服务商的状态
	GetStatus() map[string]HealthStatus
}

// HealthStatus 健康状态
type HealthStatus struct {
	Available  bool
	Latency    float64 // 毫秒
	ErrorRate  float64 // 错误率
	LastCheck  int64
}

// LoadBalancer 负载均衡器
type LoadBalancer interface {
	// SelectProvider 选择最优服务商
	SelectProvider(availableProviders []string) (string, error)

	// UpdateMetrics 更新指标
	UpdateMetrics(provider string, latency float64, success bool)
}

// StreamReader 流读取器辅助接口
type StreamReader interface {
	io.Reader
	ReadEvent() (*StreamEvent, error)
}
