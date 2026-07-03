package rpc

import (
	"context"
	"errors"
	"time"

	"connectrpc.com/connect"
	"github.com/google/uuid"

	pmsv1 "github.com/pms/backend/gen/pms/v1"
	"github.com/pms/backend/gen/pms/v1/pmsv1connect"
	"github.com/pms/backend/internal/domain/cycles"
)

type cycleHandler struct {
	pmsv1connect.UnimplementedCycleServiceHandler
	svc *cycles.Service
}

func (h *cycleHandler) CreateCycle(ctx context.Context, req *connect.Request[pmsv1.CreateCycleRequest]) (*connect.Response[pmsv1.CommandResult], error) {
	actor, err := requireActor(ctx)
	if err != nil {
		return nil, err
	}
	wsID, err := uuid.Parse(req.Msg.WorkspaceId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("invalid workspace_id"))
	}
	teamID, err := uuid.Parse(req.Msg.TeamId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("invalid team_id"))
	}
	id, err := parseOrNew(req.Msg.Id)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	starts, err1 := parseDate(req.Msg.StartsOn)
	ends, err2 := parseDate(req.Msg.EndsOn)
	if err1 != nil || err2 != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("invalid dates (YYYY-MM-DD)"))
	}
	if _, err := h.svc.CreateCycle(ctx, actor, wsID, teamID, id, req.Msg.Name, starts, ends); err != nil {
		return nil, mapErr(err)
	}
	return connect.NewResponse(&pmsv1.CommandResult{EntityId: id.String()}), nil
}

func (h *cycleHandler) UpdateCycle(ctx context.Context, req *connect.Request[pmsv1.UpdateCycleRequest]) (*connect.Response[pmsv1.CommandResult], error) {
	actor, wsID, id, err := issueRef(ctx, req.Msg.WorkspaceId, req.Msg.Id)
	if err != nil {
		return nil, err
	}
	var starts, ends *time.Time
	if req.Msg.StartsOn != nil {
		t, e := parseDate(req.Msg.GetStartsOn())
		if e != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("invalid starts_on"))
		}
		starts = &t
	}
	if req.Msg.EndsOn != nil {
		t, e := parseDate(req.Msg.GetEndsOn())
		if e != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("invalid ends_on"))
		}
		ends = &t
	}
	if _, err := h.svc.UpdateCycle(ctx, actor, wsID, id, req.Msg.Name, starts, ends); err != nil {
		return nil, mapErr(err)
	}
	return connect.NewResponse(&pmsv1.CommandResult{EntityId: id.String()}), nil
}
