# Technical notes

This document explains how the job queue works inside and why it is built this way. For setup and the API, see the [README](../README.md).

## Layers

```
GraphQL (delivery/graphql)   parses requests, maps entities to resolvers
        │
        ▼
Service (service)            worker pool, retries, idempotency, shutdown
        │
        ▼
Repository (repository/inmem) thread-safe storage, atomic state changes
        │
        ▼
Entity (entity)              Job, JobStatus, shared errors
```

Each layer depends only on the interfaces in `interface/job.go`, so the in-memory repository could be replaced by a database without touching the service.

## Job lifecycle

```
            Enqueue
               │
               ▼
          ┌─────────┐   worker claims it    ┌─────────┐
          │ pending │ ────────────────────► │ running │
          └─────────┘   (StartAttempt)      └─────────┘
               ▲                              │   │   │
               │  attempt failed,             │   │   │ success
               │  attempts < max              │   │   ▼
               └── re-queued after backoff ───┘   │ ┌───────────┐
                                                  │ │ completed │
                          attempt failed,         │ └───────────┘
                          attempts == max         ▼
                                            ┌────────┐
                                            │ failed │
                                            └────────┘
```

While a job waits for its retry it is back in `pending`, with `attempts` showing how many tries it has used.

## Enqueue

`Service.Enqueue(ctx, task, idempotencyKey)`:

1. Rejects an empty task (`ErrEmptyTask`) and rejects any job once shutdown has started (`ErrShuttingDown`).
2. Calls `Repository.Save`. If the idempotency key was already used, the repository returns the existing job and the service returns it without queueing anything. The key check and the insert happen under one lock, so two concurrent requests with the same key cannot both create a job.
3. Sends the job ID to the queue channel. If the channel stays full for `enqueueTimeout` (default 2s), or the request is cancelled, the job is deleted again and the caller gets `ErrQueueFull`. This is a deliberate choice: the client gets a clear error it can retry, and no job is left in `pending` that will never run. Deleting the job also frees its idempotency key.

## Workers

`Build()` starts `workerPool` goroutines (default 10). Each loops over:

```go
select {
case <-q.stop:          // shutdown
    return
case jobID := <-q.jobQueue:
    q.processJob(jobID)
}
```

`processJob` runs **one** attempt:

1. `Repository.StartAttempt` atomically checks that the job is `pending`, sets it to `running` and increments `attempts`. If the job is not pending (for example because the same ID was queued twice), the worker skips it. This makes processing idempotent at the execution level too.
2. The executor runs. A panic inside the executor is recovered and counted as a failed attempt, so one bad job cannot kill a worker.
3. On success the job becomes `completed`. On failure it either goes back to `pending` and a retry is scheduled, or becomes `failed` once `maxAttempts` (default 3) is reached.

## Retries without blocking workers

A retry is scheduled in its own goroutine with a timer:

```go
timer := time.NewTimer(delay)
select {
case <-timer.C:        // then put the job ID back on the queue
case <-q.stop:         // shutdown: drop the retry
}
```

The worker returns straight away, so a job waiting 2s for a retry does not hold a worker for those 2s. The default backoff is linear (`attempt × 1s`: 1s, then 2s) and can be replaced with `SetBackoff`.

## Concurrency safety

- The repository guards its map with a `sync.RWMutex`.
- **Every repository method copies jobs on the way in and out.** Workers and GraphQL resolvers each work on their own copy, so no goroutine can change a stored job without holding the lock. Returning shared pointers would let a worker write `Status` while a resolver reads it, which is a data race.
- State transitions that must not interleave (claiming a job, checking an idempotency key) are single repository calls under one lock instead of read-then-write sequences in the service.
- CI runs the whole test suite with `go test -race`.

## Graceful shutdown

On SIGINT/SIGTERM, `main.go`:

1. Stops the HTTP server (`echo.Shutdown`), so no new requests arrive.
2. Calls `Service.Shutdown(ctx)` with a 10s deadline. This closes the `stop` channel: workers finish their current attempt and exit, new enqueues are rejected and scheduled retries are dropped.
3. If the deadline expires first, the context passed to executors is cancelled so running attempts can stop, and `Shutdown` returns `context.DeadlineExceeded`.

The store is in memory, so jobs that are still pending are lost when the process exits. A persistent repository would be needed to resume them.

## DataLoader

`JobsByIds` loads jobs through a [DataLoader](https://github.com/graph-gophers/dataloader), which batches the IDs into one `FindByIDs` call. A new loader is created for every HTTP request by `EchoMiddelware`. A loader caches what it has loaded, so sharing one across requests would keep returning old job statuses. Unknown IDs are skipped instead of failing the whole batch. More detail is in [DATALOADER_BATCHING_EXPLAINED.md](DATALOADER_BATCHING_EXPLAINED.md).

## GraphQL schema

The `.graphql` files in `delivery/graphql/schema` are embedded into the binary with `go:embed`, so there is no code generation step and no schema file to ship.

## Configuration

The service builder sets these defaults, and each one can be overridden:

| Setting | Default | Builder method |
|---|---|---|
| Workers | 10 | `SetWorkerPool` |
| Queue capacity | 100 | `SetQueueSize` |
| Max attempts | 3 | `SetMaxAttempts` |
| Enqueue timeout when the queue is full | 2s | `SetEnqueueTimeout` |
| Backoff | linear, 1s steps | `SetBackoff` |
| Executor | `SimulatedExecutor` | `SetExecutor` |

The HTTP port is set in `config/config.go` from `MY_ENV`: 58579 by default, 58577 for staging and 58578 for production.

## Tests

| Package | What is covered |
|---|---|
| `repository/inmem` | copy semantics, atomic idempotency key under 100 goroutines, `StartAttempt` claims a job only once, stats, concurrent reads and writes |
| `service` | success, unstable job (3 attempts), failure after max attempts, executor panic, idempotent enqueue under concurrency, 100 concurrent jobs, full queue, retries not blocking workers, shutdown, shutdown timeout |
| `delivery/graphql` | end-to-end GraphQL: idempotency key, validation error, `JobsByIds` returning current status and skipping unknown IDs |

Run them with:

```bash
go test ./... -race -count=1
```

`-race` needs cgo. On Windows without gcc, run `go test ./...` locally and let CI run the race detector.
