package schedule

import (
	"context"
	"sync/atomic"
	"testing"
	"time"
)

func TestRunnerFiresJobOncePerMinute(t *testing.T) {
	t.Parallel()

	spec, err := ParseSpec("* * * * *")
	if err != nil {
		t.Fatalf("ParseSpec() error = %v", err)
	}

	var runs atomic.Int32
	now := time.Date(2026, 6, 14, 12, 0, 0, 0, time.UTC)
	runner, err := NewRunner(RunnerOptions{
		Jobs: []Job{{
			Name: "tick",
			Spec: spec,
			Run: func(context.Context) error {
				runs.Add(1)
				return nil
			},
		}},
		Now: func() time.Time { return now },
	})
	if err != nil {
		t.Fatalf("NewRunner() error = %v", err)
	}

	runner.evaluate(context.Background(), now)
	runner.evaluate(context.Background(), now)
	runner.evaluate(context.Background(), now.Add(time.Minute))

	if got := runs.Load(); got != 2 {
		t.Fatalf("runs = %d, want 2", got)
	}
}
