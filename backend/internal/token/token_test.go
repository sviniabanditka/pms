package token

import (
	"strings"
	"testing"
)

func TestGenerateAndHash(t *testing.T) {
	raw, hash := Generate("pms_")
	if !strings.HasPrefix(raw, "pms_") {
		t.Errorf("raw must keep prefix: %q", raw)
	}
	if len(raw) != len("pms_")+48 { // 24 random bytes → 48 hex
		t.Errorf("raw length = %d, want %d", len(raw), len("pms_")+48)
	}
	if hash != Hash(raw) {
		t.Error("returned hash must equal Hash(raw)")
	}
	if len(hash) != 64 { // sha256 hex
		t.Errorf("hash length = %d, want 64", len(hash))
	}
	if hash == raw {
		t.Error("hash must not equal the raw token")
	}
}

func TestGenerateUnique(t *testing.T) {
	a, _ := Generate("x_")
	b, _ := Generate("x_")
	if a == b {
		t.Error("two Generate calls must differ (random)")
	}
}

func TestHashDeterministic(t *testing.T) {
	if Hash("same") != Hash("same") {
		t.Error("Hash must be deterministic")
	}
	if Hash("a") == Hash("b") {
		t.Error("different inputs must hash differently")
	}
}
