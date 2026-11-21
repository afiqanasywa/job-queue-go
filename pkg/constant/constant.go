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

// MaxRetries is the maximum number of retry attempts for a job
const MaxRetries = 3
