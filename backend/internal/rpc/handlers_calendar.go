package rpc

import (
	"context"
	"errors"
	"time"

	"connectrpc.com/connect"
	"github.com/google/uuid"

	pmsv1 "github.com/pms/backend/gen/pms/v1"
	"github.com/pms/backend/gen/pms/v1/pmsv1connect"
	"github.com/pms/backend/internal/domain/calendar"
)

type calendarHandler struct {
	pmsv1connect.UnimplementedCalendarServiceHandler
	svc *calendar.Service
}

func (h *calendarHandler) CreateEvent(ctx context.Context, req *connect.Request[pmsv1.CreateEventRequest]) (*connect.Response[pmsv1.CommandResult], error) {
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
	st, en := parseTS(req.Msg.StartsAt), parseTS(req.Msg.EndsAt)
	if st.IsZero() || en.IsZero() {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("starts_at and ends_at required"))
	}
	var until *time.Time
	if u := parseTS(req.Msg.RecurrenceUntil); !u.IsZero() {
		until = &u
	}
	if err := h.svc.CreateEvent(ctx, actor, wsID, id, req.Msg.Title, req.Msg.Description, req.Msg.Kind, st, en, req.Msg.AllDay, req.Msg.Recurrence, until); err != nil {
		return nil, mapErr(err)
	}
	return connect.NewResponse(&pmsv1.CommandResult{EntityId: id.String()}), nil
}

func (h *calendarHandler) UpdateEvent(ctx context.Context, req *connect.Request[pmsv1.UpdateEventRequest]) (*connect.Response[pmsv1.CommandResult], error) {
	actor, wsID, id, err := issueRef(ctx, req.Msg.WorkspaceId, req.Msg.Id)
	if err != nil {
		return nil, err
	}
	st, en := parseTS(req.Msg.StartsAt), parseTS(req.Msg.EndsAt)
	if st.IsZero() || en.IsZero() {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("starts_at and ends_at required"))
	}
	var until *time.Time
	if u := parseTS(req.Msg.RecurrenceUntil); !u.IsZero() {
		until = &u
	}
	if err := h.svc.UpdateEvent(ctx, actor, wsID, id, req.Msg.Title, req.Msg.Description, req.Msg.Kind, st, en, req.Msg.AllDay, req.Msg.Recurrence, until); err != nil {
		return nil, mapErr(err)
	}
	return connect.NewResponse(&pmsv1.CommandResult{EntityId: id.String()}), nil
}

func (h *calendarHandler) DeleteEvent(ctx context.Context, req *connect.Request[pmsv1.EntityRef]) (*connect.Response[pmsv1.CommandResult], error) {
	actor, wsID, id, err := issueRef(ctx, req.Msg.WorkspaceId, req.Msg.Id)
	if err != nil {
		return nil, err
	}
	if err := h.svc.DeleteEvent(ctx, actor, wsID, id); err != nil {
		return nil, mapErr(err)
	}
	return connect.NewResponse(&pmsv1.CommandResult{EntityId: id.String()}), nil
}
