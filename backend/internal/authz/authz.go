// Package authz defines the workspace role model and helpers shared across
// RPC handlers. RLS isolates tenants; this layer is object/role authorization
// on top of that — see the workspaceAuthInterceptor for the membership gate.
package authz

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

// ErrForbidden — object/role authorization failure; mapErr maps it to PermissionDenied.
var ErrForbidden = errors.New("insufficient permissions")

// CanAccessTeam reports whether the actor may read/write content of a team:
// ws-admin always; a public team is open to any member; a private team requires
// membership of the team or an ancestor (downward ltree inheritance).
func CanAccessTeam(ctx context.Context, tx bun.Tx, actor, teamID uuid.UUID) (bool, error) {
	if r, _ := RoleFromContext(ctx); r.IsAdmin() {
		return true, nil
	}
	return tx.NewSelect().TableExpr("teams AS t").
		Where("t.id = ?", teamID).
		Where("(t.private = false OR EXISTS (SELECT 1 FROM team_members tm JOIN teams anc ON anc.id = tm.team_id WHERE tm.user_id = ? AND t.path <@ anc.path))", actor).
		Exists(ctx)
}

// CanManageTeam reports whether the actor may manage a team (config + membership):
// ws-admin, or a lead of the team or any ancestor (downward ltree inheritance).
// Privacy is irrelevant — management is lead/admin regardless.
func CanManageTeam(ctx context.Context, tx bun.Tx, actor, teamID uuid.UUID) (bool, error) {
	if r, _ := RoleFromContext(ctx); r.IsAdmin() {
		return true, nil
	}
	return tx.NewSelect().TableExpr("teams AS t").
		Where("t.id = ?", teamID).
		Where("EXISTS (SELECT 1 FROM team_members tm JOIN teams anc ON anc.id = tm.team_id WHERE tm.user_id = ? AND tm.role = 'lead' AND t.path <@ anc.path)", actor).
		Exists(ctx)
}

// OwnerOrAdmin reports whether the actor may mutate an object owned by `owner`:
// the owner always may; otherwise a workspace admin/owner may. Used for content
// whose edit is collaborative but whose delete/ownership is restricted.
func OwnerOrAdmin(ctx context.Context, owner, actor uuid.UUID) bool {
	if owner != uuid.Nil && owner == actor {
		return true
	}
	r, _ := RoleFromContext(ctx)
	return r.IsAdmin()
}

// Role is a workspace-level role. The canonical set matches the
// workspace_members.role / invitations.role CHECK constraint.
type Role string

const (
	RoleOwner  Role = "owner"
	RoleAdmin  Role = "admin"
	RoleMember Role = "member"
	RoleGuest  Role = "guest"
)

// rank orders roles low→high. Unknown roles rank below guest (0 default → -1 via lookup miss).
var rank = map[Role]int{RoleGuest: 0, RoleMember: 1, RoleAdmin: 2, RoleOwner: 3}

// AtLeast reports whether r sits at or above min in the hierarchy.
func (r Role) AtLeast(min Role) bool {
	rr, ok := rank[r]
	if !ok {
		return false
	}
	return rr >= rank[min]
}

// CanWrite — guests are read-only; member and up may mutate content.
func (r Role) CanWrite() bool { return r.AtLeast(RoleMember) }

// IsAdmin — owner or admin (workspace administration).
func (r Role) IsAdmin() bool { return r.AtLeast(RoleAdmin) }

// Valid reports whether r is a known role.
func (r Role) Valid() bool { _, ok := rank[r]; return ok }

type ctxKey int

const roleKey ctxKey = iota

// WithRole stashes the caller's workspace role in the context (set by the interceptor).
func WithRole(ctx context.Context, r Role) context.Context {
	return context.WithValue(ctx, roleKey, r)
}

// RoleFromContext returns the role set by the interceptor, if any.
func RoleFromContext(ctx context.Context) (Role, bool) {
	r, ok := ctx.Value(roleKey).(Role)
	return r, ok
}
