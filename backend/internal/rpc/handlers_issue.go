package rpc

import (
	"context"
	"errors"

	"connectrpc.com/connect"
	"github.com/google/uuid"

	pmsv1 "github.com/pms/backend/gen/pms/v1"
	"github.com/pms/backend/gen/pms/v1/pmsv1connect"
	"github.com/pms/backend/internal/domain/issues"
)

type issueHandler struct {
	pmsv1connect.UnimplementedIssueServiceHandler
	svc *issues.Service
}

func (h *issueHandler) CreateIssue(ctx context.Context, req *connect.Request[pmsv1.CreateIssueRequest]) (*connect.Response[pmsv1.CommandResult], error) {
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
	assignee, err := optUUID(req.Msg.AssigneeId)
	if err != nil {
		return nil, err
	}
	parent, err := optUUID(req.Msg.ParentId)
	if err != nil {
		return nil, err
	}
	if _, err := h.svc.CreateIssue(ctx, actor, wsID, teamID, id, req.Msg.Title, req.Msg.StateId, int(req.Msg.Priority), assignee, parent); err != nil {
		return nil, mapErr(err)
	}
	return connect.NewResponse(&pmsv1.CommandResult{EntityId: id.String()}), nil
}

func (h *issueHandler) UpdateIssueState(ctx context.Context, req *connect.Request[pmsv1.UpdateIssueStateRequest]) (*connect.Response[pmsv1.CommandResult], error) {
	actor, wsID, id, err := issueRef(ctx, req.Msg.WorkspaceId, req.Msg.Id)
	if err != nil {
		return nil, err
	}
	if _, err := h.svc.UpdateIssueState(ctx, actor, wsID, id, req.Msg.StateId); err != nil {
		return nil, mapErr(err)
	}
	return connect.NewResponse(&pmsv1.CommandResult{EntityId: id.String()}), nil
}

func (h *issueHandler) AssignIssue(ctx context.Context, req *connect.Request[pmsv1.AssignIssueRequest]) (*connect.Response[pmsv1.CommandResult], error) {
	actor, wsID, id, err := issueRef(ctx, req.Msg.WorkspaceId, req.Msg.Id)
	if err != nil {
		return nil, err
	}
	assignee, err := optUUID(req.Msg.AssigneeId)
	if err != nil {
		return nil, err
	}
	if _, err := h.svc.AssignIssue(ctx, actor, wsID, id, assignee); err != nil {
		return nil, mapErr(err)
	}
	return connect.NewResponse(&pmsv1.CommandResult{EntityId: id.String()}), nil
}

func (h *issueHandler) SetPriority(ctx context.Context, req *connect.Request[pmsv1.SetPriorityRequest]) (*connect.Response[pmsv1.CommandResult], error) {
	actor, wsID, id, err := issueRef(ctx, req.Msg.WorkspaceId, req.Msg.Id)
	if err != nil {
		return nil, err
	}
	if _, err := h.svc.SetPriority(ctx, actor, wsID, id, int(req.Msg.Priority)); err != nil {
		return nil, mapErr(err)
	}
	return connect.NewResponse(&pmsv1.CommandResult{EntityId: id.String()}), nil
}

func (h *issueHandler) UpdateIssue(ctx context.Context, req *connect.Request[pmsv1.UpdateIssueRequest]) (*connect.Response[pmsv1.CommandResult], error) {
	actor, wsID, id, err := issueRef(ctx, req.Msg.WorkspaceId, req.Msg.Id)
	if err != nil {
		return nil, err
	}
	// Все присутствующие поля — одним атомарным UpdateFields (один tx/changelog/дельта).
	var p issues.IssuePatch
	if req.Msg.Title != nil {
		v := req.Msg.GetTitle()
		p.Title = &v
	}
	if req.Msg.Priority != nil {
		v := int(req.Msg.GetPriority())
		p.Priority = &v
	}
	if req.Msg.AssigneeId != nil {
		assignee, err := optUUID(req.Msg.GetAssigneeId())
		if err != nil {
			return nil, err
		}
		p.Assignee = assignee
		p.AssigneeSet = true
	}
	if req.Msg.ScheduledDate != nil {
		v := req.Msg.GetScheduledDate()
		p.ScheduledDate = &v
	}
	if req.Msg.DueDate != nil {
		v := req.Msg.GetDueDate()
		p.DueDate = &v
	}
	if req.Msg.Description != nil {
		v := req.Msg.GetDescription()
		p.Description = &v
	}
	if err := h.svc.UpdateFields(ctx, actor, wsID, id, p); err != nil {
		return nil, mapErr(err)
	}
	return connect.NewResponse(&pmsv1.CommandResult{EntityId: id.String()}), nil
}

