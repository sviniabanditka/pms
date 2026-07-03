package rpc

import (
	"net/http"

	"connectrpc.com/connect"
	"github.com/go-chi/chi/v5"

	"github.com/pms/backend/gen/pms/v1/pmsv1connect"
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
	"github.com/pms/backend/internal/domain/templates"
	"github.com/pms/backend/internal/domain/timetracking"
	"github.com/pms/backend/internal/domain/views"
	"github.com/pms/backend/internal/domain/webhooks"
	"github.com/pms/backend/internal/domain/workspaces"
)

// Services — доменные зависимости connect-хендлеров.
type Services struct {
	Workspaces    *workspaces.Service
	Teams         *teams.Service
	Issues        *issues.Service
	Views         *views.Service
	Cycles        *cycles.Service
	CustomFields  *customfields.Service
	Dashboards    *dashboards.Service
	Notifications *notifications.Service
	Docs          *docs.Service
	Chats         *chats.Service
	Comments      *comments.Service
	Milestones    *milestones.Service
	Meetings      *meetings.Service
	Calendar      *calendar.Service
	Time          *timetracking.Service
	Automations   *automations.Service
	Attachments   *attachments.Service
	Search        *search.Service
	Sync          *syncsvc.Service
	ApiTokens     *apitokens.Service
	Webhooks      *webhooks.Service
	Templates     *templates.Service
}

// Mount монтирует connect-сервисы. Все unary-RPC проходят через membership-гейт
// (newWorkspaceAuthInterceptor): запрос с workspace_id → проверка членства + роль в ctx.
func Mount(r chi.Router, svcs Services) {
	wsAuth := connect.WithInterceptors(newWorkspaceAuthInterceptor(svcs.Workspaces.RoleOf))
	mount := func(p string, h http.Handler) { r.Mount(p, h) }
	mount(pmsv1connect.NewWorkspaceServiceHandler(&workspaceHandler{svc: svcs.Workspaces}, wsAuth))
	mount(pmsv1connect.NewTeamServiceHandler(&teamHandler{svc: svcs.Teams}, wsAuth))
	mount(pmsv1connect.NewIssueServiceHandler(&issueHandler{svc: svcs.Issues}, wsAuth))
	mount(pmsv1connect.NewViewServiceHandler(&viewHandler{svc: svcs.Views}, wsAuth))
	mount(pmsv1connect.NewCycleServiceHandler(&cycleHandler{svc: svcs.Cycles}, wsAuth))
	mount(pmsv1connect.NewCustomFieldServiceHandler(&customFieldHandler{svc: svcs.CustomFields}, wsAuth))
	mount(pmsv1connect.NewDashboardServiceHandler(&dashboardHandler{svc: svcs.Dashboards}, wsAuth))
	mount(pmsv1connect.NewNotificationServiceHandler(&notificationHandler{svc: svcs.Notifications}, wsAuth))
	mount(pmsv1connect.NewDocServiceHandler(&docHandler{svc: svcs.Docs}, wsAuth))
	mount(pmsv1connect.NewChatServiceHandler(&chatHandler{svc: svcs.Chats}, wsAuth))
	mount(pmsv1connect.NewCommentServiceHandler(&commentHandler{svc: svcs.Comments}, wsAuth))
	mount(pmsv1connect.NewMilestoneServiceHandler(&milestoneHandler{svc: svcs.Milestones}, wsAuth))
	mount(pmsv1connect.NewMeetingServiceHandler(&meetingHandler{svc: svcs.Meetings}, wsAuth))
	mount(pmsv1connect.NewCalendarServiceHandler(&calendarHandler{svc: svcs.Calendar}, wsAuth))
	mount(pmsv1connect.NewTimeServiceHandler(&timeHandler{svc: svcs.Time}, wsAuth))
	mount(pmsv1connect.NewAutomationServiceHandler(&automationHandler{svc: svcs.Automations}, wsAuth))
	mount(pmsv1connect.NewAttachmentServiceHandler(&attachmentHandler{svc: svcs.Attachments}, wsAuth))
	mount(pmsv1connect.NewSyncServiceHandler(&syncHandler{svc: svcs.Sync}, wsAuth))
	mount(pmsv1connect.NewQueryServiceHandler(&queryHandler{svc: svcs.Search}, wsAuth))
	mount(pmsv1connect.NewTemplateServiceHandler(&templateHandler{svc: svcs.Templates}, wsAuth))
}
