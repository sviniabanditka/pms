package auth

import (
	"strings"
	"testing"
)

func TestHashPasswordFormat(t *testing.T) {
	h, err := HashPassword("s3cret")
	if err != nil {
		t.Fatalf("hash: %v", err)
	}
	if !strings.HasPrefix(h, "$argon2id$v=19$") {
		t.Errorf("unexpected PHC prefix: %q", h)
	}
	if n := len(strings.Split(h, "$")); n != 6 {
		t.Errorf("PHC must have 6 $-segments, got %d", n)
	}
}

func TestHashPasswordSaltsDiffer(t *testing.T) {
	a, _ := HashPassword("same")
	b, _ := HashPassword("same")
	if a == b {
		t.Error("same password must hash differently (random salt)")
	}
}

func TestVerifyPasswordRoundtrip(t *testing.T) {
	h, _ := HashPassword("correct horse")
	ok, err := VerifyPassword("correct horse", h)
	if err != nil || !ok {
		t.Fatalf("correct password must verify: ok=%v err=%v", ok, err)
	}
	bad, err := VerifyPassword("wrong", h)
	if err != nil {
		t.Fatalf("verify wrong: unexpected err %v", err)
	}
	if bad {
		t.Error("wrong password must not verify")
	}
}

func TestVerifyPasswordRejectsMalformed(t *testing.T) {
	for _, bad := range []string{
		"", "plain", "$argon2id$bad", "$bcrypt$v=19$m=1,t=1,p=1$x$y",
		"$argon2id$v=19$m=1,t=1,p=1$!notbase64!$y",
	} {
		if ok, err := VerifyPassword("x", bad); err == nil || ok {
			t.Errorf("malformed hash %q must error and not verify (ok=%v err=%v)", bad, ok, err)
		}
	}
}
