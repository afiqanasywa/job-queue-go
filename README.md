# job-queue-go

A personal case study: a simple in-memory job queue in Go, with a GraphQL API.

Jobs are submitted through a GraphQL mutation, processed in the background by a pool of workers, and retried when they fail. All state is kept in memory, so there is no database to set up.

![GraphiQL](./docs/graphiql.jpg)

## Features

- **Worker pool**: 10 goroutines read job IDs from a buffered channel (capacity 100).
- **Retry with backoff**: a failed job is retried up to 3 attempts in total, waiting 1s and then 2s between attempts.
- **Simulated flaky task**: a job with task `unstable-job` fails on its first two attempts and succeeds on the third, which shows the retry path working.
- **Thread-safe storage**: the in-memory repository guards its map with a `sync.RWMutex`.
- **GraphQL API** (graph-gophers/graphql-go on Echo), with a DataLoader for batched lookups by ID.
- **Structured logging** with zap. Each enqueue, attempt, retry, success and failure is logged.

## Architecture

```
delivery/graphql   GraphQL schema, resolvers, queries, mutations, dataloader
service            Job processing: worker pool, retry logic
repository/inmem   Thread-safe in-memory storage
entity             Domain models (Job, JobStatus)
interface          Contracts between layers
pkg                Server bootstrap, handlers, constants
```

## Running it

You need Go 1.23 or newer.

```bash
go run .
```

Then open http://localhost:58579/graphiql. In VS Code you can also use the `Go-Debug` launch configuration.

## API

| Operation | Type | Description |
|---|---|---|
| `Enqueue(task: String!)` | Mutation | Creates a job and returns it with status `pending` |
| `Jobs` | Query | Lists all jobs |
| `Job(id: String!)` | Query | Gets one job by ID |
| `JobsByIds(ids: [String!]!)` | Query | Gets several jobs at once, batched by the DataLoader |
| `JobStatus` | Query | Counts jobs by status: pending, running, failed, completed |

### Example

```graphql
mutation {
  a: Enqueue(task: "send-email")   { id status attempts }
  b: Enqueue(task: "unstable-job") { id status attempts }
}
```

Run this query a few seconds later:

```graphql
{
  Jobs { task status attempts }
  JobStatus { pending running failed completed }
}
```

```json
{
  "Jobs": [
    { "task": "send-email",   "status": "completed", "attempts": 1 },
    { "task": "unstable-job", "status": "completed", "attempts": 3 }
  ],
  "JobStatus": { "pending": 0, "running": 0, "failed": 0, "completed": 2 }
}
```

[docs/playground.md](docs/playground.md) has more queries you can try.

## Known limitations and what I would improve

I wrote down these gaps while reviewing the project afterwards:

- **Shared pointers**: the repository stores and returns `*Job` pointers. A worker changes a job's fields while API readers can read the same object, so this is a data race. The repository should store and return copies instead.
- **No idempotency**: each `Enqueue` creates a new UUID, so submitting the same job twice processes it twice. An idempotency key should be accepted and checked.
- **Full queue**: when the channel buffer is full, the job is saved but never queued, so it stays `pending` forever. A fix is to block with a context timeout, return an error, or re-queue the job later.
- **Blocking backoff**: retries call `time.Sleep` inside the worker, so a retrying job keeps a worker busy. A scheduler or delayed re-queue would avoid that.
- **No graceful shutdown or tests**: workers do not stop on shutdown, and there are no unit or race tests (`go test -race`) yet.

## Further documentation

- [docs/TECHNICAL.md](docs/TECHNICAL.md): technical deep dive
- [docs/DOCUMENTATION_SUMMARY.md](docs/DOCUMENTATION_SUMMARY.md): quick reference
- [docs/DATALOADER_BATCHING_EXPLAINED.md](docs/DATALOADER_BATCHING_EXPLAINED.md): how the DataLoader batching works
