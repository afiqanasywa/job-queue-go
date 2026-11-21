package service_test

import (
	"context"
	"errors"
	"fmt"
	"jobqueue/entity"
	_interface "jobqueue/interface"
	"jobqueue/pkg/constant"
	inmemrepo "jobqueue/repository/inmem"
	"jobqueue/service"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func noBackoff(int32) time.Duration { return 0 }

func succeed(context.Context, entity.Job) error { return nil }

// build creates a service with fast defaults for tests and shuts it down when
// the test ends. configure can override any setting.
func build(t *testing.T, configure func(service.Initiator) service.Initiator) _interface.JobService {
	t.Helper()
	i := service.NewJobService().
		SetJobRepository(inmemrepo.NewJobRepository().Build()).
		SetBackoff(noBackoff).
		SetExecutor(succeed)
	if configure != nil {
		i = configure(i)
	}
	svc := i.Build()
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = svc.Shutdown(ctx)
	})
	return svc
}

func waitFor(t *testing.T, timeout time.Duration, what string, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("timed out after %s waiting for %s", timeout, what)
}

func waitForStatus(t *testing.T, svc _interface.JobService, id, status string) *entity.Job {
	t.Helper()
	var job *entity.Job
	waitFor(t, 5*time.Second, "job "+id+" to be "+status, func() bool {
		var err error
		job, err = svc.GetJobById(context.Background(), id)
		return err == nil && job.Status == status
	})
	return job
}

func enqueue(t *testing.T, svc _interface.JobService, task, key string) *entity.Job {
	t.Helper()
	job, err := svc.Enqueue(context.Background(), task, key)
	if err != nil {
		t.Fatalf("Enqueue(%q) error: %v", task, err)
	}
	return job
}

func TestEnqueueProcessesJob(t *testing.T) {
	svc := build(t, nil)

	job := enqueue(t, svc, "send-email", "")
	if job.Status != constant.JobStatusPending || job.Attempts != 0 {
		t.Fatalf("new job = %+v, want pending with 0 attempts", job)
	}

	done := waitForStatus(t, svc, job.ID, constant.JobStatusCompleted)
	if done.Attempts != 1 {
		t.Fatalf("attempts = %d, want 1", done.Attempts)
	}
}

func TestEnqueueRejectsEmptyTask(t *testing.T) {
	svc := build(t, nil)

	if _, err := svc.Enqueue(context.Background(), "   ", ""); !errors.Is(err, service.ErrEmptyTask) {
		t.Fatalf("err = %v, want ErrEmptyTask", err)
	}
}

func TestUnstableJobSucceedsOnThirdAttempt(t *testing.T) {
	svc := build(t, func(i service.Initiator) service.Initiator {
		return i.SetExecutor(service.SimulatedExecutor)
	})

	job := enqueue(t, svc, constant.TaskUnstable, "")
	done := waitForStatus(t, svc, job.ID, constant.JobStatusCompleted)
	if done.Attempts != 3 {
		t.Fatalf("attempts = %d, want 3", done.Attempts)
	}
}

func TestJobFailsAfterMaxAttempts(t *testing.T) {
	var calls atomic.Int32
	svc := build(t, func(i service.Initiator) service.Initiator {
		return i.SetExecutor(func(context.Context, entity.Job) error {
			calls.Add(1)
			return errors.New("boom")
		})
	})

	job := enqueue(t, svc, "always-broken", "")
	done := waitForStatus(t, svc, job.ID, constant.JobStatusFailed)
	if done.Attempts != constant.MaxAttempts || calls.Load() != constant.MaxAttempts {
		t.Fatalf("attempts = %d, executor calls = %d, want %d", done.Attempts, calls.Load(), constant.MaxAttempts)
	}
}

func TestExecutorPanicDoesNotCrashWorker(t *testing.T) {
	svc := build(t, func(i service.Initiator) service.Initiator {
		return i.SetWorkerPool(1).SetExecutor(func(_ context.Context, job entity.Job) error {
			if job.Task == "panics" {
				panic("something went very wrong")
			}
			return nil
		})
	})

	bad := enqueue(t, svc, "panics", "")
	waitForStatus(t, svc, bad.ID, constant.JobStatusFailed)

	// The only worker must still be alive.
	good := enqueue(t, svc, "fine", "")
	waitForStatus(t, svc, good.ID, constant.JobStatusCompleted)
}

func TestIdempotentEnqueueProcessesOnce(t *testing.T) {
	var calls atomic.Int32
	svc := build(t, func(i service.Initiator) service.Initiator {
		return i.SetExecutor(func(context.Context, entity.Job) error {
			calls.Add(1)
			return nil
		})
	})

	const requests = 50
	ids := make([]string, requests)
	var wg sync.WaitGroup
	for n := 0; n < requests; n++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			job, err := svc.Enqueue(context.Background(), "charge-card", "order-42")
			if err != nil {
				t.Error(err)
				return
			}
			ids[n] = job.ID
		}(n)
	}
	wg.Wait()

	for _, id := range ids {
		if id != ids[0] {
			t.Fatalf("requests with the same key got different jobs: %s and %s", ids[0], id)
		}
	}
	waitForStatus(t, svc, ids[0], constant.JobStatusCompleted)

	// Give any (wrong) duplicate processing a moment to show up.
	time.Sleep(50 * time.Millisecond)
	if got := calls.Load(); got != 1 {
		t.Fatalf("executor ran %d times, want 1", got)
	}
	all, _ := svc.GetAllJobs(context.Background())
	if len(all) != 1 {
		t.Fatalf("stored %d jobs, want 1", len(all))
	}
}

