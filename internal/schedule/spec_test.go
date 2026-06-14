package schedule

import (
	"testing"
	"time"
)

func TestParseSpecAutoSyncDefault(t *testing.T) {
	t.Parallel()

	spec, err := ParseSpec("0 9 * * *")
	if err != nil {
		t.Fatalf("ParseSpec() error = %v", err)
	}

	matches := []time.Time{
		time.Date(2026, 6, 14, 9, 0, 0, 0, time.UTC),
		time.Date(2026, 6, 15, 9, 0, 0, 0, time.UTC),
	}
	for _, instant := range matches {
		if !spec.Matches(instant) {
			t.Fatalf("spec should match %s", instant)
		}
	}

	if spec.Matches(time.Date(2026, 6, 14, 10, 0, 0, 0, time.UTC)) {
		t.Fatal("spec should not match 10:00")
	}
}

func TestParseSpecAutoPushEveryTwoHoursNoonToNinePM(t *testing.T) {
	t.Parallel()

	spec, err := ParseSpec("0 12-21/2 * * *")
	if err != nil {
		t.Fatalf("ParseSpec() error = %v", err)
	}

	for _, hour := range []int{12, 14, 16, 18, 20} {
		instant := time.Date(2026, 6, 14, hour, 0, 0, 0, time.UTC)
		if !spec.Matches(instant) {
			t.Fatalf("spec should match hour %d", hour)
		}
	}

	for _, hour := range []int{11, 13, 21, 22} {
		instant := time.Date(2026, 6, 14, hour, 0, 0, 0, time.UTC)
		if spec.Matches(instant) {
			t.Fatalf("spec should not match hour %d", hour)
		}
	}
}
