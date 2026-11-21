package _dataloader

import (
	"context"
	"jobqueue/entity"

	"github.com/graph-gophers/dataloader/v6"
	"go.uber.org/zap"
)

func (s *GeneralDataloader) JobBatchFunc(ctx context.Context, keys dataloader.Keys) []*dataloader.Result {
	results := make([]*dataloader.Result, len(keys))

	// ---- 1. Extract IDs from keys ----
	ids := make([]string, len(keys))
	for i, k := range keys {
		ids[i] = k.String()
	}

	zap.L().Info("JobBatchFunc called - DataLoader batching!",
		zap.Strings("job_ids", ids),
		zap.Int("batch_size", len(ids)))

	// ---- 2. Query repository in batch ----
	jobs, err := s.jobRepo.FindByIDs(ctx, ids)
	if err != nil {
		// error untuk semua result
		for i := range results {
			results[i] = &dataloader.Result{Error: err}
		}
		return results
	}

	// ---- 3. Map jobs by ID (agar cepat dicocokkan kembali dengan keys) ----
	jobMap := make(map[string]*entity.Job, len(jobs))
	for _, job := range jobs {
		jobMap[job.ID] = job
	}

	// ---- 4. Susun result sesuai urutan `keys` ----
	for i, id := range ids {
		if job, ok := jobMap[id]; ok {
			results[i] = &dataloader.Result{Data: job, Error: nil}
		} else {
			// kalau tidak ditemukan
			results[i] = &dataloader.Result{Data: nil, Error: nil}
		}
	}

	return results
}
