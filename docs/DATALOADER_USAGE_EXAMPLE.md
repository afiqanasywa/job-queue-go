# How to Trigger DataLoader in Your Job Queue System

## Problem

The `JobBatchFunc` in the DataLoader is not being triggered because your queries bypass the DataLoader and call the service directly.

## Current Implementation (Not Using DataLoader)

```go
// query/job.go - Current implementation
func (q JobQuery) Jobs(ctx context.Context) ([]resolver.JobResolver, error) {
    jobs, err := q.jobService.GetAllJobs(ctx)  // ❌ Bypasses DataLoader
    // ...
}
```

## Solution: Use DataLoader Properly

### Option 1: Create a Query That Returns Job IDs, Then Use DataLoader

Modify your GraphQL schema and implementation to leverage DataLoader:

#### Step 1: Update Your Query to Return IDs First

```go
// query/job.go
func (q JobQuery) Jobs(ctx context.Context) ([]resolver.JobResolver, error) {
    jobs, err := q.jobService.GetAllJobs(ctx)
    if err != nil {
        return nil, err
    }

    resolvers := make([]resolver.JobResolver, 0, len(jobs))
    for _, job := range jobs {
        resolvers = append(resolvers, resolver.JobResolver{
            // Only store the ID, not the full data
            Data:       entity.Job{ID: job.ID},
            JobService: q.jobService,
            Dataloader: q.dataloader,
        })
    }
    return resolvers, nil
}
```

#### Step 2: Modify Resolver to Use DataLoader

```go
// resolver/job.go
import (
    "context"
    _dataloader "jobqueue/delivery/graphql/dataloader"
    "jobqueue/entity"
    _interface "jobqueue/interface"
    "jobqueue/pkg/constant"
    "github.com/graph-gophers/dataloader/v6"
)

type JobResolver struct {
    Data       entity.Job
    JobService _interface.JobService
    Dataloader *_dataloader.GeneralDataloader
}

// Task uses DataLoader to fetch the job
func (q JobResolver) Task(ctx context.Context) (string, error) {
    // Get dataloader from context
    loader := q.Dataloader.JobLoader

    // Load job using DataLoader - THIS TRIGGERS JobBatchFunc!
    thunk := loader.Load(ctx, dataloader.StringKey(q.Data.ID))
    result, err := thunk()
    if err != nil {
        return "", err
    }

    job := result.(*entity.Job)
    return job.Task, nil
}

// Status uses DataLoader
func (q JobResolver) Status(ctx context.Context) (string, error) {
    loader := q.Dataloader.JobLoader
    thunk := loader.Load(ctx, dataloader.StringKey(q.Data.ID))
    result, err := thunk()
    if err != nil {
        return "", err
    }

    job := result.(*entity.Job)
    return job.Status, nil
}

// Attempts uses DataLoader
func (q JobResolver) Attempts(ctx context.Context) (int32, error) {
    loader := q.Dataloader.JobLoader
    thunk := loader.Load(ctx, dataloader.StringKey(q.Data.ID))
    result, err := thunk()
    if err != nil {
        return 0, err
    }

    job := result.(*entity.Job)
    return job.Attempts, nil
}
```

### Option 2: Create a Separate Query for Batch Loading

Add a new GraphQL query specifically designed to test DataLoader:

#### Step 1: Add New Query Method

```go
// query/job.go
func (q JobQuery) JobsByIds(ctx context.Context, args struct {
    IDs []string
}) ([]resolver.JobResolver, error) {
    loader := q.dataloader.JobLoader

    resolvers := make([]resolver.JobResolver, 0, len(args.IDs))

    // Load all jobs using DataLoader - THIS TRIGGERS JobBatchFunc!
    for _, id := range args.IDs {
        thunk := loader.Load(ctx, dataloader.StringKey(id))
        result, err := thunk()
        if err != nil {
            return nil, err
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
```

#### Step 2: Update GraphQL Schema

```graphql
# schema/query.graphql
type Query {
  # Existing queries
  Jobs: [Job!]!
  Job(id: String!): Job
  JobStatus: JobStatus!

  # New query to test DataLoader
  JobsByIds(ids: [String!]!): [Job!]!
}
```

## Testing DataLoader

### Test Query 1: Using JobsByIds

```graphql
query {
  JobsByIds(ids: ["job-id-1", "job-id-2", "job-id-3"]) {
    id
    task
    status
    attempts
  }
}
```

This will trigger `JobBatchFunc` with all 3 IDs in a single batch!

### Test Query 2: Multiple Individual Queries (Batched)

```graphql
query {
  job1: Job(id: "job-id-1") {
    id
    task
  }
  job2: Job(id: "job-id-2") {
    id
    task
  }
  job3: Job(id: "job-id-3") {
    id
    task
  }
}
```

If executed within the same request, DataLoader will batch these into a single `JobBatchFunc` call!

## Why DataLoader Wasn't Triggered

1. **Direct Service Calls**: Your queries call `jobService.GetAllJobs()` directly
2. **No DataLoader.Load()**: You never call `loader.Load(ctx, key)` which is what triggers batching
3. **Data Already Loaded**: You're passing the full `Job` entity to the resolver, so there's no need to lazy-load

## DataLoader Benefits

When properly implemented, DataLoader:

- ✅ Batches multiple requests into a single database/repository call
- ✅ Caches results within a single request
- ✅ Prevents N+1 query problems
- ✅ Automatically deduplicates requests

## Quick Test

To quickly test if DataLoader works, create this simple query:

```go
// Add to query/job.go
func (q JobQuery) TestDataLoader(ctx context.Context) (string, error) {
    loader := q.dataloader.JobLoader

    // Simulate loading 3 jobs
    ids := []string{"id1", "id2", "id3"}

    for _, id := range ids {
        loader.Load(ctx, dataloader.StringKey(id))
    }

    return "DataLoader test triggered - check logs!", nil
}
```

Then query:

```graphql
query {
  TestDataLoader
}
```

You should see the log from `JobBatchFunc` with all 3 IDs!
