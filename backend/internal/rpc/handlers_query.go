package rpc

import (
	"context"
	"encoding/json"
	"errors"

	"connectrpc.com/connect"
	"github.com/google/uuid"

	pmsv1 "github.com/pms/backend/gen/pms/v1"
	"github.com/pms/backend/gen/pms/v1/pmsv1connect"
	"github.com/pms/backend/internal/domain/search"
	"github.com/pms/backend/internal/filter"
)

type queryHandler struct {
	pmsv1connect.UnimplementedQueryServiceHandler
	svc *search.Service
}

func (h *queryHandler) Search(ctx context.Context, req *connect.Request[pmsv1.SearchRequest]) (*connect.Response[pmsv1.SearchResponse], error) {
	actor, err := requireActor(ctx)
	if err != nil {
		return nil, err
	}
	wsID, err := uuid.Parse(req.Msg.WorkspaceId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("invalid workspace_id"))
	}
	raw, err := h.svc.Search(ctx, actor, wsID, req.Msg.Query)
	if err != nil {
		return nil, mapErr(err)
	}
	return connect.NewResponse(&pmsv1.SearchResponse{Results: raw}), nil
}

func (h *queryHandler) RunView(ctx context.Context, req *connect.Request[pmsv1.RunViewRequest]) (*connect.Response[pmsv1.RunViewResponse], error) {
	actor, err := requireActor(ctx)
	if err != nil {
		return nil, err
	}
	wsID, err := uuid.Parse(req.Msg.WorkspaceId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("invalid workspace_id"))
	}
	def := req.Msg.Def
	if def == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("def required"))
	}
	node := protoToFilterNode(def.Filter)
	raw, err := h.svc.RunView(ctx, actor, wsID, def.DataSource, node, 0)
	if err != nil {
		return nil, mapErr(err)
	}
	return connect.NewResponse(&pmsv1.RunViewResponse{GroupedResult: raw}), nil
}

func (h *queryHandler) SemanticSearch(ctx context.Context, req *connect.Request[pmsv1.SemanticSearchRequest]) (*connect.Response[pmsv1.SemanticSearchResponse], error) {
	actor, err := requireActor(ctx)
	if err != nil {
		return nil, err
	}
	wsID, err := uuid.Parse(req.Msg.WorkspaceId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("invalid workspace_id"))
	}
	raw, err := h.svc.SemanticSearch(ctx, actor, wsID, req.Msg.Query, int(req.Msg.Limit))
	if err != nil {
		return nil, mapErr(err)
	}
	return connect.NewResponse(&pmsv1.SemanticSearchResponse{Results: raw}), nil
}

func (h *queryHandler) SimilarIssues(ctx context.Context, req *connect.Request[pmsv1.SimilarIssuesRequest]) (*connect.Response[pmsv1.SimilarIssuesResponse], error) {
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
	raw, err := h.svc.SimilarIssues(ctx, actor, wsID, issueID, int(req.Msg.Limit))
	if err != nil {
		return nil, mapErr(err)
	}
	return connect.NewResponse(&pmsv1.SimilarIssuesResponse{Results: raw}), nil
}

func (h *queryHandler) Suggest(ctx context.Context, req *connect.Request[pmsv1.SuggestRequest]) (*connect.Response[pmsv1.SuggestResponse], error) {
	actor, err := requireActor(ctx)
	if err != nil {
		return nil, err
	}
	wsID, err := uuid.Parse(req.Msg.WorkspaceId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("invalid workspace_id"))
	}
	raw, err := h.svc.Suggest(ctx, actor, wsID, req.Msg.Text, req.Msg.TeamId, int(req.Msg.Limit))
	if err != nil {
		return nil, mapErr(err)
	}
	return connect.NewResponse(&pmsv1.SuggestResponse{Result: raw}), nil
}

func (h *queryHandler) Themes(ctx context.Context, req *connect.Request[pmsv1.ThemesRequest]) (*connect.Response[pmsv1.ThemesResponse], error) {
	actor, err := requireActor(ctx)
	if err != nil {
		return nil, err
	}
	wsID, err := uuid.Parse(req.Msg.WorkspaceId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("invalid workspace_id"))
	}
	raw, err := h.svc.Themes(ctx, actor, wsID)
	if err != nil {
		return nil, mapErr(err)
	}
	return connect.NewResponse(&pmsv1.ThemesResponse{Result: raw}), nil
}

