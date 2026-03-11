package main

import (
	"context"
	"fmt"

	"github.com/boreauxx/monitoring-bot.git/internal/logger"
	"github.com/boreauxx/monitoring-bot.git/internal/postgres"
	"github.com/boreauxx/monitoring-bot.git/internal/scheduler"
	"go.uber.org/zap"
)

func run(ctx context.Context) error {
	res, err := getResources()
	if err != nil {
		return err
	}

	defer res.cleanup()

	if err = registerJobs(res); err != nil {
		return err
	}

	res.scheduler.Start(ctx)

	<-ctx.Done()

	res.logger.Info("shutting down...")

	return nil
}

type resources struct {
	logger     *zap.Logger
	cleanup    func()
	repository *postgres.Repository
	scheduler  *scheduler.Scheduler
}

func getResources() (*resources, error) {
	log, cleanup := logger.NewZapLogger("Monitoring-Bot")

	pgConf := postgres.Default()

	if err := postgres.Migrate(pgConf); err != nil {
		return nil, fmt.Errorf("error migrating database: %w", err)
	}

	repository, err := postgres.NewRepository(pgConf)
	if err != nil {
		return nil, fmt.Errorf("error creating repository repository: %w", err)
	}

	sched, err := scheduler.NewScheduler(log)
	if err != nil {
		return nil, fmt.Errorf("error creating scheduler: %w", err)
	}

	return &resources{
		logger:     log,
		cleanup:    cleanup,
		repository: repository,
		scheduler:  sched,
	}, nil
}
