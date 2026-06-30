package webhooks

import "testing"

func TestMatches(t *testing.T) {
	cases := []struct {
		events     []string
		entity, op string
		want       bool
	}{
		{[]string{"issue.create"}, "issue", "create", true},
		{[]string{"issue.create"}, "issue", "update", false},
		{[]string{"issue.*"}, "issue", "update", true},
		{[]string{"issue.*"}, "comment", "create", false},
		{[]string{"*"}, "anything", "delete", true},
		{[]string{"comment.create", "issue.update"}, "issue", "update", true},
		{[]string{}, "issue", "create", false},
		{[]string{"issue.create"}, "issue_label", "create", false}, // не префиксное совпадение
	}
	for _, c := range cases {
		if got := Matches(c.events, c.entity, c.op); got != c.want {
			t.Errorf("Matches(%v, %s, %s) = %v, want %v", c.events, c.entity, c.op, got, c.want)
		}
	}
}
