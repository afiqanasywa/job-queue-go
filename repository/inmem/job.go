package inmemrepo

import (
	"context"
	"jobqueue/entity"
	_interface "jobqueue/interface"
	"jobqueue/pkg/constant"
	"sort"
	"sync"
)

// jobRepository keeps jobs in a map guarded by a RWMutex. Jobs are copied on
// the way in and on the way out, so no caller can mutate stored state without
// going through the lock.
type jobRepository struct {
	mu             sync.RWMutex
	inMemDb        map[string]*entity.Job
	idempotencyIdx map[string]string // idempotency key -> job ID
}

func clone(job *entity.Job) *entity.Job {
	c := *job
	return &c
}

func (t *jobRepository) Save(ctx context.Context, job *entity.Job) (*entity.Job, bool, error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if job.IdempotencyKey != "" {
		if id, ok := t.idempotencyIdx[job.IdempotencyKey]; ok {
			return clone(t.inMemDb[id]), false, nil
		}
	}
	if _, exists := t.inMemDb[job.ID]; exists {
		return nil, false, entity.ErrJobExists
	}

	t.inMemDb[job.ID] = clone(job)
	if job.IdempotencyKey != "" {
		t.idempotencyIdx[job.IdempotencyKey] = job.ID
	}
	return clone(job), true, nil
}

func (t *jobRepository) Update(ctx context.Context, job *entity.Job) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if _, exists := t.inMemDb[job.ID]; !exists {
		return entity.ErrJobNotFound
	}
	t.inMemDb[job.ID] = clone(job)
	return nil
}

func (t *jobRepository) Delete(ctx context.Context, id string) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	job, exists := t.inMemDb[id]
	if !exists {
		return entity.ErrJobNotFound
	}
	if job.IdempotencyKey != "" {
		delete(t.idempotencyIdx, job.IdempotencyKey)
	}
	delete(t.inMemDb, id)
	return nil
}

func (t *jobRepository) StartAttempt(ctx context.Context, id string) (*entity.Job, error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	job, exists := t.inMemDb[id]
	if !exists {
		return nil, entity.ErrJobNotFound
	}
	if job.Status != constant.JobStatusPending {
		return nil, entity.ErrJobNotPending
	}
	job.Status = constant.JobStatusRunning
	job.Attempts++
	return clone(job), nil
}

func (t *jobRepository) FindByIDs(ctx context.Context, ids []string) ([]*entity.Job, error) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	jobs := make([]*entity.Job, 0, len(ids))
	for _, id := range ids {
		if job, exists := t.inMemDb[id]; exists {
			jobs = append(jobs, clone(job))
		}
	}
	return jobs, nil
}

func (t *jobRepository) FindByID(ctx context.Context, id string) (*entity.Job, error) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	job, exists := t.inMemDb[id]
	if !exists {
		return nil, entity.ErrJobNotFound
	}
	return clone(job), nil
}

// FindAll returns every job, oldest first.
func (t *jobRepository) FindAll(ctx context.Context) ([]*entity.Job, error) {
	t.mu.RLock()
	jobs := make([]*entity.Job, 0, len(t.inMemDb))
	for _, job := range t.inMemDb {
		jobs = append(jobs, clone(job))
	}
	t.mu.RUnlock()

	sort.Slice(jobs, func(i, j int) bool {
		if jobs[i].CreatedAt.Equal(jobs[j].CreatedAt) {
			return jobs[i].ID < jobs[j].ID
		}
		return jobs[i].CreatedAt.Before(jobs[j].CreatedAt)
	})
	return jobs, nil
}

func (t *jobRepository) GetJobStats(ctx context.Context) (*entity.JobStatus, error) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	stats := &entity.JobStatus{}
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
	r := i(&jobRepository{})
	if r.inMemDb == nil {
		r.inMemDb = make(map[string]*entity.Job)
	}
	r.idempotencyIdx = make(map[string]string)
	for id, job := range r.inMemDb {
		if job.IdempotencyKey != "" {
			r.idempotencyIdx[job.IdempotencyKey] = id
		}
	}
	return r
}