func (h *issueHandler) AddRelation(ctx context.Context, req *connect.Request[pmsv1.AddRelationRequest]) (*connect.Response[pmsv1.CommandResult], error) {
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
	source, err := uuid.Parse(req.Msg.SourceId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("invalid source_id"))
	}
	target, err := uuid.Parse(req.Msg.TargetId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("invalid target_id"))
	}
	if _, err := h.svc.AddRelation(ctx, actor, wsID, id, source, target, req.Msg.Type); err != nil {
		return nil, mapErr(err)
	}
	return connect.NewResponse(&pmsv1.CommandResult{EntityId: id.String()}), nil
}

func (h *issueHandler) RemoveRelation(ctx context.Context, req *connect.Request[pmsv1.EntityRef]) (*connect.Response[pmsv1.CommandResult], error) {
	actor, wsID, id, err := issueRef(ctx, req.Msg.WorkspaceId, req.Msg.Id)
	if err != nil {
		return nil, err
	}
	if _, err := h.svc.RemoveRelation(ctx, actor, wsID, id); err != nil {
		return nil, mapErr(err)
	}
	return connect.NewResponse(&pmsv1.CommandResult{EntityId: id.String()}), nil
}

func (h *issueHandler) ReorderIssue(ctx context.Context, req *connect.Request[pmsv1.ReorderIssueRequest]) (*connect.Response[pmsv1.CommandResult], error) {
	actor, wsID, id, err := issueRef(ctx, req.Msg.WorkspaceId, req.Msg.Id)
	if err != nil {
		return nil, err
	}
	beforeID, err := optUUID(req.Msg.BeforeId)
	if err != nil {
		return nil, err
	}
	afterID, err := optUUID(req.Msg.AfterId)
	if err != nil {
		return nil, err
	}
	if _, err := h.svc.ReorderIssue(ctx, actor, wsID, id, req.Msg.StateId, beforeID, afterID); err != nil {
		return nil, mapErr(err)
	}
	return connect.NewResponse(&pmsv1.CommandResult{EntityId: id.String()}), nil
}

func (h *issueHandler) SetCycle(ctx context.Context, req *connect.Request[pmsv1.SetCycleRequest]) (*connect.Response[pmsv1.CommandResult], error) {
	actor, wsID, id, err := issueRef(ctx, req.Msg.WorkspaceId, req.Msg.Id)
	if err != nil {
		return nil, err
	}
	cycleID, err := optUUID(req.Msg.CycleId)
	if err != nil {
		return nil, err
	}
	if _, err := h.svc.SetCycle(ctx, actor, wsID, id, cycleID); err != nil {
		return nil, mapErr(err)
	}
	return connect.NewResponse(&pmsv1.CommandResult{EntityId: id.String()}), nil
}

func (h *issueHandler) SetCustomFieldValue(ctx context.Context, req *connect.Request[pmsv1.SetCustomFieldValueRequest]) (*connect.Response[pmsv1.CommandResult], error) {
	actor, wsID, issueID, err := issueRef(ctx, req.Msg.WorkspaceId, req.Msg.IssueId)
	if err != nil {
		return nil, err
	}
	fieldID, err := uuid.Parse(req.Msg.FieldId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("invalid field_id"))
	}
	if _, err := h.svc.SetCustomFieldValue(ctx, actor, wsID, issueID, fieldID, req.Msg.Value); err != nil {
		return nil, mapErr(err)
	}
	return connect.NewResponse(&pmsv1.CommandResult{EntityId: issueID.String()}), nil
}

