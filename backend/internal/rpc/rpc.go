package rpc

import (
	"context"
	"errors"
	"net/http"
	"time"

	"connectrpc.com/connect"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	pmsv1 "github.com/pms/backend/gen/pms/v1"
	"github.com/pms/backend/gen/pms/v1/pmsv1connect"
	"github.com/pms/backend/internal/auth"
	"github.com/pms/backend/internal/domain/attachments"
	"github.com/pms/backend/internal/domain/chats"
	"github.com/pms/backend/internal/domain/comments"
	"github.com/pms/backend/internal/domain/customfields"
	"github.com/pms/backend/internal/domain/cycles"
	"github.com/pms/backend/internal/domain/dashboards"
	"github.com/pms/backend/internal/domain/docs"
	"github.com/pms/backend/internal/domain/issues"
	"github.com/pms/backend/internal/domain/meetings"
	"github.com/pms/backend/internal/domain/milestones"
	"github.com/pms/backend/internal/domain/notifications"
	"github.com/pms/backend/internal/domain/search"
	syncsvc "github.com/pms/backend/internal/domain/sync"
	"github.com/pms/backend/internal/domain/teams"
	"github.com/pms/backend/internal/domain/views"
	"github.com/pms/backend/internal/domain/workspaces"
)

// Services — доменные зависимости connect-хендлеров.
type Services struct {
	Workspaces *workspaces.Service
	Teams      *teams.Service
	Issues     *issues.Service
	Views        *views.Service
	Cycles       *cycles.Service
	CustomFields  *customfields.Service
	Dashboards    *dashboards.Service
	Notifications *notifications.Service
	Docs          *docs.Service
	Chats         *chats.Service
	Comments      *comments.Service
	Milestones    *milestones.Service
	Meetings      *meetings.Service
	Attachments   *attachments.Service
	Search        *search.Service
	Sync          *syncsvc.Service
}

// Mount монтирует connect-сервисы.
func Mount(r chi.Router, svcs Services) {
	mount := func(p string, h http.Handler) { r.Mount(p, h) }
	mount(pmsv1connect.NewWorkspaceServiceHandler(&workspaceHandler{svc: svcs.Workspaces}))
	mount(pmsv1connect.NewTeamServiceHandler(&teamHandler{svc: svcs.Teams}))
	mount(pmsv1connect.NewIssueServiceHandler(&issueHandler{svc: svcs.Issues}))
	mount(pmsv1connect.NewViewServiceHandler(&viewHandler{svc: svcs.Views}))
	mount(pmsv1connect.NewCycleServiceHandler(&cycleHandler{svc: svcs.Cycles}))
	mount(pmsv1connect.NewCustomFieldServiceHandler(&customFieldHandler{svc: svcs.CustomFields}))
	mount(pmsv1connect.NewDashboardServiceHandler(&dashboardHandler{svc: svcs.Dashboards}))
	mount(pmsv1connect.NewNotificationServiceHandler(&notificationHandler{svc: svcs.Notifications}))
	mount(pmsv1connect.NewDocServiceHandler(&docHandler{svc: svcs.Docs}))
	mount(pmsv1connect.NewChatServiceHandler(&chatHandler{svc: svcs.Chats}))
	mount(pmsv1connect.NewCommentServiceHandler(&commentHandler{svc: svcs.Comments}))
	mount(pmsv1connect.NewMilestoneServiceHandler(&milestoneHandler{svc: svcs.Milestones}))
	mount(pmsv1connect.NewMeetingServiceHandler(&meetingHandler{svc: svcs.Meetings}))
	mount(pmsv1connect.NewAttachmentServiceHandler(&attachmentHandler{svc: svcs.Attachments}))
	mount(pmsv1connect.NewSyncServiceHandler(&syncHandler{svc: svcs.Sync}))
	mount(pmsv1connect.NewQueryServiceHandler(&queryHandler{svc: svcs.Search}))
}

type queryHandler struct {
	pmsv1connect.UnimplementedQueryServiceHandler
	svc *search.Service
}

func (h *queryHandler) Search(ctx context.Context, req *connect.Request[pmsv1.SearchRequest]) (*connect.Response[pmsv1.SearchResponse], error) {
	actor, err := requireActor(ctx)
	if err != nil {
		return nil, err
	}
	wsID, err := uuid.Parse(req.Msg.WorkspaceId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("invalid workspace_id"))
	}
	raw, err := h.svc.Search(ctx, actor, wsID, req.Msg.Query)
	if err != nil {
		return nil, mapErr(err)
	}
	return connect.NewResponse(&pmsv1.SearchResponse{Results: raw}), nil
}

type syncHandler struct {
	pmsv1connect.UnimplementedSyncServiceHandler
	svc *syncsvc.Service
}

func (h *syncHandler) Bootstrap(ctx context.Context, req *connect.Request[pmsv1.BootstrapRequest], stream *connect.ServerStream[pmsv1.BootstrapChunk]) error {
	actor, err := requireActor(ctx)
	if err != nil {
		return err
	}
	wsID, err := uuid.Parse(req.Msg.WorkspaceId)
	if err != nil {
		return connect.NewError(connect.CodeInvalidArgument, errors.New("invalid workspace_id"))
	}
	chunks, cursor, err := h.svc.Bootstrap(ctx, actor, wsID)
	if err != nil {
		return mapErr(err)
	}
	for _, c := range chunks {
		if err := stream.Send(&pmsv1.BootstrapChunk{EntityType: c.EntityType, Entities: c.Entities, Cursor: cursor}); err != nil {
			return err
		}
	}
	return nil
}

func (h *syncHandler) Changes(ctx context.Context, req *connect.Request[pmsv1.ChangesRequest]) (*connect.Response[pmsv1.ChangesResponse], error) {
	actor, err := requireActor(ctx)
	if err != nil {
		return nil, err
	}
	wsID, err := uuid.Parse(req.Msg.WorkspaceId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("invalid workspace_id"))
	}
	rows, cursor, err := h.svc.Changes(ctx, actor, wsID, req.Msg.SinceCursor)
	if err != nil {
		return nil, mapErr(err)
	}
	deltas := make([]*pmsv1.Delta, 0, len(rows))
	for i := range rows {
		deltas = append(deltas, &pmsv1.Delta{
			Seq:           rows[i].Seq,
			EntityType:    rows[i].EntityType,
			EntityId:      rows[i].EntityID.String(),
			Version:       rows[i].Version,
			Op:            rows[i].Op,
			ChangedFields: rows[i].ChangedFields,
		})
	}
	return connect.NewResponse(&pmsv1.ChangesResponse{Deltas: deltas, Cursor: cursor}), nil
}

type workspaceHandler struct {
	pmsv1connect.UnimplementedWorkspaceServiceHandler
	svc *workspaces.Service
}

func (h *workspaceHandler) CreateWorkspace(ctx context.Context, req *connect.Request[pmsv1.CreateWorkspaceRequest]) (*connect.Response[pmsv1.CommandResult], error) {
	actor, err := requireActor(ctx)
	if err != nil {
		return nil, err
	}
	id, err := parseOrNew(req.Msg.Id)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	ver, err := h.svc.CreateWorkspace(ctx, actor, id, req.Msg.Slug, req.Msg.Name)
	if err != nil {
		return nil, mapErr(err)
	}
	return connect.NewResponse(&pmsv1.CommandResult{EntityId: id.String(), NewVersion: ver}), nil
}

func (h *workspaceHandler) UpdateWorkspaceSettings(ctx context.Context, req *connect.Request[pmsv1.UpdateWorkspaceSettingsRequest]) (*connect.Response[pmsv1.CommandResult], error) {
	actor, err := requireActor(ctx)
	if err != nil {
		return nil, err
	}
	wsID, err := uuid.Parse(req.Msg.WorkspaceId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("invalid workspace_id"))
	}
	var settings []byte
	if req.Msg.Settings != nil {
		settings = req.Msg.Settings
	}
	if err := h.svc.UpdateSettings(ctx, actor, wsID, req.Msg.Name, settings); err != nil {
		return nil, mapErr(err)
	}
	return connect.NewResponse(&pmsv1.CommandResult{EntityId: wsID.String()}), nil
}

