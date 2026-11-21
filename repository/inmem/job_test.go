package inmemrepo_test

import (
	"context"
	"errors"
	"fmt"
	"jobqueue/entity"
	_interface "jobqueue/interface"
	"jobqueue/pkg/constant"
	inmemrepo "jobqueue/repository/inmem"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func newRepo() _interface.JobRepository {
	return inmemrepo.NewJobRepository().Build()
}

func pendingJob(id string) *entity.Job {
	return &entity.Job{ID: id, Task: "task-" + id, Status: constant.JobStatusPending, CreatedAt: time.Now()}
}

func TestSaveAndFindReturnCopies(t *testing.T) {
	ctx := context.Background()
	repo := newRepo()

	in := pendingJob("a")
	saved, created, err := repo.Save(ctx, in)
	if err != nil || !created {
		t.Fatalf("Save() = created %v, err %v", created, err)
	}

	// Mutating what we passed in or got back must not change the stored job.
	in.Status = "mutated"
	saved.Status = "mutated"
	found, err := repo.FindByID(ctx, "a")
	if err != nil {
		t.Fatal(err)
	}
	found.Attempts = 99

	again, _ := repo.FindByID(ctx, "a")
	if again.Status != constant.JobStatusPending || again.Attempts != 0 {
		t.Fatalf("stored job was mutated through a returned pointer: %+v", again)
	}
}

func TestSaveRejectsDuplicateID(t *testing.T) {
	ctx := context.Background()
	repo := newRepo()

	if _, _, err := repo.Save(ctx, pendingJob("a")); err != nil {
		t.Fatal(err)
	}
	if _, _, err := repo.Save(ctx, pendingJob("a")); !errors.Is(err, entity.ErrJobExists) {
		t.Fatalf("err = %v, want ErrJobExists", err)
	}
}

func TestSaveIdempotencyKeyIsAtomic(t *testing.T) {
	ctx := context.Background()
	repo := newRepo()

	const goroutines = 100
	var createdCount atomic.Int32
	ids := make(chan string, goroutines)

	var wg sync.WaitGroup
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			job := pendingJob(fmt.Sprintf("job-%d", i))
			job.IdempotencyKey = "same-key"
			saved, created, err := repo.Save(ctx, job)
			if err != nil {
				t.Error(err)
				return
			}
			if created {
				createdCount.Add(1)
			}
			ids <- saved.ID
		}(i)
	}
	wg.Wait()
	close(ids)

	if got := createdCount.Load(); got != 1 {
		t.Fatalf("created %d jobs for one idempotency key, want 1", got)
	}
	first := ""
	for id := range ids {
		if first == "" {
			first = id
		}
		if id != first {
			t.Fatalf("callers got different jobs: %s and %s", first, id)
		}
	}
	all, _ := repo.FindAll(ctx)
	if len(all) != 1 {
		t.Fatalf("stored %d jobs, want 1", len(all))
	}
}

func TestDeleteFreesIdempotencyKey(t *testing.T) {
	ctx := context.Background()
	repo := newRepo()

	job := pendingJob("a")
	job.IdempotencyKey = "k"
	if _, _, err := repo.Save(ctx, job); err != nil {
		t.Fatal(err)
	}
	if err := repo.Delete(ctx, "a"); err != nil {
		t.Fatal(err)
	}

	job2 := pendingJob("b")
	job2.IdempotencyKey = "k"
	saved, created, err := repo.Save(ctx, job2)
	if err != nil || !created || saved.ID != "b" {
		t.Fatalf("Save() after Delete = %+v, created %v, err %v", saved, created, err)
	}
}

func TestStartAttemptOnlyOnce(t *testing.T) {
	ctx := context.Background()
	repo := newRepo()
	if _, _, err := repo.Save(ctx, pendingJob("a")); err != nil {
		t.Fatal(err)
	}

	const goroutines = 50
	var started atomic.Int32
	var wg sync.WaitGroup
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := repo.StartAttempt(ctx, "a")
			switch {
			case err == nil:
				started.Add(1)
			case !errors.Is(err, entity.ErrJobNotPending):
				t.Errorf("unexpected error: %v", err)
			}
		}()
	}
	wg.Wait()

	if got := started.Load(); got != 1 {
		t.Fatalf("job started %d times, want 1", got)
	}
	job, _ := repo.FindByID(ctx, "a")
	if job.Status != constant.JobStatusRunning || job.Attempts != 1 {
		t.Fatalf("job = %+v, want running with 1 attempt", job)
	}
}

func TestFindByIDsSkipsUnknown(t *testing.T) {
	ctx := context.Background()
	repo := newRepo()
	for _, id := range []string{"a", "b"} {
		if _, _, err := repo.Save(ctx, pendingJob(id)); err != nil {
			t.Fatal(err)
		}
	}

	jobs, err := repo.FindByIDs(ctx, []string{"a", "missing", "b"})
	if err != nil {
		t.Fatal(err)
	}
	if len(jobs) != 2 {
		t.Fatalf("got %d jobs, want 2", len(jobs))
	}
}

func TestFindAllIsOrderedByCreation(t *testing.T) {
	ctx := context.Background()
	repo := newRepo()
	base := time.Now()
	for i, id := range []string{"c", "a", "b"} {
		job := pendingJob(id)
		job.CreatedAt = base.Add(time.Duration(i) * time.Second)
		if _, _, err := repo.Save(ctx, job); err != nil {
			t.Fatal(err)
		}
	}

	jobs, _ := repo.FindAll(ctx)
	got := [3]string{jobs[0].ID, jobs[1].ID, jobs[2].ID}
	if got != [3]string{"c", "a", "b"} {
		t.Fatalf("order = %v, want [c a b]", got)
	}
}

func TestGetJobStats(t *testing.T) {
	ctx := context.Background()
	repo := newRepo()
	statuses := []string{
		constant.JobStatusPending, constant.JobStatusRunning,
		constant.JobStatusCompleted, constant.JobStatusCompleted,
		constant.JobStatusFailed,
	}
	for i, s := range statuses {
		job := pendingJob(fmt.Sprint(i))
		job.Status = s
		if _, _, err := repo.Save(ctx, job); err != nil {
			t.Fatal(err)
		}
	}

	stats, _ := repo.GetJobStats(ctx)
	want := entity.JobStatus{Pending: 1, Running: 1, Completed: 2, Failed: 1}
	if *stats != want {
		t.Fatalf("stats = %+v, want %+v", *stats, want)
	}
}

func TestConcurrentReadsAndWrites(t *testing.T) {
	ctx := context.Background()
	repo := newRepo()

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		id := fmt.Sprint(i)
		wg.Add(3)
		go func() {
			defer wg.Done()
			if _, _, err := repo.Save(ctx, pendingJob(id)); err != nil {
				t.Error(err)
			}
			if job, err := repo.StartAttempt(ctx, id); err == nil {
				job.Status = constant.JobStatusCompleted
				_ = repo.Update(ctx, job)
			}
		}()
		go func() { defer wg.Done(); _, _ = repo.FindAll(ctx) }()
		go func() { defer wg.Done(); _, _ = repo.GetJobStats(ctx) }()
	}
	wg.Wait()

	stats, _ := repo.GetJobStats(ctx)
	if stats.Completed != 50 {
		t.Fatalf("completed = %d, want 50", stats.Completed)
	}
}
