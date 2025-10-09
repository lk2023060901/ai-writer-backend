package workerpool

import (
	"container/heap"
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/panjf2000/ants/v2"
	"go.uber.org/zap"
)

// Priority 优先级定义
type Priority int

const (
	PriorityLow    Priority = 0
	PriorityNormal Priority = 5
	PriorityHigh   Priority = 10
)

var (
	ErrPoolClosed = errors.New("worker pool is closed")
	ErrTimeout    = errors.New("task execution timeout")
)

// TaskResult 任务结果
type TaskResult struct {
	Data  interface{}
	Error error
}

// ============= 配置 =============

// Config Worker Pool 配置
type Config struct {
	// 基础配置
	InitialWorkers int  // 初始 worker 数量
	QueueSize      int  // 队列缓冲区大小
	EnablePriority bool // 是否启用优先级队列

	// 自动扩缩容配置（可选）
	AutoScaling *AutoScalingConfig
}

// AutoScalingConfig 自动扩缩容配置
type AutoScalingConfig struct {
	Enable                    bool          // 是否启用自动扩缩容
	MinWorkers                int           // 最小 worker 数
	MaxWorkers                int           // 最大 worker 数
	ScaleUpQueueThreshold     int           // 扩容队列阈值
	ScaleUpUtilizationRatio   float64       // 扩容利用率阈值
	ScaleDownUtilizationRatio float64       // 缩容利用率阈值
	ScaleUpStep               int           // 扩容步长
	ScaleDownStep             int           // 缩容步长
	CooldownPeriod            time.Duration // 冷却时间
	EnablePredictive          bool          // 启用预测性扩容
}

// DefaultConfig 默认配置
func DefaultConfig() *Config {
	return &Config{
		InitialWorkers: 80,
		QueueSize:      1000,
		EnablePriority: false,
		AutoScaling:    nil, // 默认不启用自动扩缩容
	}
}

// DefaultAutoScalingConfig 默认自动扩缩容配置
func DefaultAutoScalingConfig() *AutoScalingConfig {
	return &AutoScalingConfig{
		Enable:                    true,
		MinWorkers:                10,
		MaxWorkers:                200,
		ScaleUpQueueThreshold:     100,
		ScaleUpUtilizationRatio:   0.8,
		ScaleDownUtilizationRatio: 0.3,
		ScaleUpStep:               10,
		ScaleDownStep:             5,
		CooldownPeriod:            30 * time.Second,
		EnablePredictive:          false,
	}
}

// ============= 统计信息 =============

// Statistics 统计信息
type Statistics struct {
	mu sync.RWMutex

	Submitted int64 // 已提交
	Completed int64 // 已完成
	Failed    int64 // 失败
	Running   int64 // 运行中

	HighPriority   int64
	NormalPriority int64
	LowPriority    int64
}

func (s *Statistics) incSubmitted(priority Priority) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Submitted++
	switch priority {
	case PriorityHigh:
		s.HighPriority++
	case PriorityNormal:
		s.NormalPriority++
	case PriorityLow:
		s.LowPriority++
	}
}

func (s *Statistics) incRunning() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Running++
}

func (s *Statistics) decRunning() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Running--
}

func (s *Statistics) incCompleted() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Completed++
}

func (s *Statistics) incFailed() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Failed++
}

func (s *Statistics) Get() Statistics {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return *s
}

// Metrics 监控指标（用于自动扩缩容）
type Metrics struct {
	mu sync.RWMutex

	QueueLength    int
	MaxQueueLength int
	TotalWorkers   int
	RunningWorkers int
	IdleWorkers    int

	SubmittedTasks int64
	CompletedTasks int64
	FailedTasks    int64

	QueueHistory []int // 用于预测
}

func (m *Metrics) update(queueLen, totalWorkers, runningWorkers int, stats Statistics) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.QueueLength = queueLen
	if queueLen > m.MaxQueueLength {
		m.MaxQueueLength = queueLen
	}

	m.TotalWorkers = totalWorkers
	m.RunningWorkers = runningWorkers
	m.IdleWorkers = totalWorkers - runningWorkers

	m.SubmittedTasks = stats.Submitted
	m.CompletedTasks = stats.Completed
	m.FailedTasks = stats.Failed

	if len(m.QueueHistory) >= 60 {
		m.QueueHistory = m.QueueHistory[1:]
	}
	m.QueueHistory = append(m.QueueHistory, queueLen)
}

