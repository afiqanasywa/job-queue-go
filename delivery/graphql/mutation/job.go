package mutation

import (
	"context"
	_dataloader "jobqueue/delivery/graphql/dataloader"
	"jobqueue/delivery/graphql/resolver"
	_interface "jobqueue/interface"
)

type JobMutation struct {
	jobService _interface.JobService
	dataloader *_dataloader.GeneralDataloader
}

type EnqueueArgs struct {
	Task           string
	IdempotencyKey *string
}

func (q JobMutation) Enqueue(ctx context.Context, args EnqueueArgs) (*resolver.JobResolver, error) {
	key := ""
	if args.IdempotencyKey != nil {
		key = *args.IdempotencyKey
	}

	job, err := q.jobService.Enqueue(ctx, args.Task, key)
	if err != nil {
		return nil, err
	}
	return &resolver.JobResolver{
		Data:       *job,
		JobService: q.jobService,
		Dataloader: q.dataloader,
	}, nil
}

func NewJobMutation(jobService _interface.JobService, dataloader *_dataloader.GeneralDataloader) JobMutation {
	return JobMutation{
		jobService: jobService,
		dataloader: dataloader,
	}
}
