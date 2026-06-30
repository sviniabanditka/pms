package dashboards

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/pms/backend/internal/authz"
	"github.com/pms/backend/internal/changelog"
	"github.com/pms/backend/internal/db"
	"github.com/pms/backend/internal/models"
	"github.com/pms/backend/internal/realtime"
	"github.com/uptrace/bun"
)

var (
	ErrName       = errors.New("dashboard name required")
	ErrWidgetType = errors.New("invalid widget type")
)

var widgetTypes = map[string]bool{"view": true, "stat": true, "chart": true, "note": true}

type Service struct {
	db  *bun.DB
	pub *realtime.Publisher
}

func NewService(bdb *bun.DB, pub *realtime.Publisher) *Service { return &Service{db: bdb, pub: pub} }

func (s *Service) CreateDashboard(ctx context.Context, actor, wsID, id uuid.UUID, name string) (int64, error) {
	if strings.TrimSpace(name) == "" {
		return 0, ErrName
	}
	now := time.Now().UTC()
	err := db.WithWorkspace(ctx, s.db, wsID, func(ctx context.Context, tx bun.Tx) error {
		dsh := &models.Dashboard{ID: id, WorkspaceID: wsID, Name: strings.TrimSpace(name), ScopeKind: "workspace", OwnerID: &actor, CreatedAt: now, UpdatedAt: now}
		if _, err := tx.NewInsert().Model(dsh).Exec(ctx); err != nil {
			return err
		}
		return changelog.Record(ctx, tx, wsID, "dashboard", id, 0, "create", map[string]any{"name": name}, actor)
	})
	if err != nil {
		return 0, err
	}
	_ = s.pub.PublishDelta(ctx, wsID, realtime.Delta{EntityType: "dashboard", EntityID: id.String(), Op: "create"})
	return 0, nil
}

// checkDashOwner — мутировать дашборд/виджет может владелец дашборда или админ.
func checkDashOwner(ctx context.Context, owner *uuid.UUID, actor uuid.UUID) error {
	o := uuid.Nil
	if owner != nil {
		o = *owner
	}
	if !authz.OwnerOrAdmin(ctx, o, actor) {
		return authz.ErrForbidden
	}
	return nil
}

func requireDashboardOwner(ctx context.Context, tx bun.Tx, dashboardID, actor uuid.UUID) error {
	var owner *uuid.UUID
	if err := tx.NewSelect().Model((*models.Dashboard)(nil)).Column("owner_id").Where("id = ?", dashboardID).Scan(ctx, &owner); err != nil {
		return err
	}
	return checkDashOwner(ctx, owner, actor)
}

func requireWidgetOwner(ctx context.Context, tx bun.Tx, widgetID, actor uuid.UUID) error {
	var owner *uuid.UUID
	if err := tx.NewSelect().TableExpr("dashboard_widgets AS dw").
		Join("JOIN dashboards AS d ON d.id = dw.dashboard_id").
		Column("d.owner_id").Where("dw.id = ?", widgetID).Scan(ctx, &owner); err != nil {
		return err
	}
	return checkDashOwner(ctx, owner, actor)
}

func (s *Service) AddWidget(ctx context.Context, actor, wsID, id, dashboardID uuid.UUID, typ, title string, config []byte) (int64, error) {
	if !widgetTypes[typ] {
		return 0, ErrWidgetType
	}
	if len(config) == 0 {
		config = []byte("{}")
	}
	now := time.Now().UTC()
	err := db.WithWorkspace(ctx, s.db, wsID, func(ctx context.Context, tx bun.Tx) error {
		if err := requireDashboardOwner(ctx, tx, dashboardID, actor); err != nil {
			return err
		}
		w := &models.DashboardWidget{ID: id, WorkspaceID: wsID, DashboardID: dashboardID, Type: typ, Config: config, W: 1, H: 1, CreatedAt: now, UpdatedAt: now}
		if title != "" {
			w.Title = &title
		}
		if _, err := tx.NewInsert().Model(w).Exec(ctx); err != nil {
			return err
		}
		return changelog.Record(ctx, tx, wsID, "dashboard_widget", id, 0, "create", nil, actor)
	})
	if err != nil {
		return 0, err
	}
	_ = s.pub.PublishDelta(ctx, wsID, realtime.Delta{EntityType: "dashboard_widget", EntityID: id.String(), Op: "create"})
	return 0, nil
}

func (s *Service) UpdateWidget(ctx context.Context, actor, wsID, id uuid.UUID, title *string, config []byte, x, y, w, h *int32) (int64, error) {
	err := db.WithWorkspace(ctx, s.db, wsID, func(ctx context.Context, tx bun.Tx) error {
		if err := requireWidgetOwner(ctx, tx, id, actor); err != nil {
			return err
		}
		q := tx.NewUpdate().Model((*models.DashboardWidget)(nil)).Where("id = ?", id).
			Set("updated_at = now()").Set("version = version + 1")
		if title != nil {
			q.Set("title = ?", *title)
		}
		if config != nil {
			q.Set("config = ?", config)
		}
		if x != nil {
			q.Set("x = ?", *x)
		}
		if y != nil {
			q.Set("y = ?", *y)
		}
		if w != nil {
			q.Set("w = ?", *w)
		}
		if h != nil {
			q.Set("h = ?", *h)
		}
		if _, err := q.Exec(ctx); err != nil {
			return err
		}
		return changelog.Record(ctx, tx, wsID, "dashboard_widget", id, 0, "update", nil, actor)
	})
	if err != nil {
		return 0, err
	}
	_ = s.pub.PublishDelta(ctx, wsID, realtime.Delta{EntityType: "dashboard_widget", EntityID: id.String(), Op: "update"})
	return 0, nil
}

func (s *Service) RemoveWidget(ctx context.Context, actor, wsID, id uuid.UUID) (int64, error) {
	err := db.WithWorkspace(ctx, s.db, wsID, func(ctx context.Context, tx bun.Tx) error {
		if err := requireWidgetOwner(ctx, tx, id, actor); err != nil {
			return err
		}
		if _, err := tx.NewDelete().Model((*models.DashboardWidget)(nil)).Where("id = ?", id).Exec(ctx); err != nil {
			return err
		}
		return changelog.Record(ctx, tx, wsID, "dashboard_widget", id, 0, "delete", nil, actor)
	})
	if err != nil {
		return 0, err
	}
	_ = s.pub.PublishDelta(ctx, wsID, realtime.Delta{EntityType: "dashboard_widget", EntityID: id.String(), Op: "delete"})
	return 0, nil
}
