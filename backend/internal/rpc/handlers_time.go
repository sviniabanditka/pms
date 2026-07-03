package rpc

import (
	"context"
	"errors"

	"connectrpc.com/connect"
	"github.com/google/uuid"

	pmsv1 "github.com/pms/backend/gen/pms/v1"
	"github.com/pms/backend/gen/pms/v1/pmsv1connect"
	"github.com/pms/backend/internal/domain/timetracking"
)

type timeHandler struct {
	pmsv1connect.UnimplementedTimeServiceHandler
	svc *timetracking.Service
}

func (h *timeHandler) StartTimer(ctx context.Context, req *connect.Request[pmsv1.StartTimerRequest]) (*connect.Response[pmsv1.CommandResult], error) {
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
	issueID, err := optUUID(req.Msg.IssueId)
	if err != nil {
		return nil, err
	}
	if err := h.svc.StartTimer(ctx, actor, wsID, id, issueID, req.Msg.Description); err != nil {
		return nil, mapErr(err)
	}
	return connect.NewResponse(&pmsv1.CommandResult{EntityId: id.String()}), nil
}

func (h *timeHandler) StopTimer(ctx context.Context, req *connect.Request[pmsv1.EntityRef]) (*connect.Response[pmsv1.CommandResult], error) {
	actor, err := requireActor(ctx)
	if err != nil {
		return nil, err
	}
	wsID, err := uuid.Parse(req.Msg.WorkspaceId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("invalid workspace_id"))
	}
	if err := h.svc.StopTimer(ctx, actor, wsID); err != nil {
		return nil, mapErr(err)
	}
	return connect.NewResponse(&pmsv1.CommandResult{}), nil
}

func (h *timeHandler) AddEntry(ctx context.Context, req *connect.Request[pmsv1.AddEntryRequest]) (*connect.Response[pmsv1.CommandResult], error) {
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
	st, en := parseTS(req.Msg.StartedAt), parseTS(req.Msg.EndedAt)
	if st.IsZero() || en.IsZero() {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("started_at and ended_at required"))
	}
	issueID, err := optUUID(req.Msg.IssueId)
	if err != nil {
		return nil, err
	}
	if err := h.svc.AddEntry(ctx, actor, wsID, id, issueID, req.Msg.Description, st, en); err != nil {
		return nil, mapErr(err)
	}
	return connect.NewResponse(&pmsv1.CommandResult{EntityId: id.String()}), nil
}

func (h *timeHandler) Report(ctx context.Context, req *connect.Request[pmsv1.ReportRequest]) (*connect.Response[pmsv1.ReportResponse], error) {
	actor, err := requireActor(ctx)
	if err != nil {
		return nil, err
	}
	wsID, err := uuid.Parse(req.Msg.WorkspaceId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("invalid workspace_id"))
	}
	from, to := parseTS(req.Msg.From), parseTS(req.Msg.To)
	if from.IsZero() || to.IsZero() {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("from and to required"))
	}
	issueID, err := optUUID(req.Msg.IssueId)
	if err != nil {
		return nil, err
	}
	userID, err := optUUID(req.Msg.UserId)
	if err != nil {
		return nil, err
	}
	rows, total, err := h.svc.Report(ctx, actor, wsID, from, to, req.Msg.GroupBy, issueID, userID)
	if err != nil {
		return nil, mapErr(err)
	}
	out := make([]*pmsv1.ReportRow, len(rows))
	for i, r := range rows {
		out[i] = &pmsv1.ReportRow{Key: r.Key, Minutes: r.Minutes, Count: r.Count}
	}
	return connect.NewResponse(&pmsv1.ReportResponse{Rows: out, TotalMinutes: total}), nil
}

func (h *timeHandler) DeleteEntry(ctx context.Context, req *connect.Request[pmsv1.EntityRef]) (*connect.Response[pmsv1.CommandResult], error) {
	actor, wsID, id, err := issueRef(ctx, req.Msg.WorkspaceId, req.Msg.Id)
	if err != nil {
		return nil, err
	}
	if err := h.svc.DeleteEntry(ctx, actor, wsID, id); err != nil {
		return nil, mapErr(err)
	}
	return connect.NewResponse(&pmsv1.CommandResult{EntityId: id.String()}), nil
}