func (h *workspaceHandler) RemoveMember(ctx context.Context, req *connect.Request[pmsv1.RemoveMemberRequest]) (*connect.Response[pmsv1.CommandResult], error) {
	actor, err := requireActor(ctx)
	if err != nil {
		return nil, err
	}
	wsID, err := uuid.Parse(req.Msg.WorkspaceId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("invalid workspace_id"))
	}
	userID, err := uuid.Parse(req.Msg.UserId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("invalid user_id"))
	}
	if err := h.svc.RemoveMember(ctx, actor, wsID, userID); err != nil {
		return nil, mapErr(err)
	}
	return connect.NewResponse(&pmsv1.CommandResult{EntityId: userID.String()}), nil
}

func (h *workspaceHandler) DeleteWorkspace(ctx context.Context, req *connect.Request[pmsv1.EntityRef]) (*connect.Response[pmsv1.CommandResult], error) {
	actor, wsID, _, err := issueRef(ctx, req.Msg.WorkspaceId, req.Msg.Id)
	if err != nil {
		return nil, err
	}
	if err := h.svc.DeleteWorkspace(ctx, actor, wsID); err != nil {
		return nil, mapErr(err)
	}
	return connect.NewResponse(&pmsv1.CommandResult{EntityId: wsID.String()}), nil
}

func (h *workspaceHandler) InviteMember(ctx context.Context, req *connect.Request[pmsv1.InviteMemberRequest]) (*connect.Response[pmsv1.CommandResult], error) {
	actor, err := requireActor(ctx)
	if err != nil {
		return nil, err
	}
	wsID, err := uuid.Parse(req.Msg.WorkspaceId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("invalid workspace_id"))
	}
	id, err := parseOrNew(req.Msg.Id)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	if err := h.svc.InviteMember(ctx, actor, wsID, id, req.Msg.Email, req.Msg.Role); err != nil {
		return nil, mapErr(err)
	}
	return connect.NewResponse(&pmsv1.CommandResult{EntityId: id.String()}), nil
}

func (h *workspaceHandler) UpdateMemberRole(ctx context.Context, req *connect.Request[pmsv1.UpdateMemberRoleRequest]) (*connect.Response[pmsv1.CommandResult], error) {
	actor, err := requireActor(ctx)
	if err != nil {
		return nil, err
	}
	wsID, err := uuid.Parse(req.Msg.WorkspaceId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("invalid workspace_id"))
	}
	userID, err := uuid.Parse(req.Msg.UserId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("invalid user_id"))
	}
	if err := h.svc.UpdateMemberRole(ctx, actor, wsID, userID, req.Msg.Role); err != nil {
		return nil, mapErr(err)
	}
	return connect.NewResponse(&pmsv1.CommandResult{EntityId: userID.String()}), nil
}

func (h *workspaceHandler) UpdateSectionsConfig(ctx context.Context, req *connect.Request[pmsv1.UpdateSectionsConfigRequest]) (*connect.Response[pmsv1.CommandResult], error) {
	actor, err := requireActor(ctx)
	if err != nil {
		return nil, err
	}
	wsID, err := uuid.Parse(req.Msg.WorkspaceId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("invalid workspace_id"))
	}
	if _, err := h.svc.UpdateSectionsConfig(ctx, actor, wsID, req.Msg.Config); err != nil {
		return nil, mapErr(err)
	}
	return connect.NewResponse(&pmsv1.CommandResult{EntityId: wsID.String()}), nil
}

type teamHandler struct {
	pmsv1connect.UnimplementedTeamServiceHandler
	svc *teams.Service
}

func (h *teamHandler) CreateTeam(ctx context.Context, req *connect.Request[pmsv1.CreateTeamRequest]) (*connect.Response[pmsv1.CommandResult], error) {
	actor, err := requireActor(ctx)
	if err != nil {
		return nil, err
	}
	wsID, err := uuid.Parse(req.Msg.WorkspaceId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("invalid workspace_id"))
	}
	id, err := parseOrNew(req.Msg.Id)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	var parent *uuid.UUID
	if req.Msg.ParentId != "" {
		p, err := uuid.Parse(req.Msg.ParentId)
		if err != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("invalid parent_id"))
		}
		parent = &p
	}
	ver, err := h.svc.CreateTeam(ctx, actor, wsID, id, parent, req.Msg.Key, req.Msg.Name, req.Msg.Private)
	if err != nil {
		return nil, mapErr(err)
	}
	return connect.NewResponse(&pmsv1.CommandResult{EntityId: id.String(), NewVersion: ver}), nil
}

func (h *teamHandler) UpdateTeam(ctx context.Context, req *connect.Request[pmsv1.UpdateTeamRequest]) (*connect.Response[pmsv1.CommandResult], error) {
	actor, wsID, id, err := issueRef(ctx, req.Msg.WorkspaceId, req.Msg.Id)
	if err != nil {
		return nil, err
	}
	if err := h.svc.UpdateTeam(ctx, actor, wsID, id, req.Msg.Name, req.Msg.Private); err != nil {
		return nil, mapErr(err)
	}
	return connect.NewResponse(&pmsv1.CommandResult{EntityId: id.String()}), nil
}

func (h *teamHandler) MoveTeam(ctx context.Context, req *connect.Request[pmsv1.MoveTeamRequest]) (*connect.Response[pmsv1.CommandResult], error) {
	actor, err := requireActor(ctx)
	if err != nil {
		return nil, err
	}
	wsID, err := uuid.Parse(req.Msg.WorkspaceId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("invalid workspace_id"))
	}
	id, err := uuid.Parse(req.Msg.Id)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("invalid id"))
	}
	var parent *uuid.UUID
	if req.Msg.NewParentId != "" {
		p, err := uuid.Parse(req.Msg.NewParentId)
		if err != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("invalid new_parent_id"))
		}
		parent = &p
	}
	ver, err := h.svc.MoveTeam(ctx, actor, wsID, id, parent)
	if err != nil {
		return nil, mapErr(err)
	}
	return connect.NewResponse(&pmsv1.CommandResult{EntityId: id.String(), NewVersion: ver}), nil
}

type issueHandler struct {
	pmsv1connect.UnimplementedIssueServiceHandler
	svc *issues.Service
}

func (h *issueHandler) CreateIssue(ctx context.Context, req *connect.Request[pmsv1.CreateIssueRequest]) (*connect.Response[pmsv1.CommandResult], error) {
	actor, err := requireActor(ctx)
	if err != nil {
		return nil, err
	}
	wsID, err := uuid.Parse(req.Msg.WorkspaceId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("invalid workspace_id"))
	}
	teamID, err := uuid.Parse(req.Msg.TeamId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("invalid team_id"))
	}
	id, err := parseOrNew(req.Msg.Id)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	assignee := optUUID(req.Msg.AssigneeId)
	parent := optUUID(req.Msg.ParentId)
	if _, err := h.svc.CreateIssue(ctx, actor, wsID, teamID, id, req.Msg.Title, req.Msg.StateId, int(req.Msg.Priority), assignee, parent); err != nil {
		return nil, mapErr(err)
	}
	return connect.NewResponse(&pmsv1.CommandResult{EntityId: id.String()}), nil
}

func (h *issueHandler) UpdateIssueState(ctx context.Context, req *connect.Request[pmsv1.UpdateIssueStateRequest]) (*connect.Response[pmsv1.CommandResult], error) {
	actor, wsID, id, err := issueRef(ctx, req.Msg.WorkspaceId, req.Msg.Id)
	if err != nil {
		return nil, err
	}
	if _, err := h.svc.UpdateIssueState(ctx, actor, wsID, id, req.Msg.StateId); err != nil {
		return nil, mapErr(err)
	}
	return connect.NewResponse(&pmsv1.CommandResult{EntityId: id.String()}), nil
}

func (h *issueHandler) AssignIssue(ctx context.Context, req *connect.Request[pmsv1.AssignIssueRequest]) (*connect.Response[pmsv1.CommandResult], error) {
	actor, wsID, id, err := issueRef(ctx, req.Msg.WorkspaceId, req.Msg.Id)
	if err != nil {
		return nil, err
	}
	if _, err := h.svc.AssignIssue(ctx, actor, wsID, id, optUUID(req.Msg.AssigneeId)); err != nil {
		return nil, mapErr(err)
	}
	return connect.NewResponse(&pmsv1.CommandResult{EntityId: id.String()}), nil
}

