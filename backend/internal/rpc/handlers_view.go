package rpc

import (
	"context"
	"errors"

	"connectrpc.com/connect"
	"github.com/google/uuid"

	pmsv1 "github.com/pms/backend/gen/pms/v1"
	"github.com/pms/backend/gen/pms/v1/pmsv1connect"
	"github.com/pms/backend/internal/domain/views"
)

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
	teamID, err := optUUID(req.Msg.TeamId)
	if err != nil {
		return nil, err
	}
	_, err = h.svc.CreateView(ctx, actor, wsID, views.CreateInput{
		ID: id, Name: req.Msg.Name, DataSource: req.Msg.DataSource, ScopeKind: req.Msg.ScopeKind,
		TeamID: teamID, Filter: req.Msg.Filter, Grouping: req.Msg.Grouping, Layout: req.Msg.Layout,
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
