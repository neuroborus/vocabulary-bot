package schedule

import (
	"context"
	"testing"

	"github.com/neuroborus/vocabulary-bot/internal/config"
)

type stubScheduledRunner struct{}

func (stubScheduledRunner) RunAutoSync(context.Context) error { return nil }
func (stubScheduledRunner) RunAutoPush(context.Context) error { return nil }
func (stubScheduledRunner) RunAutoLogs(context.Context) error { return nil }

func TestJobsFromConfigIncludesAutoLogs(t *testing.T) {
	t.Parallel()

	jobs, err := JobsFromConfig(config.ScheduleConfig{
		AutoSyncCron: "0 9 * * *",
		AutoPushCron: "0 12 * * *",
		AutoLogsCron: "0 21 * * 5",
	}, stubScheduledRunner{})
	if err != nil {
		t.Fatalf("JobsFromConfig() error = %v", err)
	}

	names := make([]string, 0, len(jobs))
	for _, job := range jobs {
		names = append(names, job.Name)
	}

	want := []string{"auto_sync", "auto_push", "auto_logs"}
	if len(names) != len(want) {
		t.Fatalf("job count = %d, want %d (%v)", len(names), len(want), names)
	}
	for i, name := range want {
		if names[i] != name {
			t.Fatalf("jobs[%d] = %q, want %q", i, names[i], name)
		}
	}
}

func TestJobsFromConfigDisablesAutoLogsWhenCronEmpty(t *testing.T) {
	t.Parallel()

	jobs, err := JobsFromConfig(config.ScheduleConfig{
		AutoLogsCron: "",
	}, stubScheduledRunner{})
	if err != nil {
		t.Fatalf("JobsFromConfig() error = %v", err)
	}
	if len(jobs) != 0 {
		t.Fatalf("jobs = %v, want none", jobs)
	}
}
