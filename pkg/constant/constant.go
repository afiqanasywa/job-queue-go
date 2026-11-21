package constant

type contextKey string

const DataloaderContextKey contextKey = "dataloader"

// Job Status Constants
const (
	JobStatusPending   = "pending"
	JobStatusRunning   = "running"
	JobStatusCompleted = "completed"
	JobStatusFailed    = "failed"
)

// MaxAttempts is the default number of times a job is tried before it is
// marked as failed (the first run plus retries).
const MaxAttempts = 3

// Task names with simulated behaviour in the default executor.
const (
	// TaskUnstable fails twice and succeeds on the third attempt.
	TaskUnstable = "unstable-job"
	// TaskFailing always fails, so it ends in the failed status.
	TaskFailing = "failing-job"
)
