package _dataloader

import (
	"context"

	_interface "jobqueue/interface"
	"jobqueue/pkg/constant"

	"github.com/graph-gophers/dataloader/v6"
	"github.com/labstack/echo/v4"
)

// GeneralDataloader creates DataLoaders. Loaders cache what they load, so a
// fresh one is created for every request; a shared loader would keep serving
// stale job statuses.
type GeneralDataloader struct {
	jobRepo _interface.JobRepository
}

// NewJobLoader returns a loader that batches job lookups by ID.
func (g *GeneralDataloader) NewJobLoader() *dataloader.Loader {
	return dataloader.NewBatchedLoader(g.JobBatchFunc, dataloader.WithCache(dataloader.NewCache()))
}

// WithJobLoader stores a new per-request job loader in ctx.
func (g *GeneralDataloader) WithJobLoader(ctx context.Context) context.Context {
	return context.WithValue(ctx, constant.DataloaderContextKey, g.NewJobLoader())
}

// JobLoader returns the request's loader from ctx, or a new one if the request
// did not go through the middleware.
func (g *GeneralDataloader) JobLoader(ctx context.Context) *dataloader.Loader {
	if l, ok := ctx.Value(constant.DataloaderContextKey).(*dataloader.Loader); ok {
		return l
	}
	return g.NewJobLoader()
}

// EchoMiddelware attaches a per-request job loader to the request context.
func (g *GeneralDataloader) EchoMiddelware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		req := c.Request()
		c.SetRequest(req.WithContext(g.WithJobLoader(req.Context())))
		return next(c)
	}
}

// Initiator ...
type Initiator func(r *GeneralDataloader) *GeneralDataloader

// New ...
func New() Initiator {
	return func(r *GeneralDataloader) *GeneralDataloader {
		return r
	}
}

// SetJobRepository ...
func (i Initiator) SetJobRepository(repo _interface.JobRepository) Initiator {
	return func(s *GeneralDataloader) *GeneralDataloader {
		i(s).jobRepo = repo
		return s
	}
}

// Build ...
func (i Initiator) Build() *GeneralDataloader {
	return i(&GeneralDataloader{})
}
