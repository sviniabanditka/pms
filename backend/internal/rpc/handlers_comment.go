package rpc

import (
	"context"
	"errors"

	"connectrpc.com/connect"
	"github.com/google/uuid"

	pmsv1 "github.com/pms/backend/gen/pms/v1"
	"github.com/pms/backend/gen/pms/v1/pmsv1connect"
	"github.com/pms/backend/internal/domain/comments"
)

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

func (h *commentHandler) ToggleReaction(ctx context.Context, req *connect.Request[pmsv1.ToggleCommentReactionRequest]) (*connect.Response[pmsv1.CommandResult], error) {
	actor, wsID, cid, err := issueRef(ctx, req.Msg.WorkspaceId, req.Msg.CommentId)
	if err != nil {
		return nil, err
	}
	if err := h.svc.ToggleReaction(ctx, actor, wsID, cid, req.Msg.Emoji); err != nil {
		return nil, mapErr(err)
	}
	return connect.NewResponse(&pmsv1.CommandResult{EntityId: cid.String()}), nil
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
