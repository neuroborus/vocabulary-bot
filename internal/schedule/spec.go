package schedule

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// Spec is a standard 5-field cron expression: minute hour day month weekday.
type Spec struct {
	minute  field
	hour    field
	day     field
	month   field
	weekday field
}

func ParseSpec(expression string) (Spec, error) {
	expression = strings.TrimSpace(expression)
	if expression == "" {
		return Spec{}, fmt.Errorf("cron expression is empty")
	}

	parts := strings.Fields(expression)
	if len(parts) != 5 {
		return Spec{}, fmt.Errorf("cron expression %q: want 5 fields, got %d", expression, len(parts))
	}

	minute, err := parseField(parts[0], 0, 59)
	if err != nil {
		return Spec{}, fmt.Errorf("cron minute: %w", err)
	}
	hour, err := parseField(parts[1], 0, 23)
	if err != nil {
		return Spec{}, fmt.Errorf("cron hour: %w", err)
	}
	day, err := parseField(parts[2], 1, 31)
	if err != nil {
		return Spec{}, fmt.Errorf("cron day: %w", err)
	}
	month, err := parseField(parts[3], 1, 12)
	if err != nil {
		return Spec{}, fmt.Errorf("cron month: %w", err)
	}
	weekday, err := parseField(parts[4], 0, 7)
	if err != nil {
		return Spec{}, fmt.Errorf("cron weekday: %w", err)
	}

	return Spec{
		minute:  minute,
		hour:    hour,
		day:     day,
		month:   month,
		weekday: weekday,
	}, nil
}

func (s Spec) Matches(t time.Time) bool {
	return s.minute.matches(t.Minute()) &&
		s.hour.matches(t.Hour()) &&
		s.day.matches(t.Day()) &&
		s.month.matches(int(t.Month())) &&
		s.weekday.matches(weekdayValue(t.Weekday()))
}

type field struct {
	any    bool
	values map[int]struct{}
}

func parseField(expr string, minValue, maxValue int) (field, error) {
	expr = strings.TrimSpace(expr)
	if expr == "*" {
		return field{any: true}, nil
	}

	result := field{values: make(map[int]struct{})}
	segments := strings.Split(expr, ",")
	for _, segment := range segments {
		segment = strings.TrimSpace(segment)
		if segment == "" {
			return field{}, fmt.Errorf("empty segment in %q", expr)
		}

		step := 1
		rangeExpr := segment
		if before, after, ok := strings.Cut(segment, "/"); ok {
			rangeExpr = strings.TrimSpace(before)
			parsedStep, err := strconv.Atoi(strings.TrimSpace(after))
			if err != nil || parsedStep <= 0 {
				return field{}, fmt.Errorf("invalid step in %q", segment)
			}
			step = parsedStep
		}

		switch {
		case rangeExpr == "*":
			for value := minValue; value <= maxValue; value += step {
				result.values[value] = struct{}{}
			}
		case strings.Contains(rangeExpr, "-"):
			startText, endText, ok := strings.Cut(rangeExpr, "-")
			if !ok {
				return field{}, fmt.Errorf("invalid range %q", rangeExpr)
			}
			start, err := parseBound(startText, minValue, maxValue)
			if err != nil {
				return field{}, err
			}
			end, err := parseBound(endText, minValue, maxValue)
			if err != nil {
				return field{}, err
			}
			if start > end {
				return field{}, fmt.Errorf("invalid range %q", rangeExpr)
			}
			for value := start; value <= end; value += step {
				result.values[value] = struct{}{}
			}
		default:
			value, err := parseBound(rangeExpr, minValue, maxValue)
			if err != nil {
				return field{}, err
			}
			result.values[value] = struct{}{}
		}
	}

	if len(result.values) == 0 {
		return field{}, fmt.Errorf("field %q matches nothing", expr)
	}

	return result, nil
}

func parseBound(value string, minValue, maxValue int) (int, error) {
	parsed, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil {
		return 0, fmt.Errorf("invalid number %q", value)
	}
	if parsed < minValue || parsed > maxValue {
		return 0, fmt.Errorf("value %d out of range %d-%d", parsed, minValue, maxValue)
	}

	return parsed, nil
}

func (f field) matches(value int) bool {
	if f.any {
		return true
	}
	_, ok := f.values[value]
	return ok
}

func weekdayValue(weekday time.Weekday) int {
	// Cron uses 0 or 7 for Sunday; Go uses Sunday = 0.
	return int(weekday)
}
