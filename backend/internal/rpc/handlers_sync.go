package rpc

import (
	"context"
	"errors"

	"connectrpc.com/connect"
	"github.com/google/uuid"

	pmsv1 "github.com/pms/backend/gen/pms/v1"
	"github.com/pms/backend/gen/pms/v1/pmsv1connect"
	syncsvc "github.com/pms/backend/internal/domain/sync"
)

type syncHandler struct {
	pmsv1connect.UnimplementedSyncServiceHandler
	svc *syncsvc.Service
}

func (h *syncHandler) Bootstrap(ctx context.Context, req *connect.Request[pmsv1.BootstrapRequest], stream *connect.ServerStream[pmsv1.BootstrapChunk]) error {
	actor, err := requireActor(ctx)
	if err != nil {
		return err
	}
	wsID, err := uuid.Parse(req.Msg.WorkspaceId)
	if err != nil {
		return connect.NewError(connect.CodeInvalidArgument, errors.New("invalid workspace_id"))
	}
	chunks, cursor, err := h.svc.Bootstrap(ctx, actor, wsID, req.Msg.Types)
	if err != nil {
		return mapErr(err)
	}
	for _, c := range chunks {
		if err := stream.Send(&pmsv1.BootstrapChunk{EntityType: c.EntityType, Entities: c.Entities, Cursor: cursor}); err != nil {
			return err
		}
	}
	return nil
}

func (h *syncHandler) Changes(ctx context.Context, req *connect.Request[pmsv1.ChangesRequest]) (*connect.Response[pmsv1.ChangesResponse], error) {
	actor, err := requireActor(ctx)
	if err != nil {
		return nil, err
	}
	wsID, err := uuid.Parse(req.Msg.WorkspaceId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("invalid workspace_id"))
	}
	rows, cursor, err := h.svc.Changes(ctx, actor, wsID, req.Msg.SinceCursor)
	if err != nil {
		return nil, mapErr(err)
	}
	deltas := make([]*pmsv1.Delta, 0, len(rows))
	for i := range rows {
		deltas = append(deltas, &pmsv1.Delta{
			Seq:           rows[i].Seq,
			EntityType:    rows[i].EntityType,
			EntityId:      rows[i].EntityID.String(),
			Version:       rows[i].Version,
			Op:            rows[i].Op,
			ChangedFields: rows[i].ChangedFields,
		})
	}
	return connect.NewResponse(&pmsv1.ChangesResponse{Deltas: deltas, Cursor: cursor}), nil
}
