package main

import (
	"context"
	"errors"
	"jobqueue/config"
	"jobqueue/delivery/graphql"
	_dataloader "jobqueue/delivery/graphql/dataloader"
	"jobqueue/delivery/graphql/mutation"
	"jobqueue/delivery/graphql/query"
	"jobqueue/delivery/graphql/schema"
	"jobqueue/entity"
	"jobqueue/pkg/handler"
	"jobqueue/pkg/server"
	inmemrepo "jobqueue/repository/inmem"
	"jobqueue/service"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	_graphql "github.com/graph-gophers/graphql-go"
	"github.com/graph-gophers/graphql-go/relay"

	"github.com/labstack/echo/v4/middleware"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func main() {
	setupLogger()
	defer func() { _ = zap.L().Sync() }()

	e := server.New(config.Data.Server)
	e.Echo.HideBanner = true
	e.Echo.Use(middleware.LoggerWithConfig(middleware.LoggerConfig{
		Format: "${remote_ip} ${time_rfc3339_nano} \"${method} ${path}\" ${status} ${bytes_out} \"${referer}\" \"${user_agent}\"\n",
	}))
	e.Echo.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: []string{"*"},
		AllowMethods: []string{http.MethodGet, http.MethodPost, http.MethodOptions},
	}))

	//graphql schema
	opts := make([]_graphql.SchemaOpt, 0)
	opts = append(opts, _graphql.SubscribeResolverTimeout(10*time.Second))

	//initialize in mem database
	inMemDb := make(map[string]*entity.Job)

	//set job repository
	jobRepository := inmemrepo.
		NewJobRepository().
		SetInMemConnection(inMemDb).
		Build()
	dataloader := _dataloader.
		New().
		SetJobRepository(jobRepository).
		Build()

	//set job service
	jobService := service.NewJobService().
		SetJobRepository(jobRepository).
		Build()

	jobMutation := mutation.NewJobMutation(jobService, dataloader)
	jobQuery := query.NewJobQuery(jobService, dataloader)

	rootResolver := graphql.
		New().
		SetJobMutation(jobMutation).
		SetJobQuery(jobQuery).
		Build()

	graphqlSchema := _graphql.MustParseSchema(schema.String(), rootResolver, opts...)
	e.Echo.POST("/graphql",
		handler.GraphQLHandler(&relay.Handler{Schema: graphqlSchema}),
		dataloader.EchoMiddelware,
	)
	e.Echo.GET("/graphql",
		handler.GraphQLHandler(&relay.Handler{Schema: graphqlSchema}),
		dataloader.EchoMiddelware,
	)
	e.Echo.GET("/graphiql", handler.GraphiQLHandler)

	// Run the server until SIGINT/SIGTERM, then shut down gracefully: stop
	// accepting HTTP requests first, then let the job workers finish.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		if err := e.Start(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			zap.L().Fatal("HTTP server error", zap.Error(err))
		}
	}()

	<-ctx.Done()
	zap.L().Info("Shutdown signal received")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := e.Echo.Shutdown(shutdownCtx); err != nil {
		zap.L().Error("HTTP server shutdown error", zap.Error(err))
	}
	if err := jobService.Shutdown(shutdownCtx); err != nil {
		zap.L().Error("Job service shutdown error", zap.Error(err))
	}
	zap.L().Info("Bye")
}

func setupLogger() {
	configLogger := zap.NewDevelopmentConfig()
	configLogger.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	configLogger.DisableStacktrace = true
	logger, _ := configLogger.Build()
	zap.ReplaceGlobals(logger)
}