func (h *queryHandler) Anomalies(ctx context.Context, req *connect.Request[pmsv1.AnomaliesRequest]) (*connect.Response[pmsv1.AnomaliesResponse], error) {
	actor, err := requireActor(ctx)
	if err != nil {
		return nil, err
	}
	wsID, err := uuid.Parse(req.Msg.WorkspaceId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("invalid workspace_id"))
	}
	raw, err := h.svc.Anomalies(ctx, actor, wsID, int(req.Msg.Limit))
	if err != nil {
		return nil, mapErr(err)
	}
	return connect.NewResponse(&pmsv1.AnomaliesResponse{Result: raw}), nil
}

func (h *queryHandler) Summarize(ctx context.Context, req *connect.Request[pmsv1.SummarizeRequest]) (*connect.Response[pmsv1.SummarizeResponse], error) {
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
	raw, err := h.svc.Summarize(ctx, actor, wsID, issueID, int(req.Msg.MaxSentences))
	if err != nil {
		return nil, mapErr(err)
	}
	return connect.NewResponse(&pmsv1.SummarizeResponse{Result: raw}), nil
}

func (h *queryHandler) Mood(ctx context.Context, req *connect.Request[pmsv1.MoodRequest]) (*connect.Response[pmsv1.MoodResponse], error) {
	actor, err := requireActor(ctx)
	if err != nil {
		return nil, err
	}
	wsID, err := uuid.Parse(req.Msg.WorkspaceId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("invalid workspace_id"))
	}
	raw, err := h.svc.Mood(ctx, actor, wsID)
	if err != nil {
		return nil, mapErr(err)
	}
	return connect.NewResponse(&pmsv1.MoodResponse{Result: raw}), nil
}

// protoToFilterNode конвертирует protobuf Filter-AST в JSON-форму компилятора (filter.Node).
func protoToFilterNode(n *pmsv1.FilterNode) *filter.Node {
	if n == nil {
		return nil
	}
	switch {
	case n.GetAnd() != nil:
		return &filter.Node{Type: "and", Children: protoChildren(n.GetAnd().GetChildren())}
	case n.GetOr() != nil:
		return &filter.Node{Type: "or", Children: protoChildren(n.GetOr().GetChildren())}
	case n.GetNot() != nil:
		return &filter.Node{Type: "not", Child: protoToFilterNode(n.GetNot())}
	case n.GetPredicate() != nil:
		p := n.GetPredicate()
		return &filter.Node{Type: "predicate", Field: p.GetField().GetBuiltin(), Op: opName(p.GetOp()), Value: filterValueJSON(p.GetValue())}
	}
	return nil
}

func protoChildren(in []*pmsv1.FilterNode) []filter.Node {
	out := make([]filter.Node, 0, len(in))
	for _, c := range in {
		if n := protoToFilterNode(c); n != nil {
			out = append(out, *n)
		}
	}
	return out
}

func opName(op pmsv1.Operator) string {
	switch op {
	case pmsv1.Operator_EQ:
		return "eq"
	case pmsv1.Operator_NEQ:
		return "neq"
	case pmsv1.Operator_IN:
		return "in"
	case pmsv1.Operator_NIN:
		return "nin"
	case pmsv1.Operator_CONTAINS:
		return "contains"
	case pmsv1.Operator_IS_SET:
		return "isSet"
	case pmsv1.Operator_IS_NOT_SET:
		return "isNotSet"
	default:
		return "" // неподдерживаемые (LT/GT/даты) → компилятор трактует как TRUE
	}
}

func filterValueJSON(v *pmsv1.FilterValue) json.RawMessage {
	if v == nil {
		return nil
	}
	var raw []byte
	switch t := v.GetV().(type) {
	case *pmsv1.FilterValue_S:
		raw, _ = json.Marshal(t.S)
	case *pmsv1.FilterValue_N:
		raw, _ = json.Marshal(t.N)
	case *pmsv1.FilterValue_B:
		raw, _ = json.Marshal(t.B)
	case *pmsv1.FilterValue_List:
		raw, _ = json.Marshal(t.List.GetValues())
	case *pmsv1.FilterValue_NullV:
		raw = []byte("null")
	default:
		return nil
	}
	return raw
}
