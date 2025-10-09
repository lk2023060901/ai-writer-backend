package rewrite

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"
)

// QueryRewriter 查询重写器
type QueryRewriter struct {
	cache      *RewriteCache
	llmClient  LLMClient
	enabled    bool
	maxRetries int
}

// LLMClient LLM 客户端接口
type LLMClient interface {
	Rewrite(ctx context.Context, query string, chatHistory string) (string, error)
}

// RewriteCache 重写结果缓存
type RewriteCache struct {
	cache map[string]*CacheEntry
	mu    sync.RWMutex
	ttl   time.Duration
}

// CacheEntry 缓存条目
type CacheEntry struct {
	Rewritten string
	Timestamp time.Time
}

// NewQueryRewriter 创建查询重写器
func NewQueryRewriter(llmClient LLMClient, enabled bool) *QueryRewriter {
	return &QueryRewriter{
		cache: &RewriteCache{
			cache: make(map[string]*CacheEntry),
			ttl:   24 * time.Hour, // 缓存 24 小时
		},
		llmClient:  llmClient,
		enabled:    enabled,
		maxRetries: 2,
	}
}

// Rewrite 重写查询（智能路由）
func (r *QueryRewriter) Rewrite(ctx context.Context, query string, chatHistory string) (string, error) {
	// 如果未启用，直接返回原始查询
	if !r.enabled || r.llmClient == nil {
		return query, nil
	}

	// 检查是否需要重写
	if !r.shouldRewrite(query) {
		return query, nil
	}

	// 检查缓存
	if cached := r.cache.Get(query); cached != "" {
		return cached, nil
	}

	// LLM 重写
	rewritten, err := r.llmClient.Rewrite(ctx, query, chatHistory)
	if err != nil {
		// 重写失败，返回原始查询（降级处理）
		return query, nil
	}

	// 缓存结果
	r.cache.Set(query, rewritten)

	return rewritten, nil
}

// shouldRewrite 判断是否需要重写
func (r *QueryRewriter) shouldRewrite(query string) bool {
	query = strings.TrimSpace(query)

	// 1. 太短的查询不需要重写
	if len(query) < 5 {
		return false
	}

	// 2. 简单的英文查询不需要重写（如 "What is Docker?"）
	if r.isSimpleEnglishQuery(query) {
		return false
	}

	// 3. 口语化查询需要重写（如 "Docker是啥？"）
	if r.isColloquialQuery(query) {
		return true
	}

	// 4. 中英混合查询需要重写
	if r.isMixedLanguageQuery(query) {
		return true
	}

	// 5. 包含代词的查询需要重写（依赖上下文）
	if r.hasPronoun(query) {
		return true
	}

	// 默认不重写
	return false
}

// isSimpleEnglishQuery 检查是否为简单英文查询
func (r *QueryRewriter) isSimpleEnglishQuery(query string) bool {
	// 标准的英文问题模式
	patterns := []string{
		`^(What|How|Why|When|Where|Who|Which) (is|are|was|were|do|does|did) `,
		`^(Explain|Describe|Tell me about) `,
	}

	for _, pattern := range patterns {
		matched, _ := regexp.MatchString(pattern, query)
		if matched {
			return true
		}
	}

	return false
}

// isColloquialQuery 检查是否为口语化查询
func (r *QueryRewriter) isColloquialQuery(query string) bool {
	colloquialMarkers := []string{
		"是啥", "咋", "怎么弄", "咋整", "咋办",
		"是什么东西", "干啥的", "有啥用",
	}

	queryLower := strings.ToLower(query)
	for _, marker := range colloquialMarkers {
		if strings.Contains(queryLower, marker) {
			return true
		}
	}

	return false
}

// isMixedLanguageQuery 检查是否为中英混合查询
func (r *QueryRewriter) isMixedLanguageQuery(query string) bool {
	hasChinese := regexp.MustCompile(`[\p{Han}]`).MatchString(query)
	hasEnglish := regexp.MustCompile(`[a-zA-Z]`).MatchString(query)
	return hasChinese && hasEnglish
}

// hasPronoun 检查是否包含代词（需要上下文）
func (r *QueryRewriter) hasPronoun(query string) bool {
	// 中文代词（直接包含即可）
	chinesePronouns := []string{
		"它", "他", "她", "这个", "那个", "这些", "那些", "它们",
	}

	for _, pronoun := range chinesePronouns {
		if strings.Contains(query, pronoun) {
			return true
		}
	}

	// 英文代词（需要词边界）
	englishPronouns := []string{
		"it", "this", "that", "these", "those", "they", "them",
	}

	queryLower := strings.ToLower(query)
	for _, pronoun := range englishPronouns {
		pattern := fmt.Sprintf(`\b%s\b`, pronoun)
		matched, _ := regexp.MatchString(pattern, queryLower)
		if matched {
			return true
		}
	}

	return false
}

// Get 从缓存获取
func (c *RewriteCache) Get(query string) string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, exists := c.cache[query]
	if !exists {
		return ""
	}

	// 检查是否过期
	if time.Since(entry.Timestamp) > c.ttl {
		return ""
	}

	return entry.Rewritten
}

// Set 设置缓存
func (c *RewriteCache) Set(query, rewritten string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.cache[query] = &CacheEntry{
		Rewritten: rewritten,
		Timestamp: time.Now(),
	}

	// 简单的缓存清理（当缓存过大时）
	if len(c.cache) > 10000 {
		c.cleanup()
	}
}

// cleanup 清理过期缓存
func (c *RewriteCache) cleanup() {
	now := time.Now()
	for key, entry := range c.cache {
		if now.Sub(entry.Timestamp) > c.ttl {
			delete(c.cache, key)
		}
	}
}