func (h *issueHandler) SetPriority(ctx context.Context, req *connect.Request[pmsv1.SetPriorityRequest]) (*connect.Response[pmsv1.CommandResult], error) {
	actor, wsID, id, err := issueRef(ctx, req.Msg.WorkspaceId, req.Msg.Id)
	if err != nil {
		return nil, err
	}
	if _, err := h.svc.SetPriority(ctx, actor, wsID, id, int(req.Msg.Priority)); err != nil {
		return nil, mapErr(err)
	}
	return connect.NewResponse(&pmsv1.CommandResult{EntityId: id.String()}), nil
}

func (h *issueHandler) UpdateIssue(ctx context.Context, req *connect.Request[pmsv1.UpdateIssueRequest]) (*connect.Response[pmsv1.CommandResult], error) {
	actor, wsID, id, err := issueRef(ctx, req.Msg.WorkspaceId, req.Msg.Id)
	if err != nil {
		return nil, err
	}
	// M1: применяем присутствующие поля title/priority/assignee (даты — позже).
	if req.Msg.Title != nil {
		if _, err := h.svc.SetTitle(ctx, actor, wsID, id, req.Msg.GetTitle()); err != nil {
			return nil, mapErr(err)
		}
	}
	if req.Msg.Priority != nil {
		if _, err := h.svc.SetPriority(ctx, actor, wsID, id, int(req.Msg.GetPriority())); err != nil {
			return nil, mapErr(err)
		}
	}
	if req.Msg.AssigneeId != nil {
		if _, err := h.svc.AssignIssue(ctx, actor, wsID, id, optUUID(req.Msg.GetAssigneeId())); err != nil {
			return nil, mapErr(err)
		}
	}
	if req.Msg.ScheduledDate != nil {
		if _, err := h.svc.SetScheduledDate(ctx, actor, wsID, id, req.Msg.GetScheduledDate()); err != nil {
			return nil, mapErr(err)
		}
	}
	if req.Msg.DueDate != nil {
		if _, err := h.svc.SetDueDate(ctx, actor, wsID, id, req.Msg.GetDueDate()); err != nil {
			return nil, mapErr(err)
		}
	}
	if req.Msg.Description != nil {
		if _, err := h.svc.SetDescription(ctx, actor, wsID, id, req.Msg.GetDescription()); err != nil {
			return nil, mapErr(err)
		}
	}
	return connect.NewResponse(&pmsv1.CommandResult{EntityId: id.String()}), nil
}

func (h *issueHandler) AddRelation(ctx context.Context, req *connect.Request[pmsv1.AddRelationRequest]) (*connect.Response[pmsv1.CommandResult], error) {
	actor, err := requireActor(ctx)
	if err != nil {
		return nil, err
	}
	wsID, err := uuid.Parse(req.Msg.WorkspaceId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("invalid workspace_id"))
	}
	id, err := parseOrNew(req.Msg.Id)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	source, err := uuid.Parse(req.Msg.SourceId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("invalid source_id"))
	}
	target, err := uuid.Parse(req.Msg.TargetId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("invalid target_id"))
	}
	if _, err := h.svc.AddRelation(ctx, actor, wsID, id, source, target, req.Msg.Type); err != nil {
		return nil, mapErr(err)
	}
	return connect.NewResponse(&pmsv1.CommandResult{EntityId: id.String()}), nil
}

func (h *issueHandler) RemoveRelation(ctx context.Context, req *connect.Request[pmsv1.EntityRef]) (*connect.Response[pmsv1.CommandResult], error) {
	actor, wsID, id, err := issueRef(ctx, req.Msg.WorkspaceId, req.Msg.Id)
	if err != nil {
		return nil, err
	}
	if _, err := h.svc.RemoveRelation(ctx, actor, wsID, id); err != nil {
		return nil, mapErr(err)
	}
	return connect.NewResponse(&pmsv1.CommandResult{EntityId: id.String()}), nil
}

func (h *issueHandler) ReorderIssue(ctx context.Context, req *connect.Request[pmsv1.ReorderIssueRequest]) (*connect.Response[pmsv1.CommandResult], error) {
	actor, wsID, id, err := issueRef(ctx, req.Msg.WorkspaceId, req.Msg.Id)
	if err != nil {
		return nil, err
	}
	if _, err := h.svc.ReorderIssue(ctx, actor, wsID, id, req.Msg.StateId, optUUID(req.Msg.BeforeId), optUUID(req.Msg.AfterId)); err != nil {
		return nil, mapErr(err)
	}
	return connect.NewResponse(&pmsv1.CommandResult{EntityId: id.String()}), nil
}

func (h *issueHandler) SetCycle(ctx context.Context, req *connect.Request[pmsv1.SetCycleRequest]) (*connect.Response[pmsv1.CommandResult], error) {
	actor, wsID, id, err := issueRef(ctx, req.Msg.WorkspaceId, req.Msg.Id)
	if err != nil {
		return nil, err
	}
	if _, err := h.svc.SetCycle(ctx, actor, wsID, id, optUUID(req.Msg.CycleId)); err != nil {
		return nil, mapErr(err)
	}
	return connect.NewResponse(&pmsv1.CommandResult{EntityId: id.String()}), nil
}

func (h *issueHandler) SetCustomFieldValue(ctx context.Context, req *connect.Request[pmsv1.SetCustomFieldValueRequest]) (*connect.Response[pmsv1.CommandResult], error) {
	actor, wsID, issueID, err := issueRef(ctx, req.Msg.WorkspaceId, req.Msg.IssueId)
	if err != nil {
		return nil, err
	}
	fieldID, err := uuid.Parse(req.Msg.FieldId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("invalid field_id"))
	}
	if _, err := h.svc.SetCustomFieldValue(ctx, actor, wsID, issueID, fieldID, req.Msg.Value); err != nil {
		return nil, mapErr(err)
	}
	return connect.NewResponse(&pmsv1.CommandResult{EntityId: issueID.String()}), nil
}

func (h *issueHandler) ArchiveIssue(ctx context.Context, req *connect.Request[pmsv1.EntityRef]) (*connect.Response[pmsv1.CommandResult], error) {
	actor, wsID, id, err := issueRef(ctx, req.Msg.WorkspaceId, req.Msg.Id)
	if err != nil {
		return nil, err
	}
	if _, err := h.svc.ArchiveIssue(ctx, actor, wsID, id); err != nil {
		return nil, mapErr(err)
	}
	return connect.NewResponse(&pmsv1.CommandResult{EntityId: id.String()}), nil
}

type viewHandler struct {
	pmsv1connect.UnimplementedViewServiceHandler
	svc *views.Service
}

func (h *viewHandler) CreateView(ctx context.Context, req *connect.Request[pmsv1.CreateViewRequest]) (*connect.Response[pmsv1.CommandResult], error) {
	actor, err := requireActor(ctx)
	if err != nil {
		return nil, err
	}
	wsID, err := uuid.Parse(req.Msg.WorkspaceId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("invalid workspace_id"))
	}
	id, err := parseOrNew(req.Msg.Id)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	_, err = h.svc.CreateView(ctx, actor, wsID, views.CreateInput{
		ID: id, Name: req.Msg.Name, DataSource: req.Msg.DataSource, ScopeKind: req.Msg.ScopeKind,
		TeamID: optUUID(req.Msg.TeamId), Filter: req.Msg.Filter, Grouping: req.Msg.Grouping, Layout: req.Msg.Layout,
	})
	if err != nil {
		return nil, mapErr(err)
	}
	return connect.NewResponse(&pmsv1.CommandResult{EntityId: id.String()}), nil
}

func (h *viewHandler) UpdateView(ctx context.Context, req *connect.Request[pmsv1.UpdateViewRequest]) (*connect.Response[pmsv1.CommandResult], error) {
	actor, wsID, id, err := issueRef(ctx, req.Msg.WorkspaceId, req.Msg.Id)
	if err != nil {
		return nil, err
	}
	p := views.UpdatePatch{Name: req.Msg.Name, Layout: req.Msg.Layout}
	if req.Msg.Filter != nil {
		p.Filter = req.Msg.Filter
	}
	if req.Msg.Grouping != nil {
		p.Grouping = req.Msg.Grouping
	}
	if _, err := h.svc.UpdateView(ctx, actor, wsID, id, p); err != nil {
		return nil, mapErr(err)
	}
	return connect.NewResponse(&pmsv1.CommandResult{EntityId: id.String()}), nil
}

