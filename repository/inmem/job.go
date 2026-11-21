package inmemrepo

import (
	"context"
	"errors"
	"jobqueue/entity"
	_interface "jobqueue/interface"
	"jobqueue/pkg/constant"
	"sync"
)

type jobRepository struct {
	mu      sync.RWMutex
	inMemDb map[string]*entity.Job
}

func (t *jobRepository) Save(ctx context.Context, job *entity.Job) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.inMemDb[job.ID] = job
	return nil
}

func (t *jobRepository) Update(ctx context.Context, job *entity.Job) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if _, exists := t.inMemDb[job.ID]; !exists {
		return errors.New("job not found")
	}
	t.inMemDb[job.ID] = job
	return nil
}

func (t *jobRepository) FindByIDs(ctx context.Context, ids []string) ([]*entity.Job, error) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	var jobs []*entity.Job
	for _, id := range ids {
		job, exists := t.inMemDb[id]
		if !exists {
			return nil, errors.New("job not found")
		}
		jobs = append(jobs, job)
	}
	return jobs, nil
}

func (t *jobRepository) FindByID(ctx context.Context, id string) (*entity.Job, error) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	job, exists := t.inMemDb[id]
	if !exists {
		return nil, errors.New("job not found")
	}
	return job, nil
}

func (t *jobRepository) FindAll(ctx context.Context) ([]*entity.Job, error) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	var jobs []*entity.Job
	for _, job := range t.inMemDb {
		jobs = append(jobs, job)
	}
	return jobs, nil
}

func (t *jobRepository) GetJobStats(ctx context.Context) (*entity.JobStatus, error) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	stats := &entity.JobStatus{
		Pending:   0,
		Running:   0,
		Failed:    0,
		Completed: 0,
	}

	for _, job := range t.inMemDb {
		switch job.Status {
		case constant.JobStatusPending:
			stats.Pending++
		case constant.JobStatusRunning:
			stats.Running++
		case constant.JobStatusFailed:
			stats.Failed++
		case constant.JobStatusCompleted:
			stats.Completed++
		}
	}

	return stats, nil
}

type Initiator func(s *jobRepository) *jobRepository

func NewJobRepository() Initiator {
	return func(q *jobRepository) *jobRepository {
		return q
	}
}

func (i Initiator) SetInMemConnection(inMemDb map[string]*entity.Job) Initiator {
	return func(s *jobRepository) *jobRepository {
		i(s).inMemDb = inMemDb
		return s
	}
}

func (i Initiator) Build() _interface.JobRepository {
	return i(&jobRepository{})
}
