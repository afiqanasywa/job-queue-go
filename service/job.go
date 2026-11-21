package service

import (
	"context"
	"errors"
	"fmt"
	"jobqueue/entity"
	_interface "jobqueue/interface"
	"jobqueue/pkg/constant"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

var (
	ErrEmptyTask    = errors.New("task must not be empty")
	ErrQueueFull    = errors.New("job queue is full, try again later")
	ErrShuttingDown = errors.New("job service is shutting down")
)

// Executor runs a single attempt of a job. Returning an error marks the
// attempt as failed.
type Executor func(ctx context.Context, job entity.Job) error

// BackoffFunc returns how long to wait before retrying after the given
// (1-based) failed attempt.
type BackoffFunc func(attempt int32) time.Duration

// LinearBackoff waits step, 2*step, 3*step, ... between attempts.
func LinearBackoff(step time.Duration) BackoffFunc {
	return func(attempt int32) time.Duration {
		return time.Duration(attempt) * step
	}
}

// SimulatedExecutor pretends to do 100ms of work. The task names in
// constant.TaskUnstable and constant.TaskFailing simulate failures.
func SimulatedExecutor(ctx context.Context, job entity.Job) error {
	select {
	case <-time.After(100 * time.Millisecond):
	case <-ctx.Done():
		return ctx.Err()
	}

	switch job.Task {
	case constant.TaskUnstable:
		if job.Attempts < 3 {
			return fmt.Errorf("simulated failure on attempt %d", job.Attempts)
		}
	case constant.TaskFailing:
		return errors.New("simulated permanent failure")
	}
	return nil
}

type jobService struct {
	jobRepo        _interface.JobRepository
	jobQueue       chan string
	workerPool     int
	queueSize      int
	maxAttempts    int32
	enqueueTimeout time.Duration
	backoff        BackoffFunc
	execute        Executor

	// stop is closed by Shutdown: workers stop taking jobs, Enqueue is
	// rejected and pending retries are dropped.
	stop     chan struct{}
	stopOnce sync.Once
	// runCtx is passed to executors. It is only cancelled when Shutdown's own
	// deadline expires, so in-flight attempts normally get to finish.
	runCtx    context.Context
	cancelRun context.CancelFunc
	workers   sync.WaitGroup
	retries   sync.WaitGroup
}

func (q *jobService) startWorkers() {
	zap.L().Info("Starting worker pool",
		zap.Int("workers", q.workerPool),
		zap.Int("queue_size", q.queueSize))
	for i := 0; i < q.workerPool; i++ {
		q.workers.Add(1)
		go q.worker(i)
	}
}

func (q *jobService) worker(id int) {
	defer q.workers.Done()
	for {
		select {
		case <-q.stop:
			return
		case jobID := <-q.jobQueue:
			zap.L().Info("Worker picked up job", zap.Int("worker_id", id), zap.String("job_id", jobID))
			q.processJob(jobID)
		}
	}
}

// processJob runs exactly one attempt. On a retryable failure the job goes
// back to pending and is re-queued after the backoff delay, so the worker is
// free to process other jobs in the meantime.
func (q *jobService) processJob(jobID string) {
	ctx := q.runCtx

	job, err := q.jobRepo.StartAttempt(ctx, jobID)
	if err != nil {
		// Not pending means another worker already handled it; this keeps
		// processing idempotent even if an ID is queued twice.
		zap.L().Warn("Skipping job", zap.String("job_id", jobID), zap.Error(err))
		return
	}

	zap.L().Info("Attempting job execution",
		zap.String("job_id", jobID),
		zap.String("task", job.Task),
		zap.Int32("attempt", job.Attempts),
		zap.Int32("max_attempts", q.maxAttempts))

	execErr := q.safeExecute(ctx, *job)

	switch {
	case execErr == nil:
		job.Status = constant.JobStatusCompleted
		q.saveJob(job)
		zap.L().Info("Job completed",
			zap.String("job_id", jobID),
			zap.Int32("attempts", job.Attempts))

	case job.Attempts < q.maxAttempts:
		job.Status = constant.JobStatusPending
		q.saveJob(job)
		delay := q.backoff(job.Attempts)
		zap.L().Warn("Job attempt failed, will retry",
			zap.String("job_id", jobID),
			zap.Int32("attempt", job.Attempts),
			zap.Duration("retry_in", delay),
			zap.Error(execErr))
		q.scheduleRetry(jobID, delay)

	default:
		job.Status = constant.JobStatusFailed
		q.saveJob(job)
		zap.L().Error("Job failed after max attempts",
			zap.String("job_id", jobID),
			zap.Int32("attempts", job.Attempts),
			zap.Error(execErr))
	}
}

// safeExecute turns a panicking executor into an ordinary failed attempt.
func (q *jobService) safeExecute(ctx context.Context, job entity.Job) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("executor panic: %v", r)
		}
	}()
	return q.execute(ctx, job)
}

func (q *jobService) saveJob(job *entity.Job) {
	if err := q.jobRepo.Update(q.runCtx, job); err != nil {
		zap.L().Error("Failed to update job", zap.String("job_id", job.ID), zap.Error(err))
	}
}

func (q *jobService) scheduleRetry(jobID string, delay time.Duration) {
	q.retries.Add(1)
	go func() {
		defer q.retries.Done()

		timer := time.NewTimer(delay)
		defer timer.Stop()
		select {
		case <-timer.C:
		case <-q.stop:
			return
		}

		select {
		case q.jobQueue <- jobID:
		case <-q.stop:
		}
	}()
}