func (h *viewHandler) DeleteView(ctx context.Context, req *connect.Request[pmsv1.EntityRef]) (*connect.Response[pmsv1.CommandResult], error) {
	actor, wsID, id, err := issueRef(ctx, req.Msg.WorkspaceId, req.Msg.Id)
	if err != nil {
		return nil, err
	}
	if _, err := h.svc.DeleteView(ctx, actor, wsID, id); err != nil {
		return nil, mapErr(err)
	}
	return connect.NewResponse(&pmsv1.CommandResult{EntityId: id.String()}), nil
}

type cycleHandler struct {
	pmsv1connect.UnimplementedCycleServiceHandler
	svc *cycles.Service
}

func (h *cycleHandler) CreateCycle(ctx context.Context, req *connect.Request[pmsv1.CreateCycleRequest]) (*connect.Response[pmsv1.CommandResult], error) {
	actor, err := requireActor(ctx)
	if err != nil {
		return nil, err
	}
	wsID, err := uuid.Parse(req.Msg.WorkspaceId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("invalid workspace_id"))
	}
	teamID, err := uuid.Parse(req.Msg.TeamId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("invalid team_id"))
	}
	id, err := parseOrNew(req.Msg.Id)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	starts, err1 := parseDate(req.Msg.StartsOn)
	ends, err2 := parseDate(req.Msg.EndsOn)
	if err1 != nil || err2 != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("invalid dates (YYYY-MM-DD)"))
	}
	if _, err := h.svc.CreateCycle(ctx, actor, wsID, teamID, id, req.Msg.Name, starts, ends); err != nil {
		return nil, mapErr(err)
	}
	return connect.NewResponse(&pmsv1.CommandResult{EntityId: id.String()}), nil
}

func (h *cycleHandler) UpdateCycle(ctx context.Context, req *connect.Request[pmsv1.UpdateCycleRequest]) (*connect.Response[pmsv1.CommandResult], error) {
	actor, wsID, id, err := issueRef(ctx, req.Msg.WorkspaceId, req.Msg.Id)
	if err != nil {
		return nil, err
	}
	var starts, ends *time.Time
	if req.Msg.StartsOn != nil {
		t, e := parseDate(req.Msg.GetStartsOn())
		if e != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("invalid starts_on"))
		}
		starts = &t
	}
	if req.Msg.EndsOn != nil {
		t, e := parseDate(req.Msg.GetEndsOn())
		if e != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("invalid ends_on"))
		}
		ends = &t
	}
	if _, err := h.svc.UpdateCycle(ctx, actor, wsID, id, req.Msg.Name, starts, ends); err != nil {
		return nil, mapErr(err)
	}
	return connect.NewResponse(&pmsv1.CommandResult{EntityId: id.String()}), nil
}

type customFieldHandler struct {
	pmsv1connect.UnimplementedCustomFieldServiceHandler
	svc *customfields.Service
}

func (h *customFieldHandler) CreateCustomField(ctx context.Context, req *connect.Request[pmsv1.CreateCustomFieldRequest]) (*connect.Response[pmsv1.CommandResult], error) {
	actor, err := requireActor(ctx)
	if err != nil {
		return nil, err
	}
	wsID, err := uuid.Parse(req.Msg.WorkspaceId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("invalid workspace_id"))
	}
	id, err := parseOrNew(req.Msg.Id)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	if _, err := h.svc.CreateCustomField(ctx, actor, wsID, id, optUUID(req.Msg.TeamId), req.Msg.Name, req.Msg.Type, req.Msg.Config); err != nil {
		return nil, mapErr(err)
	}
	return connect.NewResponse(&pmsv1.CommandResult{EntityId: id.String()}), nil
}

func (h *customFieldHandler) ArchiveCustomField(ctx context.Context, req *connect.Request[pmsv1.EntityRef]) (*connect.Response[pmsv1.CommandResult], error) {
	actor, wsID, id, err := issueRef(ctx, req.Msg.WorkspaceId, req.Msg.Id)
	if err != nil {
		return nil, err
	}
	if _, err := h.svc.ArchiveCustomField(ctx, actor, wsID, id); err != nil {
		return nil, mapErr(err)
	}
	return connect.NewResponse(&pmsv1.CommandResult{EntityId: id.String()}), nil
}

type dashboardHandler struct {
	pmsv1connect.UnimplementedDashboardServiceHandler
	svc *dashboards.Service
}

func (h *dashboardHandler) CreateDashboard(ctx context.Context, req *connect.Request[pmsv1.CreateDashboardRequest]) (*connect.Response[pmsv1.CommandResult], error) {
	actor, err := requireActor(ctx)
	if err != nil {
		return nil, err
	}
	wsID, err := uuid.Parse(req.Msg.WorkspaceId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("invalid workspace_id"))
	}
	id, err := parseOrNew(req.Msg.Id)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	if _, err := h.svc.CreateDashboard(ctx, actor, wsID, id, req.Msg.Name); err != nil {
		return nil, mapErr(err)
	}
	return connect.NewResponse(&pmsv1.CommandResult{EntityId: id.String()}), nil
}

func (h *dashboardHandler) AddWidget(ctx context.Context, req *connect.Request[pmsv1.AddWidgetRequest]) (*connect.Response[pmsv1.CommandResult], error) {
	actor, err := requireActor(ctx)
	if err != nil {
		return nil, err
	}
	wsID, err := uuid.Parse(req.Msg.WorkspaceId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("invalid workspace_id"))
	}
	id, err := parseOrNew(req.Msg.Id)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	dashID, err := uuid.Parse(req.Msg.DashboardId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("invalid dashboard_id"))
	}
	if _, err := h.svc.AddWidget(ctx, actor, wsID, id, dashID, req.Msg.Type, req.Msg.Title, req.Msg.Config); err != nil {
		return nil, mapErr(err)
	}
	return connect.NewResponse(&pmsv1.CommandResult{EntityId: id.String()}), nil
}

func (h *dashboardHandler) UpdateWidget(ctx context.Context, req *connect.Request[pmsv1.UpdateWidgetRequest]) (*connect.Response[pmsv1.CommandResult], error) {
	actor, wsID, id, err := issueRef(ctx, req.Msg.WorkspaceId, req.Msg.Id)
	if err != nil {
		return nil, err
	}
	var cfg []byte
	if req.Msg.Config != nil {
		cfg = req.Msg.Config
	}
	if _, err := h.svc.UpdateWidget(ctx, actor, wsID, id, req.Msg.Title, cfg, req.Msg.X, req.Msg.Y, req.Msg.W, req.Msg.H); err != nil {
		return nil, mapErr(err)
	}
	return connect.NewResponse(&pmsv1.CommandResult{EntityId: id.String()}), nil
}

func (h *dashboardHandler) RemoveWidget(ctx context.Context, req *connect.Request[pmsv1.EntityRef]) (*connect.Response[pmsv1.CommandResult], error) {
	actor, wsID, id, err := issueRef(ctx, req.Msg.WorkspaceId, req.Msg.Id)
	if err != nil {
		return nil, err
	}
	if _, err := h.svc.RemoveWidget(ctx, actor, wsID, id); err != nil {
		return nil, mapErr(err)
	}
	return connect.NewResponse(&pmsv1.CommandResult{EntityId: id.String()}), nil
}

type notificationHandler struct {
	pmsv1connect.UnimplementedNotificationServiceHandler
	svc *notifications.Service
}

func (h *notificationHandler) MarkRead(ctx context.Context, req *connect.Request[pmsv1.EntityRef]) (*connect.Response[pmsv1.CommandResult], error) {
	actor, wsID, id, err := issueRef(ctx, req.Msg.WorkspaceId, req.Msg.Id)
	if err != nil {
		return nil, err
	}
	if _, err := h.svc.MarkRead(ctx, actor, wsID, id); err != nil {
		return nil, mapErr(err)
	}
	return connect.NewResponse(&pmsv1.CommandResult{EntityId: id.String()}), nil
}

