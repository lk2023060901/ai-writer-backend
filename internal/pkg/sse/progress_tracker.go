package sse

import (
	"fmt"
	"sync/atomic"
)

// ProgressTracker 批量任务进度跟踪器
type ProgressTracker struct {
	stream       *Stream
	total        int
	completed    atomic.Int32
	successCount atomic.Int32
	failedCount  atomic.Int32
}

// NewProgressTracker 创建进度跟踪器
func NewProgressTracker(stream *Stream, total int) *ProgressTracker {
	return &ProgressTracker{
		stream: stream,
		total:  total,
	}
}

// Start 发送开始事件
func (t *ProgressTracker) Start() error {
	return t.stream.Send("batch-start", map[string]interface{}{
		"total_count": t.total,
		"message":     fmt.Sprintf("Starting batch processing of %d items", t.total),
	})
}

// RecordSuccess 记录成功并推送事件
func (t *ProgressTracker) RecordSuccess(index int, itemName string, data interface{}) error {
	t.successCount.Add(1)
	completed := t.completed.Add(1)

	eventData := map[string]interface{}{
		"index":     index + 1,
		"total":     t.total,
		"completed": int(completed),
		"item_name": itemName,
		"message":   fmt.Sprintf("Item '%s' processed successfully", itemName),
	}

	// 如果有额外数据,添加到事件中
	if data != nil {
		eventData["data"] = data
	}

	return t.stream.Send("item-success", eventData)
}

// RecordFailure 记录失败并推送事件
func (t *ProgressTracker) RecordFailure(index int, itemName string, err error) error {
	t.failedCount.Add(1)
	completed := t.completed.Add(1)

	return t.stream.Send("item-failed", map[string]interface{}{
		"index":     index + 1,
		"total":     t.total,
		"completed": int(completed),
		"item_name": itemName,
		"error":     err.Error(),
		"message":   fmt.Sprintf("Item '%s' processing failed: %s", itemName, err.Error()),
	})
}

// Complete 发送完成事件
func (t *ProgressTracker) Complete() error {
	success := int(t.successCount.Load())
	failed := int(t.failedCount.Load())

	return t.stream.Send("batch-complete", map[string]interface{}{
		"total_count":   t.total,
		"success_count": success,
		"failed_count":  failed,
		"message":       fmt.Sprintf("Batch processing completed: %d succeeded, %d failed", success, failed),
	})
}

// GetStats 获取当前统计信息
func (t *ProgressTracker) GetStats() (completed, success, failed int) {
	return int(t.completed.Load()), int(t.successCount.Load()), int(t.failedCount.Load())
}

// GetSuccessRate 获取成功率(0-100)
func (t *ProgressTracker) GetSuccessRate() float64 {
	completed := int(t.completed.Load())
	if completed == 0 {
		return 0
	}
	success := int(t.successCount.Load())
	return float64(success) / float64(completed) * 100
}
