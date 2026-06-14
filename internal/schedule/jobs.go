package schedule

import (
	"context"
	"fmt"

	"github.com/neuroborus/vocabulary-bot/internal/config"
)

type AutoSyncRunner interface {
	RunAutoSync(ctx context.Context) error
}

type AutoPushRunner interface {
	RunAutoPush(ctx context.Context) error
}

func JobsFromConfig(cfg config.ScheduleConfig, syncRunner AutoSyncRunner, pushRunner AutoPushRunner) ([]Job, error) {
	jobs := make([]Job, 0, 2)

	if cfg.AutoSyncCron != "" {
		spec, err := ParseSpec(cfg.AutoSyncCron)
		if err != nil {
			return nil, fmt.Errorf("auto sync cron: %w", err)
		}
		jobs = append(jobs, Job{
			Name: "auto_sync",
			Spec: spec,
			Run:  syncRunner.RunAutoSync,
		})
	}

	if cfg.AutoPushCron != "" {
		spec, err := ParseSpec(cfg.AutoPushCron)
		if err != nil {
			return nil, fmt.Errorf("auto push cron: %w", err)
		}
		jobs = append(jobs, Job{
			Name: "auto_push",
			Spec: spec,
			Run:  pushRunner.RunAutoPush,
		})
	}

	return jobs, nil
}
