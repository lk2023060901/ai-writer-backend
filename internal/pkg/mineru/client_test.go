package mineru

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/lk2023060901/ai-writer-backend/internal/pkg/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testBaseURL  = "https://mineru.net"
	testAPIKey   = "eyJ0eXBlIjoiSldUIiwiYWxnIjoiSFM1MTIifQ.eyJqdGkiOiI2MjEwMDIzMiIsInJvbCI6IlJPTEVfUkVHSVNURVIiLCJpc3MiOiJPcGVuWExhYiIsImlhdCI6MTc1OTMxNjE3MCwiY2xpZW50SWQiOiJsa3pkeDU3bnZ5MjJqa3BxOXgydyIsInBob25lIjoiIiwib3BlbklkIjpudWxsLCJ1dWlkIjoiMWUzY2Q2OGItZDQ1MS00NDUzLWFmNjktZWQ5NmRmODYyYmJiIiwiZW1haWwiOiIiLCJleHAiOjE3NjA1MjU3NzB9.NW7unuR50pO9zJIGCdD3nSZDXQkYoCcIRgS41sRg6Y7DUihH8Cp4kZVJpt8t1ECIBiDwa4OMt-1mVq4t_focVg"
	testFilePath = "/Volumes/work/cherry-studio/益禾堂/益禾堂9月8日调研.docx"
)

func setupTestClient(t *testing.T) *Client {
	cfg := &Config{
		BaseURL:         testBaseURL,
		APIKey:          testAPIKey,
		Timeout:         30 * time.Second,
		MaxRetries:      3,
		DefaultLanguage: "ch",
		EnableFormula:   true,
		EnableTable:     true,
		ModelVersion:    "pipeline",
	}

	log, err := logger.New(&logger.Config{
		Level:  "debug",
		Format: "json",
		Output: "console",
	})
	require.NoError(t, err)

	client, err := New(cfg, log)
	require.NoError(t, err)

	return client
}

func TestNew(t *testing.T) {
	log, err := logger.New(&logger.Config{
		Level:  "info",
		Format: "json",
		Output: "console",
	})
	require.NoError(t, err)

	tests := []struct {
		name    string
		config  *Config
		wantErr bool
	}{
		{
			name: "valid config with real credentials",
			config: &Config{
				BaseURL: testBaseURL,
				APIKey:  testAPIKey,
			},
			wantErr: false,
		},
		{
			name: "missing base url",
			config: &Config{
				APIKey: testAPIKey,
			},
			wantErr: true,
		},
		{
			name: "missing api key",
			config: &Config{
				BaseURL: testBaseURL,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := New(tt.config, log)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, client)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, client)
				if client != nil {
					client.Close()
				}
			}
		})
	}
}

func TestCreateTaskWithFile_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skip integration test")
	}

	client := setupTestClient(t)
	defer client.Close()

	t.Run("upload and parse document", func(t *testing.T) {
		if _, err := os.Stat(testFilePath); os.IsNotExist(err) {
			t.Skipf("test file not found: %s", testFilePath)
		}

		fileData, err := os.ReadFile(testFilePath)
		require.NoError(t, err)

		result, err := client.CreateTaskWithFile(
			context.Background(),
			"test-document.docx",
			fileData,
			&CreateTaskRequest{
				IsOCR:    true,
				Language: "ch",
				// ExtraFormats: []string{"md", "docx"}, // API 不支持这些格式
			},
		)

		require.NoError(t, err)

		// 记录 API 返回的完整结果
		t.Logf("=== MinerU API Response ===")
		t.Logf("TaskID: %s", result.TaskID)
		t.Logf("DataID: %s", result.DataID)
		t.Logf("State: %s", result.State)
		t.Logf("FullZipURL: %s", result.FullZipURL)
		t.Logf("ErrMsg: %s", result.ErrMsg)
		if result.ExtractProgress != nil {
			t.Logf("ExtractProgress.ExtractedPages: %d", result.ExtractProgress.ExtractedPages)
			t.Logf("ExtractProgress.TotalPages: %d", result.ExtractProgress.TotalPages)
			t.Logf("ExtractProgress.StartTime: %s", result.ExtractProgress.StartTime)
		} else {
			t.Logf("ExtractProgress: nil")
		}
		t.Logf("===========================")

		assert.NotNil(t, result)
		assert.NotEmpty(t, result.TaskID)
		assert.Equal(t, TaskStateDone, result.State)
		assert.NotEmpty(t, result.FullZipURL)
	})
}

func TestApplyDefaultTaskConfig(t *testing.T) {
	client := setupTestClient(t)
	defer client.Close()

	t.Run("apply defaults when fields are empty", func(t *testing.T) {
		req := &CreateTaskRequest{
			URL: "https://example.com/test.pdf",
		}

		client.applyDefaultTaskConfig(req)

		assert.Equal(t, "ch", req.Language)
		assert.NotNil(t, req.EnableFormula)
		assert.True(t, *req.EnableFormula)
		assert.NotNil(t, req.EnableTable)
		assert.True(t, *req.EnableTable)
		assert.Equal(t, "pipeline", req.ModelVersion)
	})

	t.Run("keep existing values", func(t *testing.T) {
		enableFormula := false
		enableTable := true
		req := &CreateTaskRequest{
			URL:           "https://example.com/test.pdf",
			Language:      "en",
			EnableFormula: &enableFormula,
			EnableTable:   &enableTable,
			ModelVersion:  "vlm",
		}

		client.applyDefaultTaskConfig(req)

		assert.Equal(t, "en", req.Language)
		assert.False(t, *req.EnableFormula)
		assert.True(t, *req.EnableTable)
		assert.Equal(t, "vlm", req.ModelVersion)
	})
}

func TestClose(t *testing.T) {
	client := setupTestClient(t)
	err := client.Close()
	assert.NoError(t, err)
}
