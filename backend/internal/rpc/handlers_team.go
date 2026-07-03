package rpc

import (
	"context"
	"errors"

	"connectrpc.com/connect"
	"github.com/google/uuid"

	pmsv1 "github.com/pms/backend/gen/pms/v1"
	"github.com/pms/backend/gen/pms/v1/pmsv1connect"
	"github.com/pms/backend/internal/domain/teams"
)

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

func (h *teamHandler) AddTeamMember(ctx context.Context, req *connect.Request[pmsv1.AddTeamMemberRequest]) (*connect.Response[pmsv1.CommandResult], error) {
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
	userID, err := uuid.Parse(req.Msg.UserId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("invalid user_id"))
	}
	if err := h.svc.AddMember(ctx, actor, wsID, id, teamID, userID, req.Msg.Role); err != nil {
		return nil, mapErr(err)
	}
	return connect.NewResponse(&pmsv1.CommandResult{EntityId: id.String()}), nil
}

func (h *teamHandler) RemoveTeamMember(ctx context.Context, req *connect.Request[pmsv1.RemoveTeamMemberRequest]) (*connect.Response[pmsv1.CommandResult], error) {
	actor, wsID, teamID, err := issueRef(ctx, req.Msg.WorkspaceId, req.Msg.TeamId)
	if err != nil {
		return nil, err
	}
	userID, err := uuid.Parse(req.Msg.UserId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("invalid user_id"))
	}
	if err := h.svc.RemoveMember(ctx, actor, wsID, teamID, userID); err != nil {
		return nil, mapErr(err)
	}
	return connect.NewResponse(&pmsv1.CommandResult{EntityId: userID.String()}), nil
}

func (h *teamHandler) LeaveTeam(ctx context.Context, req *connect.Request[pmsv1.LeaveTeamRequest]) (*connect.Response[pmsv1.CommandResult], error) {
	actor, wsID, teamID, err := issueRef(ctx, req.Msg.WorkspaceId, req.Msg.TeamId)
	if err != nil {
		return nil, err
	}
	if err := h.svc.LeaveTeam(ctx, actor, wsID, teamID); err != nil {
		return nil, mapErr(err)
	}
	return connect.NewResponse(&pmsv1.CommandResult{EntityId: teamID.String()}), nil
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
	teamID, err := optUUID(req.Msg.TeamId)
	if err != nil {
		return nil, err
	}
	if err := h.svc.CreateLabel(ctx, actor, wsID, id, teamID, req.Msg.Name, req.Msg.ColorToken); err != nil {
		return nil, mapErr(err)
	}
	return connect.NewResponse(&pmsv1.CommandResult{EntityId: id.String()}), nil
}
