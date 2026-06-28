package teams

import (
	"strings"
	"testing"

	"github.com/google/uuid"
)

func TestLtreeLabel(t *testing.T) {
	id := uuid.MustParse("019f09fa-5230-7655-8a6b-61d696b27f3f")
	got := ltreeLabel(id)
	if strings.Contains(got, "-") {
		t.Fatalf("ltree label must not contain '-': %q", got)
	}
	if got != "019f09fa_5230_7655_8a6b_61d696b27f3f" {
		t.Fatalf("unexpected label: %q", got)
	}
}

func TestDefaultStatesCoverAllCategories(t *testing.T) {
	want := map[string]bool{"backlog": true, "unstarted": true, "started": true, "completed": true, "canceled": true}
	if len(defaultStateDefs) != len(want) {
		t.Fatalf("expected %d default states, got %d", len(want), len(defaultStateDefs))
	}
	seen := map[string]bool{}
	for _, d := range defaultStateDefs {
		if !want[d.Category] {
			t.Fatalf("unexpected category %q", d.Category)
		}
		if seen[d.Category] {
			t.Fatalf("duplicate category %q", d.Category)
		}
		seen[d.Category] = true
		if d.Pos == "" || d.Color == "" || d.Name == "" {
			t.Fatalf("incomplete default state def: %+v", d)
		}
	}
}

func TestKeyRegex(t *testing.T) {
	ok := []string{"BLD", "A", "UI2", "ABCDEFGHIJ"}
	bad := []string{"", "bld", "1AB", "TOOLONGKEY1", "A-B", "A B"}
	for _, s := range ok {
		if !keyRe.MatchString(s) {
			t.Errorf("expected %q valid", s)
		}
	}
	for _, s := range bad {
		if keyRe.MatchString(s) {
			t.Errorf("expected %q invalid", s)
		}
	}
}