func (q *jobService) GetAllJobs(ctx context.Context) ([]*entity.Job, error) {
	return q.jobRepo.FindAll(ctx)
}

func (q *jobService) GetJobById(ctx context.Context, id string) (*entity.Job, error) {
	return q.jobRepo.FindByID(ctx, id)
}

func (q *jobService) GetJobStatus(ctx context.Context) (*entity.JobStatus, error) {
	return q.jobRepo.GetJobStats(ctx)
}

func (q *jobService) Enqueue(ctx context.Context, taskName string, idempotencyKey string) (*entity.Job, error) {
	taskName = strings.TrimSpace(taskName)
	if taskName == "" {
		return nil, ErrEmptyTask
	}
	select {
	case <-q.stop:
		return nil, ErrShuttingDown
	default:
	}

	job, created, err := q.jobRepo.Save(ctx, &entity.Job{
		ID:             uuid.New().String(),
		Task:           taskName,
		Status:         constant.JobStatusPending,
		IdempotencyKey: idempotencyKey,
		CreatedAt:      time.Now(),
	})
	if err != nil {
		zap.L().Error("Failed to save job", zap.String("task", taskName), zap.Error(err))
		return nil, fmt.Errorf("failed to save job: %w", err)
	}
	if !created {
		zap.L().Info("Duplicate enqueue, returning existing job",
			zap.String("job_id", job.ID),
			zap.String("idempotency_key", idempotencyKey))
		return job, nil
	}

	if err := q.dispatch(ctx, job.ID); err != nil {
		// Roll back so the job does not sit in pending forever and the
		// idempotency key can be used again.
		if delErr := q.jobRepo.Delete(context.Background(), job.ID); delErr != nil {
			zap.L().Error("Failed to roll back job", zap.String("job_id", job.ID), zap.Error(delErr))
		}
		zap.L().Warn("Job rejected", zap.String("job_id", job.ID), zap.Error(err))
		return nil, err
	}

	zap.L().Info("Job enqueued", zap.String("job_id", job.ID), zap.String("task", taskName))
	return job, nil
}

// dispatch puts the job on the queue, waiting up to enqueueTimeout for space.
func (q *jobService) dispatch(ctx context.Context, jobID string) error {
	timer := time.NewTimer(q.enqueueTimeout)
	defer timer.Stop()

	select {
	case q.jobQueue <- jobID:
		return nil
	case <-timer.C:
		return ErrQueueFull
	case <-ctx.Done():
		return ctx.Err()
	case <-q.stop:
		return ErrShuttingDown
	}
}

func (q *jobService) Shutdown(ctx context.Context) error {
	q.stopOnce.Do(func() {
		zap.L().Info("Shutting down job service")
		close(q.stop)
	})

	done := make(chan struct{})
	go func() {
		q.workers.Wait()
		q.retries.Wait()
		close(done)
	}()

	select {
	case <-done:
		q.cancelRun()
		zap.L().Info("Job service stopped")
		return nil
	case <-ctx.Done():
		q.cancelRun() // interrupt attempts that are still running
		return ctx.Err()
	}
}

type Initiator func(s *jobService) *jobService

// NewJobService returns a builder with the default settings: 10 workers, a
// queue of 100, 3 attempts, linear backoff of 1s and the simulated executor.
func NewJobService() Initiator {
	return func(s *jobService) *jobService {
		s.workerPool = 10
		s.queueSize = 100
		s.maxAttempts = constant.MaxAttempts
		s.enqueueTimeout = 2 * time.Second
		s.backoff = LinearBackoff(time.Second)
		s.execute = SimulatedExecutor
		return s
	}
}

func (i Initiator) SetJobRepository(jobRepository _interface.JobRepository) Initiator {
	return func(s *jobService) *jobService {
		i(s).jobRepo = jobRepository
		return s
	}
}

func (i Initiator) SetWorkerPool(n int) Initiator {
	return func(s *jobService) *jobService {
		i(s).workerPool = n
		return s
	}
}

func (i Initiator) SetQueueSize(n int) Initiator {
	return func(s *jobService) *jobService {
		i(s).queueSize = n
		return s
	}
}

func (i Initiator) SetMaxAttempts(n int32) Initiator {
	return func(s *jobService) *jobService {
		i(s).maxAttempts = n
		return s
	}
}

func (i Initiator) SetEnqueueTimeout(d time.Duration) Initiator {
	return func(s *jobService) *jobService {
		i(s).enqueueTimeout = d
		return s
	}
}

func (i Initiator) SetBackoff(b BackoffFunc) Initiator {
	return func(s *jobService) *jobService {
		i(s).backoff = b
		return s
	}
}

func (i Initiator) SetExecutor(e Executor) Initiator {
	return func(s *jobService) *jobService {
		i(s).execute = e
		return s
	}
}

// Build validates the settings and starts the worker pool.
func (i Initiator) Build() _interface.JobService {
	s := i(&jobService{})
	if s.jobRepo == nil {
		panic("service: job repository is required")
	}
	if s.workerPool < 1 {
		s.workerPool = 1
	}
	if s.queueSize < 0 {
		s.queueSize = 0
	}
	if s.maxAttempts < 1 {
		s.maxAttempts = 1
	}

	s.jobQueue = make(chan string, s.queueSize)
	s.stop = make(chan struct{})
	s.runCtx, s.cancelRun = context.WithCancel(context.Background())
	s.startWorkers()
	return s
}