func (h *issueHandler) ArchiveIssue(ctx context.Context, req *connect.Request[pmsv1.EntityRef]) (*connect.Response[pmsv1.CommandResult], error) {
	actor, wsID, id, err := issueRef(ctx, req.Msg.WorkspaceId, req.Msg.Id)
	if err != nil {
		return nil, err
	}
	if _, err := h.svc.ArchiveIssue(ctx, actor, wsID, id); err != nil {
		return nil, mapErr(err)
	}
	return connect.NewResponse(&pmsv1.CommandResult{EntityId: id.String()}), nil
}

func (h *issueHandler) SetEstimate(ctx context.Context, req *connect.Request[pmsv1.SetEstimateRequest]) (*connect.Response[pmsv1.CommandResult], error) {
	actor, wsID, id, err := issueRef(ctx, req.Msg.WorkspaceId, req.Msg.Id)
	if err != nil {
		return nil, err
	}
	if _, err := h.svc.SetEstimate(ctx, actor, wsID, id, req.Msg.Estimate); err != nil {
		return nil, mapErr(err)
	}
	return connect.NewResponse(&pmsv1.CommandResult{EntityId: id.String()}), nil
}

func (h *issueHandler) SetMilestone(ctx context.Context, req *connect.Request[pmsv1.SetMilestoneRequest]) (*connect.Response[pmsv1.CommandResult], error) {
	actor, wsID, id, err := issueRef(ctx, req.Msg.WorkspaceId, req.Msg.Id)
	if err != nil {
		return nil, err
	}
	milestoneID, err := optUUID(req.Msg.MilestoneId)
	if err != nil {
		return nil, err
	}
	if _, err := h.svc.SetMilestone(ctx, actor, wsID, id, milestoneID); err != nil {
		return nil, mapErr(err)
	}
	return connect.NewResponse(&pmsv1.CommandResult{EntityId: id.String()}), nil
}

func (h *issueHandler) GetActivity(ctx context.Context, req *connect.Request[pmsv1.GetActivityRequest]) (*connect.Response[pmsv1.GetActivityResponse], error) {
	actor, err := requireActor(ctx)
	if err != nil {
		return nil, err
	}
	wsID, err := uuid.Parse(req.Msg.WorkspaceId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("invalid workspace_id"))
	}
	entityID, err := uuid.Parse(req.Msg.EntityId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("invalid entity_id"))
	}
	raw, desc, err := h.svc.GetActivity(ctx, actor, wsID, entityID)
	if err != nil {
		return nil, mapErr(err)
	}
	return connect.NewResponse(&pmsv1.GetActivityResponse{Activity: raw, Description: desc}), nil
}

func (h *issueHandler) AddLabel(ctx context.Context, req *connect.Request[pmsv1.LabelRequest]) (*connect.Response[pmsv1.CommandResult], error) {
	actor, wsID, issueID, err := issueRef(ctx, req.Msg.WorkspaceId, req.Msg.IssueId)
	if err != nil {
		return nil, err
	}
	labelID, err := uuid.Parse(req.Msg.LabelId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("invalid label_id"))
	}
	if err := h.svc.AddLabel(ctx, actor, wsID, issueID, labelID); err != nil {
		return nil, mapErr(err)
	}
	return connect.NewResponse(&pmsv1.CommandResult{EntityId: issueID.String()}), nil
}

func (h *issueHandler) RemoveLabel(ctx context.Context, req *connect.Request[pmsv1.LabelRequest]) (*connect.Response[pmsv1.CommandResult], error) {
	actor, wsID, issueID, err := issueRef(ctx, req.Msg.WorkspaceId, req.Msg.IssueId)
	if err != nil {
		return nil, err
	}
	labelID, err := uuid.Parse(req.Msg.LabelId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("invalid label_id"))
	}
	if err := h.svc.RemoveLabel(ctx, actor, wsID, issueID, labelID); err != nil {
		return nil, mapErr(err)
	}
	return connect.NewResponse(&pmsv1.CommandResult{EntityId: issueID.String()}), nil
}
