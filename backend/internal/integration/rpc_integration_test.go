//go:build integration

package integration

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	pmsv1 "github.com/pms/backend/gen/pms/v1"
	"github.com/pms/backend/gen/pms/v1/pmsv1connect"
	"github.com/pms/backend/internal/auth"
	"github.com/pms/backend/internal/domain/apitokens"
	"github.com/pms/backend/internal/domain/attachments"
	"github.com/pms/backend/internal/domain/automations"
	"github.com/pms/backend/internal/domain/calendar"
	"github.com/pms/backend/internal/domain/chats"
	"github.com/pms/backend/internal/domain/comments"
	"github.com/pms/backend/internal/domain/customfields"
	"github.com/pms/backend/internal/domain/cycles"
	"github.com/pms/backend/internal/domain/dashboards"
	"github.com/pms/backend/internal/domain/docs"
	"github.com/pms/backend/internal/domain/issues"
	"github.com/pms/backend/internal/domain/meetings"
	"github.com/pms/backend/internal/domain/milestones"
	"github.com/pms/backend/internal/domain/notifications"
	"github.com/pms/backend/internal/domain/search"
	syncsvc "github.com/pms/backend/internal/domain/sync"
	"github.com/pms/backend/internal/domain/teams"
	"github.com/pms/backend/internal/domain/timetracking"
	"github.com/pms/backend/internal/domain/views"
	"github.com/pms/backend/internal/domain/workspaces"
	"github.com/pms/backend/internal/rpc"
)

func buildServices() rpc.Services {
	return rpc.Services{
		Workspaces:    workspaces.NewService(appDB, pub, jobsDB),
		Teams:         teams.NewService(appDB, pub),
		Issues:        issues.NewService(appDB, pub),
		Views:         views.NewService(appDB, pub),
		Cycles:        cycles.NewService(appDB, pub),
		CustomFields:  customfields.NewService(appDB, pub),
		Dashboards:    dashboards.NewService(appDB, pub),
		Notifications: notifications.NewService(appDB, pub),
		Docs:          docs.NewService(appDB, pub),
		Chats:         chats.NewService(appDB, pub),
		Comments:      comments.NewService(appDB, pub),
		Milestones:    milestones.NewService(appDB, pub),
		Meetings:      meetings.NewService(appDB, pub, meetings.LiveKit{}),
		Calendar:      calendar.NewService(appDB, pub, jobsDB),
		Time:          timetracking.NewService(appDB, pub),
		Automations:   automations.NewService(appDB, pub),
		Attachments:   attachments.NewService(appDB, pub, nil),
		Search:        search.NewService(appDB),
		Sync:          syncsvc.NewService(appDB),
	}
}

// rpcServer поднимает реальный connect-стек (rpc.Mount c membership-интерсептором).
// X-Test-User-заголовок инжектит аутентифицированного юзера (вместо cookie/session).
func rpcServer(t *testing.T) *httptest.Server {
	t.Helper()
	r := chi.NewRouter()
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			if h := req.Header.Get("X-Test-User"); h != "" {
				if uid, err := uuid.Parse(h); err == nil {
					req = req.WithContext(auth.WithUserID(req.Context(), uid))
				}
			}
			next.ServeHTTP(w, req)
		})
	})
	rpc.Mount(r, buildServices())
	srv := httptest.NewServer(r)
	t.Cleanup(srv.Close)
	return srv
}

// mkWS — воркспейс с owner-членом (created_by FK).
func mkWS(t *testing.T) uuid.UUID {
	t.Helper()
	ws, owner := uuid.New(), newUser(t)
	mustExec(t, `INSERT INTO workspaces (id, slug, name, created_by) VALUES (?, ?, 'ws', ?)`, ws, "ws-"+ws.String()[:8], owner)
	mustExec(t, `INSERT INTO workspace_members (id, workspace_id, user_id, role) VALUES (?, ?, ?, 'owner')`, uuid.New(), ws, owner)
	return ws
}

func addMember(t *testing.T, ws uuid.UUID, role string) uuid.UUID {
	t.Helper()
	u := newUser(t)
	mustExec(t, `INSERT INTO workspace_members (id, workspace_id, user_id, role) VALUES (?, ?, ?, ?)`, uuid.New(), ws, u, role)
	return u
}