func (m *Metrics) Get() Metrics {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return *m
}

// ============= 优先级队列 =============

type priorityTask struct {
	Priority  Priority
	Task      func()
	Timestamp time.Time
	index     int
}

type priorityQueue []*priorityTask

func (pq priorityQueue) Len() int { return len(pq) }

func (pq priorityQueue) Less(i, j int) bool {
	if pq[i].Priority != pq[j].Priority {
		return pq[i].Priority > pq[j].Priority
	}
	return pq[i].Timestamp.Before(pq[j].Timestamp)
}

func (pq priorityQueue) Swap(i, j int) {
	pq[i], pq[j] = pq[j], pq[i]
	pq[i].index = i
	pq[j].index = j
}

func (pq *priorityQueue) Push(x interface{}) {
	n := len(*pq)
	task := x.(*priorityTask)
	task.index = n
	*pq = append(*pq, task)
}

func (pq *priorityQueue) Pop() interface{} {
	old := *pq
	n := len(old)
	task := old[n-1]
	old[n-1] = nil
	task.index = -1
	*pq = old[0 : n-1]
	return task
}

// ============= Worker Pool =============

// Pool 企业级 Worker Pool
// 支持：优先级队列 + 动态扩缩容
type Pool struct {
	pool *ants.Pool

	// 配置
	config *Config

	// 优先级队列（可选）
	priorityQueue *priorityQueue
	queueMu       sync.Mutex
	notEmpty      chan struct{}

	// 自动扩缩容（可选）
	currentWorkers int
	lastScaleTime  time.Time
	scaleMu        sync.RWMutex

	// 监控
	stats   *Statistics
	metrics *Metrics

	// 控制
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup

	logger *zap.Logger
}

