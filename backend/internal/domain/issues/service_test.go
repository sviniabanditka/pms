package issues

import (
	"testing"
	"time"

	"github.com/pms/backend/internal/frac"
)

// Позиции теперь фракционные строки (D20) — реордер через frac.Between (см. internal/frac тесты).
func TestFracReorder(t *testing.T) {
	// append в конец: всегда больше предыдущего
	a := frac.Between("", "")
	b := frac.Between(a, "")
	if !(a < b) {
		t.Fatalf("append: %q !< %q", a, b)
	}
	// вставка между соседями — строго между
	mid := frac.Between(a, b)
	if !(a < mid && mid < b) {
		t.Fatalf("between: %q < %q < %q violated", a, mid, b)
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
