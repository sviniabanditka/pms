// Package calendar — персональные события календаря (event/dayoff). Видны всем в
// воркспейсе (free/busy), редактирует владелец.
package calendar

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/pms/backend/internal/changelog"
	"github.com/pms/backend/internal/db"
	"github.com/pms/backend/internal/models"
	"github.com/pms/backend/internal/realtime"
	"github.com/uptrace/bun"
)

var (
	ErrDates    = errors.New("starts_at and ends_at required (ends not before starts)")
	ErrNotOwner = errors.New("only the owner can modify this event")
	ErrBadToken = errors.New("unknown feed token")
)

type Service struct {
	db   *bun.DB
	pub  *realtime.Publisher
	jobs *bun.DB // BYPASSRLS-пул для cross-workspace iCal-фида по токену
}

func NewService(bdb *bun.DB, pub *realtime.Publisher, jobs *bun.DB) *Service {
	return &Service{db: bdb, pub: pub, jobs: jobs}
}

func (s *Service) delta(ctx context.Context, wsID, id uuid.UUID, op string) {
	_ = s.pub.PublishDelta(ctx, wsID, realtime.Delta{EntityType: "calendar_event", EntityID: id.String(), Op: op})
}

func norm(kind string) string {
	if kind == "dayoff" {
		return "dayoff"
	}
	return "event"
}

func normRec(r string) string {
	switch r {
	case "daily", "weekly", "monthly":
		return r
	default:
		return "none"
	}
}

func (s *Service) CreateEvent(ctx context.Context, actor, wsID, id uuid.UUID, title, desc, kind string, startsAt, endsAt time.Time, allDay bool, recurrence string, until *time.Time) error {
	if endsAt.Before(startsAt) {
		return ErrDates
	}
	now := time.Now().UTC()
	err := db.WithWorkspace(ctx, s.db, wsID, func(ctx context.Context, tx bun.Tx) error {
		e := &models.CalendarEvent{
			ID: id, WorkspaceID: wsID, UserID: actor, Kind: norm(kind),
			Title: strings.TrimSpace(title), Description: strings.TrimSpace(desc),
			StartsAt: startsAt, EndsAt: endsAt, AllDay: allDay,
			Recurrence: normRec(recurrence), RecurrenceUntil: until,
			CreatedBy: actor, CreatedAt: now, UpdatedAt: now,
		}
		if _, err := tx.NewInsert().Model(e).Exec(ctx); err != nil {
			return err
		}
		return changelog.Record(ctx, tx, wsID, "calendar_event", id, 0, "create", nil, actor)
	})
	if err != nil {
		return err
	}
	s.delta(ctx, wsID, id, "create")
	return nil
}

func (s *Service) UpdateEvent(ctx context.Context, actor, wsID, id uuid.UUID, title, desc, kind string, startsAt, endsAt time.Time, allDay bool, recurrence string, until *time.Time) error {
	if endsAt.Before(startsAt) {
		return ErrDates
	}
	err := db.WithWorkspace(ctx, s.db, wsID, func(ctx context.Context, tx bun.Tx) error {
		var owner uuid.UUID
		if err := tx.NewSelect().Model((*models.CalendarEvent)(nil)).Column("user_id").
			Where("id = ?", id).Where("deleted_at IS NULL").Scan(ctx, &owner); err != nil {
			return err
		}
		if owner != actor {
			return ErrNotOwner
		}
		_, err := tx.NewUpdate().Model((*models.CalendarEvent)(nil)).Where("id = ?", id).
			Set("title = ?", strings.TrimSpace(title)).Set("description = ?", strings.TrimSpace(desc)).
			Set("kind = ?", norm(kind)).Set("starts_at = ?", startsAt).Set("ends_at = ?", endsAt).
			Set("all_day = ?", allDay).Set("recurrence = ?", normRec(recurrence)).Set("recurrence_until = ?", until).
			Set("updated_at = now()").Set("version = version + 1").Exec(ctx)
		if err != nil {
			return err
		}
		return changelog.Record(ctx, tx, wsID, "calendar_event", id, 0, "update", nil, actor)
	})
	if err != nil {
		return err
	}
	s.delta(ctx, wsID, id, "update")
	return nil
}