func (h *notificationHandler) MarkAllRead(ctx context.Context, req *connect.Request[pmsv1.MarkAllReadRequest]) (*connect.Response[pmsv1.CommandResult], error) {
	actor, err := requireActor(ctx)
	if err != nil {
		return nil, err
	}
	wsID, err := uuid.Parse(req.Msg.WorkspaceId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("invalid workspace_id"))
	}
	if _, err := h.svc.MarkAllRead(ctx, actor, wsID); err != nil {
		return nil, mapErr(err)
	}
	return connect.NewResponse(&pmsv1.CommandResult{EntityId: wsID.String()}), nil
}

type docHandler struct {
	pmsv1connect.UnimplementedDocServiceHandler
	svc *docs.Service
}

func (h *docHandler) CreateDoc(ctx context.Context, req *connect.Request[pmsv1.CreateDocRequest]) (*connect.Response[pmsv1.CommandResult], error) {
	actor, err := requireActor(ctx)
	if err != nil {
		return nil, err
	}
	wsID, err := uuid.Parse(req.Msg.WorkspaceId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("invalid workspace_id"))
	}
	id, err := parseOrNew(req.Msg.Id)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	if _, err := h.svc.CreateDoc(ctx, actor, wsID, id, optUUID(req.Msg.TeamId), req.Msg.Title, optUUID(req.Msg.ParentId)); err != nil {
		return nil, mapErr(err)
	}
	return connect.NewResponse(&pmsv1.CommandResult{EntityId: id.String()}), nil
}

func (h *docHandler) UpdateDoc(ctx context.Context, req *connect.Request[pmsv1.UpdateDocRequest]) (*connect.Response[pmsv1.CommandResult], error) {
	actor, wsID, id, err := issueRef(ctx, req.Msg.WorkspaceId, req.Msg.Id)
	if err != nil {
		return nil, err
	}
	var body []byte
	if req.Msg.Body != nil {
		body = req.Msg.Body
	}
	if _, err := h.svc.UpdateDoc(ctx, actor, wsID, id, req.Msg.Title, body); err != nil {
		return nil, mapErr(err)
	}
	return connect.NewResponse(&pmsv1.CommandResult{EntityId: id.String()}), nil
}

func (h *docHandler) DeleteDoc(ctx context.Context, req *connect.Request[pmsv1.EntityRef]) (*connect.Response[pmsv1.CommandResult], error) {
	actor, wsID, id, err := issueRef(ctx, req.Msg.WorkspaceId, req.Msg.Id)
	if err != nil {
		return nil, err
	}
	if _, err := h.svc.DeleteDoc(ctx, actor, wsID, id); err != nil {
		return nil, mapErr(err)
	}
	return connect.NewResponse(&pmsv1.CommandResult{EntityId: id.String()}), nil
}

func (h *teamHandler) CreateWorkflowState(ctx context.Context, req *connect.Request[pmsv1.CreateWorkflowStateRequest]) (*connect.Response[pmsv1.CommandResult], error) {
	actor, err := requireActor(ctx)
	if err != nil {
		return nil, err
	}
	wsID, err := uuid.Parse(req.Msg.WorkspaceId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("invalid workspace_id"))
	}
	id, err := parseOrNew(req.Msg.Id)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	teamID, err := uuid.Parse(req.Msg.TeamId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("invalid team_id"))
	}
	if err := h.svc.CreateWorkflowState(ctx, actor, wsID, id, teamID, req.Msg.Category, req.Msg.Name, req.Msg.ColorToken); err != nil {
		return nil, mapErr(err)
	}
	return connect.NewResponse(&pmsv1.CommandResult{EntityId: id.String()}), nil
}

func (h *teamHandler) UpdateWorkflowState(ctx context.Context, req *connect.Request[pmsv1.UpdateWorkflowStateRequest]) (*connect.Response[pmsv1.CommandResult], error) {
	actor, wsID, id, err := issueRef(ctx, req.Msg.WorkspaceId, req.Msg.Id)
	if err != nil {
		return nil, err
	}
	if err := h.svc.UpdateWorkflowState(ctx, actor, wsID, id, req.Msg.Name, req.Msg.ColorToken, req.Msg.Category); err != nil {
		return nil, mapErr(err)
	}
	return connect.NewResponse(&pmsv1.CommandResult{EntityId: id.String()}), nil
}

func (h *teamHandler) DeleteWorkflowState(ctx context.Context, req *connect.Request[pmsv1.EntityRef]) (*connect.Response[pmsv1.CommandResult], error) {
	actor, wsID, id, err := issueRef(ctx, req.Msg.WorkspaceId, req.Msg.Id)
	if err != nil {
		return nil, err
	}
	if err := h.svc.DeleteWorkflowState(ctx, actor, wsID, id); err != nil {
		return nil, mapErr(err)
	}
	return connect.NewResponse(&pmsv1.CommandResult{EntityId: id.String()}), nil
}

func (h *teamHandler) UpdateLabel(ctx context.Context, req *connect.Request[pmsv1.UpdateLabelRequest]) (*connect.Response[pmsv1.CommandResult], error) {
	actor, wsID, id, err := issueRef(ctx, req.Msg.WorkspaceId, req.Msg.Id)
	if err != nil {
		return nil, err
	}
	if err := h.svc.UpdateLabel(ctx, actor, wsID, id, req.Msg.Name, req.Msg.ColorToken); err != nil {
		return nil, mapErr(err)
	}
	return connect.NewResponse(&pmsv1.CommandResult{EntityId: id.String()}), nil
}

func (h *teamHandler) DeleteLabel(ctx context.Context, req *connect.Request[pmsv1.EntityRef]) (*connect.Response[pmsv1.CommandResult], error) {
	actor, wsID, id, err := issueRef(ctx, req.Msg.WorkspaceId, req.Msg.Id)
	if err != nil {
		return nil, err
	}
	if err := h.svc.DeleteLabel(ctx, actor, wsID, id); err != nil {
		return nil, mapErr(err)
	}
	return connect.NewResponse(&pmsv1.CommandResult{EntityId: id.String()}), nil
}

func (h *teamHandler) DeleteTeam(ctx context.Context, req *connect.Request[pmsv1.EntityRef]) (*connect.Response[pmsv1.CommandResult], error) {
	actor, wsID, id, err := issueRef(ctx, req.Msg.WorkspaceId, req.Msg.Id)
	if err != nil {
		return nil, err
	}
	if err := h.svc.DeleteTeam(ctx, actor, wsID, id); err != nil {
		return nil, mapErr(err)
	}
	return connect.NewResponse(&pmsv1.CommandResult{EntityId: id.String()}), nil
}

func (h *teamHandler) CreateLabel(ctx context.Context, req *connect.Request[pmsv1.CreateLabelRequest]) (*connect.Response[pmsv1.CommandResult], error) {
	actor, err := requireActor(ctx)
	if err != nil {
		return nil, err
	}
	wsID, err := uuid.Parse(req.Msg.WorkspaceId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("invalid workspace_id"))
	}
	id, err := parseOrNew(req.Msg.Id)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	if err := h.svc.CreateLabel(ctx, actor, wsID, id, optUUID(req.Msg.TeamId), req.Msg.Name, req.Msg.ColorToken); err != nil {
		return nil, mapErr(err)
	}
	return connect.NewResponse(&pmsv1.CommandResult{EntityId: id.String()}), nil
}

func (h *issueHandler) SetEstimate(ctx context.Context, req *connect.Request[pmsv1.SetEstimateRequest]) (*connect.Response[pmsv1.CommandResult], error) {
	actor, wsID, id, err := issueRef(ctx, req.Msg.WorkspaceId, req.Msg.Id)
	if err != nil {
		return nil, err
	}
	if _, err := h.svc.SetEstimate(ctx, actor, wsID, id, req.Msg.Estimate); err != nil {
		return nil, mapErr(err)
	}
	return connect.NewResponse(&pmsv1.CommandResult{EntityId: id.String()}), nil
}

func (h *issueHandler) SetMilestone(ctx context.Context, req *connect.Request[pmsv1.SetMilestoneRequest]) (*connect.Response[pmsv1.CommandResult], error) {
	actor, wsID, id, err := issueRef(ctx, req.Msg.WorkspaceId, req.Msg.Id)
	if err != nil {
		return nil, err
	}
	if _, err := h.svc.SetMilestone(ctx, actor, wsID, id, optUUID(req.Msg.MilestoneId)); err != nil {
		return nil, mapErr(err)
	}
	return connect.NewResponse(&pmsv1.CommandResult{EntityId: id.String()}), nil
}

