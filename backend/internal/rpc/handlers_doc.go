package rpc

import (
	"context"
	"errors"

	"connectrpc.com/connect"
	"github.com/google/uuid"

	pmsv1 "github.com/pms/backend/gen/pms/v1"
	"github.com/pms/backend/gen/pms/v1/pmsv1connect"
	"github.com/pms/backend/internal/domain/docs"
)

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
	teamID, err := optUUID(req.Msg.TeamId)
	if err != nil {
		return nil, err
	}
	parentID, err := optUUID(req.Msg.ParentId)
	if err != nil {
		return nil, err
	}
	if _, err := h.svc.CreateDoc(ctx, actor, wsID, id, teamID, req.Msg.Title, parentID); err != nil {
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
