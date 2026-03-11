package scheduler

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/go-co-op/gocron/v2"
	"go.uber.org/zap"
)

type Scheduler struct {
	cron   gocron.Scheduler
	logger *zap.Logger
}

func NewScheduler(logger *zap.Logger) (*Scheduler, error) {
	sch, err := gocron.NewScheduler()
	if err != nil {
		return nil, fmt.Errorf("error creating scheduler: %w", err)
	}

	return &Scheduler{
		cron:   sch,
		logger: logger,
	}, nil
}

func (scheduler *Scheduler) Start(ctx context.Context) {
	scheduler.logger.Info("starting scheduler")
	scheduler.cron.Start()

	<-ctx.Done()

	if err := scheduler.cron.Shutdown(); err != nil {
		scheduler.logger.Error("error shutting down scheduler", zap.Error(err))
	} else {
		scheduler.logger.Info("scheduler stopped")
	}
}

func (scheduler *Scheduler) Register(definition gocron.JobDefinition, task gocron.Task, options ...gocron.JobOption) error {
	_, err := scheduler.cron.NewJob(definition, task, options...)
	if err != nil {
		scheduler.logger.Error("error creating job", zap.Error(err))
		return err
	}

	return nil
}

func (scheduler *Scheduler) ScheduleOnce(delay time.Duration, name string, task func()) error {
	if delay < 0 {
		return errors.New("scheduler: delay must be non-negative")
	}

	definition := gocron.OneTimeJob(gocron.OneTimeJobStartImmediately())
	if delay > 0 {
		definition = gocron.OneTimeJob(
			gocron.OneTimeJobStartDateTime(time.Now().Add(delay)),
		)
	}

	options := make([]gocron.JobOption, 0, 1)
	if name != "" {
		options = append(options, gocron.WithName(name))
	}

	return scheduler.Register(definition, gocron.NewTask(task), options...)
}

func (scheduler *Scheduler) ScheduleDailyAt(name string, hour, minute, second int, task func()) error {
	if hour < 0 || minute < 0 || second < 0 {
		return errors.New("scheduler: daily time values must be non-negative")
	}

	options := make([]gocron.JobOption, 0, 1)
	if name != "" {
		options = append(options, gocron.WithName(name))
	}

	return scheduler.Register(
		gocron.DailyJob(
			1,
			gocron.NewAtTimes(gocron.NewAtTime(uint(hour), uint(minute), uint(second))),
		),
		gocron.NewTask(task),
		options...,
	)
}
