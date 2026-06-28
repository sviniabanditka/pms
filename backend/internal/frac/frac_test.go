package frac

import "testing"

func TestBetweenOrdering(t *testing.T) {
	// First < append chain strictly increases.
	prev := First()
	for i := 0; i < 200; i++ {
		next := Between(prev, "")
		if !(prev < next) {
			t.Fatalf("append not increasing: %q !< %q", prev, next)
		}
		prev = next
	}
	// Prepend chain strictly decreases.
	first := First()
	for i := 0; i < 200; i++ {
		p := Between("", first)
		if !(p < first) {
			t.Fatalf("prepend not decreasing: %q !< %q", p, first)
		}
		first = p
	}
}

func TestBetweenInsertSamePoint(t *testing.T) {
	// Repeatedly insert between the same two keys → no precision exhaustion (D20 core).
	lo, hi := "a", "b"
	for i := 0; i < 500; i++ {
		m := Between(lo, hi)
		if !(lo < m && m < hi) {
			t.Fatalf("iter %d: %q < %q < %q violated", i, lo, m, hi)
		}
		hi = m // keep squeezing toward lo
	}
}
