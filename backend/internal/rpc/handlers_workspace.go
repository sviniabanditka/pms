package rpc

import (
	"context"
	"errors"

	"connectrpc.com/connect"
	"github.com/google/uuid"

	pmsv1 "github.com/pms/backend/gen/pms/v1"
	"github.com/pms/backend/gen/pms/v1/pmsv1connect"
	"github.com/pms/backend/internal/domain/workspaces"
)

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

func (h *workspaceHandler) RevokeInvitation(ctx context.Context, req *connect.Request[pmsv1.EntityRef]) (*connect.Response[pmsv1.CommandResult], error) {
	actor, wsID, id, err := issueRef(ctx, req.Msg.WorkspaceId, req.Msg.Id)
	if err != nil {
		return nil, err
	}
	if err := h.svc.RevokeInvitation(ctx, actor, wsID, id); err != nil {
		return nil, mapErr(err)
	}
	return connect.NewResponse(&pmsv1.CommandResult{EntityId: id.String()}), nil
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
	token, err := h.svc.InviteMember(ctx, actor, wsID, id, req.Msg.Email, req.Msg.Role)
	if err != nil {
		return nil, mapErr(err)
	}
	return connect.NewResponse(&pmsv1.CommandResult{EntityId: id.String(), Token: token}), nil
}

func (h *workspaceHandler) AcceptInvitation(ctx context.Context, req *connect.Request[pmsv1.AcceptInvitationRequest]) (*connect.Response[pmsv1.CommandResult], error) {
	actor, err := requireActor(ctx)
	if err != nil {
		return nil, err
	}
	wsID, err := h.svc.AcceptInvitation(ctx, actor, req.Msg.Token)
	if err != nil {
		return nil, mapErr(err)
	}
	return connect.NewResponse(&pmsv1.CommandResult{EntityId: wsID.String()}), nil
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
