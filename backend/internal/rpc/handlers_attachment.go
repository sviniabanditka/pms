package rpc

import (
	"context"
	"errors"

	"connectrpc.com/connect"
	"github.com/google/uuid"

	pmsv1 "github.com/pms/backend/gen/pms/v1"
	"github.com/pms/backend/gen/pms/v1/pmsv1connect"
	"github.com/pms/backend/internal/domain/attachments"
)

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
	actor, err := requireActor(ctx)
	if err != nil {
		return nil, err
	}
	url, err := h.svc.URL(ctx, actor, req.Msg.S3Key)
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
