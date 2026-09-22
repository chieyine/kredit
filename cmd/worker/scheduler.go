package main

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"
)

// Each activity owns one goroutine. A slow run never overlaps its next run or
// holds up another activity. Deadlines are passed through to the actual work.
type periodicTask struct {
	name     string
	interval time.Duration
	budget   time.Duration
	critical bool
	work     func(context.Context) error

	mu          sync.Mutex
	lastSuccess time.Time
	started     time.Time
	running     bool
}

type workerSchedule struct {
	tasks  []*periodicTask
	cancel context.CancelFunc
	done   chan struct{}
}

func startWorkerSchedule(ctx context.Context, logger *slog.Logger, tasks []*periodicTask) *workerSchedule {
	ctx, cancel := context.WithCancel(ctx)
	schedule := &workerSchedule{tasks: tasks, cancel: cancel, done: make(chan struct{})}
	var group sync.WaitGroup
	for _, task := range tasks {
		group.Add(1)
		go func(task *periodicTask) {
			defer group.Done()
			ticker := time.NewTicker(task.interval)
			defer ticker.Stop()
			for {
				if ctx.Err() != nil {
					return
				}
				task.mu.Lock()
				task.started, task.running = time.Now(), true
				task.mu.Unlock()
				runCtx, cancel := context.WithTimeout(ctx, task.budget)
				err := task.work(runCtx)
				if err == nil {
					err = runCtx.Err()
				}
				cancel()
				task.mu.Lock()
				task.running = false
				if err == nil {
					task.lastSuccess = time.Now()
				}
				task.mu.Unlock()
				if err != nil && ctx.Err() == nil {
					logger.Error("worker scheduled activity failed", "activity", task.name, "error", err)
				}
				select {
				case <-ctx.Done():
					return
				case <-ticker.C:
				}
			}
		}(task)
	}
	go func() {
		group.Wait()
		close(schedule.done)
	}()
	return schedule
}

// Ready reports local scheduling progress, not completion of queued River jobs
// or the health of an entire deployment. Optional providers do not gate it.
func (s *workerSchedule) Ready() error {
	now := time.Now()
	for _, task := range s.tasks {
		if !task.critical {
			continue
		}
		task.mu.Lock()
		lastSuccess, started, running := task.lastSuccess, task.started, task.running
		task.mu.Unlock()
		if lastSuccess.IsZero() {
			return fmt.Errorf("worker activity %s has not completed its first successful run", task.name)
		}
		if (running && now.Sub(started) > task.budget+5*time.Second) || now.Sub(lastSuccess) > 2*task.interval+task.budget {
			return fmt.Errorf("worker activity %s has stopped making progress", task.name)
		}
	}
	return nil
}

func (s *workerSchedule) Stop(ctx context.Context) error {
	s.cancel()
	select {
	case <-s.done:
		return nil
	case <-ctx.Done():
		return fmt.Errorf("wait for worker scheduling shutdown: %w", ctx.Err())
	}
}
