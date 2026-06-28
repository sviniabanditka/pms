package issues

import (
	"testing"
	"time"
)

func TestMidpoint(t *testing.T) {
	f := func(v float64) *float64 { return &v }
	if got := midpoint(nil, nil); got != 1 {
		t.Fatalf("empty: got %v want 1", got)
	}
	if got := midpoint(nil, f(5)); got != 4 {
		t.Fatalf("top: got %v want 4", got)
	}
	if got := midpoint(f(5), nil); got != 6 {
		t.Fatalf("bottom: got %v want 6", got)
	}
	if got := midpoint(f(2), f(4)); got != 3 {
		t.Fatalf("between: got %v want 3", got)
	}
	// строго между
	mid := midpoint(f(1), f(2))
	if !(mid > 1 && mid < 2) {
		t.Fatalf("strictly between: got %v", mid)
	}
}

func TestStateTimestamps(t *testing.T) {
	now := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	earlier := now.Add(-time.Hour)

	cases := []struct {
		name              string
		category          string
		curStarted        *time.Time
		curCompleted      *time.Time
		wantStartedSet    bool
		wantCompletedSet  bool
		wantStartedIsPrev bool
	}{
		{"backlog clears", "backlog", &earlier, &earlier, false, false, false},
		{"unstarted clears", "unstarted", &earlier, &earlier, false, false, false},
		{"started sets started, clears completed", "started", nil, &earlier, true, false, false},
		{"started keeps existing started", "started", &earlier, nil, true, false, true},
		{"completed sets both", "completed", nil, nil, true, true, false},
		{"canceled sets completed, keeps started", "canceled", &earlier, nil, true, true, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			s, comp := stateTimestamps(c.category, now, c.curStarted, c.curCompleted)
			if (s != nil) != c.wantStartedSet {
				t.Fatalf("started set=%v want %v", s != nil, c.wantStartedSet)
			}
			if (comp != nil) != c.wantCompletedSet {
				t.Fatalf("completed set=%v want %v", comp != nil, c.wantCompletedSet)
			}
			if c.wantStartedIsPrev && s != nil && !s.Equal(earlier) {
				t.Fatalf("expected started preserved at %v, got %v", earlier, *s)
			}
		})
	}
}