// New 创建 Worker Pool
func New(config *Config, logger *zap.Logger) (*Pool, error) {
	if config == nil {
		config = DefaultConfig()
	}

	// 创建 ants pool
	initialSize := config.InitialWorkers
	if config.AutoScaling != nil && config.AutoScaling.Enable {
		initialSize = config.AutoScaling.MinWorkers
	}

	antsPool, err := ants.NewPool(initialSize,
		ants.WithPanicHandler(func(err interface{}) {
			logger.Error("worker panic", zap.Any("error", err))
		}),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create ants pool: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	p := &Pool{
		pool:           antsPool,
		config:         config,
		currentWorkers: initialSize,
		lastScaleTime:  time.Now(),
		stats:          &Statistics{},
		metrics:        &Metrics{},
		ctx:            ctx,
		cancel:         cancel,
		logger:         logger,
	}

	// 初始化优先级队列（如果启用）
	if config.EnablePriority {
		pq := make(priorityQueue, 0, config.QueueSize)
		heap.Init(&pq)
		p.priorityQueue = &pq
		p.notEmpty = make(chan struct{}, 1)

		// 启动调度器
		p.wg.Add(1)
		go p.scheduler()
	}

	// 启动自动扩缩容（如果启用）
	if config.AutoScaling != nil && config.AutoScaling.Enable {
		p.wg.Add(2)
		go p.metricsCollector()
		go p.autoScaler()
	}

	return p, nil
}

// Submit 提交任务
func (p *Pool) Submit(task func()) error {
	return p.SubmitWithPriority(PriorityNormal, task)
}

// SubmitWithPriority 提交带优先级的任务
func (p *Pool) SubmitWithPriority(priority Priority, task func()) error {
	select {
	case <-p.ctx.Done():
		return ErrPoolClosed
	default:
	}

	// 如果启用了优先级队列
	if p.config.EnablePriority {
		pt := &priorityTask{
			Priority:  priority,
			Task:      task,
			Timestamp: time.Now(),
		}

		p.queueMu.Lock()
		heap.Push(p.priorityQueue, pt)
		p.queueMu.Unlock()

		p.stats.incSubmitted(priority)

		select {
		case p.notEmpty <- struct{}{}:
		default:
		}

		return nil
	}

	// 否则直接提交给 ants
	p.stats.incSubmitted(priority)
	return p.pool.Submit(func() {
		p.stats.incRunning()
		defer func() {
			p.stats.decRunning()
			p.stats.incCompleted()
		}()
		task()
	})
}

// SubmitWithResult 提交任务并获取结果
func (p *Pool) SubmitWithResult(task func() (interface{}, error)) <-chan TaskResult {
	return p.SubmitWithPriorityAndResult(PriorityNormal, task)
}

// SubmitWithPriorityAndResult 提交带优先级的任务并获取结果
func (p *Pool) SubmitWithPriorityAndResult(
	priority Priority,
	task func() (interface{}, error),
) <-chan TaskResult {
	resultCh := make(chan TaskResult, 1)

	_ = p.SubmitWithPriority(priority, func() {
		result, err := task()
		resultCh <- TaskResult{Data: result, Error: err}
		close(resultCh)
	})

	return resultCh
}

// scheduler 调度器（仅在启用优先级队列时运行）
func (p *Pool) scheduler() {
	defer p.wg.Done()

	for {
		select {
		case <-p.ctx.Done():
			return
		case <-p.notEmpty:
			p.dispatch()
		}
	}
}

func (p *Pool) dispatch() {
	for {
		select {
		case <-p.ctx.Done():
			return
		default:
		}

		p.queueMu.Lock()
		if p.priorityQueue.Len() == 0 {
			p.queueMu.Unlock()
			return
		}

		pt := heap.Pop(p.priorityQueue).(*priorityTask)
		p.queueMu.Unlock()

		task := pt.Task
		err := p.pool.Submit(func() {
			p.stats.incRunning()
			defer func() {
				p.stats.decRunning()
				p.stats.incCompleted()
			}()
			task()
		})

		if err != nil {
			p.queueMu.Lock()
			heap.Push(p.priorityQueue, pt)
			p.queueMu.Unlock()
			p.stats.incFailed()
			time.Sleep(10 * time.Millisecond)
			return
		}
	}
}

// ============= 自动扩缩容 =============

func (p *Pool) metricsCollector() {
	defer p.wg.Done()

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-p.ctx.Done():
			return
		case <-ticker.C:
			p.collectMetrics()
		}
	}
}

func (p *Pool) collectMetrics() {
	stats := p.stats.Get()
	queueLen := p.QueueLength()
	runningWorkers := p.pool.Running()

	p.scaleMu.RLock()
	totalWorkers := p.currentWorkers
	p.scaleMu.RUnlock()

	p.metrics.update(queueLen, totalWorkers, runningWorkers, stats)
}

func (p *Pool) autoScaler() {
	defer p.wg.Done()

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-p.ctx.Done():
			return
		case <-ticker.C:
			p.evaluateAndScale()
		}
	}
}

func (p *Pool) evaluateAndScale() {
	if p.config.AutoScaling == nil || !p.config.AutoScaling.Enable {
		return
	}

	p.scaleMu.Lock()
	defer p.scaleMu.Unlock()

	if time.Since(p.lastScaleTime) < p.config.AutoScaling.CooldownPeriod {
		return
	}

	m := p.metrics.Get()
	utilization := float64(m.RunningWorkers) / float64(m.TotalWorkers)

	decision := p.makeScalingDecision(m.QueueLength, utilization)

	switch decision {
	case scaleUp:
		p.scaleUp(m.QueueLength)
	case scaleDown:
		p.scaleDown()
	}
}

type scalingDecision int

const (
	noChange scalingDecision = iota
	scaleUp
	scaleDown
)

func (p *Pool) makeScalingDecision(queueLen int, utilization float64) scalingDecision {
	cfg := p.config.AutoScaling

	// 扩容条件
	if queueLen > cfg.ScaleUpQueueThreshold {
		return scaleUp
	}

	if utilization > cfg.ScaleUpUtilizationRatio {
		return scaleUp
	}

	// 预测性扩容
	if cfg.EnablePredictive {
		predicted := p.predictQueueGrowth()
		if predicted > cfg.ScaleUpQueueThreshold {
			p.logger.Info("predictive scale up",
				zap.Int("current_queue", queueLen),
				zap.Int("predicted_queue", predicted))
			return scaleUp
		}
	}

	// 缩容条件
	if utilization < cfg.ScaleDownUtilizationRatio &&
		queueLen < cfg.ScaleUpQueueThreshold/2 &&
		p.currentWorkers > cfg.MinWorkers {
		return scaleDown
	}

	return noChange
}

