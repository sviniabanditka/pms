package rpc

import (
	"context"
	"errors"

	"connectrpc.com/connect"
	"github.com/google/uuid"

	pmsv1 "github.com/pms/backend/gen/pms/v1"
	"github.com/pms/backend/gen/pms/v1/pmsv1connect"
	"github.com/pms/backend/internal/domain/chats"
)

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
	if err := h.svc.CreateChannel(ctx, actor, wsID, id, req.Msg.Name, req.Msg.Private); err != nil {
		return nil, mapErr(err)
	}
	return connect.NewResponse(&pmsv1.CommandResult{EntityId: id.String()}), nil
}

func (h *chatHandler) AddChannelMember(ctx context.Context, req *connect.Request[pmsv1.ChannelMemberRequest]) (*connect.Response[pmsv1.CommandResult], error) {
	actor, wsID, chID, err := issueRef(ctx, req.Msg.WorkspaceId, req.Msg.ChannelId)
	if err != nil {
		return nil, err
	}
	uid, err := uuid.Parse(req.Msg.UserId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("invalid user_id"))
	}
	if err := h.svc.AddChannelMember(ctx, actor, wsID, chID, uid); err != nil {
		return nil, mapErr(err)
	}
	return connect.NewResponse(&pmsv1.CommandResult{EntityId: chID.String()}), nil
}

func (h *chatHandler) RemoveChannelMember(ctx context.Context, req *connect.Request[pmsv1.ChannelMemberRequest]) (*connect.Response[pmsv1.CommandResult], error) {
	actor, wsID, chID, err := issueRef(ctx, req.Msg.WorkspaceId, req.Msg.ChannelId)
	if err != nil {
		return nil, err
	}
	uid, err := uuid.Parse(req.Msg.UserId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("invalid user_id"))
	}
	if err := h.svc.RemoveChannelMember(ctx, actor, wsID, chID, uid); err != nil {
		return nil, mapErr(err)
	}
	return connect.NewResponse(&pmsv1.CommandResult{EntityId: chID.String()}), nil
}

func (h *chatHandler) ArchiveChannel(ctx context.Context, req *connect.Request[pmsv1.EntityRef]) (*connect.Response[pmsv1.CommandResult], error) {
	actor, wsID, chID, err := issueRef(ctx, req.Msg.WorkspaceId, req.Msg.Id)
	if err != nil {
		return nil, err
	}
	if err := h.svc.ArchiveChannel(ctx, actor, wsID, chID); err != nil {
		return nil, mapErr(err)
	}
	return connect.NewResponse(&pmsv1.CommandResult{EntityId: chID.String()}), nil
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
	parentID, err := optUUID(req.Msg.ParentId)
	if err != nil {
		return nil, err
	}
	if err := h.svc.SendMessage(ctx, actor, wsID, id, channelID, req.Msg.Body, parentID); err != nil {
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
	parentID, err := optUUID(req.Msg.ParentId)
	if err != nil {
		return nil, err
	}
	raw, hasMore, err := h.svc.GetMessages(ctx, actor, wsID, channelID, parentID, req.Msg.Before, int(req.Msg.Limit))
	if err != nil {
		return nil, mapErr(err)
	}
	return connect.NewResponse(&pmsv1.GetMessagesResponse{Messages: raw, HasMore: hasMore}), nil
}
