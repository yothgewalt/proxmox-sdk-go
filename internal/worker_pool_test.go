package internal

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestWorkerPool_Basic(t *testing.T) {
	config := WorkerPoolConfig{
		MaxWorkers: 5,
	}

	pool := NewWorkerPool(config)
	defer pool.Close()

	var counter int64
	var wg sync.WaitGroup

	// Submit 10 tasks
	for i := 0; i < 10; i++ {
		wg.Add(1)
		err := pool.Submit(func() {
			defer wg.Done()
			atomic.AddInt64(&counter, 1)
			time.Sleep(10 * time.Millisecond)
		})

		if err != nil {
			t.Errorf("Failed to submit task: %v", err)
		}
	}

	wg.Wait()

	if atomic.LoadInt64(&counter) != 10 {
		t.Errorf("Expected 10 tasks completed, got %d", atomic.LoadInt64(&counter))
	}
}

func TestWorkerPool_Concurrency(t *testing.T) {
	config := WorkerPoolConfig{
		MaxWorkers: 3,
	}

	pool := NewWorkerPool(config)
	defer pool.Close()

	start := time.Now()
	var wg sync.WaitGroup

	// Submit 6 tasks that take 100ms each
	for i := 0; i < 6; i++ {
		wg.Add(1)
		err := pool.Submit(func() {
			defer wg.Done()
			time.Sleep(100 * time.Millisecond)
		})

		if err != nil {
			t.Errorf("Failed to submit task: %v", err)
		}
	}

	wg.Wait()
	elapsed := time.Since(start)

	// With 3 workers and 6 tasks of 100ms each, it should take ~200ms
	// Allow some variance for test execution
	if elapsed < 150*time.Millisecond || elapsed > 300*time.Millisecond {
		t.Errorf("Expected ~200ms execution time, got %v", elapsed)
	}
}

func TestWorkerPool_Resize(t *testing.T) {
	config := WorkerPoolConfig{
		MaxWorkers: 2,
	}

	pool := NewWorkerPool(config)
	defer pool.Close()

	// Check initial stats
	_, _, max := pool.GetStats()
	if max != 2 {
		t.Errorf("Expected max workers 2, got %d", max)
	}

	// Resize to 5
	pool.Resize(5)

	_, _, max = pool.GetStats()
	if max != 5 {
		t.Errorf("Expected max workers 5 after resize, got %d", max)
	}

	// Resize to 1
	pool.Resize(1)

	_, _, max = pool.GetStats()
	if max != 1 {
		t.Errorf("Expected max workers 1 after resize, got %d", max)
	}
}

func TestWorkerPool_WithContext(t *testing.T) {
	config := WorkerPoolConfig{
		MaxWorkers: 2,
	}

	pool := NewWorkerPool(config)
	defer pool.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	var completed int64
	var wg sync.WaitGroup

	// Submit a task that should be cancelled
	wg.Add(1)
	err := pool.SubmitWithContext(ctx, func(taskCtx context.Context) {
		defer wg.Done()
		select {
		case <-taskCtx.Done():
			// Task was cancelled
			return
		case <-time.After(100 * time.Millisecond):
			atomic.AddInt64(&completed, 1)
		}
	})

	if err != nil {
		t.Errorf("Failed to submit task with context: %v", err)
	}

	wg.Wait()

	// Task should not have completed due to context cancellation
	if atomic.LoadInt64(&completed) != 0 {
		t.Errorf("Expected task to be cancelled, but it completed")
	}
}

func TestWorkerPool_CloseWithTimeout(t *testing.T) {
	config := WorkerPoolConfig{
		MaxWorkers: 2,
	}

	pool := NewWorkerPool(config)

	var wg sync.WaitGroup

	// Submit a long-running task
	wg.Add(1)
	err := pool.Submit(func() {
		defer wg.Done()
		time.Sleep(200 * time.Millisecond)
	})

	if err != nil {
		t.Errorf("Failed to submit task: %v", err)
	}

	// Try to close with short timeout
	start := time.Now()
	err = pool.CloseWithTimeout(50 * time.Millisecond)
	elapsed := time.Since(start)

	if err != context.DeadlineExceeded {
		t.Errorf("Expected timeout error, got %v", err)
	}

	if elapsed < 40*time.Millisecond || elapsed > 70*time.Millisecond {
		t.Errorf("Expected ~50ms timeout, got %v", elapsed)
	}

	// Wait for task to complete before test ends
	wg.Wait()
}

func TestWorkerPool_Stats(t *testing.T) {
	config := WorkerPoolConfig{
		MaxWorkers: 3,
	}

	pool := NewWorkerPool(config)
	defer pool.Close()

	// Initial stats
	active, total, max := pool.GetStats()
	if active != 0 || total != 0 || max != 3 {
		t.Errorf("Expected initial stats (0, 0, 3), got (%d, %d, %d)", active, total, max)
	}

	var wg sync.WaitGroup

	// Submit some tasks
	for i := 0; i < 5; i++ {
		wg.Add(1)
		err := pool.Submit(func() {
			defer wg.Done()
			time.Sleep(50 * time.Millisecond)
		})

		if err != nil {
			t.Errorf("Failed to submit task: %v", err)
		}
	}

	// Check stats while tasks are running
	time.Sleep(10 * time.Millisecond) // Let tasks start
	active, total, max = pool.GetStats()

	if total != 5 {
		t.Errorf("Expected 5 total tasks, got %d", total)
	}

	if max != 3 {
		t.Errorf("Expected max workers 3, got %d", max)
	}

	// Active should be <= 3 (max workers)
	if active > 3 {
		t.Errorf("Expected active workers <= 3, got %d", active)
	}

	wg.Wait()

	// After completion, active should be 0
	time.Sleep(10 * time.Millisecond) // Let workers finish
	active, _, _ = pool.GetStats()

	if active != 0 {
		t.Errorf("Expected 0 active workers after completion, got %d", active)
	}
}

