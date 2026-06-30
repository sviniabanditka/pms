package mentions

import "testing"

func TestParse(t *testing.T) {
	id1 := "11111111-1111-1111-1111-111111111111"
	id2 := "22222222-2222-2222-2222-222222222222"

	t.Run("empty", func(t *testing.T) {
		if got := Parse(""); len(got) != 0 {
			t.Errorf("empty body → %v, want none", got)
		}
	})

	t.Run("no mentions", func(t *testing.T) {
		if got := Parse("just text, email a@b.com, #ENG-1"); len(got) != 0 {
			t.Errorf("no @[..](id) → %v, want none", got)
		}
	})

	t.Run("single mention", func(t *testing.T) {
		got := Parse("hi @[Maya](" + id1 + ") there")
		if len(got) != 1 || got[0].String() != id1 {
			t.Errorf("got %v, want [%s]", got, id1)
		}
	})

	t.Run("multiple distinct, preserves order", func(t *testing.T) {
		got := Parse("@[A](" + id1 + ") and @[B](" + id2 + ")")
		if len(got) != 2 || got[0].String() != id1 || got[1].String() != id2 {
			t.Errorf("got %v, want [%s %s]", got, id1, id2)
		}
	})

	t.Run("dedups repeated mention", func(t *testing.T) {
		got := Parse("@[A](" + id1 + ") @[A again](" + id1 + ")")
		if len(got) != 1 {
			t.Errorf("repeated id must dedup, got %v", got)
		}
	})

	t.Run("ignores malformed id", func(t *testing.T) {
		// 36 chars but not a valid uuid → uuid.Parse fails → skipped
		got := Parse("@[X](zzzzzzzz-zzzz-zzzz-zzzz-zzzzzzzzzzzz)")
		if len(got) != 0 {
			t.Errorf("malformed uuid must be skipped, got %v", got)
		}
	})
}
