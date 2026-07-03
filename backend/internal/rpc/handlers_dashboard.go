package rpc

import (
	"context"
	"errors"

	"connectrpc.com/connect"
	"github.com/google/uuid"

	pmsv1 "github.com/pms/backend/gen/pms/v1"
	"github.com/pms/backend/gen/pms/v1/pmsv1connect"
	"github.com/pms/backend/internal/domain/dashboards"
)

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
