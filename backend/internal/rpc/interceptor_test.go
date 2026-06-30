package rpc

import (
	"testing"

	"github.com/google/uuid"

	pmsv1 "github.com/pms/backend/gen/pms/v1"
)

func TestWorkspaceIDOf(t *testing.T) {
	id := uuid.New()

	// message carrying a valid workspace_id
	if got, ok := workspaceIDOf(&pmsv1.CreateIssueRequest{WorkspaceId: id.String()}); !ok || got != id {
		t.Errorf("workspaceIDOf(CreateIssue) = %v,%v; want %v,true", got, ok, id)
	}
	// empty workspace_id → no scope
	if _, ok := workspaceIDOf(&pmsv1.CreateIssueRequest{}); ok {
		t.Error("empty workspace_id must yield ok=false")
	}
	// invalid uuid → no scope
	if _, ok := workspaceIDOf(&pmsv1.CreateIssueRequest{WorkspaceId: "not-a-uuid"}); ok {
		t.Error("invalid workspace_id must yield ok=false")
	}
	// message with NO workspace_id field (token-authorized) → pass through
	if _, ok := workspaceIDOf(&pmsv1.AcceptInvitationRequest{Token: "tok"}); ok {
		t.Error("AcceptInvitation has no workspace_id → ok=false")
	}
	// non-proto value
	if _, ok := workspaceIDOf("nope"); ok {
		t.Error("non-proto must yield ok=false")
	}
}

func TestIsReadProcedure(t *testing.T) {
	reads := []string{
		"/pms.v1.QueryService/Search",
		"/pms.v1.QueryService/RunView",
		"/pms.v1.ChatService/GetMessages",
		"/pms.v1.IssueService/GetActivity",
		"/pms.v1.TimeService/Report",
		"/pms.v1.SyncService/Bootstrap",
		"/pms.v1.SyncService/Changes",
	}
	for _, p := range reads {
		if !isReadProcedure(p) {
			t.Errorf("%s must be a read procedure (guest-allowed)", p)
		}
	}
	writes := []string{
		"/pms.v1.IssueService/CreateIssue",
		"/pms.v1.IssueService/UpdateIssueState",
		"/pms.v1.IssueService/ArchiveIssue",
		"/pms.v1.ChatService/SendMessage",
		"/pms.v1.ChatService/ToggleReaction",
		"/pms.v1.ChatService/MarkChannelRead",
		"/pms.v1.WorkspaceService/InviteMember",
		"/pms.v1.TeamService/AddTeamMember",
	}
	for _, p := range writes {
		if isReadProcedure(p) {
			t.Errorf("%s must NOT be a read procedure (guests denied)", p)
		}
	}
}

func TestAdminProcedures(t *testing.T) {
	admin := []string{
		"/pms.v1.AutomationService/CreateAutomation",
		"/pms.v1.AutomationService/UpdateAutomation",
		"/pms.v1.AutomationService/SetAutomationEnabled",
		"/pms.v1.AutomationService/DeleteAutomation",
		"/pms.v1.CustomFieldService/CreateCustomField",
		"/pms.v1.CustomFieldService/ArchiveCustomField",
		"/pms.v1.AttachmentService/SetWorkspaceLogo",
	}
	for _, p := range admin {
		if !adminProcedures[p] {
			t.Errorf("%s must require admin", p)
		}
	}
	// collaborative content must NOT be admin-gated
	for _, p := range []string{
		"/pms.v1.IssueService/CreateIssue",
		"/pms.v1.DocService/CreateDoc",
		"/pms.v1.CommentService/AddComment",
	} {
		if adminProcedures[p] {
			t.Errorf("%s must NOT be admin-gated (collaborative)", p)
		}
	}
}