func (h *issueHandler) GetActivity(ctx context.Context, req *connect.Request[pmsv1.GetActivityRequest]) (*connect.Response[pmsv1.GetActivityResponse], error) {
	actor, err := requireActor(ctx)
	if err != nil {
		return nil, err
	}
	wsID, err := uuid.Parse(req.Msg.WorkspaceId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("invalid workspace_id"))
	}
	entityID, err := uuid.Parse(req.Msg.EntityId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("invalid entity_id"))
	}
	raw, err := h.svc.GetActivity(ctx, actor, wsID, entityID)
	if err != nil {
		return nil, mapErr(err)
	}
	return connect.NewResponse(&pmsv1.GetActivityResponse{Activity: raw}), nil
}

func (h *issueHandler) AddLabel(ctx context.Context, req *connect.Request[pmsv1.LabelRequest]) (*connect.Response[pmsv1.CommandResult], error) {
	actor, wsID, issueID, err := issueRef(ctx, req.Msg.WorkspaceId, req.Msg.IssueId)
	if err != nil {
		return nil, err
	}
	labelID, err := uuid.Parse(req.Msg.LabelId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("invalid label_id"))
	}
	if err := h.svc.AddLabel(ctx, actor, wsID, issueID, labelID); err != nil {
		return nil, mapErr(err)
	}
	return connect.NewResponse(&pmsv1.CommandResult{EntityId: issueID.String()}), nil
}

func (h *issueHandler) RemoveLabel(ctx context.Context, req *connect.Request[pmsv1.LabelRequest]) (*connect.Response[pmsv1.CommandResult], error) {
	actor, wsID, issueID, err := issueRef(ctx, req.Msg.WorkspaceId, req.Msg.IssueId)
	if err != nil {
		return nil, err
	}
	labelID, err := uuid.Parse(req.Msg.LabelId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("invalid label_id"))
	}
	if err := h.svc.RemoveLabel(ctx, actor, wsID, issueID, labelID); err != nil {
		return nil, mapErr(err)
	}
	return connect.NewResponse(&pmsv1.CommandResult{EntityId: issueID.String()}), nil
}

type chatHandler struct {
	pmsv1connect.UnimplementedChatServiceHandler
	svc *chats.Service
}

func (h *chatHandler) CreateChannel(ctx context.Context, req *connect.Request[pmsv1.CreateChannelRequest]) (*connect.Response[pmsv1.CommandResult], error) {
	actor, err := requireActor(ctx)
	if err != nil {
		return nil, err
	}
	wsID, err := uuid.Parse(req.Msg.WorkspaceId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("invalid workspace_id"))
	}
	id, err := parseOrNew(req.Msg.Id)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	if err := h.svc.CreateChannel(ctx, actor, wsID, id, req.Msg.Name); err != nil {
		return nil, mapErr(err)
	}
	return connect.NewResponse(&pmsv1.CommandResult{EntityId: id.String()}), nil
}

func (h *chatHandler) CreateDM(ctx context.Context, req *connect.Request[pmsv1.CreateDMRequest]) (*connect.Response[pmsv1.CommandResult], error) {
	actor, err := requireActor(ctx)
	if err != nil {
		return nil, err
	}
	wsID, err := uuid.Parse(req.Msg.WorkspaceId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("invalid workspace_id"))
	}
	id, err := parseOrNew(req.Msg.Id)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	other, err := uuid.Parse(req.Msg.OtherUserId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("invalid other_user_id"))
	}
	cid, err := h.svc.CreateDM(ctx, actor, wsID, id, other)
	if err != nil {
		return nil, mapErr(err)
	}
	return connect.NewResponse(&pmsv1.CommandResult{EntityId: cid.String()}), nil
}

func (h *chatHandler) SendMessage(ctx context.Context, req *connect.Request[pmsv1.SendMessageRequest]) (*connect.Response[pmsv1.CommandResult], error) {
	actor, err := requireActor(ctx)
	if err != nil {
		return nil, err
	}
	wsID, err := uuid.Parse(req.Msg.WorkspaceId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("invalid workspace_id"))
	}
	id, err := parseOrNew(req.Msg.Id)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	channelID, err := uuid.Parse(req.Msg.ChannelId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("invalid channel_id"))
	}
	if err := h.svc.SendMessage(ctx, actor, wsID, id, channelID, req.Msg.Body, optUUID(req.Msg.ParentId)); err != nil {
		return nil, mapErr(err)
	}
	return connect.NewResponse(&pmsv1.CommandResult{EntityId: id.String()}), nil
}

func (h *chatHandler) EditMessage(ctx context.Context, req *connect.Request[pmsv1.EditMessageRequest]) (*connect.Response[pmsv1.CommandResult], error) {
	actor, wsID, id, err := issueRef(ctx, req.Msg.WorkspaceId, req.Msg.Id)
	if err != nil {
		return nil, err
	}
	if err := h.svc.EditMessage(ctx, actor, wsID, id, req.Msg.Body); err != nil {
		return nil, mapErr(err)
	}
	return connect.NewResponse(&pmsv1.CommandResult{EntityId: id.String()}), nil
}

func (h *chatHandler) DeleteMessage(ctx context.Context, req *connect.Request[pmsv1.EntityRef]) (*connect.Response[pmsv1.CommandResult], error) {
	actor, wsID, id, err := issueRef(ctx, req.Msg.WorkspaceId, req.Msg.Id)
	if err != nil {
		return nil, err
	}
	if err := h.svc.DeleteMessage(ctx, actor, wsID, id); err != nil {
		return nil, mapErr(err)
	}
	return connect.NewResponse(&pmsv1.CommandResult{EntityId: id.String()}), nil
}

func (h *chatHandler) ToggleReaction(ctx context.Context, req *connect.Request[pmsv1.ToggleReactionRequest]) (*connect.Response[pmsv1.CommandResult], error) {
	actor, wsID, mid, err := issueRef(ctx, req.Msg.WorkspaceId, req.Msg.MessageId)
	if err != nil {
		return nil, err
	}
	if err := h.svc.ToggleReaction(ctx, actor, wsID, mid, req.Msg.Emoji); err != nil {
		return nil, mapErr(err)
	}
	return connect.NewResponse(&pmsv1.CommandResult{EntityId: mid.String()}), nil
}

func (h *chatHandler) SendTyping(ctx context.Context, req *connect.Request[pmsv1.SendTypingRequest]) (*connect.Response[pmsv1.CommandResult], error) {
	actor, wsID, channelID, err := issueRef(ctx, req.Msg.WorkspaceId, req.Msg.ChannelId)
	if err != nil {
		return nil, err
	}
	if err := h.svc.SendTyping(ctx, actor, wsID, channelID); err != nil {
		return nil, mapErr(err)
	}
	return connect.NewResponse(&pmsv1.CommandResult{EntityId: channelID.String()}), nil
}

func (h *chatHandler) MarkChannelRead(ctx context.Context, req *connect.Request[pmsv1.MarkChannelReadRequest]) (*connect.Response[pmsv1.CommandResult], error) {
	actor, wsID, channelID, err := issueRef(ctx, req.Msg.WorkspaceId, req.Msg.ChannelId)
	if err != nil {
		return nil, err
	}
	if err := h.svc.MarkChannelRead(ctx, actor, wsID, channelID); err != nil {
		return nil, mapErr(err)
	}
	return connect.NewResponse(&pmsv1.CommandResult{EntityId: channelID.String()}), nil
}

func (h *chatHandler) GetMessages(ctx context.Context, req *connect.Request[pmsv1.GetMessagesRequest]) (*connect.Response[pmsv1.GetMessagesResponse], error) {
	actor, err := requireActor(ctx)
	if err != nil {
		return nil, err
	}
	wsID, err := uuid.Parse(req.Msg.WorkspaceId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("invalid workspace_id"))
	}
	channelID, err := uuid.Parse(req.Msg.ChannelId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("invalid channel_id"))
	}
	raw, hasMore, err := h.svc.GetMessages(ctx, actor, wsID, channelID, optUUID(req.Msg.ParentId), req.Msg.Before, int(req.Msg.Limit))
	if err != nil {
		return nil, mapErr(err)
	}
	return connect.NewResponse(&pmsv1.GetMessagesResponse{Messages: raw, HasMore: hasMore}), nil
}

type commentHandler struct {
	pmsv1connect.UnimplementedCommentServiceHandler
	svc *comments.Service
}

