package sse

import (
	"context"
	"fmt"
	"reflect"
)

// WorkerPool 工作池接口(兼容现有的 workerpool)
type WorkerPool interface {
	Submit(task func()) error
}

// ItemNamer 可命名的项目(用于获取项目名称)
type ItemNamer interface {
	GetName() string
}

// BatchUploader 批量上传器(泛型实现)
type BatchUploader[T any] struct {
	stream       *Stream
	tracker      *ProgressTracker
	items        []T
	processFunc  func(ctx context.Context, item T) (interface{}, error)
	workerPool   WorkerPool
	onSuccess    func(index int, item T, result interface{}) error
	onFailure    func(index int, item T, err error) error
	getItemName  func(T) string
	eventPrefix  string // 事件前缀(默认 "item-", 可自定义为 "file-" 等)
}

// NewBatchUploader 创建批量上传器
func NewBatchUploader[T any](stream *Stream, totalItems int) *BatchUploader[T] {
	tracker := NewProgressTracker(stream, totalItems)
	return &BatchUploader[T]{
		stream:      stream,
		tracker:     tracker,
		eventPrefix: "item", // 默认事件前缀
	}
}

// WithEventPrefix 设置事件前缀(如 "file" 会生成 "file-success", "file-failed")
func (u *BatchUploader[T]) WithEventPrefix(prefix string) *BatchUploader[T] {
	u.eventPrefix = prefix
	return u
}

// Process 设置处理函数
func (u *BatchUploader[T]) Process(items []T, fn func(ctx context.Context, item T) (interface{}, error)) *BatchUploader[T] {
	u.items = items
	u.processFunc = fn
	return u
}

// WithWorkerPool 设置工作池
func (u *BatchUploader[T]) WithWorkerPool(pool WorkerPool) *BatchUploader[T] {
	u.workerPool = pool
	return u
}

// OnSuccess 设置成功回调
func (u *BatchUploader[T]) OnSuccess(fn func(index int, item T, result interface{}) error) *BatchUploader[T] {
	u.onSuccess = fn
	return u
}

// OnFailure 设置失败回调
func (u *BatchUploader[T]) OnFailure(fn func(index int, item T, err error) error) *BatchUploader[T] {
	u.onFailure = fn
	return u
}

// WithItemNamer 设置项目名称提取器
func (u *BatchUploader[T]) WithItemNamer(fn func(T) string) *BatchUploader[T] {
	u.getItemName = fn
	return u
}

// Run 执行批量处理(阻塞直到所有任务完成)
func (u *BatchUploader[T]) Run(ctx context.Context) error {
	if u.processFunc == nil {
		return fmt.Errorf("process function not set")
	}

	if u.workerPool == nil {
		return fmt.Errorf("worker pool not set")
	}

	// 如果没有设置名称提取器,使用默认实现
	if u.getItemName == nil {
		u.getItemName = u.defaultItemNamer
	}

	// 发送开始事件
	u.tracker.Start()

	// 结果 channel
	type result struct {
		Index  int
		Item   T
		Result interface{}
		Error  error
	}
	resultCh := make(chan result, len(u.items))

	// 提交所有任务到工作池
	for i, item := range u.items {
		idx := i
		it := item
		if err := u.workerPool.Submit(func() {
			res, err := u.processFunc(ctx, it)
			resultCh <- result{
				Index:  idx,
				Item:   it,
				Result: res,
				Error:  err,
			}
		}); err != nil {
			// 如果提交失败,直接记录为失败结果
			resultCh <- result{
				Index: idx,
				Item:  it,
				Error: fmt.Errorf("failed to submit task: %w", err),
			}
		}
	}

	// 收集结果并推送事件
	for range u.items {
		r := <-resultCh
		itemName := u.getItemName(r.Item)

		if r.Error != nil {
			// 记录失败
			u.sendFailureEvent(r.Index, itemName, r.Error)

			// 触发失败回调
			if u.onFailure != nil {
				if err := u.onFailure(r.Index, r.Item, r.Error); err != nil {
					// 回调错误只记录日志,不中断处理
					u.stream.onError(fmt.Errorf("onFailure callback error: %w", err))
				}
			}
		} else {
			// 记录成功
			u.sendSuccessEvent(r.Index, itemName, r.Result)

			// 触发成功回调
			if u.onSuccess != nil {
				if err := u.onSuccess(r.Index, r.Item, r.Result); err != nil {
					// 回调错误只记录日志,不中断处理
					if u.stream.onError != nil {
						u.stream.onError(fmt.Errorf("onSuccess callback error: %w", err))
					}
				}
			}
		}
	}

	close(resultCh)

	// 发送完成事件
	return u.tracker.Complete()
}

// sendSuccessEvent 发送成功事件
func (u *BatchUploader[T]) sendSuccessEvent(index int, itemName string, data interface{}) error {
	u.tracker.successCount.Add(1)
	completed := u.tracker.completed.Add(1)

	eventData := map[string]interface{}{
		"index":     index + 1,
		"total":     u.tracker.total,
		"completed": int(completed),
		"item_name": itemName,
		"message":   fmt.Sprintf("Item '%s' processed successfully", itemName),
	}

	// 如果有额外数据,添加到事件中
	if data != nil {
		eventData["data"] = data
	}

	// 使用自定义事件类型
	eventType := fmt.Sprintf("%s-success", u.eventPrefix)
	return u.stream.Send(eventType, eventData)
}

// sendFailureEvent 发送失败事件
func (u *BatchUploader[T]) sendFailureEvent(index int, itemName string, err error) error {
	u.tracker.failedCount.Add(1)
	completed := u.tracker.completed.Add(1)

	eventType := fmt.Sprintf("%s-failed", u.eventPrefix)
	return u.stream.Send(eventType, map[string]interface{}{
		"index":     index + 1,
		"total":     u.tracker.total,
		"completed": int(completed),
		"item_name": itemName,
		"error":     err.Error(),
		"message":   fmt.Sprintf("Item '%s' processing failed: %s", itemName, err.Error()),
	})
}

// defaultItemNamer 默认的项目名称提取器
func (u *BatchUploader[T]) defaultItemNamer(item T) string {
	// 1. 尝试断言为 ItemNamer 接口
	if namer, ok := any(item).(ItemNamer); ok {
		return namer.GetName()
	}

	// 2. 尝试通过反射获取 FileName 或 Name 字段
	v := reflect.ValueOf(item)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	if v.Kind() == reflect.Struct {
		// 优先查找 FileName 字段
		if field := v.FieldByName("FileName"); field.IsValid() && field.Kind() == reflect.String {
			return field.String()
		}
		// 其次查找 Name 字段
		if field := v.FieldByName("Name"); field.IsValid() && field.Kind() == reflect.String {
			return field.String()
		}
	}

	// 3. 返回类型名称
	return fmt.Sprintf("%T", item)
}

// GetStats 获取当前统计信息
func (u *BatchUploader[T]) GetStats() (completed, success, failed int) {
	return u.tracker.GetStats()
}
