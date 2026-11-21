package _interface

import (
	"context"
	"jobqueue/entity"
)

type JobService interface {
	// Enqueue creates a job and queues it for processing. When idempotencyKey is
	// not empty and a job with the same key already exists, that job is returned
	// and nothing new is queued.
	Enqueue(ctx context.Context, taskName string, idempotencyKey string) (*entity.Job, error)
	GetAllJobs(ctx context.Context) ([]*entity.Job, error)
	GetJobById(ctx context.Context, id string) (*entity.Job, error)
	GetJobStatus(ctx context.Context) (*entity.JobStatus, error)
	// Shutdown stops accepting jobs and waits for running work to finish or for
	// ctx to expire.
	Shutdown(ctx context.Context) error
}

// JobRepository stores jobs. Every method takes and returns copies, so callers
// never share memory with the store.
type JobRepository interface {
	// Save stores a new job. If job.IdempotencyKey is set and already used, the
	// existing job is returned with created == false.
	Save(ctx context.Context, job *entity.Job) (saved *entity.Job, created bool, err error)
	Update(ctx context.Context, job *entity.Job) error
	Delete(ctx context.Context, id string) error
	// StartAttempt atomically moves a pending job to running and increments its
	// attempt counter. It returns ErrJobNotPending if the job is not pending, so a
	// job can never be processed by two workers at once.
	StartAttempt(ctx context.Context, id string) (*entity.Job, error)
	FindByID(ctx context.Context, id string) (*entity.Job, error)
	// FindByIDs returns the jobs that exist; unknown IDs are skipped.
	FindByIDs(ctx context.Context, ids []string) ([]*entity.Job, error)
	FindAll(ctx context.Context) ([]*entity.Job, error)
	GetJobStats(ctx context.Context) (*entity.JobStatus, error)
}