func (h *commentHandler) AddComment(ctx context.Context, req *connect.Request[pmsv1.AddCommentRequest]) (*connect.Response[pmsv1.CommandResult], error) {
	actor, err := requireActor(ctx)
	if err != nil {
		return nil, err
	}
	wsID, err := uuid.Parse(req.Msg.WorkspaceId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("invalid workspace_id"))
	}
	id, err := parseOrNew(req.Msg.Id)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	issueID, err := uuid.Parse(req.Msg.IssueId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("invalid issue_id"))
	}
	if err := h.svc.AddComment(ctx, actor, wsID, id, issueID, req.Msg.Body); err != nil {
		return nil, mapErr(err)
	}
	return connect.NewResponse(&pmsv1.CommandResult{EntityId: id.String()}), nil
}

func (h *commentHandler) EditComment(ctx context.Context, req *connect.Request[pmsv1.EditCommentRequest]) (*connect.Response[pmsv1.CommandResult], error) {
	actor, wsID, id, err := issueRef(ctx, req.Msg.WorkspaceId, req.Msg.Id)
	if err != nil {
		return nil, err
	}
	if err := h.svc.EditComment(ctx, actor, wsID, id, req.Msg.Body); err != nil {
		return nil, mapErr(err)
	}
	return connect.NewResponse(&pmsv1.CommandResult{EntityId: id.String()}), nil
}

func (h *commentHandler) DeleteComment(ctx context.Context, req *connect.Request[pmsv1.EntityRef]) (*connect.Response[pmsv1.CommandResult], error) {
	actor, wsID, id, err := issueRef(ctx, req.Msg.WorkspaceId, req.Msg.Id)
	if err != nil {
		return nil, err
	}
	if err := h.svc.DeleteComment(ctx, actor, wsID, id); err != nil {
		return nil, mapErr(err)
	}
	return connect.NewResponse(&pmsv1.CommandResult{EntityId: id.String()}), nil
}

func (h *commentHandler) GetComments(ctx context.Context, req *connect.Request[pmsv1.GetCommentsRequest]) (*connect.Response[pmsv1.GetCommentsResponse], error) {
	actor, err := requireActor(ctx)
	if err != nil {
		return nil, err
	}
	wsID, err := uuid.Parse(req.Msg.WorkspaceId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("invalid workspace_id"))
	}
	issueID, err := uuid.Parse(req.Msg.IssueId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("invalid issue_id"))
	}
	raw, err := h.svc.GetComments(ctx, actor, wsID, issueID)
	if err != nil {
		return nil, mapErr(err)
	}
	return connect.NewResponse(&pmsv1.GetCommentsResponse{Comments: raw}), nil
}

type milestoneHandler struct {
	pmsv1connect.UnimplementedMilestoneServiceHandler
	svc *milestones.Service
}

func (h *milestoneHandler) CreateMilestone(ctx context.Context, req *connect.Request[pmsv1.CreateMilestoneRequest]) (*connect.Response[pmsv1.CommandResult], error) {
	actor, err := requireActor(ctx)
	if err != nil {
		return nil, err
	}
	wsID, err := uuid.Parse(req.Msg.WorkspaceId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("invalid workspace_id"))
	}
	id, err := parseOrNew(req.Msg.Id)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	teamID, err := uuid.Parse(req.Msg.TeamId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("invalid team_id"))
	}
	if err := h.svc.Create(ctx, actor, wsID, id, teamID, req.Msg.Name, req.Msg.TargetDate); err != nil {
		return nil, mapErr(err)
	}
	return connect.NewResponse(&pmsv1.CommandResult{EntityId: id.String()}), nil
}

func (h *milestoneHandler) UpdateMilestone(ctx context.Context, req *connect.Request[pmsv1.UpdateMilestoneRequest]) (*connect.Response[pmsv1.CommandResult], error) {
	actor, wsID, id, err := issueRef(ctx, req.Msg.WorkspaceId, req.Msg.Id)
	if err != nil {
		return nil, err
	}
	if err := h.svc.Update(ctx, actor, wsID, id, req.Msg.Name, req.Msg.TargetDate, req.Msg.Status); err != nil {
		return nil, mapErr(err)
	}
	return connect.NewResponse(&pmsv1.CommandResult{EntityId: id.String()}), nil
}

func (h *milestoneHandler) DeleteMilestone(ctx context.Context, req *connect.Request[pmsv1.EntityRef]) (*connect.Response[pmsv1.CommandResult], error) {
	actor, wsID, id, err := issueRef(ctx, req.Msg.WorkspaceId, req.Msg.Id)
	if err != nil {
		return nil, err
	}
	if err := h.svc.Delete(ctx, actor, wsID, id); err != nil {
		return nil, mapErr(err)
	}
	return connect.NewResponse(&pmsv1.CommandResult{EntityId: id.String()}), nil
}

func parseDate(s string) (time.Time, error) { return time.Parse("2006-01-02", s) }

// --- helpers ---

// issueRef извлекает actor + парсит workspace/issue id.
func issueRef(ctx context.Context, wsRaw, idRaw string) (uuid.UUID, uuid.UUID, uuid.UUID, error) {
	actor, err := requireActor(ctx)
	if err != nil {
		return uuid.Nil, uuid.Nil, uuid.Nil, err
	}
	wsID, err := uuid.Parse(wsRaw)
	if err != nil {
		return uuid.Nil, uuid.Nil, uuid.Nil, connect.NewError(connect.CodeInvalidArgument, errors.New("invalid workspace_id"))
	}
	id, err := uuid.Parse(idRaw)
	if err != nil {
		return uuid.Nil, uuid.Nil, uuid.Nil, connect.NewError(connect.CodeInvalidArgument, errors.New("invalid id"))
	}
	return actor, wsID, id, nil
}

// optUUID: пустая строка → nil, иначе парсит (невалидное → nil, мягко).
func optUUID(s string) *uuid.UUID {
	if s == "" {
		return nil
	}
	u, err := uuid.Parse(s)
	if err != nil {
		return nil
	}
	return &u
}

func requireActor(ctx context.Context) (uuid.UUID, error) {
	uid, ok := auth.UserID(ctx)
	if !ok {
		return uuid.Nil, connect.NewError(connect.CodeUnauthenticated, errors.New("authentication required"))
	}
	return uid, nil
}

// parseOrNew принимает client-generated UUID; пустой → генерим UUIDv7 (удобно для ручных вызовов).
func parseOrNew(s string) (uuid.UUID, error) {
	if s == "" {
		return uuid.NewV7()
	}
	return uuid.Parse(s)
}

func mapErr(err error) error {
	switch {
	case errors.Is(err, workspaces.ErrInvalidSlug),
		errors.Is(err, workspaces.ErrInvalidName),
		errors.Is(err, teams.ErrInvalidKey),
		errors.Is(err, teams.ErrInvalidName),
		errors.Is(err, teams.ErrParentNotFound),
		errors.Is(err, teams.ErrCycle),
		errors.Is(err, teams.ErrTeamNotFound),
		errors.Is(err, issues.ErrTitleRequired),
		errors.Is(err, issues.ErrTeamNotFound),
		errors.Is(err, issues.ErrStateNotFound),
		errors.Is(err, issues.ErrIssueNotFound),
		errors.Is(err, issues.ErrInvalidRelation),
		errors.Is(err, views.ErrNameRequired),
		errors.Is(err, cycles.ErrDates),
		errors.Is(err, customfields.ErrName),
		errors.Is(err, customfields.ErrType),
		errors.Is(err, dashboards.ErrName),
		errors.Is(err, dashboards.ErrWidgetType),
		errors.Is(err, docs.ErrTitle),
		errors.Is(err, chats.ErrEmpty),
		errors.Is(err, comments.ErrEmpty),
		errors.Is(err, milestones.ErrName),
		errors.Is(err, workspaces.ErrEmail):
		return connect.NewError(connect.CodeInvalidArgument, err)
	case errors.Is(err, workspaces.ErrSlugTaken),
		errors.Is(err, teams.ErrKeyTaken):
		return connect.NewError(connect.CodeAlreadyExists, err)
	case errors.Is(err, syncsvc.ErrForbidden),
		errors.Is(err, chats.ErrForbidden),
		errors.Is(err, comments.ErrForbidden):
		return connect.NewError(connect.CodePermissionDenied, err)
	case errors.Is(err, teams.ErrStateInUse),
		errors.Is(err, teams.ErrTeamHasIssues):
		return connect.NewError(connect.CodeFailedPrecondition, err)
	default:
		return connect.NewError(connect.CodeInternal, err)
	}
}

