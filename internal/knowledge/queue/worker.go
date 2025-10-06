package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/lk2023060901/ai-writer-backend/internal/knowledge/biz"
	pkgredis "github.com/lk2023060901/ai-writer-backend/internal/pkg/redis"
	"github.com/lk2023060901/ai-writer-backend/internal/pkg/sse"
	"go.uber.org/zap"
)

const (
	DocumentProcessQueue = "queue:document:process"
	ProcessingSet        = "set:document:processing"
)

// DocumentTask 文档处理任务
type DocumentTask struct {
	DocumentID string `json:"document_id"`
	RetryCount int    `json:"retry_count"`
}

// Worker 任务处理Worker
type Worker struct {
	redis       *pkgredis.Client
	docUseCase  *biz.DocumentUseCase
	sseHub      *sse.Hub
	logger      *zap.Logger
	workerCount int
	wg          sync.WaitGroup
	stopCh      chan struct{}
	mu          sync.Mutex
	running     bool
}

// NewWorker 创建Worker
func NewWorker(
	redis *pkgredis.Client,
	docUseCase *biz.DocumentUseCase,
	sseHub *sse.Hub,
	logger *zap.Logger,
	workerCount int,
) *Worker {
	return &Worker{
		redis:       redis,
		docUseCase:  docUseCase,
		sseHub:      sseHub,
		logger:      logger,
		workerCount: workerCount,
		stopCh:      make(chan struct{}),
		running:     false,
	}
}

// Start 启动Worker
func (w *Worker) Start(ctx context.Context) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.running {
		return fmt.Errorf("worker already running")
	}

	w.running = true
	w.logger.Info("starting document processing workers", zap.Int("worker_count", w.workerCount))

	for i := 0; i < w.workerCount; i++ {
		w.wg.Add(1)
		go w.processLoop(ctx, i)
	}

	return nil
}

// Stop 停止Worker
func (w *Worker) Stop() {
	w.mu.Lock()
	defer w.mu.Unlock()

	if !w.running {
		return
	}

	w.logger.Info("stopping document processing workers")
	close(w.stopCh)
	w.wg.Wait()
	w.running = false
	w.logger.Info("all workers stopped")
}

// EnqueueDocument 将文档加入处理队列
func (w *Worker) EnqueueDocument(ctx context.Context, documentID string) error {
	task := &DocumentTask{
		DocumentID: documentID,
		RetryCount: 0,
	}

	taskJSON, err := json.Marshal(task)
	if err != nil {
		return fmt.Errorf("failed to marshal task: %w", err)
	}

	_, err = w.redis.LPush(ctx, DocumentProcessQueue, string(taskJSON))
	if err != nil {
		return fmt.Errorf("failed to enqueue task: %w", err)
	}

	w.logger.Info("document enqueued for processing", zap.String("document_id", documentID))
	return nil
}

// processLoop 处理循环
func (w *Worker) processLoop(ctx context.Context, workerID int) {
	defer w.wg.Done()

	logger := w.logger.With(zap.Int("worker_id", workerID))
	logger.Info("worker started")

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-w.stopCh:
			logger.Info("worker stopping")
			return
		case <-ctx.Done():
			logger.Info("context cancelled, worker stopping")
			return
		case <-ticker.C:
			// 尝试从队列获取任务
			taskJSON, err := w.redis.RPop(ctx, DocumentProcessQueue)
			if err != nil || taskJSON == "" {
				continue
			}

			var task DocumentTask
			if err := json.Unmarshal([]byte(taskJSON), &task); err != nil {
				logger.Error("failed to unmarshal task", zap.Error(err))
				continue
			}

			// 处理任务
			w.processTask(ctx, &task, logger)
		}
	}
}

// processTask 处理单个任务
func (w *Worker) processTask(ctx context.Context, task *DocumentTask, logger *zap.Logger) {
	logger = logger.With(zap.String("document_id", task.DocumentID))
	logger.Info("processing document task")

	resource := "doc:" + task.DocumentID

	// 标记为处理中
	_, err := w.redis.SAdd(ctx, ProcessingSet, task.DocumentID)
	if err != nil {
		logger.Error("failed to mark document as processing", zap.Error(err))
	}

	// SSE 广播: 开始处理
	w.sseHub.Broadcast(resource, sse.Event{
		Type: "status",
		Data: map[string]interface{}{
			"document_id": task.DocumentID,
			"status":      "processing",
			"message":     "Document processing started",
		},
	})

	// 执行处理
	err = w.docUseCase.ProcessDocument(ctx, task.DocumentID)

	// 从处理集合中移除
	_, _ = w.redis.SRem(ctx, ProcessingSet, task.DocumentID)

	if err != nil {
		logger.Error("failed to process document",
			zap.Error(err),
			zap.Int("retry_count", task.RetryCount))

		// 重试逻辑（最多3次）
		if task.RetryCount < 3 {
			task.RetryCount++
			taskJSON, _ := json.Marshal(task)
			_, _ = w.redis.LPush(ctx, DocumentProcessQueue, string(taskJSON))
			logger.Info("document re-enqueued for retry", zap.Int("retry_count", task.RetryCount))

			// SSE 广播: 重试中
			w.sseHub.Broadcast(resource, sse.Event{
				Type: "status",
				Data: map[string]interface{}{
					"document_id": task.DocumentID,
					"status":      "retrying",
					"retry_count": task.RetryCount,
					"error":       err.Error(),
				},
			})
		} else {
			logger.Error("document processing failed after max retries")

			// SSE 广播: 失败
			w.sseHub.Broadcast(resource, sse.Event{
				Type: "status",
				Data: map[string]interface{}{
					"document_id": task.DocumentID,
					"status":      "failed",
					"error":       err.Error(),
				},
			})
		}
	} else {
		logger.Info("document processed successfully")

		// 获取最新文档信息
		doc, _ := w.docUseCase.DocumentRepo.GetByID(ctx, task.DocumentID)

		// SSE 广播: 完成
		w.sseHub.Broadcast(resource, sse.Event{
			Type: "status",
			Data: map[string]interface{}{
				"document_id": task.DocumentID,
				"status":      "completed",
				"chunk_count": doc.ChunkCount,
				"message":     "Document processing completed successfully",
			},
		})
	}
}

// GetQueueSize 获取队列大小
func (w *Worker) GetQueueSize(ctx context.Context) (int64, error) {
	return w.redis.LLen(ctx, DocumentProcessQueue)
}

// GetProcessingCount 获取处理中的文档数量
func (w *Worker) GetProcessingCount(ctx context.Context) (int64, error) {
	return w.redis.SCard(ctx, ProcessingSet)
}