func TestManyConcurrentJobs(t *testing.T) {
	var calls atomic.Int32
	svc := build(t, func(i service.Initiator) service.Initiator {
		return i.SetExecutor(func(context.Context, entity.Job) error {
			calls.Add(1)
			time.Sleep(10 * time.Millisecond)
			return nil
		})
	})

	const jobs = 100
	var wg sync.WaitGroup
	for n := 0; n < jobs; n++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			if _, err := svc.Enqueue(context.Background(), fmt.Sprintf("job-%d", n), ""); err != nil {
				t.Error(err)
			}
		}(n)
	}
	wg.Wait()

	waitFor(t, 10*time.Second, "all jobs to complete", func() bool {
		stats, _ := svc.GetJobStatus(context.Background())
		return stats.Completed == jobs
	})
	if got := calls.Load(); got != jobs {
		t.Fatalf("executor ran %d times, want %d", got, jobs)
	}
}

func TestQueueFullRejectsJob(t *testing.T) {
	release := make(chan struct{})
	svc := build(t, func(i service.Initiator) service.Initiator {
		return i.SetWorkerPool(1).
			SetQueueSize(1).
			SetEnqueueTimeout(50 * time.Millisecond).
			SetExecutor(func(ctx context.Context, _ entity.Job) error {
				select {
				case <-release:
				case <-ctx.Done():
				}
				return nil
			})
	})
	ctx := context.Background()

	first := enqueue(t, svc, "first", "")
	waitForStatus(t, svc, first.ID, constant.JobStatusRunning) // the worker is now busy
	second := enqueue(t, svc, "second", "")                    // fills the buffer

	_, err := svc.Enqueue(ctx, "third", "retry-me")
	if !errors.Is(err, service.ErrQueueFull) {
		t.Fatalf("err = %v, want ErrQueueFull", err)
	}
	all, _ := svc.GetAllJobs(ctx)
	if len(all) != 2 {
		t.Fatalf("stored %d jobs, want 2 (rejected job must not be left pending)", len(all))
	}

	close(release)
	waitForStatus(t, svc, first.ID, constant.JobStatusCompleted)
	waitForStatus(t, svc, second.ID, constant.JobStatusCompleted)

	// The rejected job's idempotency key is free again.
	third := enqueue(t, svc, "third", "retry-me")
	waitForStatus(t, svc, third.ID, constant.JobStatusCompleted)
}

func TestRetryBackoffDoesNotBlockWorker(t *testing.T) {
	svc := build(t, func(i service.Initiator) service.Initiator {
		return i.SetWorkerPool(1).
			SetBackoff(func(int32) time.Duration { return 300 * time.Millisecond }).
			SetExecutor(func(_ context.Context, job entity.Job) error {
				if job.Task == "flaky" && job.Attempts == 1 {
					return errors.New("first attempt fails")
				}
				return nil
			})
	})
	ctx := context.Background()

	flaky := enqueue(t, svc, "flaky", "")
	waitFor(t, time.Second, "first attempt to fail", func() bool {
		job, _ := svc.GetJobById(ctx, flaky.ID)
		return job.Attempts == 1 && job.Status == constant.JobStatusPending
	})

	// While flaky waits for its retry, the single worker must be free.
	fast := enqueue(t, svc, "fast", "")
	waitForStatus(t, svc, fast.ID, constant.JobStatusCompleted)
	if job, _ := svc.GetJobById(ctx, flaky.ID); job.Status != constant.JobStatusPending {
		t.Fatalf("flaky status = %s, want pending (still waiting for retry)", job.Status)
	}

	done := waitForStatus(t, svc, flaky.ID, constant.JobStatusCompleted)
	if done.Attempts != 2 {
		t.Fatalf("flaky attempts = %d, want 2", done.Attempts)
	}
}

func TestShutdownStopsWorkersAndRejectsNewJobs(t *testing.T) {
	svc := build(t, func(i service.Initiator) service.Initiator {
		return i.
			SetBackoff(func(int32) time.Duration { return time.Hour }).
			SetExecutor(func(context.Context, entity.Job) error { return errors.New("fail") })
	})

	// Leave a retry scheduled an hour from now; Shutdown must not wait for it.
	job := enqueue(t, svc, "waits-for-retry", "")
	waitFor(t, time.Second, "first attempt to fail", func() bool {
		j, _ := svc.GetJobById(context.Background(), job.ID)
		return j.Attempts == 1 && j.Status == constant.JobStatusPending
	})

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := svc.Shutdown(ctx); err != nil {
		t.Fatalf("Shutdown() error: %v", err)
	}

	if _, err := svc.Enqueue(context.Background(), "too-late", ""); !errors.Is(err, service.ErrShuttingDown) {
		t.Fatalf("err = %v, want ErrShuttingDown", err)
	}
}

func TestShutdownTimeoutCancelsRunningJob(t *testing.T) {
	cancelled := make(chan struct{})
	svc := build(t, func(i service.Initiator) service.Initiator {
		return i.SetExecutor(func(ctx context.Context, _ entity.Job) error {
			<-ctx.Done()
			close(cancelled)
			return ctx.Err()
		})
	})

	job := enqueue(t, svc, "slow", "")
	waitForStatus(t, svc, job.ID, constant.JobStatusRunning)

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	if err := svc.Shutdown(ctx); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("Shutdown() = %v, want DeadlineExceeded", err)
	}

	select {
	case <-cancelled:
	case <-time.After(time.Second):
		t.Fatal("running job was not cancelled after the shutdown deadline")
	}
}
