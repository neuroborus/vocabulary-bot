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

type AutoLogsRunner interface {
	RunAutoLogs(ctx context.Context) error
}

type ScheduledRunner interface {
	AutoSyncRunner
	AutoPushRunner
	AutoLogsRunner
}

func JobsFromConfig(cfg config.ScheduleConfig, runner ScheduledRunner) ([]Job, error) {
	jobs := make([]Job, 0, 3)

	if cfg.AutoSyncCron != "" {
		spec, err := ParseSpec(cfg.AutoSyncCron)
		if err != nil {
			return nil, fmt.Errorf("auto sync cron: %w", err)
		}
		jobs = append(jobs, Job{
			Name: "auto_sync",
			Spec: spec,
			Run:  runner.RunAutoSync,
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
			Run:  runner.RunAutoPush,
		})
	}

	if cfg.AutoLogsCron != "" {
		spec, err := ParseSpec(cfg.AutoLogsCron)
		if err != nil {
			return nil, fmt.Errorf("auto logs cron: %w", err)
		}
		jobs = append(jobs, Job{
			Name: "auto_logs",
			Spec: spec,
			Run:  runner.RunAutoLogs,
		})
	}

	return jobs, nil
}