func (s *Service) DeleteEvent(ctx context.Context, actor, wsID, id uuid.UUID) error {
	err := db.WithWorkspace(ctx, s.db, wsID, func(ctx context.Context, tx bun.Tx) error {
		var owner uuid.UUID
		if err := tx.NewSelect().Model((*models.CalendarEvent)(nil)).Column("user_id").
			Where("id = ?", id).Where("deleted_at IS NULL").Scan(ctx, &owner); err != nil {
			return err
		}
		if owner != actor {
			return ErrNotOwner
		}
		_, err := tx.NewUpdate().Model((*models.CalendarEvent)(nil)).Where("id = ?", id).
			Set("deleted_at = now()").Exec(ctx)
		if err != nil {
			return err
		}
		return changelog.Record(ctx, tx, wsID, "calendar_event", id, 0, "delete", nil, actor)
	})
	if err != nil {
		return err
	}
	s.delta(ctx, wsID, id, "delete")
	return nil
}

// --- iCal feed ---

func newToken() string {
	b := make([]byte, 24)
	_, _ = rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)
}

// EnsureFeedToken возвращает (создаёт при отсутствии) персональный токен iCal-фида.
func (s *Service) EnsureFeedToken(ctx context.Context, userID uuid.UUID) (string, error) {
	var tok *string
	if err := s.jobs.NewSelect().Model((*models.User)(nil)).Column("ical_token").Where("id = ?", userID).Scan(ctx, &tok); err != nil {
		return "", err
	}
	if tok != nil && *tok != "" {
		return *tok, nil
	}
	return s.RotateFeedToken(ctx, userID)
}

// RotateFeedToken выдаёт новый токен (старые ссылки перестают работать).
func (s *Service) RotateFeedToken(ctx context.Context, userID uuid.UUID) (string, error) {
	t := newToken()
	if _, err := s.jobs.NewUpdate().Model((*models.User)(nil)).Set("ical_token = ?", t).Where("id = ?", userID).Exec(ctx); err != nil {
		return "", err
	}
	return t, nil
}

// Feed строит .ics для владельца токена: его события + назначенные ему дедлайны задач + митинги-участником.
// Cross-workspace (jobs-пул, BYPASSRLS) — токен сам по себе и есть авторизация.
func (s *Service) Feed(ctx context.Context, token string) (string, error) {
	if token == "" {
		return "", ErrBadToken
	}
	var u models.User
	if err := s.jobs.NewSelect().Model(&u).Where("ical_token = ?", token).Where("deleted_at IS NULL").Limit(1).Scan(ctx); err != nil {
		return "", ErrBadToken
	}

	var evs []models.CalendarEvent
	if err := s.jobs.NewSelect().Model(&evs).Where("user_id = ?", u.ID).Where("deleted_at IS NULL").Scan(ctx); err != nil {
		return "", err
	}

	var iss []models.Issue
	if err := s.jobs.NewSelect().Model(&iss).
		Where("assignee_id = ?", u.ID).Where("archived_at IS NULL").Where("deleted_at IS NULL").
		Where("due_date IS NOT NULL OR scheduled_date IS NOT NULL").Scan(ctx); err != nil {
		return "", err
	}

	var mts []models.Meeting
	if err := s.jobs.NewSelect().Model(&mts).
		Join("JOIN meeting_participants mp ON mp.meeting_id = mt.id").
		Where("mp.user_id = ?", u.ID).Where("mt.scheduled_at IS NOT NULL").
		Where("mt.deleted_at IS NULL").Where("mt.status != 'ended'").Scan(ctx); err != nil {
		return "", err
	}

	var b strings.Builder
	b.WriteString("BEGIN:VCALENDAR\r\nVERSION:2.0\r\nPRODID:-//PMS//Calendar//EN\r\nCALSCALE:GREGORIAN\r\nMETHOD:PUBLISH\r\n")
	name := u.Name
	if name == "" {
		name = u.Email
	}
	b.WriteString("X-WR-CALNAME:" + icsEsc("PMS — "+name) + "\r\n")
	stamp := time.Now().UTC().Format("20060102T150405Z")

	for _, e := range evs {
		writeEvent(&b, "ev-"+e.ID.String(), stamp, e.Title, e.Description, e.StartsAt, e.EndsAt, e.AllDay, rrule(e.Recurrence, e.RecurrenceUntil))
	}
	for _, i := range iss {
		d := i.DueDate
		if i.ScheduledDate != nil {
			d = i.ScheduledDate
		}
		if d == nil {
			continue
		}
		writeEvent(&b, "iss-"+i.ID.String(), stamp, "["+i.Identifier+"] "+i.Title, "", *d, *d, true, "")
	}
	for _, m := range mts {
		end := m.ScheduledAt.Add(time.Hour)
		if m.EndsAt != nil {
			end = *m.EndsAt
		}
		writeEvent(&b, "mtg-"+m.ID.String(), stamp, "📹 "+nonEmpty(m.Title, "Meeting"), "", *m.ScheduledAt, end, false, "")
	}

	b.WriteString("END:VCALENDAR\r\n")
	return b.String(), nil
}

