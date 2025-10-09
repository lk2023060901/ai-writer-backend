package intent

import (
	"regexp"
	"strings"
)

// IntentType 意图类型
type IntentType string

const (
	IntentNeedSearch    IntentType = "need_search"    // 需要搜索
	IntentNoSearch      IntentType = "no_search"      // 不需要搜索
	IntentUncertain     IntentType = "uncertain"      // 不确定
)

// Intent 意图识别结果
type Intent struct {
	Type       IntentType // 意图类型
	Confidence float64    // 置信度 (0.0-1.0)
	Reason     string     // 判断原因
}

// Recognizer 意图识别器（规则版本）
type Recognizer struct {
	// 问候语模式
	greetingPatterns []*regexp.Regexp
	// 写作任务模式
	writingPatterns []*regexp.Regexp
	// 问题模式
	questionPatterns []*regexp.Regexp
}

// NewRecognizer 创建意图识别器
func NewRecognizer() *Recognizer {
	return &Recognizer{
		greetingPatterns: []*regexp.Regexp{
			regexp.MustCompile(`(?i)how are you|how's it going|how do you do`),
			regexp.MustCompile(`(?i)^(hi|hello|hey|你好|您好|早上好|下午好|晚上好)[\s,!\?！。\.]*`),
			regexp.MustCompile(`(?i)^(good morning|good afternoon|good evening)[\s\?！!。\.]*$`),
		},
		writingPatterns: []*regexp.Regexp{
			regexp.MustCompile(`(?i)^(写|帮我写|请写|生成|创建|create|write|generate|compose)`),
			regexp.MustCompile(`(?i)(文章|报告|邮件|email|essay|article|document|story|poem|code)`),
			regexp.MustCompile(`(?i)(翻译|translate|paraphrase|改写|rewrite|summarize|总结)`),
		},
		questionPatterns: []*regexp.Regexp{
			regexp.MustCompile(`(?i)^(what|who|when|where|why|how|which|什么|为什么|怎么|如何|哪个|谁|何时|哪里)`),
			regexp.MustCompile(`(?i)\?$|？$`), // 以问号结尾
			regexp.MustCompile(`(?i)(explain|describe|tell me|介绍|解释|说明|讲解)`),
		},
	}
}

// Recognize 识别用户意图
func (r *Recognizer) Recognize(query string) *Intent {
	query = strings.TrimSpace(query)

	// 空查询
	if query == "" {
		return &Intent{
			Type:       IntentNoSearch,
			Confidence: 1.0,
			Reason:     "empty query",
		}
	}

	// 检查是否为问候语
	if r.isGreeting(query) {
		return &Intent{
			Type:       IntentNoSearch,
			Confidence: 0.9,
			Reason:     "greeting message",
		}
	}

	// 检查是否为写作任务
	if r.isWritingTask(query) {
		return &Intent{
			Type:       IntentNoSearch,
			Confidence: 0.8,
			Reason:     "writing task",
		}
	}

	// 检查是否为问题
	if r.isQuestion(query) {
		return &Intent{
			Type:       IntentNeedSearch,
			Confidence: 0.7,
			Reason:     "question pattern detected",
		}
	}

	// 查询较长（>20个字符）可能需要搜索
	if len(query) > 20 {
		return &Intent{
			Type:       IntentNeedSearch,
			Confidence: 0.6,
			Reason:     "long query, might need context",
		}
	}

	// 默认不确定
	return &Intent{
		Type:       IntentUncertain,
		Confidence: 0.5,
		Reason:     "unclear intent",
	}
}

// isGreeting 检查是否为问候语
func (r *Recognizer) isGreeting(query string) bool {
	for _, pattern := range r.greetingPatterns {
		if pattern.MatchString(query) {
			return true
		}
	}
	return false
}

// isWritingTask 检查是否为写作任务
func (r *Recognizer) isWritingTask(query string) bool {
	for _, pattern := range r.writingPatterns {
		if pattern.MatchString(query) {
			return true
		}
	}
	return false
}

// isQuestion 检查是否为问题
func (r *Recognizer) isQuestion(query string) bool {
	for _, pattern := range r.questionPatterns {
		if pattern.MatchString(query) {
			return true
		}
	}
	return false
}
