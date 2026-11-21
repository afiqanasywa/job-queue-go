package query

import (
	"context"
	_dataloader "jobqueue/delivery/graphql/dataloader"
	"jobqueue/delivery/graphql/resolver"
	"jobqueue/entity"
	_interface "jobqueue/interface"

	"github.com/graph-gophers/dataloader/v6"
)

type JobQuery struct {
	jobService _interface.JobService
	dataloader *_dataloader.GeneralDataloader
}

func (q JobQuery) Jobs(ctx context.Context) ([]resolver.JobResolver, error) {
	jobs, err := q.jobService.GetAllJobs(ctx)
	if err != nil {
		return nil, err
	}

	resolvers := make([]resolver.JobResolver, 0, len(jobs))
	for _, job := range jobs {
		resolvers = append(resolvers, resolver.JobResolver{
			Data:       *job,
			JobService: q.jobService,
			Dataloader: q.dataloader,
		})
	}
	return resolvers, nil
}

func (q JobQuery) Job(ctx context.Context, args struct {
	ID string
}) (*resolver.JobResolver, error) {
	job, err := q.jobService.GetJobById(ctx, args.ID)
	if err != nil {
		return nil, err
	}

	return &resolver.JobResolver{
		Data:       *job,
		JobService: q.jobService,
		Dataloader: q.dataloader,
	}, nil
}

func (q JobQuery) JobStatus(ctx context.Context) (resolver.JobStatusResolver, error) {
	stats, err := q.jobService.GetJobStatus(ctx)
	if err != nil {
		return resolver.JobStatusResolver{}, err
	}

	return resolver.JobStatusResolver{
		Data:       *stats,
		JobService: q.jobService,
		Dataloader: q.dataloader,
	}, nil
}

func (q JobQuery) JobsByIds(ctx context.Context, args struct {
	IDs []string
}) ([]resolver.JobResolver, error) {
	loader := q.dataloader.JobLoader(ctx)

	thunks := make([]dataloader.Thunk, 0, len(args.IDs))
	for _, id := range args.IDs {
		thunk := loader.Load(ctx, dataloader.StringKey(id))
		thunks = append(thunks, thunk)
	}

	resolvers := make([]resolver.JobResolver, 0, len(args.IDs))
	for _, thunk := range thunks {
		result, err := thunk()
		if err != nil {
			return nil, err
		}

		if result == nil {
			continue
		}

		job := result.(*entity.Job)
		resolvers = append(resolvers, resolver.JobResolver{
			Data:       *job,
			JobService: q.jobService,
			Dataloader: q.dataloader,
		})
	}

	return resolvers, nil
}

func NewJobQuery(jobService _interface.JobService,
	dataloader *_dataloader.GeneralDataloader) JobQuery {
	return JobQuery{
		jobService: jobService,
		dataloader: dataloader,
	}
}
