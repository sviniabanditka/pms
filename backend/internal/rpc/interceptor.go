package rpc

import (
	"context"
	"errors"
	"strings"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"

	"github.com/pms/backend/internal/auth"
	"github.com/pms/backend/internal/authz"
)

// adminProcedures require owner/admin — workspace-wide configuration that any
// member must not be able to change. Team-scoped config (teams, labels, workflow
// states, cycles, milestones) is gated by team role in the teams layer; per-object
// rules (delete = creator|admin) live in the services where the row is loaded.
var adminProcedures = map[string]bool{
	"/pms.v1.AutomationService/CreateAutomation":     true,
	"/pms.v1.AutomationService/UpdateAutomation":     true,
	"/pms.v1.AutomationService/SetAutomationEnabled": true,
	"/pms.v1.AutomationService/DeleteAutomation":     true,
	"/pms.v1.CustomFieldService/CreateCustomField":   true,
	"/pms.v1.CustomFieldService/ArchiveCustomField":  true,
	"/pms.v1.AttachmentService/SetWorkspaceLogo":     true,
}

// readVerbPrefixes mark a procedure as a read. Guests (read-only) may call only
// these; anything else is treated as a mutation and denied. New read RPCs must
// follow the convention (Get*/List*/Search*/…) to be guest-accessible.
var readVerbPrefixes = []string{"Get", "List", "Search", "Semantic", "Similar", "Run", "Report", "Feed", "Bootstrap", "Changes", "Export", "Stream", "Watch"}

func isReadProcedure(procedure string) bool {
	method := procedure[strings.LastIndex(procedure, "/")+1:]
	for _, p := range readVerbPrefixes {
		if strings.HasPrefix(method, p) {
			return true
		}
	}
	return false
}

// roleLookup returns the actor's active role in a workspace, or an error if not a member.
type roleLookup func(ctx context.Context, wsID, actor uuid.UUID) (string, error)

// newWorkspaceAuthInterceptor is the central membership gate. For every unary RPC
// whose request carries a non-empty workspace_id, it verifies the authenticated
// caller is an active member of that workspace and stashes their role in the
// context for downstream RBAC. RPCs without a workspace_id (CreateWorkspace,
// AcceptInvitation by token) pass through and authorize themselves.
//
// This closes the cross-tenant hole where WithWorkspace set app.workspace_id from
// a client-supplied id with no membership check — RLS keys on that id, not the user.
func newWorkspaceAuthInterceptor(lookup roleLookup) connect.UnaryInterceptorFunc {
	return func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			wsID, ok := workspaceIDOf(req.Any())
			if !ok {
				return next(ctx, req)
			}
			actor, ok := auth.UserID(ctx)
			if !ok {
				return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("authentication required"))
			}
			roleStr, err := lookup(ctx, wsID, actor)
			if err != nil {
				return nil, connect.NewError(connect.CodePermissionDenied, errors.New("not a workspace member"))
			}
			role := authz.Role(roleStr)
			proc := req.Spec().Procedure
			if auth.IsReadOnly(ctx) && !isReadProcedure(proc) {
				return nil, connect.NewError(connect.CodePermissionDenied, errors.New("read-only API token"))
			}
			if role == authz.RoleGuest && !isReadProcedure(proc) {
				return nil, connect.NewError(connect.CodePermissionDenied, errors.New("guests are read-only"))
			}
			if adminProcedures[proc] && !role.IsAdmin() {
				return nil, connect.NewError(connect.CodePermissionDenied, errors.New("requires workspace admin"))
			}
			return next(authz.WithRole(ctx, role), req)
		}
	}
}

// workspaceIDOf extracts a non-empty, parseable workspace_id string field from a
// request message via protoreflect. Returns false if the field is absent/empty/invalid.
func workspaceIDOf(msg any) (uuid.UUID, bool) {
	pm, ok := msg.(proto.Message)
	if !ok {
		return uuid.Nil, false
	}
	fd := pm.ProtoReflect().Descriptor().Fields().ByName("workspace_id")
	if fd == nil || fd.Kind() != protoreflect.StringKind {
		return uuid.Nil, false
	}
	s := pm.ProtoReflect().Get(fd).String()
	if s == "" {
		return uuid.Nil, false
	}
	id, err := uuid.Parse(s)
	if err != nil {
		return uuid.Nil, false
	}
	return id, true
}
