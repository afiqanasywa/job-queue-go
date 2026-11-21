package entity

import "time"

type Job struct {
	ID             string    `json:"id"`
	Task           string    `json:"task"`
	Status         string    `json:"status"`
	Attempts       int32     `json:"attempts"`
	IdempotencyKey string    `json:"idempotencyKey,omitempty"`
	CreatedAt      time.Time `json:"createdAt"`
}

type JobStatus struct {
	Pending   int32 `json:"pending"`
	Running   int32 `json:"running"`
	Failed    int32 `json:"failed"`
	Completed int32 `json:"completed"`
}