type meetingHandler struct {
	pmsv1connect.UnimplementedMeetingServiceHandler
	svc *meetings.Service
}

func (h *meetingHandler) CreateMeeting(ctx context.Context, req *connect.Request[pmsv1.CreateMeetingRequest]) (*connect.Response[pmsv1.CommandResult], error) {
	actor, err := requireActor(ctx)
	if err != nil {
		return nil, err
	}
	wsID, err := uuid.Parse(req.Msg.WorkspaceId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("invalid workspace_id"))
	}
	id, err := parseOrNew(req.Msg.Id)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	var schedAt *time.Time
	if s := req.Msg.ScheduledAt; s != "" {
		if t, perr := time.Parse(time.RFC3339, s); perr == nil {
			schedAt = &t
		}
	}
	parts := make([]uuid.UUID, 0, len(req.Msg.ParticipantIds))
	for _, p := range req.Msg.ParticipantIds {
		if u, perr := uuid.Parse(p); perr == nil {
			parts = append(parts, u)
		}
	}
	channelID := optUUID(req.Msg.ChannelId)
	if err := h.svc.CreateMeeting(ctx, actor, wsID, id, req.Msg.Title, schedAt, parts, channelID); err != nil {
		return nil, mapErr(err)
	}
	return connect.NewResponse(&pmsv1.CommandResult{EntityId: id.String()}), nil
}

func (h *meetingHandler) StartHuddle(ctx context.Context, req *connect.Request[pmsv1.StartHuddleRequest]) (*connect.Response[pmsv1.CommandResult], error) {
	actor, err := requireActor(ctx)
	if err != nil {
		return nil, err
	}
	wsID, err := uuid.Parse(req.Msg.WorkspaceId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("invalid workspace_id"))
	}
	id, err := parseOrNew(req.Msg.Id)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	channelID, err := uuid.Parse(req.Msg.ChannelId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("invalid channel_id"))
	}
	mid, err := h.svc.StartHuddle(ctx, actor, wsID, id, channelID)
	if err != nil {
		return nil, mapErr(err)
	}
	return connect.NewResponse(&pmsv1.CommandResult{EntityId: mid.String()}), nil
}

func (h *meetingHandler) JoinMeeting(ctx context.Context, req *connect.Request[pmsv1.JoinMeetingRequest]) (*connect.Response[pmsv1.JoinMeetingResponse], error) {
	actor, wsID, mid, err := issueRef(ctx, req.Msg.WorkspaceId, req.Msg.MeetingId)
	if err != nil {
		return nil, err
	}
	peers, err := h.svc.JoinMeeting(ctx, actor, wsID, mid)
	if err != nil {
		return nil, mapErr(err)
	}
	return connect.NewResponse(&pmsv1.JoinMeetingResponse{PeerIds: peers}), nil
}

func (h *meetingHandler) LeaveMeeting(ctx context.Context, req *connect.Request[pmsv1.EntityRef]) (*connect.Response[pmsv1.CommandResult], error) {
	actor, wsID, mid, err := issueRef(ctx, req.Msg.WorkspaceId, req.Msg.Id)
	if err != nil {
		return nil, err
	}
	if err := h.svc.LeaveMeeting(ctx, actor, wsID, mid); err != nil {
		return nil, mapErr(err)
	}
	return connect.NewResponse(&pmsv1.CommandResult{EntityId: mid.String()}), nil
}

func (h *meetingHandler) EndMeeting(ctx context.Context, req *connect.Request[pmsv1.EntityRef]) (*connect.Response[pmsv1.CommandResult], error) {
	actor, wsID, mid, err := issueRef(ctx, req.Msg.WorkspaceId, req.Msg.Id)
	if err != nil {
		return nil, err
	}
	if err := h.svc.EndMeeting(ctx, actor, wsID, mid); err != nil {
		return nil, mapErr(err)
	}
	return connect.NewResponse(&pmsv1.CommandResult{EntityId: mid.String()}), nil
}

func (h *meetingHandler) SendSignal(ctx context.Context, req *connect.Request[pmsv1.SendSignalRequest]) (*connect.Response[pmsv1.CommandResult], error) {
	actor, wsID, mid, err := issueRef(ctx, req.Msg.WorkspaceId, req.Msg.MeetingId)
	if err != nil {
		return nil, err
	}
	toUser, err := uuid.Parse(req.Msg.ToUserId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("invalid to_user_id"))
	}
	if err := h.svc.SendSignal(ctx, actor, wsID, mid, toUser, req.Msg.Kind, req.Msg.Payload); err != nil {
		return nil, mapErr(err)
	}
	return connect.NewResponse(&pmsv1.CommandResult{EntityId: mid.String()}), nil
}

type attachmentHandler struct {
	pmsv1connect.UnimplementedAttachmentServiceHandler
	svc *attachments.Service
}

func (h *attachmentHandler) CreateAttachment(ctx context.Context, req *connect.Request[pmsv1.CreateAttachmentRequest]) (*connect.Response[pmsv1.CommandResult], error) {
	actor, err := requireActor(ctx)
	if err != nil {
		return nil, err
	}
	wsID, err := uuid.Parse(req.Msg.WorkspaceId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("invalid workspace_id"))
	}
	id, err := parseOrNew(req.Msg.Id)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	entityID, err := uuid.Parse(req.Msg.EntityId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("invalid entity_id"))
	}
	if err := h.svc.Create(ctx, actor, wsID, id, req.Msg.EntityType, entityID, req.Msg.Filename, req.Msg.ContentType, req.Msg.Size, req.Msg.S3Key); err != nil {
		return nil, mapErr(err)
	}
	return connect.NewResponse(&pmsv1.CommandResult{EntityId: id.String()}), nil
}

func (h *attachmentHandler) DeleteAttachment(ctx context.Context, req *connect.Request[pmsv1.EntityRef]) (*connect.Response[pmsv1.CommandResult], error) {
	actor, wsID, id, err := issueRef(ctx, req.Msg.WorkspaceId, req.Msg.Id)
	if err != nil {
		return nil, err
	}
	if err := h.svc.Delete(ctx, actor, wsID, id); err != nil {
		return nil, mapErr(err)
	}
	return connect.NewResponse(&pmsv1.CommandResult{EntityId: id.String()}), nil
}

func (h *attachmentHandler) GetAttachmentURL(ctx context.Context, req *connect.Request[pmsv1.GetAttachmentURLRequest]) (*connect.Response[pmsv1.URLResponse], error) {
	if _, err := requireActor(ctx); err != nil {
		return nil, err
	}
	url, err := h.svc.URL(ctx, req.Msg.S3Key)
	if err != nil {
		return nil, mapErr(err)
	}
	return connect.NewResponse(&pmsv1.URLResponse{Url: url}), nil
}

func (h *attachmentHandler) SetUserAvatar(ctx context.Context, req *connect.Request[pmsv1.SetKeyRequest]) (*connect.Response[pmsv1.CommandResult], error) {
	actor, err := requireActor(ctx)
	if err != nil {
		return nil, err
	}
	wsID, err := uuid.Parse(req.Msg.WorkspaceId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("invalid workspace_id"))
	}
	if err := h.svc.SetUserAvatar(ctx, actor, wsID, req.Msg.S3Key); err != nil {
		return nil, mapErr(err)
	}
	return connect.NewResponse(&pmsv1.CommandResult{EntityId: actor.String()}), nil
}

func (h *attachmentHandler) SetWorkspaceLogo(ctx context.Context, req *connect.Request[pmsv1.SetKeyRequest]) (*connect.Response[pmsv1.CommandResult], error) {
	actor, err := requireActor(ctx)
	if err != nil {
		return nil, err
	}
	wsID, err := uuid.Parse(req.Msg.WorkspaceId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("invalid workspace_id"))
	}
	if err := h.svc.SetWorkspaceLogo(ctx, actor, wsID, req.Msg.S3Key); err != nil {
		return nil, mapErr(err)
	}
	return connect.NewResponse(&pmsv1.CommandResult{EntityId: wsID.String()}), nil
}
