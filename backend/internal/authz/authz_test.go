package authz

import (
	"context"
	"testing"

	"github.com/google/uuid"
)

func TestRoleHierarchy(t *testing.T) {
	cases := []struct {
		r                       Role
		atLeastMember, canWrite bool
		isAdmin, valid          bool
	}{
		{RoleOwner, true, true, true, true},
		{RoleAdmin, true, true, true, true},
		{RoleMember, true, true, false, true},
		{RoleGuest, false, false, false, true},
		{Role("bogus"), false, false, false, false},
		{Role(""), false, false, false, false},
	}
	for _, c := range cases {
		if got := c.r.AtLeast(RoleMember); got != c.atLeastMember {
			t.Errorf("%q.AtLeast(member) = %v, want %v", c.r, got, c.atLeastMember)
		}
		if got := c.r.CanWrite(); got != c.canWrite {
			t.Errorf("%q.CanWrite() = %v, want %v", c.r, got, c.canWrite)
		}
		if got := c.r.IsAdmin(); got != c.isAdmin {
			t.Errorf("%q.IsAdmin() = %v, want %v", c.r, got, c.isAdmin)
		}
		if got := c.r.Valid(); got != c.valid {
			t.Errorf("%q.Valid() = %v, want %v", c.r, got, c.valid)
		}
	}

	// elevation-bound semantics: an actor cannot grant/touch a role above its own.
	if RoleAdmin.AtLeast(RoleOwner) {
		t.Error("admin must not be AtLeast(owner) — admin can't mint owners")
	}
	if !RoleOwner.AtLeast(RoleOwner) {
		t.Error("owner must be AtLeast(owner)")
	}
	if !RoleAdmin.AtLeast(RoleAdmin) {
		t.Error("admin must be AtLeast(admin) — can manage peers")
	}
}

func TestOwnerOrAdmin(t *testing.T) {
	owner, actor, other := uuid.New(), uuid.New(), uuid.New()

	// owner of the object always may, regardless of role in ctx
	if !OwnerOrAdmin(context.Background(), actor, actor) {
		t.Error("object owner must be allowed even with no role in ctx")
	}
	// nil owner must never auto-match a nil actor
	if OwnerOrAdmin(WithRole(context.Background(), RoleMember), uuid.Nil, uuid.Nil) {
		t.Error("nil owner must not match nil actor")
	}
	// non-owner: admin yes, member no
	adminCtx := WithRole(context.Background(), RoleAdmin)
	if !OwnerOrAdmin(adminCtx, owner, actor) {
		t.Error("admin must mutate others' objects")
	}
	memberCtx := WithRole(context.Background(), RoleMember)
	if OwnerOrAdmin(memberCtx, owner, actor) {
		t.Error("member must NOT mutate others' objects")
	}
	// guard against the wrong owner being honored
	if OwnerOrAdmin(memberCtx, owner, other) {
		t.Error("non-owner member must be denied")
	}
}

func TestRoleContext(t *testing.T) {
	if _, ok := RoleFromContext(context.Background()); ok {
		t.Error("empty ctx must have no role")
	}
	ctx := WithRole(context.Background(), RoleOwner)
	if r, ok := RoleFromContext(ctx); !ok || r != RoleOwner {
		t.Errorf("RoleFromContext = %q,%v; want owner,true", r, ok)
	}
}
