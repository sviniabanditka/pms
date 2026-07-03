package rpc

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"time"

	"connectrpc.com/connect"
	"github.com/google/uuid"

	"github.com/pms/backend/internal/auth"
	"github.com/pms/backend/internal/authz"
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
	syncsvc "github.com/pms/backend/internal/domain/sync"
	"github.com/pms/backend/internal/domain/teams"
	"github.com/pms/backend/internal/domain/timetracking"
	"github.com/pms/backend/internal/domain/views"
	"github.com/pms/backend/internal/domain/workspaces"
)

func parseDate(s string) (time.Time, error) { return time.Parse("2006-01-02", s) }

// --- helpers ---

// issueRef извлекает actor + парсит workspace/issue id.
func issueRef(ctx context.Context, wsRaw, idRaw string) (uuid.UUID, uuid.UUID, uuid.UUID, error) {
	actor, err := requireActor(ctx)
	if err != nil {
		return uuid.Nil, uuid.Nil, uuid.Nil, err
	}
	wsID, err := uuid.Parse(wsRaw)
	if err != nil {
		return uuid.Nil, uuid.Nil, uuid.Nil, connect.NewError(connect.CodeInvalidArgument, errors.New("invalid workspace_id"))
	}
	id, err := uuid.Parse(idRaw)
	if err != nil {
		return uuid.Nil, uuid.Nil, uuid.Nil, connect.NewError(connect.CodeInvalidArgument, errors.New("invalid id"))
	}
	return actor, wsID, id, nil
}

// optUUID: пустая строка → nil, иначе парсит (невалидное → nil, мягко).
// optUUID: "" → nil (осознанное «снять значение»). Невалидный uuid — ошибка,
// а не тихий nil: иначе кривой assignee_id молча РАЗ-назначает issue.
func optUUID(s string) (*uuid.UUID, error) {
	if s == "" {
		return nil, nil
	}
	u, err := uuid.Parse(s)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("invalid uuid: "+s))
	}
	return &u, nil
}

func requireActor(ctx context.Context) (uuid.UUID, error) {
	uid, ok := auth.UserID(ctx)
	if !ok {
		return uuid.Nil, connect.NewError(connect.CodeUnauthenticated, errors.New("authentication required"))
	}
	return uid, nil
}

// parseOrNew принимает client-generated UUID; пустой → генерим UUIDv7 (удобно для ручных вызовов).
func parseOrNew(s string) (uuid.UUID, error) {
	if s == "" {
		return uuid.NewV7()
	}
	return uuid.Parse(s)
}

func mapErr(err error) error {
	switch {
	case errors.Is(err, workspaces.ErrInvalidSlug),
		errors.Is(err, workspaces.ErrInvalidName),
		errors.Is(err, teams.ErrInvalidKey),
		errors.Is(err, teams.ErrInvalidName),
		errors.Is(err, teams.ErrParentNotFound),
		errors.Is(err, teams.ErrCycle),
		errors.Is(err, teams.ErrTeamNotFound),
		errors.Is(err, issues.ErrTitleRequired),
		errors.Is(err, issues.ErrTeamNotFound),
		errors.Is(err, issues.ErrStateNotFound),
		errors.Is(err, issues.ErrIssueNotFound),
		errors.Is(err, issues.ErrInvalidRelation),
		errors.Is(err, views.ErrNameRequired),
		errors.Is(err, cycles.ErrDates),
		errors.Is(err, customfields.ErrName),
		errors.Is(err, customfields.ErrType),
		errors.Is(err, dashboards.ErrName),
		errors.Is(err, dashboards.ErrWidgetType),
		errors.Is(err, docs.ErrTitle),
		errors.Is(err, chats.ErrEmpty),
		errors.Is(err, comments.ErrEmpty),
		errors.Is(err, milestones.ErrName),
		errors.Is(err, calendar.ErrDates),
		errors.Is(err, timetracking.ErrDates),
		errors.Is(err, automations.ErrName),
		errors.Is(err, workspaces.ErrInviteEmail),
		errors.Is(err, workspaces.ErrInviteExpired),
		errors.Is(err, workspaces.ErrInvalidRole),
		errors.Is(err, workspaces.ErrEmail):
		return connect.NewError(connect.CodeInvalidArgument, err)
	case errors.Is(err, workspaces.ErrInviteNotFound):
		return connect.NewError(connect.CodeNotFound, err)
	case errors.Is(err, workspaces.ErrSlugTaken),
		errors.Is(err, teams.ErrKeyTaken):
		return connect.NewError(connect.CodeAlreadyExists, err)
	case errors.Is(err, authz.ErrForbidden),
		errors.Is(err, syncsvc.ErrForbidden),
		errors.Is(err, chats.ErrForbidden),
		errors.Is(err, comments.ErrForbidden),
		errors.Is(err, workspaces.ErrForbidden),
		errors.Is(err, meetings.ErrForbidden),
		errors.Is(err, attachments.ErrForbidden),
		errors.Is(err, calendar.ErrNotOwner):
		return connect.NewError(connect.CodePermissionDenied, err)
	case errors.Is(err, teams.ErrStateInUse),
		errors.Is(err, teams.ErrTeamHasIssues),
		errors.Is(err, workspaces.ErrLastOwner):
		return connect.NewError(connect.CodeFailedPrecondition, err)
	default:
		// не отдаём наружу сырые pg/bun-сообщения (имена таблиц, констрейнты, SQL)
		slog.Error("rpc: internal error", "err", err)
		return connect.NewError(connect.CodeInternal, errors.New("internal error"))
	}
}

// parseTS: ISO datetime → time; пустое/невалидное → zero (хендлер решает).
func parseTS(s string) time.Time {
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t.UTC()
	}
	return time.Time{}
}

// jsonRaw: пустая строка → nil, иначе json.RawMessage.
func jsonRaw(s string) json.RawMessage {
	if s == "" {
		return nil
	}
	return json.RawMessage(s)
}
