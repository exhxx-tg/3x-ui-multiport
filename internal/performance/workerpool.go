package performance

import (
	"context"
	"sync"
	"sync/atomic"

	"github.com/exhxx-tg/3x-ui-multiport/internal/logger"
)

type Job func(ctx context.Context)

type WorkerPool struct {
	name       string
	size       int
	queue      chan Job
	ctx        context.Context
	cancel     context.CancelFunc
	wg         sync.WaitGroup
	submitted  atomic.Int64
	completed  atomic.Int64
	failed     atomic.Int64
	running    atomic.Int64
	started    bool
	mu         sync.Mutex
}

func NewWorkerPool(name string, size int, queueDepth int) *WorkerPool {
	if size < 0 {
		size = defaultWorkerPoolSize
	}
	if size == 0 {
		size = 1
	}
	if queueDepth < 0 {
		queueDepth = defaultWorkerQueueDepth
	}

	ctx, cancel := context.WithCancel(context.Background())
	return &WorkerPool{
		name:   name,
		size:   size,
		queue:  make(chan Job, queueDepth),
		ctx:    ctx,
		cancel: cancel,
	}
}

func (wp *WorkerPool) Start() {
	wp.mu.Lock()
	defer wp.mu.Unlock()

	if wp.started {
		return
	}
	wp.started = true

	for i := 0; i < wp.size; i++ {
		wp.wg.Add(1)
		go wp.worker()
	}

	logger.Infof("worker pool [%s]: started %d workers, queue depth %d", wp.name, wp.size, cap(wp.queue))
}

func (wp *WorkerPool) Stop() {
	wp.mu.Lock()
	defer wp.mu.Unlock()

	if !wp.started {
		return
	}
	wp.cancel()
	wp.wg.Wait()
	wp.started = false

	logger.Infof("worker pool [%s]: stopped (submitted=%d completed=%d failed=%d)",
		wp.name, wp.submitted.Load(), wp.completed.Load(), wp.failed.Load())
}

func (wp *WorkerPool) Submit(job Job) error {
	select {
	case wp.queue <- job:
		wp.submitted.Add(1)
		return nil
	case <-wp.ctx.Done():
		return wp.ctx.Err()
	default:
		return ErrPoolQueueFull
	}
}

func (wp *WorkerPool) SubmitWait(job Job) {
	wp.queue <- job
	wp.submitted.Add(1)
}

func (wp *WorkerPool) TrySubmit(job Job) bool {
	select {
	case wp.queue <- job:
		wp.submitted.Add(1)
		return true
	default:
		return false
	}
}

func (wp *WorkerPool) worker() {
	defer wp.wg.Done()

	for {
		select {
		case <-wp.ctx.Done():
			return
		case job, ok := <-wp.queue:
			if !ok {
				return
			}
			wp.running.Add(1)
			func() {
				defer func() {
					if r := recover(); r != nil {
						wp.failed.Add(1)
						logger.Errorf("worker pool [%s]: job panicked: %v", wp.name, r)
					}
					wp.running.Add(-1)
					wp.completed.Add(1)
				}()
				job(wp.ctx)
			}()
		}
	}
}

func (wp *WorkerPool) Stats() PoolStats {
	return PoolStats{
		Name:      wp.name,
		Size:      wp.size,
		QueueSize: len(wp.queue),
		QueueCap:  cap(wp.queue),
		Submitted: wp.submitted.Load(),
		Completed: wp.completed.Load(),
		Failed:    wp.failed.Load(),
		Running:   wp.running.Load(),
	}
}

type PoolStats struct {
	Name      string `json:"name"`
	Size      int    `json:"size"`
	QueueSize int    `json:"queueSize"`
	QueueCap  int    `json:"queueCap"`
	Submitted int64  `json:"submitted"`
	Completed int64  `json:"completed"`
	Failed    int64  `json:"failed"`
	Running   int64  `json:"running"`
}

var (
	globalHealthPool   *WorkerPool
	globalMetricsPool  *WorkerPool
	globalAsyncPool    *WorkerPool
	globalPoolInitOnce sync.Once
)

func InitGlobalWorkerPools(cfg Config) {
	globalPoolInitOnce.Do(func() {
		globalHealthPool = NewWorkerPool("health", cfg.HealthConcurrency, cfg.WorkerQueueDepth)
		globalMetricsPool = NewWorkerPool("metrics", 4, 128)
		globalAsyncPool = NewWorkerPool("async", cfg.WorkerPoolSize, cfg.WorkerQueueDepth)

		globalHealthPool.Start()
		globalMetricsPool.Start()
		globalAsyncPool.Start()

		logger.Infof("performance: global worker pools initialized (health=%d, metrics=4, async=%d)",
			cfg.HealthConcurrency, cfg.WorkerPoolSize)
	})
}

func GlobalHealthPool() *WorkerPool     { return globalHealthPool }
func GlobalMetricsPool() *WorkerPool    { return globalMetricsPool }
func GlobalAsyncPool() *WorkerPool      { return globalAsyncPool }

func StopAllWorkerPools() {
	if globalHealthPool != nil {
		globalHealthPool.Stop()
	}
	if globalMetricsPool != nil {
		globalMetricsPool.Stop()
	}
	if globalAsyncPool != nil {
		globalAsyncPool.Stop()
	}
}
