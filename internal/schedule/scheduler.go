package schedule

import (
	"context"
	"fmt"
	"log/slog"
	"time"
)

type Job struct {
	Name string
	Spec Spec
	Run  func(context.Context) error
}

type Runner struct {
	location *time.Location
	jobs     []Job
	logger   *slog.Logger
	now      func() time.Time
	lastRun  map[string]time.Time
}

type RunnerOptions struct {
	Timezone string
	Jobs     []Job
	Logger   *slog.Logger
	Now      func() time.Time
}

func NewRunner(options RunnerOptions) (*Runner, error) {
	location := time.Local
	if options.Timezone != "" {
		loaded, err := time.LoadLocation(options.Timezone)
		if err != nil {
			return nil, fmt.Errorf("load schedule timezone %q: %w", options.Timezone, err)
		}
		location = loaded
	}

	logger := options.Logger
	if logger == nil {
		logger = slog.Default()
	}
	now := options.Now
	if now == nil {
		now = time.Now
	}

	return &Runner{
		location: location,
		jobs:     options.Jobs,
		logger:   logger,
		now:      now,
		lastRun:  make(map[string]time.Time),
	}, nil
}

func (r *Runner) Run(ctx context.Context) {
	if len(r.jobs) == 0 {
		return
	}

	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case tick := <-ticker.C:
			r.evaluate(ctx, tick.In(r.location))
		}
	}
}

func (r *Runner) evaluate(ctx context.Context, now time.Time) {
	currentMinute := now.Truncate(time.Minute)

	for _, job := range r.jobs {
		if !job.Spec.Matches(currentMinute) {
			continue
		}
		if r.lastRun[job.Name].Equal(currentMinute) {
			continue
		}

		r.lastRun[job.Name] = currentMinute
		r.logger.Info("scheduled job started", slog.String("job", job.Name))

		if err := job.Run(ctx); err != nil {
			r.logger.Error(
				"scheduled job failed",
				slog.String("job", job.Name),
				slog.String("error", err.Error()),
			)
			continue
		}

		r.logger.Info("scheduled job completed", slog.String("job", job.Name))
	}
}
