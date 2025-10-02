package mineru

import "time"

// TaskState 任务状态
type TaskState string

const (
	TaskStateDone        TaskState = "done"        // 完成
	TaskStatePending     TaskState = "pending"     // 排队中
	TaskStateRunning     TaskState = "running"     // 正在解析
	TaskStateFailed      TaskState = "failed"      // 解析失败
	TaskStateConverting  TaskState = "converting"  // 格式转换中
	TaskStateWaitingFile TaskState = "waiting-file" // 等待文件上传
)

// APIResponse 通用 API 响应
type APIResponse struct {
	Code    int    `json:"code"`              // 旧版 API 使用
	MsgCode string `json:"msgCode,omitempty"` // 新版 API 使用
	Msg     string `json:"msg"`
	TraceID string `json:"trace_id,omitempty"` // 旧版 API 使用
	TraceId string `json:"traceId,omitempty"`  // 新版 API 使用（注意大小写）
	Success *bool  `json:"success,omitempty"`  // 新版 API 使用
}

// CreateTaskRequest 创建单个解析任务请求
type CreateTaskRequest struct {
	// URL 文件 URL (必填)
	URL string `json:"url"`

	// IsOCR 是否启用 OCR
	IsOCR bool `json:"is_ocr,omitempty"`

	// EnableFormula 是否开启公式识别
	EnableFormula *bool `json:"enable_formula,omitempty"`

	// EnableTable 是否开启表格识别
	EnableTable *bool `json:"enable_table,omitempty"`

	// Language 指定文档语言
	Language string `json:"language,omitempty"`

	// DataID 解析对象对应的数据 ID
	DataID string `json:"data_id,omitempty"`

	// Callback 解析结果回调 URL
	Callback string `json:"callback,omitempty"`

	// Seed 随机字符串，用于回调签名
	Seed string `json:"seed,omitempty"`

	// ExtraFormats 额外导出格式 (docx, html, latex)
	ExtraFormats []string `json:"extra_formats,omitempty"`

	// PageRanges 指定页码范围
	PageRanges string `json:"page_ranges,omitempty"`

	// ModelVersion 模型版本 (pipeline/vlm)
	ModelVersion string `json:"model_version,omitempty"`
}

// CreateTaskResponse 创建任务响应
type CreateTaskResponse struct {
	APIResponse
	Data struct {
		TaskID string `json:"task_id"`
	} `json:"data"`
}

// ExtractProgress 解析进度
type ExtractProgress struct {
	ExtractedPages int    `json:"extracted_pages"` // 已解析页数
	TotalPages     int    `json:"total_pages"`     // 总页数
	StartTime      string `json:"start_time"`      // 开始时间
}

// TaskResult 任务结果
type TaskResult struct {
	TaskID          string           `json:"task_id"`
	DataID          string           `json:"data_id,omitempty"`
	State           TaskState        `json:"state"`
	FullZipURL      string           `json:"full_zip_url,omitempty"`
	ErrMsg          string           `json:"err_msg,omitempty"`
	ExtractProgress *ExtractProgress `json:"extract_progress,omitempty"`
}

// GetTaskResultResponse 获取任务结果响应
type GetTaskResultResponse struct {
	APIResponse
	Data TaskResult `json:"data"`
}

// BatchFileInfo 批量文件信息
type BatchFileInfo struct {
	// Name 文件名 (文件上传模式必填)
	Name string `json:"name,omitempty"`

	// URL 文件 URL (URL 模式必填)
	URL string `json:"url,omitempty"`

	// IsOCR 是否启用 OCR
	IsOCR bool `json:"is_ocr,omitempty"`

	// DataID 解析对象对应的数据 ID
	DataID string `json:"data_id,omitempty"`

	// PageRanges 指定页码范围
	PageRanges string `json:"page_ranges,omitempty"`
}

// BatchUploadRequest 批量文件上传请求
type BatchUploadRequest struct {
	// EnableFormula 是否开启公式识别
	EnableFormula *bool `json:"enable_formula,omitempty"`

	// EnableTable 是否开启表格识别
	EnableTable *bool `json:"enable_table,omitempty"`

	// Language 指定文档语言
	Language string `json:"language,omitempty"`

	// Files 文件列表
	Files []BatchFileInfo `json:"files"`

	// Callback 解析结果回调 URL
	Callback string `json:"callback,omitempty"`

	// Seed 随机字符串，用于回调签名
	Seed string `json:"seed,omitempty"`

	// ExtraFormats 额外导出格式
	ExtraFormats []string `json:"extra_formats,omitempty"`

	// ModelVersion 模型版本
	ModelVersion string `json:"model_version,omitempty"`
}

// BatchUploadResponse 批量上传响应
type BatchUploadResponse struct {
	APIResponse
	Data struct {
		BatchID  string   `json:"batch_id"`
		FileURLs []string `json:"file_urls"`
	} `json:"data"`
}

// BatchTaskRequest 批量 URL 解析请求
type BatchTaskRequest struct {
	// EnableFormula 是否开启公式识别
	EnableFormula *bool `json:"enable_formula,omitempty"`

	// EnableTable 是否开启表格识别
	EnableTable *bool `json:"enable_table,omitempty"`

	// Language 指定文档语言
	Language string `json:"language,omitempty"`

	// Files 文件列表
	Files []BatchFileInfo `json:"files"`

	// Callback 解析结果回调 URL
	Callback string `json:"callback,omitempty"`

	// Seed 随机字符串，用于回调签名
	Seed string `json:"seed,omitempty"`

	// ExtraFormats 额外导出格式
	ExtraFormats []string `json:"extra_formats,omitempty"`

	// ModelVersion 模型版本
	ModelVersion string `json:"model_version,omitempty"`
}

// BatchTaskResponse 批量任务响应
type BatchTaskResponse struct {
	APIResponse
	Data struct {
		BatchID string `json:"batch_id"`
	} `json:"data"`
}

// BatchExtractResult 批量解析结果
type BatchExtractResult struct {
	FileName        string           `json:"file_name"`
	State           TaskState        `json:"state"`
	FullZipURL      string           `json:"full_zip_url,omitempty"`
	ErrMsg          string           `json:"err_msg,omitempty"`
	DataID          string           `json:"data_id,omitempty"`
	ExtractProgress *ExtractProgress `json:"extract_progress,omitempty"`
}

// GetBatchResultsResponse 获取批量结果响应
type GetBatchResultsResponse struct {
	APIResponse
	Data struct {
		BatchID       string               `json:"batch_id"`
		ExtractResult []BatchExtractResult `json:"extract_result"`
	} `json:"data"`
}

// PollOptions 轮询选项
type PollOptions struct {
	// Interval 轮询间隔
	Interval time.Duration

	// Timeout 轮询超时时间
	Timeout time.Duration

	// OnProgress 进度回调
	OnProgress func(progress *ExtractProgress)
}

// DefaultPollOptions 默认轮询选项
func DefaultPollOptions() *PollOptions {
	return &PollOptions{
		Interval: 5 * time.Second,
		Timeout:  10 * time.Minute,
	}
}