func nonEmpty(s, def string) string {
	if strings.TrimSpace(s) == "" {
		return def
	}
	return s
}

// writeEvent — один VEVENT. all_day → DATE-форма (DTEND = следующий день, эксклюзивно).
// ponytail: без line-folding (>75 октетов) — современные клиенты (Google/Apple) терпят; добавить при жалобах.
// rrule → строка RRULE для VEVENT (пусто если без повтора).
func rrule(rec string, until *time.Time) string {
	var freq string
	switch rec {
	case "daily":
		freq = "DAILY"
	case "weekly":
		freq = "WEEKLY"
	case "monthly":
		freq = "MONTHLY"
	default:
		return ""
	}
	r := "FREQ=" + freq
	if until != nil {
		r += ";UNTIL=" + until.UTC().Format("20060102T150405Z")
	}
	return r
}

func writeEvent(b *strings.Builder, uid, stamp, summary, desc string, start, end time.Time, allDay bool, rruleStr string) {
	b.WriteString("BEGIN:VEVENT\r\n")
	b.WriteString("UID:" + uid + "@pms\r\n")
	b.WriteString("DTSTAMP:" + stamp + "\r\n")
	if rruleStr != "" {
		b.WriteString("RRULE:" + rruleStr + "\r\n")
	}
	if allDay {
		b.WriteString("DTSTART;VALUE=DATE:" + start.UTC().Format("20060102") + "\r\n")
		b.WriteString("DTEND;VALUE=DATE:" + end.UTC().AddDate(0, 0, 1).Format("20060102") + "\r\n")
	} else {
		b.WriteString("DTSTART:" + start.UTC().Format("20060102T150405Z") + "\r\n")
		b.WriteString("DTEND:" + end.UTC().Format("20060102T150405Z") + "\r\n")
	}
	b.WriteString("SUMMARY:" + icsEsc(summary) + "\r\n")
	if strings.TrimSpace(desc) != "" {
		b.WriteString("DESCRIPTION:" + icsEsc(desc) + "\r\n")
	}
	b.WriteString("END:VEVENT\r\n")
}

// icsEsc экранирует спецсимволы RFC 5545 (TEXT): \ ; , и переводы строк.
func icsEsc(s string) string {
	r := strings.NewReplacer("\\", "\\\\", ";", "\\;", ",", "\\,", "\n", "\\n", "\r", "")
	return r.Replace(s)
}