func TestWorkerPool_DetailedStats(t *testing.T) {
	config := WorkerPoolConfig{
		MaxWorkers: 5,
	}

	pool := NewWorkerPool(config)
	defer pool.Close()

	// Submit some tasks
	var wg sync.WaitGroup
	for i := 0; i < 3; i++ {
		wg.Add(1)
		pool.Submit(func() {
			defer wg.Done()
			time.Sleep(50 * time.Millisecond)
		})
	}

	time.Sleep(10 * time.Millisecond) // Let tasks start

	stats := pool.GetDetailedStats()

	if totalTasks := stats["total_tasks"].(int64); totalTasks != 3 {
		t.Errorf("Expected 3 total tasks, got %d", totalTasks)
	}

	if maxWorkers := stats["max_workers"].(int); maxWorkers != 5 {
		t.Errorf("Expected 5 max workers, got %d", maxWorkers)
	}

	if utilization := stats["utilization"].(float64); utilization < 0 || utilization > 100 {
		t.Errorf("Expected utilization between 0-100%%, got %.2f", utilization)
	}

	wg.Wait()
}

func TestWorkerPool_IsOverloaded(t *testing.T) {
	config := WorkerPoolConfig{
		MaxWorkers: 2,
	}

	pool := NewWorkerPool(config)
	defer pool.Close()

	// Initially not overloaded
	if pool.IsOverloaded() {
		t.Error("Expected pool to not be overloaded initially")
	}

	// Fill the pool
	var wg sync.WaitGroup
	for i := 0; i < 2; i++ {
		wg.Add(1)
		pool.Submit(func() {
			defer wg.Done()
			time.Sleep(100 * time.Millisecond)
		})
	}

	time.Sleep(10 * time.Millisecond) // Let tasks start

	// Should be overloaded now
	if !pool.IsOverloaded() {
		t.Error("Expected pool to be overloaded")
	}

	wg.Wait()

	// Should not be overloaded after completion
	time.Sleep(10 * time.Millisecond)
	if pool.IsOverloaded() {
		t.Error("Expected pool to not be overloaded after tasks complete")
	}
}

func TestWorkerPool_WaitForCapacity(t *testing.T) {
	config := WorkerPoolConfig{
		MaxWorkers: 1,
	}

	pool := NewWorkerPool(config)
	defer pool.Close()

	// Should have capacity initially
	err := pool.WaitForCapacity(10 * time.Millisecond)
	if err != nil {
		t.Errorf("Expected capacity to be available, got error: %v", err)
	}

	// Fill the pool
	var wg sync.WaitGroup
	wg.Add(1)
	pool.Submit(func() {
		defer wg.Done()
		time.Sleep(100 * time.Millisecond)
	})

	time.Sleep(10 * time.Millisecond) // Let task start

	// Should timeout waiting for capacity
	start := time.Now()
	err = pool.WaitForCapacity(20 * time.Millisecond)
	elapsed := time.Since(start)

	if err == nil {
		t.Error("Expected timeout error when waiting for capacity")
	}

	if elapsed < 15*time.Millisecond || elapsed > 30*time.Millisecond {
		t.Errorf("Expected ~20ms wait, got %v", elapsed)
	}

	wg.Wait()
}

func TestWorkerPool_SubmitWithPriority(t *testing.T) {
	config := WorkerPoolConfig{
		MaxWorkers: 1,
	}

	pool := NewWorkerPool(config)
	defer pool.Close()

	var executed []int
	var mu sync.Mutex

	// Submit regular task that blocks
	var wg sync.WaitGroup
	wg.Add(1)
	pool.Submit(func() {
		defer wg.Done()
		time.Sleep(50 * time.Millisecond)
		mu.Lock()
		executed = append(executed, 1)
		mu.Unlock()
	})

	time.Sleep(10 * time.Millisecond) // Let task start

	// Submit priority task
	wg.Add(1)
	err := pool.SubmitWithPriority(func() {
		defer wg.Done()
		mu.Lock()
		executed = append(executed, 2)
		mu.Unlock()
	})

	if err != nil {
		t.Errorf("Failed to submit priority task: %v", err)
	}

	wg.Wait()

	mu.Lock()
	defer mu.Unlock()

	if len(executed) != 2 {
		t.Errorf("Expected 2 tasks executed, got %d", len(executed))
	}
}

func BenchmarkWorkerPool_Submit(b *testing.B) {
	config := WorkerPoolConfig{
		MaxWorkers: 10,
	}

	pool := NewWorkerPool(config)
	defer pool.Close()

	b.ResetTimer()

	var wg sync.WaitGroup
	for i := 0; i < b.N; i++ {
		wg.Add(1)
		pool.Submit(func() {
			defer wg.Done()
			// Minimal work
		})
	}

	wg.Wait()
}

func BenchmarkWorkerPool_HighConcurrency(b *testing.B) {
	config := WorkerPoolConfig{
		MaxWorkers: 100,
	}

	pool := NewWorkerPool(config)
	defer pool.Close()

	b.ResetTimer()

	var wg sync.WaitGroup
	for i := 0; i < b.N; i++ {
		wg.Add(1)
		pool.Submit(func() {
			defer wg.Done()
			time.Sleep(1 * time.Microsecond)
		})
	}

	wg.Wait()
}
