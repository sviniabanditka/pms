package rpc

import (
	"context"
	"errors"

	"connectrpc.com/connect"
	"github.com/google/uuid"

	pmsv1 "github.com/pms/backend/gen/pms/v1"
	"github.com/pms/backend/gen/pms/v1/pmsv1connect"
	"github.com/pms/backend/internal/domain/templates"
)

type templateHandler struct {
	pmsv1connect.UnimplementedTemplateServiceHandler
	svc *templates.Service
}

func templateInput(in *pmsv1.TemplateInput) (templates.Input, error) {
	if in == nil {
		return templates.Input{}, errors.New("input required")
	}
	teamID, err := uuid.Parse(in.TeamId)
	if err != nil {
		return templates.Input{}, errors.New("invalid team_id")
	}
	assigneeID, err := optUUID(in.AssigneeId)
	if err != nil {
		return templates.Input{}, err
	}
	out := templates.Input{
		TeamID: teamID, Name: in.Name, Title: in.Title, DescriptionMD: in.DescriptionMd,
		Priority: int(in.Priority), AssigneeID: assigneeID, Recurrence: in.Recurrence,
	}
	if in.Estimate > 0 {
		e := in.Estimate
		out.Estimate = &e
	}
	for _, l := range in.LabelIds {
		if id, e := uuid.Parse(l); e == nil {
			out.LabelIDs = append(out.LabelIDs, id)
		}
	}
	return out, nil
}

func (h *templateHandler) CreateTemplate(ctx context.Context, req *connect.Request[pmsv1.CreateTemplateRequest]) (*connect.Response[pmsv1.CreateTemplateResponse], error) {
	actor, err := requireActor(ctx)
	if err != nil {
		return nil, err
	}
	wsID, err := uuid.Parse(req.Msg.WorkspaceId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("invalid workspace_id"))
	}
	in, err := templateInput(req.Msg.Input)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	id, err := h.svc.Create(ctx, actor, wsID, in)
	if err != nil {
		return nil, mapErr(err)
	}
	return connect.NewResponse(&pmsv1.CreateTemplateResponse{Id: id.String()}), nil
}

func (h *templateHandler) UpdateTemplate(ctx context.Context, req *connect.Request[pmsv1.UpdateTemplateRequest]) (*connect.Response[pmsv1.UpdateTemplateResponse], error) {
	actor, wsID, id, err := issueRef(ctx, req.Msg.WorkspaceId, req.Msg.Id)
	if err != nil {
		return nil, err
	}
	in, err := templateInput(req.Msg.Input)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	if err := h.svc.Update(ctx, actor, wsID, id, in); err != nil {
		return nil, mapErr(err)
	}
	return connect.NewResponse(&pmsv1.UpdateTemplateResponse{}), nil
}

func (h *templateHandler) DeleteTemplate(ctx context.Context, req *connect.Request[pmsv1.DeleteTemplateRequest]) (*connect.Response[pmsv1.DeleteTemplateResponse], error) {
	actor, wsID, id, err := issueRef(ctx, req.Msg.WorkspaceId, req.Msg.Id)
	if err != nil {
		return nil, err
	}
	if err := h.svc.Delete(ctx, actor, wsID, id); err != nil {
		return nil, mapErr(err)
	}
	return connect.NewResponse(&pmsv1.DeleteTemplateResponse{}), nil
}

func (h *templateHandler) SpawnTemplate(ctx context.Context, req *connect.Request[pmsv1.SpawnTemplateRequest]) (*connect.Response[pmsv1.SpawnTemplateResponse], error) {
	actor, wsID, id, err := issueRef(ctx, req.Msg.WorkspaceId, req.Msg.Id)
	if err != nil {
		return nil, err
	}
	issueID, err := h.svc.Spawn(ctx, actor, wsID, id)
	if err != nil {
		return nil, mapErr(err)
	}
	return connect.NewResponse(&pmsv1.SpawnTemplateResponse{IssueId: issueID.String()}), nil
}
