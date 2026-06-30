// Package authz defines the workspace role model and helpers shared across
// RPC handlers. RLS isolates tenants; this layer is object/role authorization
// on top of that — see the workspaceAuthInterceptor for the membership gate.
package authz

import (
	"context"
	"errors"

	"github.com/google/uuid"
)

// ErrForbidden — object/role authorization failure; mapErr maps it to PermissionDenied.
var ErrForbidden = errors.New("insufficient permissions")

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
