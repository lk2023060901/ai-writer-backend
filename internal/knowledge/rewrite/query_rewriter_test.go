package rewrite

import (
	"testing"
	"time"
)

func TestQueryRewriter_shouldRewrite(t *testing.T) {
	rewriter := NewQueryRewriter(nil, true)

	tests := []struct {
		name     string
		query    string
		expected bool
		reason   string
	}{
		// 不需要重写的情况
		{
			name:     "simple english query",
			query:    "What is Docker?",
			expected: false,
			reason:   "标准英文问题，不需要重写",
		},
		{
			name:     "explain query",
			query:    "Explain the concept of Kubernetes",
			expected: false,
			reason:   "标准英文查询",
		},
		{
			name:     "too short",
			query:    "Hi",
			expected: false,
			reason:   "太短",
		},

		// 需要重写的情况
		{
			name:     "colloquial chinese",
			query:    "Docker是啥？",
			expected: true,
			reason:   "口语化中文",
		},
		{
			name:     "mixed language",
			query:    "用 Kubernetes 部署 app",
			expected: true,
			reason:   "中英混合",
		},
		{
			name:     "has pronoun",
			query:    "它的优势是什么？",
			expected: true,
			reason:   "包含代词，需要上下文",
		},
		{
			name:     "colloquial marker",
			query:    "这个东西是干啥的？",
			expected: true,
			reason:   "口语化",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := rewriter.shouldRewrite(tt.query)
			if result != tt.expected {
				t.Errorf("shouldRewrite(%q) = %v, expected %v (reason: %s)",
					tt.query, result, tt.expected, tt.reason)
			}
		})
	}
}

func TestQueryRewriter_isSimpleEnglishQuery(t *testing.T) {
	rewriter := NewQueryRewriter(nil, true)

	simpleQueries := []string{
		"What is Docker?",
		"How does Kubernetes work?",
		"Why is this important?",
		"Explain the concept",
		"Describe the process",
	}

	for _, query := range simpleQueries {
		if !rewriter.isSimpleEnglishQuery(query) {
			t.Errorf("Expected %q to be simple English query", query)
		}
	}
}

func TestQueryRewriter_isColloquialQuery(t *testing.T) {
	rewriter := NewQueryRewriter(nil, true)

	colloquialQueries := []string{
		"Docker是啥？",
		"这个咋用？",
		"有啥用？",
		"是什么东西？",
	}

	for _, query := range colloquialQueries {
		if !rewriter.isColloquialQuery(query) {
			t.Errorf("Expected %q to be colloquial query", query)
		}
	}
}

func TestQueryRewriter_isMixedLanguageQuery(t *testing.T) {
	rewriter := NewQueryRewriter(nil, true)

	mixedQueries := []string{
		"用 Docker 部署应用",
		"Kubernetes 编排容器",
		"如何使用 Redis",
	}

	for _, query := range mixedQueries {
		if !rewriter.isMixedLanguageQuery(query) {
			t.Errorf("Expected %q to be mixed language query", query)
		}
	}
}

func TestQueryRewriter_hasPronoun(t *testing.T) {
	rewriter := NewQueryRewriter(nil, true)

	queriesWithPronouns := []string{
		"它的作用是什么？",
		"这个怎么用？",
		"那些功能有哪些？",
		"What does it do?",
		"How does this work?",
	}

	for _, query := range queriesWithPronouns {
		if !rewriter.hasPronoun(query) {
			t.Errorf("Expected %q to have pronoun", query)
		}
	}
}

func TestRewriteCache(t *testing.T) {
	cache := &RewriteCache{
		cache: make(map[string]*CacheEntry),
		ttl:   1 * time.Hour,
	}

	// 测试设置和获取
	cache.Set("test query", "rewritten query")
	result := cache.Get("test query")
	if result != "rewritten query" {
		t.Errorf("Expected 'rewritten query', got %q", result)
	}

	// 测试不存在的key
	result = cache.Get("nonexistent")
	if result != "" {
		t.Errorf("Expected empty string for nonexistent key, got %q", result)
	}
}
