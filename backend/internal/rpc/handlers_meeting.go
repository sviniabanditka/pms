package rpc

import (
	"context"
	"errors"
	"time"

	"connectrpc.com/connect"
	"github.com/google/uuid"

	pmsv1 "github.com/pms/backend/gen/pms/v1"
	"github.com/pms/backend/gen/pms/v1/pmsv1connect"
	"github.com/pms/backend/internal/domain/meetings"
)

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
	var schedAt, endsAt *time.Time
	if s := req.Msg.ScheduledAt; s != "" {
		if t, perr := time.Parse(time.RFC3339, s); perr == nil {
			schedAt = &t
		}
	}
	if s := req.Msg.EndsAt; s != "" {
		if t, perr := time.Parse(time.RFC3339, s); perr == nil {
			endsAt = &t
		}
	}
	parts := make([]uuid.UUID, 0, len(req.Msg.ParticipantIds))
	for _, p := range req.Msg.ParticipantIds {
		if u, perr := uuid.Parse(p); perr == nil {
			parts = append(parts, u)
		}
	}
	channelID, err := optUUID(req.Msg.ChannelId)
	if err != nil {
		return nil, err
	}
	if err := h.svc.CreateMeeting(ctx, actor, wsID, id, req.Msg.Title, schedAt, endsAt, parts, channelID); err != nil {
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
	url, token, err := h.svc.JoinMeeting(ctx, actor, wsID, mid)
	if err != nil {
		return nil, mapErr(err)
	}
	return connect.NewResponse(&pmsv1.JoinMeetingResponse{LivekitUrl: url, Token: token}), nil
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
