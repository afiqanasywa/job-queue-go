package service

import (
	"context"
	"fmt"
	"jobqueue/entity"
	_interface "jobqueue/interface"
	"jobqueue/pkg/constant"
	"sync"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

type jobService struct {
	jobRepo    _interface.JobRepository
	jobQueue   chan string
	workerPool int
	once       sync.Once
}

type Initiator func(s *jobService) *jobService

func (q *jobService) initWorkers() {
	q.once.Do(func() {
		zap.L().Info("Starting worker pool", zap.Int("workers", q.workerPool))
		for i := 0; i < q.workerPool; i++ {
			go q.worker(i)
		}
	})
}

func (q *jobService) worker(id int) {
	for jobID := range q.jobQueue {
		zap.L().Info("Worker processing job", zap.Int("worker_id", id), zap.String("job_id", jobID))
		q.processJob(context.Background(), jobID)
	}
}

func (q *jobService) processJob(ctx context.Context, jobID string) {
	job, err := q.jobRepo.FindByID(ctx, jobID)
	if err != nil {
		zap.L().Error("Failed to find job", zap.String("job_id", jobID), zap.Error(err))
		return
	}

	job.Status = constant.JobStatusRunning
	if err := q.jobRepo.Update(ctx, job); err != nil {
		zap.L().Error("Failed to update job status", zap.String("job_id", jobID), zap.Error(err))
		return
	}

	for job.Attempts < constant.MaxRetries {
		job.Attempts++

		zap.L().Info("Attempting job execution",
			zap.String("job_id", jobID),
			zap.String("task", job.Task),
			zap.Int32("attempt", job.Attempts))

		success := q.executeTask(job.Task, job.Attempts)

		if success {
			job.Status = constant.JobStatusCompleted
			if err := q.jobRepo.Update(ctx, job); err != nil {
				zap.L().Error("Failed to update job status", zap.String("job_id", jobID), zap.Error(err))
			}
			zap.L().Info("Job completed successfully",
				zap.String("job_id", jobID),
				zap.Int32("attempts", job.Attempts))
			return
		}

		if job.Attempts < constant.MaxRetries {
			zap.L().Warn("Job failed, will retry",
				zap.String("job_id", jobID),
				zap.Int32("attempt", job.Attempts),
				zap.Int("max_retries", constant.MaxRetries))

			if err := q.jobRepo.Update(ctx, job); err != nil {
				zap.L().Error("Failed to update job attempts", zap.String("job_id", jobID), zap.Error(err))
			}

			backoffDuration := time.Duration(job.Attempts) * time.Second
			time.Sleep(backoffDuration)
		} else {
			job.Status = constant.JobStatusFailed
			if err := q.jobRepo.Update(ctx, job); err != nil {
				zap.L().Error("Failed to update job status", zap.String("job_id", jobID), zap.Error(err))
			}
			zap.L().Error("Job failed after max retries",
				zap.String("job_id", jobID),
				zap.Int32("attempts", job.Attempts))
			return
		}
	}
}


func (q *jobService) executeTask(task string, attempt int32) bool {
	time.Sleep(100 * time.Millisecond)

	if task == "unstable-job" {
		if attempt < 3 {
			zap.L().Warn("Unstable job intentionally failing",
				zap.String("task", task),
				zap.Int32("attempt", attempt))
			return false
		}
		zap.L().Info("Unstable job succeeding on attempt 3",
			zap.String("task", task),
			zap.Int32("attempt", attempt))
		return true
	}

	return true
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

func (q *jobService) Enqueue(ctx context.Context, taskName string) (*entity.Job, error) {
	
	jobID := uuid.New().String()

	job := &entity.Job{
		ID:       jobID,
		Task:     taskName,
		Status:   constant.JobStatusPending,
		Attempts: 0,
	}

	if err := q.jobRepo.Save(ctx, job); err != nil {
		zap.L().Error("Failed to save job", zap.String("job_id", jobID), zap.Error(err))
		return nil, fmt.Errorf("failed to save job: %w", err)
	}

	zap.L().Info("Job enqueued", zap.String("job_id", jobID), zap.String("task", taskName))

	select {
	case q.jobQueue <- jobID:
	default:
		zap.L().Warn("Job queue full, job saved but not queued immediately", zap.String("job_id", jobID))
	}

	return job, nil
}

func NewJobService() Initiator {
	return func(s *jobService) *jobService {
		s.jobQueue = make(chan string, 100)
		s.workerPool = 10
		s.initWorkers()
		return s
	}
}

func (i Initiator) SetJobRepository(jobRepository _interface.JobRepository) Initiator {
	return func(s *jobService) *jobService {
		i(s).jobRepo = jobRepository
		return s
	}
}

func (i Initiator) Build() _interface.JobService {
	return i(&jobService{})
}
