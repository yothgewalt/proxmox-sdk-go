package internal

import (
	"context"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
)

// WorkerPool represents a reusable worker pool for concurrent task execution
type WorkerPool struct {
	// Channel to control concurrency
	semaphore chan struct{}

	// Worker pool configuration
	maxWorkers int

	// Statistics
	activeWorkers int64
	totalTasks    int64

	// Mutex for thread-safe operations
	mutex sync.RWMutex

	// Context for graceful shutdown
	ctx    context.Context
	cancel context.CancelFunc

	// WaitGroup for shutdown coordination
	wg sync.WaitGroup
}

// WorkerPoolConfig holds configuration for worker pool creation
type WorkerPoolConfig struct {
	MaxWorkers int
	// Optional timeout for tasks
	TaskTimeout time.Duration
}

// NewWorkerPool creates a new worker pool with the specified configuration
func NewWorkerPool(config WorkerPoolConfig) *WorkerPool {
	if config.MaxWorkers <= 0 {
		config.MaxWorkers = runtime.NumCPU() * 2 // Default to 2x CPU cores
	}

	ctx, cancel := context.WithCancel(context.Background())

	pool := &WorkerPool{
		semaphore:  make(chan struct{}, config.MaxWorkers),
		maxWorkers: config.MaxWorkers,
		ctx:        ctx,
		cancel:     cancel,
	}

	return pool
}

// Submit submits a task to the worker pool
func (wp *WorkerPool) Submit(task func()) error {
	select {
	case <-wp.ctx.Done():
		return wp.ctx.Err()
	default:
	}

	wp.semaphore <- struct{}{}
	atomic.AddInt64(&wp.activeWorkers, 1)
	atomic.AddInt64(&wp.totalTasks, 1)

	wp.wg.Add(1)
	go func() {
		defer func() {
			<-wp.semaphore
			atomic.AddInt64(&wp.activeWorkers, -1)
			wp.wg.Done()
		}()

		task()
	}()

	return nil
}

// SubmitWithContext submits a task with context for timeout/cancellation
func (wp *WorkerPool) SubmitWithContext(ctx context.Context, task func(ctx context.Context)) error {
	select {
	case <-wp.ctx.Done():
		return wp.ctx.Err()
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	wp.semaphore <- struct{}{}
	atomic.AddInt64(&wp.activeWorkers, 1)
	atomic.AddInt64(&wp.totalTasks, 1)

	wp.wg.Add(1)
	go func() {
		defer func() {
			<-wp.semaphore
			atomic.AddInt64(&wp.activeWorkers, -1)
			wp.wg.Done()
		}()

		task(ctx)
	}()

	return nil
}

// Resize dynamically adjusts the worker pool size
func (wp *WorkerPool) Resize(newSize int) {
	if newSize <= 0 {
		return
	}

	wp.mutex.Lock()
	defer wp.mutex.Unlock()

	oldSize := wp.maxWorkers
	wp.maxWorkers = newSize

	newSemaphore := make(chan struct{}, newSize)

	if newSize > oldSize {
		for i := 0; i < len(wp.semaphore); i++ {
			select {
			case token := <-wp.semaphore:
				newSemaphore <- token
			default:
				break
			}
		}
	} else {
		for i := 0; i < newSize && i < len(wp.semaphore); i++ {
			select {
			case token := <-wp.semaphore:
				newSemaphore <- token
			default:
				break
			}
		}
	}

	wp.semaphore = newSemaphore
}

// GetStats returns current worker pool statistics
func (wp *WorkerPool) GetStats() (activeWorkers, totalTasks int64, maxWorkers int) {
	return atomic.LoadInt64(&wp.activeWorkers),
		atomic.LoadInt64(&wp.totalTasks),
		wp.maxWorkers
}

// GetDetailedStats returns comprehensive worker pool statistics
func (wp *WorkerPool) GetDetailedStats() map[string]interface{} {
	active := atomic.LoadInt64(&wp.activeWorkers)
	total := atomic.LoadInt64(&wp.totalTasks)

	return map[string]interface{}{
		"active_workers":    active,
		"total_tasks":       total,
		"max_workers":       wp.maxWorkers,
		"pending_tasks":     len(wp.semaphore),
		"available_workers": wp.maxWorkers - len(wp.semaphore),
		"utilization":       float64(active) / float64(wp.maxWorkers) * 100,
	}
}

// IsOverloaded checks if the worker pool is at capacity
func (wp *WorkerPool) IsOverloaded() bool {
	return len(wp.semaphore) >= wp.maxWorkers
}

// WaitForCapacity waits until worker capacity is available or timeout
func (wp *WorkerPool) WaitForCapacity(timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	select {
	case wp.semaphore <- struct{}{}:
		<-wp.semaphore
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// SubmitWithPriority submits a high-priority task (bypasses queue when possible)
func (wp *WorkerPool) SubmitWithPriority(task func()) error {
	select {
	case <-wp.ctx.Done():
		return wp.ctx.Err()
	default:
	}

	select {
	case wp.semaphore <- struct{}{}:
		atomic.AddInt64(&wp.activeWorkers, 1)
		atomic.AddInt64(&wp.totalTasks, 1)

		wp.wg.Add(1)
		go func() {
			defer func() {
				<-wp.semaphore
				atomic.AddInt64(&wp.activeWorkers, -1)
				wp.wg.Done()
			}()
			task()
		}()
		return nil
	default:
		return wp.Submit(task)
	}
}

// Close gracefully shuts down the worker pool
func (wp *WorkerPool) Close() error {
	wp.cancel()
	wp.wg.Wait()
	return nil
}

// CloseWithTimeout shuts down the worker pool with a timeout
func (wp *WorkerPool) CloseWithTimeout(timeout time.Duration) error {
	wp.cancel()

	done := make(chan struct{})
	go func() {
		wp.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-time.After(timeout):
		return context.DeadlineExceeded
	}
}
