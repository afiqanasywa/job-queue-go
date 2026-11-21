package graphql_test

import (
	"context"
	"encoding/json"
	"jobqueue/delivery/graphql"
	_dataloader "jobqueue/delivery/graphql/dataloader"
	"jobqueue/delivery/graphql/mutation"
	"jobqueue/delivery/graphql/query"
	"jobqueue/delivery/graphql/schema"
	inmemrepo "jobqueue/repository/inmem"
	"jobqueue/service"
	"strings"
	"testing"
	"time"

	_graphql "github.com/graph-gophers/graphql-go"
)

type api struct {
	schema *_graphql.Schema
	loader *_dataloader.GeneralDataloader
}

func newAPI(t *testing.T) *api {
	t.Helper()
	repo := inmemrepo.NewJobRepository().Build()
	loader := _dataloader.New().SetJobRepository(repo).Build()
	svc := service.NewJobService().
		SetJobRepository(repo).
		SetBackoff(func(int32) time.Duration { return 0 }).
		Build()
	t.Cleanup(func() { _ = svc.Shutdown(context.Background()) })

	root := graphql.New().
		SetJobMutation(mutation.NewJobMutation(svc, loader)).
		SetJobQuery(query.NewJobQuery(svc, loader)).
		Build()
	return &api{schema: _graphql.MustParseSchema(schema.String(), root), loader: loader}
}

// exec runs a query like one HTTP request would: with a fresh dataloader.
func (a *api) exec(t *testing.T, q string, vars map[string]interface{}, out interface{}) {
	t.Helper()
	ctx := a.loader.WithJobLoader(context.Background())
	resp := a.schema.Exec(ctx, q, "", vars)
	if len(resp.Errors) > 0 {
		t.Fatalf("query errors: %v", resp.Errors)
	}
	if err := json.Unmarshal(resp.Data, out); err != nil {
		t.Fatal(err)
	}
}

type job struct {
	ID             string  `json:"id"`
	Task           string  `json:"task"`
	Status         string  `json:"status"`
	Attempts       int     `json:"attempts"`
	IdempotencyKey *string `json:"idempotencyKey"`
}

const enqueueMutation = `mutation($task: String!, $key: String) {
  Enqueue(task: $task, idempotencyKey: $key) { id task status attempts idempotencyKey }
}`

func TestEnqueueWithIdempotencyKey(t *testing.T) {
	a := newAPI(t)
	vars := map[string]interface{}{"task": "send-invoice", "key": "invoice-7"}

	var first, second struct{ Enqueue job }
	a.exec(t, enqueueMutation, vars, &first)
	a.exec(t, enqueueMutation, vars, &second)

	if first.Enqueue.ID == "" || first.Enqueue.ID != second.Enqueue.ID {
		t.Fatalf("same key returned different jobs: %q and %q", first.Enqueue.ID, second.Enqueue.ID)
	}
	if first.Enqueue.IdempotencyKey == nil || *first.Enqueue.IdempotencyKey != "invoice-7" {
		t.Fatalf("idempotencyKey = %v, want invoice-7", first.Enqueue.IdempotencyKey)
	}

	var all struct{ Jobs []job }
	a.exec(t, `{ Jobs { id } }`, nil, &all)
	if len(all.Jobs) != 1 {
		t.Fatalf("Jobs returned %d jobs, want 1", len(all.Jobs))
	}
}

func TestEnqueueEmptyTaskReturnsError(t *testing.T) {
	a := newAPI(t)
	resp := a.schema.Exec(context.Background(), enqueueMutation, "", map[string]interface{}{"task": " "})
	if len(resp.Errors) == 0 || !strings.Contains(resp.Errors[0].Message, "task must not be empty") {
		t.Fatalf("errors = %v, want 'task must not be empty'", resp.Errors)
	}
}

func TestJobsByIdsSeesFreshStatus(t *testing.T) {
	a := newAPI(t)

	var created struct{ Enqueue job }
	a.exec(t, enqueueMutation, map[string]interface{}{"task": "report"}, &created)
	id := created.Enqueue.ID

	// Each exec gets its own loader, so a later request must see the job's
	// new status rather than a cached copy from an earlier request.
	deadline := time.Now().Add(5 * time.Second)
	for {
		var res struct{ JobsByIds []job }
		a.exec(t, `query($ids: [String!]!) { JobsByIds(ids: $ids) { id status } }`,
			map[string]interface{}{"ids": []interface{}{id, "does-not-exist"}}, &res)

		if len(res.JobsByIds) != 1 {
			t.Fatalf("JobsByIds returned %d jobs, want 1 (unknown IDs are skipped)", len(res.JobsByIds))
		}
		if res.JobsByIds[0].Status == "completed" {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("job never showed as completed, last status %q", res.JobsByIds[0].Status)
		}
		time.Sleep(20 * time.Millisecond)
	}

	var stats struct {
		JobStatus struct{ Completed int } `json:"JobStatus"`
	}
	a.exec(t, `{ JobStatus { completed } }`, nil, &stats)
	if stats.JobStatus.Completed != 1 {
		t.Fatalf("completed = %d, want 1", stats.JobStatus.Completed)
	}
}
