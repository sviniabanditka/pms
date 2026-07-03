package rpc

import (
	"context"
	"errors"

	"connectrpc.com/connect"
	"github.com/google/uuid"

	pmsv1 "github.com/pms/backend/gen/pms/v1"
	"github.com/pms/backend/gen/pms/v1/pmsv1connect"
	"github.com/pms/backend/internal/domain/automations"
)

type automationHandler struct {
	pmsv1connect.UnimplementedAutomationServiceHandler
	svc *automations.Service
}

func (h *automationHandler) CreateAutomation(ctx context.Context, req *connect.Request[pmsv1.CreateAutomationRequest]) (*connect.Response[pmsv1.CommandResult], error) {
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
	if err := h.svc.Create(ctx, actor, wsID, id, teamID, req.Msg.Name, req.Msg.Enabled,
		jsonRaw(req.Msg.Trigger), jsonRaw(req.Msg.Conditions), jsonRaw(req.Msg.Actions)); err != nil {
		return nil, mapErr(err)
	}
	return connect.NewResponse(&pmsv1.CommandResult{EntityId: id.String()}), nil
}

func (h *automationHandler) UpdateAutomation(ctx context.Context, req *connect.Request[pmsv1.UpdateAutomationRequest]) (*connect.Response[pmsv1.CommandResult], error) {
	actor, wsID, id, err := issueRef(ctx, req.Msg.WorkspaceId, req.Msg.Id)
	if err != nil {
		return nil, err
	}
	teamID, err := optUUID(req.Msg.TeamId)
	if err != nil {
		return nil, err
	}
	if err := h.svc.Update(ctx, actor, wsID, id, teamID, req.Msg.Name, req.Msg.Enabled,
		jsonRaw(req.Msg.Trigger), jsonRaw(req.Msg.Conditions), jsonRaw(req.Msg.Actions)); err != nil {
		return nil, mapErr(err)
	}
	return connect.NewResponse(&pmsv1.CommandResult{EntityId: id.String()}), nil
}

func (h *automationHandler) SetAutomationEnabled(ctx context.Context, req *connect.Request[pmsv1.SetAutomationEnabledRequest]) (*connect.Response[pmsv1.CommandResult], error) {
	actor, wsID, id, err := issueRef(ctx, req.Msg.WorkspaceId, req.Msg.Id)
	if err != nil {
		return nil, err
	}
	if err := h.svc.SetEnabled(ctx, actor, wsID, id, req.Msg.Enabled); err != nil {
		return nil, mapErr(err)
	}
	return connect.NewResponse(&pmsv1.CommandResult{EntityId: id.String()}), nil
}

func (h *automationHandler) DeleteAutomation(ctx context.Context, req *connect.Request[pmsv1.EntityRef]) (*connect.Response[pmsv1.CommandResult], error) {
	actor, wsID, id, err := issueRef(ctx, req.Msg.WorkspaceId, req.Msg.Id)
	if err != nil {
		return nil, err
	}
	if err := h.svc.Delete(ctx, actor, wsID, id); err != nil {
		return nil, mapErr(err)
	}
	return connect.NewResponse(&pmsv1.CommandResult{EntityId: id.String()}), nil
}
