package intent

import (
	"testing"
)

func TestRecognizer_Recognize(t *testing.T) {
	recognizer := NewRecognizer()

	tests := []struct {
		name           string
		query          string
		expectedType   IntentType
		minConfidence  float64
	}{
		// 问候语测试
		{
			name:          "greeting - hi",
			query:         "Hi",
			expectedType:  IntentNoSearch,
			minConfidence: 0.8,
		},
		{
			name:          "greeting - hello",
			query:         "Hello, how are you?",
			expectedType:  IntentNoSearch,
			minConfidence: 0.8,
		},
		{
			name:          "greeting - chinese",
			query:         "你好",
			expectedType:  IntentNoSearch,
			minConfidence: 0.8,
		},
		{
			name:          "greeting - good morning",
			query:         "Good morning",
			expectedType:  IntentNoSearch,
			minConfidence: 0.8,
		},

		// 写作任务测试
		{
			name:          "writing - write article",
			query:         "Write an article about AI",
			expectedType:  IntentNoSearch,
			minConfidence: 0.7,
		},
		{
			name:          "writing - chinese",
			query:         "帮我写一篇关于人工智能的文章",
			expectedType:  IntentNoSearch,
			minConfidence: 0.7,
		},
		{
			name:          "writing - generate code",
			query:         "Generate a Python script for data analysis",
			expectedType:  IntentNoSearch,
			minConfidence: 0.7,
		},
		{
			name:          "writing - translate",
			query:         "Translate this text to English",
			expectedType:  IntentNoSearch,
			minConfidence: 0.7,
		},

		// 问题测试
		{
			name:          "question - what",
			query:         "What is Docker?",
			expectedType:  IntentNeedSearch,
			minConfidence: 0.6,
		},
		{
			name:          "question - how",
			query:         "How does Kubernetes work?",
			expectedType:  IntentNeedSearch,
			minConfidence: 0.6,
		},
		{
			name:          "question - chinese",
			query:         "什么是微服务架构？",
			expectedType:  IntentNeedSearch,
			minConfidence: 0.6,
		},
		{
			name:          "question - explain",
			query:         "Explain the concept of vector embeddings",
			expectedType:  IntentNeedSearch,
			minConfidence: 0.6,
		},
		{
			name:          "question - ending with ?",
			query:         "Can you tell me about Redis?",
			expectedType:  IntentNeedSearch,
			minConfidence: 0.6,
		},

		// 边界情况
		{
			name:          "empty query",
			query:         "",
			expectedType:  IntentNoSearch,
			minConfidence: 1.0,
		},
		{
			name:          "short query",
			query:         "Redis",
			expectedType:  IntentUncertain,
			minConfidence: 0.4,
		},
		{
			name:          "long query without clear pattern",
			query:         "I'm working on a project that needs to handle large amounts of data",
			expectedType:  IntentNeedSearch,
			minConfidence: 0.5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := recognizer.Recognize(tt.query)

			if result.Type != tt.expectedType {
				t.Errorf("Expected intent type %s, got %s (reason: %s)",
					tt.expectedType, result.Type, result.Reason)
			}

			if result.Confidence < tt.minConfidence {
				t.Errorf("Expected confidence >= %.2f, got %.2f",
					tt.minConfidence, result.Confidence)
			}

			t.Logf("Query: %q -> Type: %s, Confidence: %.2f, Reason: %s",
				tt.query, result.Type, result.Confidence, result.Reason)
		})
	}
}

func TestRecognizer_isGreeting(t *testing.T) {
	recognizer := NewRecognizer()

	greetings := []string{
		"Hi",
		"Hello",
		"Hey",
		"你好",
		"Good morning",
		"How are you?",
	}

	for _, greeting := range greetings {
		if !recognizer.isGreeting(greeting) {
			t.Errorf("Expected %q to be recognized as greeting", greeting)
		}
	}
}

func TestRecognizer_isWritingTask(t *testing.T) {
	recognizer := NewRecognizer()

	writingTasks := []string{
		"Write an article",
		"帮我写一封邮件",
		"Generate a report",
		"Translate this",
		"Create a document",
	}

	for _, task := range writingTasks {
		if !recognizer.isWritingTask(task) {
			t.Errorf("Expected %q to be recognized as writing task", task)
		}
	}
}

func TestRecognizer_isQuestion(t *testing.T) {
	recognizer := NewRecognizer()

	questions := []string{
		"What is AI?",
		"How does it work?",
		"什么是机器学习？",
		"Explain the concept",
		"Can you describe this?",
	}

	for _, question := range questions {
		if !recognizer.isQuestion(question) {
			t.Errorf("Expected %q to be recognized as question", question)
		}
	}
}
