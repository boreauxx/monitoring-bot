package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/boreauxx/monitoring-bot.git/internal/postgres"
	"github.com/boreauxx/monitoring-bot.git/internal/probe"
	"github.com/go-co-op/gocron/v2"
	"go.uber.org/zap"
)

func registerJobs(res *resources) error {
	if err := registerProbeJob(res); err != nil {
		return err
	}
	if err := registerProbeCleanupJob(res); err != nil {
		return err
	}
	return nil
}

// ---------------------------------------------------------------------------------------------------------------------

func registerProbeJob(res *resources) error {
	type settings struct {
		url      string
		name     string
		interval string
		timeout  string
	}

	var setts settings

	flag.StringVar(&setts.url, "url", "https://petguides.bg", "URL of the target to be monitored")
	flag.StringVar(&setts.name, "name", "Pet Guides", "name of the target")
	flag.StringVar(&setts.interval, "interval", "30s", "interval in between requests")
	flag.StringVar(&setts.timeout, "timeout", "10s", "timeout for requests")
	flag.Parse()

	interval, err := time.ParseDuration(setts.interval)
	if err != nil {
		return fmt.Errorf("error parsing interval: %w", err)
	}

	timeout, err := time.ParseDuration(setts.timeout)
	if err != nil {
		return fmt.Errorf("error parsing timeout: %w", err)
	}

	asset := &postgres.Asset{
		Name:            setts.name,
		Address:         setts.url,
		IntervalSeconds: int(interval.Seconds()),
		TimeoutSeconds:  int(timeout.Seconds()),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	stored, err := res.repository.StoreAsset(ctx, asset)
	if err != nil {
		return err
	}

	if err = res.scheduler.Register(
		gocron.DurationJob(interval),
		newProbeTask(res, stored),
		gocron.WithSingletonMode(gocron.LimitModeReschedule),
	); err != nil {
		return fmt.Errorf("error registering probes job: %w", err)
	}

	return nil
}

func newProbeTask(res *resources, asset *postgres.Asset) gocron.Task {
	return gocron.NewTask(
		func() {
			timeout := time.Duration(asset.TimeoutSeconds)*time.Second + 5*time.Second

			ctx, cancel := context.WithTimeout(context.Background(), timeout)
			defer cancel()

			pr := probe.HttpProbe{}
			result := pr.Probe(ctx, asset)

			processed, err := res.repository.ProcessProbe(ctx, asset, result)
			if err != nil {
				res.logger.Error("error executing probe task", zap.Error(err))
				return
			}

			if processed.Incident != nil {
				res.logger.Warn("incident occurred: ", zap.Any("incident", processed.Incident))
			}
			if processed.Notification != nil {
				if err = sendNotification(processed.Notification); err != nil {
					res.logger.Error("error sending notification", zap.Error(err))
				}
			}
		})
}

func sendNotification(notification *postgres.Notification) error {
	botToken := os.Getenv("BOT_TOKEN")
	chatID := os.Getenv("CHAT_ID")

	message := fmt.Sprintf(
		"Asset: %s\nDetails: %s\nEvent: %s\nTimestamp: %s",
		notification.AssetName,
		notification.Event,
		notification.Details,
		notification.Timestamp.Format(time.DateTime),
	)

	target := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", botToken)

	data := url.Values{}
	data.Set("chat_id", chatID)
	data.Set("text", message)

	response, err := http.PostForm(target, data)
	if err != nil {
		return err
	}

	return response.Body.Close()
}

// ---------------------------------------------------------------------------------------------------------------------

func registerProbeCleanupJob(res *resources) error {
	if err := res.scheduler.ScheduleDailyAt(
		"cleanup probes",
		0, 0, 0,
		func() {
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()

			rows, err := res.repository.CleanupProbes(ctx, 7)
			if err != nil {
				res.logger.Error("error executing cleanup probes", zap.Error(err))
				return
			}

			res.logger.Info(fmt.Sprintf("cleaned up probes: %d", rows))
		},
	); err != nil {
		return fmt.Errorf("error registering cleanup probes job: %w", err)
	}

	return nil
}

// ---------------------------------------------------------------------------------------------------------------------