func TestRPC_MembershipInterceptor(t *testing.T) {
	srv := rpcServer(t)
	wsA := mkWS(t)
	wsB := mkWS(t)
	memberU := addMember(t, wsA, "member")
	guestU := addMember(t, wsA, "guest")

	qc := pmsv1connect.NewQueryServiceClient(srv.Client(), srv.URL)
	ic := pmsv1connect.NewIssueServiceClient(srv.Client(), srv.URL)
	ac := pmsv1connect.NewAutomationServiceClient(srv.Client(), srv.URL)

	withUser := func(req connect.AnyRequest, u uuid.UUID) {
		if u != uuid.Nil {
			req.Header().Set("X-Test-User", u.String())
		}
	}

	t.Run("member reads own workspace → ok", func(t *testing.T) {
		req := connect.NewRequest(&pmsv1.SearchRequest{WorkspaceId: wsA.String(), Query: "x"})
		withUser(req, memberU)
		if _, err := qc.Search(context.Background(), req); err != nil {
			t.Fatalf("own-ws search must succeed, got %v", err)
		}
	})

	t.Run("cross-tenant: member with another workspace id → permission_denied", func(t *testing.T) {
		req := connect.NewRequest(&pmsv1.SearchRequest{WorkspaceId: wsB.String(), Query: "x"})
		withUser(req, memberU)
		_, err := qc.Search(context.Background(), req)
		if connect.CodeOf(err) != connect.CodePermissionDenied {
			t.Fatalf("foreign-ws search must be PermissionDenied, got %v (code %v)", err, connect.CodeOf(err))
		}
	})

	t.Run("unauthenticated → unauthenticated", func(t *testing.T) {
		req := connect.NewRequest(&pmsv1.SearchRequest{WorkspaceId: wsA.String(), Query: "x"})
		_, err := qc.Search(context.Background(), req)
		if connect.CodeOf(err) != connect.CodeUnauthenticated {
			t.Fatalf("no auth must be Unauthenticated, got %v (code %v)", err, connect.CodeOf(err))
		}
	})

	t.Run("guest write → permission_denied (read-only)", func(t *testing.T) {
		req := connect.NewRequest(&pmsv1.CreateIssueRequest{WorkspaceId: wsA.String(), TeamId: uuid.New().String(), Title: "x"})
		withUser(req, guestU)
		_, err := ic.CreateIssue(context.Background(), req)
		if connect.CodeOf(err) != connect.CodePermissionDenied {
			t.Fatalf("guest write must be PermissionDenied, got %v (code %v)", err, connect.CodeOf(err))
		}
	})

	t.Run("member admin-only procedure → permission_denied (requires admin)", func(t *testing.T) {
		req := connect.NewRequest(&pmsv1.CreateAutomationRequest{WorkspaceId: wsA.String(), Name: "rule"})
		withUser(req, memberU)
		_, err := ac.CreateAutomation(context.Background(), req)
		if connect.CodeOf(err) != connect.CodePermissionDenied {
			t.Fatalf("member admin-proc must be PermissionDenied, got %v (code %v)", err, connect.CodeOf(err))
		}
	})
}

// rpcServerReal монтирует НАСТОЯЩИЙ auth.Authenticator (резолвит Bearer pms_-токены).
// sessions с nil-redis: для pms_-токенов ветка идёт в resolveAPIToken (db), redis не трогается.
func rpcServerReal(t *testing.T) *httptest.Server {
	t.Helper()
	authSvc := auth.NewService(appDB, auth.NewSessionStore(nil, time.Hour), false)
	r := chi.NewRouter()
	r.Use(authSvc.Authenticator)
	rpc.Mount(r, buildServices())
	srv := httptest.NewServer(r)
	t.Cleanup(srv.Close)
	return srv
}

func TestRPC_ApiTokenReadOnly(t *testing.T) {
	srv := rpcServerReal(t)
	ws, owner := newWS(t)
	member := addMember(t, ws, "member")
	team := makeTeam(t, ws, owner, "RW") // public team → member имеет доступ
	ats := apitokens.NewService(appDB)
	rwTok, err := ats.Create(context.Background(), member, ws, uuid.New(), "rw", false, nil)
	if err != nil {
		t.Fatalf("create rw: %v", err)
	}
	roTok, err := ats.Create(context.Background(), member, ws, uuid.New(), "ro", true, nil)
	if err != nil {
		t.Fatalf("create ro: %v", err)
	}

	qc := pmsv1connect.NewQueryServiceClient(srv.Client(), srv.URL)
	ic := pmsv1connect.NewIssueServiceClient(srv.Client(), srv.URL)
	bearer := func(req connect.AnyRequest, tok string) { req.Header().Set("Authorization", "Bearer "+tok) }

	t.Run("read-only token can read", func(t *testing.T) {
		req := connect.NewRequest(&pmsv1.SearchRequest{WorkspaceId: ws.String(), Query: "x"})
		bearer(req, roTok)
		if _, err := qc.Search(context.Background(), req); err != nil {
			t.Fatalf("ro read must work, got %v", err)
		}
	})
	t.Run("read-only token cannot write", func(t *testing.T) {
		req := connect.NewRequest(&pmsv1.CreateIssueRequest{WorkspaceId: ws.String(), TeamId: team.String(), Title: "x"})
		bearer(req, roTok)
		_, err := ic.CreateIssue(context.Background(), req)
		if connect.CodeOf(err) != connect.CodePermissionDenied {
			t.Fatalf("ro write must be PermissionDenied, got %v (%v)", err, connect.CodeOf(err))
		}
	})
	t.Run("read-write token can write", func(t *testing.T) {
		req := connect.NewRequest(&pmsv1.CreateIssueRequest{WorkspaceId: ws.String(), TeamId: team.String(), Title: "ok"})
		bearer(req, rwTok)
		if _, err := ic.CreateIssue(context.Background(), req); err != nil {
			t.Fatalf("rw write must pass, got %v", err)
		}
	})
}
