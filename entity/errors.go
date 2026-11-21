package entity

import "errors"

var (
	ErrJobNotFound   = errors.New("job not found")
	ErrJobExists     = errors.New("job already exists")
	ErrJobNotPending = errors.New("job is not pending")
)