func (p *Pool) predictQueueGrowth() int {
	m := p.metrics.Get()
	history := m.QueueHistory

	if len(history) < 10 {
		return m.QueueLength
	}

	n := len(history)
	var sumX, sumY, sumXY, sumX2 float64

	for i, y := range history[n-10:] {
		x := float64(i)
		sumX += x
		sumY += float64(y)
		sumXY += x * float64(y)
		sumX2 += x * x
	}

	slope := (10*sumXY - sumX*sumY) / (10*sumX2 - sumX*sumX)
	intercept := (sumY - slope*sumX) / 10

	predicted := int(slope*float64(n) + intercept)
	if predicted < 0 {
		return 0
	}

	return predicted
}

func (p *Pool) scaleUp(queueLen int) {
	cfg := p.config.AutoScaling

	// 多级扩容策略
	step := cfg.ScaleUpStep
	if queueLen > cfg.ScaleUpQueueThreshold*3 {
		step = 50
	} else if queueLen > cfg.ScaleUpQueueThreshold*2 {
		step = 20
	}

	newSize := p.currentWorkers + step
	if newSize > cfg.MaxWorkers {
		newSize = cfg.MaxWorkers
	}

	if newSize == p.currentWorkers {
		return
	}

	p.logger.Info("scaling up workers",
		zap.Int("from", p.currentWorkers),
		zap.Int("to", newSize),
		zap.Int("queue_length", queueLen))

	p.pool.Tune(newSize)
	p.currentWorkers = newSize
	p.lastScaleTime = time.Now()
}

func (p *Pool) scaleDown() {
	cfg := p.config.AutoScaling

	newSize := p.currentWorkers - cfg.ScaleDownStep
	if newSize < cfg.MinWorkers {
		newSize = cfg.MinWorkers
	}

	if newSize == p.currentWorkers {
		return
	}

	p.logger.Info("scaling down workers",
		zap.Int("from", p.currentWorkers),
		zap.Int("to", newSize))

	p.pool.Tune(newSize)
	p.currentWorkers = newSize
	p.lastScaleTime = time.Now()
}

// ============= 公共方法 =============

// QueueLength 获取队列长度
func (p *Pool) QueueLength() int {
	if p.config.EnablePriority {
		p.queueMu.Lock()
		defer p.queueMu.Unlock()
		return p.priorityQueue.Len()
	}
	return 0
}

// Running 获取运行中的 worker 数量
func (p *Pool) Running() int {
	return p.pool.Running()
}

// Free 获取空闲 worker 数量
func (p *Pool) Free() int {
	return p.pool.Free()
}

// Stats 获取统计信息
func (p *Pool) Stats() Statistics {
	return p.stats.Get()
}

// Metrics 获取监控指标
func (p *Pool) Metrics() Metrics {
	return p.metrics.Get()
}

// ManualScale 手动调整 worker 数量
func (p *Pool) ManualScale(newSize int) error {
	if p.config.AutoScaling == nil {
		return errors.New("auto scaling not enabled")
	}

	p.scaleMu.Lock()
	defer p.scaleMu.Unlock()

	cfg := p.config.AutoScaling
	if newSize < cfg.MinWorkers || newSize > cfg.MaxWorkers {
		return fmt.Errorf("invalid size: must be between %d and %d",
			cfg.MinWorkers, cfg.MaxWorkers)
	}

	p.logger.Warn("manual scaling",
		zap.Int("from", p.currentWorkers),
		zap.Int("to", newSize))

	p.pool.Tune(newSize)
	p.currentWorkers = newSize
	p.lastScaleTime = time.Now()

	return nil
}

// Shutdown 关闭
func (p *Pool) Shutdown() {
	p.cancel()
	p.wg.Wait()
	p.pool.Release()
}
